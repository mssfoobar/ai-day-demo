// Package aohotel is the thin AOH OpenTelemetry "distribution" for Go
// services. It wires the OTEL TracerProvider, MeterProvider and LoggerProvider
// from OTEL_* environment variables plus a small Config, sets the global
// providers and the W3C propagators, and returns a single shutdown func.
//
// It deliberately stays a config layer: services create spans with the plain
// OTEL API. Backend selection is entirely the collector's concern — services
// only ever point OTEL_EXPORTER_OTLP_ENDPOINT at the local gateway collector.
//
// If OTEL_EXPORTER_OTLP_ENDPOINT is unset, Init installs only the W3C
// propagators (so trace context still flows across hops) and returns a no-op
// shutdown — telemetry export is off, with no code change required (AOH-7540).
package aohotel

import (
	"context"
	"errors"
	"os"

	"github.com/google/uuid"
	aohlog "github.com/mssfoobar/ops-hub/packages/aoh-golib/logger"
	runtimemetrics "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otellogglobal "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap/zapcore"
)

// ServiceNamespace is the fixed OTEL service.namespace for all AOH services.
const ServiceNamespace = "aoh"

// Config carries the per-service identity. Everything else (exporter endpoint,
// headers, protocol, sampling, extra resource attributes via
// OTEL_RESOURCE_ATTRIBUTES — including deployment.environment.name) is read
// from the standard OTEL_* environment variables.
type Config struct {
	// ServiceName sets resource service.name. Falls back to OTEL_SERVICE_NAME,
	// then "unknown-service". Use the AOH convention: "<module>-service" or
	// "web-<app>".
	ServiceName string
	// ServiceVersion sets resource service.version (optional).
	ServiceVersion string
}

// Init configures global OTEL providers + propagators and returns a shutdown
// func that flushes and closes them. Call shutdown on process exit.
func Init(ctx context.Context, cfg Config) (shutdown func(context.Context) error, err error) {
	// Always install W3C trace-context + baggage propagators so context flows
	// across services even when exporting is disabled.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		// Export disabled: propagation only, no providers, no-op shutdown.
		return func(context.Context) error { return nil }, nil
	}

	res, err := newResource(ctx, cfg)
	if err != nil {
		return nil, err
	}

	var shutdownFns []func(context.Context) error
	shutdown = func(ctx context.Context) error {
		var errs error
		for _, fn := range shutdownFns {
			errs = errors.Join(errs, fn(ctx))
		}
		shutdownFns = nil
		return errs
	}
	// On partial-setup failure, flush whatever was created, then return.
	bail := func(e error) (func(context.Context) error, error) {
		return nil, errors.Join(e, shutdown(ctx))
	}

	// Traces. The gRPC exporter auto-configures from OTEL_EXPORTER_OTLP_*.
	traceExp, err := otlptracegrpc.New(ctx)
	if err != nil {
		return bail(err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(traceExp),
	)
	shutdownFns = append(shutdownFns, tp.Shutdown)
	otel.SetTracerProvider(tp)

	// Metrics.
	metricExp, err := otlpmetricgrpc.New(ctx)
	if err != nil {
		return bail(err)
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp)),
	)
	shutdownFns = append(shutdownFns, mp.Shutdown)
	otel.SetMeterProvider(mp)

	// Go runtime metrics (GC pauses, heap/memory, goroutine count) on the same
	// meter provider — the "runtime metrics" half of phase-1 observability,
	// alongside the RED/HTTP metrics from otelhttp.
	if err := runtimemetrics.Start(runtimemetrics.WithMeterProvider(mp)); err != nil {
		return bail(err)
	}

	// Logs. Honours the standard OTEL_LOGS_EXPORTER: "none" disables log export
	// entirely (no provider, no bridge); anything else uses OTLP. Using the
	// spec'd variable rather than an AOH-specific flag keeps the off-switch
	// discoverable to anyone who knows OTEL.
	if os.Getenv("OTEL_LOGS_EXPORTER") != "none" {
		logExp, err := otlploggrpc.New(ctx)
		if err != nil {
			return bail(err)
		}
		lp := sdklog.NewLoggerProvider(
			sdklog.WithResource(res),
			sdklog.WithProcessor(sdklog.NewBatchProcessor(logExp)),
		)
		otellogglobal.SetLoggerProvider(lp)

		// Bridge aohlog -> OTLP. Until this line existed, building the
		// LoggerProvider above was INERT: ZapCore had zero callers repo-wide, and
		// the alternative the docs described (structured stdout scraped by a
		// Collector `filelog` receiver) was never actually built — so
		// signoz_logs.distributed_logs_v2 held 0 rows all-time while traces and
		// metrics flowed normally.
		//
		// Registered as a FACTORY, not a pre-built core: aohlog discards and
		// rebuilds its logger on SetDevelopment/SetProduction, and wfe-engine
		// calls SetDevelopment *after* Init — a one-shot tee would be silently
		// dropped there. See aohlog.SetExtraCoreFactory. The level argument is
		// unused here because aohlog gates the returned core itself.
		//
		// KNOWN GAP: aohlog.Fatal/Fatalf records do NOT reach the backend. zap's
		// default terminal hook is WriteThenFatal -> os.Exit(1), so the deferred
		// shutdown below never runs and the BatchProcessor never flushes; and
		// otelzap's Core.Sync() is a no-op (`return nil`), so syncing would not
		// help either. Closing it needs a flush hook this package registers with
		// aohlog for use from zap.WithFatalHook — a second seam, tracked as a
		// follow-up. Until then stdout is the source of truth for fatal exits.
		aohlog.SetExtraCoreFactory(func(zapcore.Level) zapcore.Core {
			return ZapCore(cfg.ServiceName)
		})
		// Detach the bridge before the provider is torn down, so any log call
		// during shutdown goes to stdout only rather than a dead exporter.
		//
		// ORDER IS LOAD-BEARING: `shutdown` above runs shutdownFns in REGISTRATION
		// order, not reverse order, so the detach must be appended BEFORE
		// lp.Shutdown to actually run first. Detaching does not lose anything
		// already emitted — records buffered in the BatchProcessor are still
		// flushed by lp.Shutdown on the next line; the detach only stops NEW
		// records from being handed to a provider that is about to die.
		shutdownFns = append(shutdownFns, func(context.Context) error {
			aohlog.SetExtraCoreFactory(nil)
			return nil
		})
		shutdownFns = append(shutdownFns, lp.Shutdown)
	}

	return shutdown, nil
}

// newResource builds the OTEL resource with the standard AOH attributes.
// deployment.environment.name and any other extra attributes are supplied via
// OTEL_RESOURCE_ATTRIBUTES (picked up by resource.WithFromEnv).
func newResource(ctx context.Context, cfg Config) (*resource.Resource, error) {
	name := cfg.ServiceName
	if name == "" {
		name = os.Getenv("OTEL_SERVICE_NAME")
	}
	if name == "" {
		name = "unknown-service"
	}

	attrs := []attribute.KeyValue{
		attribute.String("service.namespace", ServiceNamespace),
		attribute.String("service.name", name),
		attribute.String("service.instance.id", uuid.NewString()),
	}
	if cfg.ServiceVersion != "" {
		attrs = append(attrs, attribute.String("service.version", cfg.ServiceVersion))
	}

	return resource.New(ctx,
		resource.WithTelemetrySDK(),
		resource.WithFromEnv(), // OTEL_RESOURCE_ATTRIBUTES, OTEL_SERVICE_NAME
		resource.WithAttributes(attrs...),
	)
}
