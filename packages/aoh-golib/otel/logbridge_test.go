package aohotel

// Tests that Init actually installs the aohlog -> OTLP log bridge (AOH-7540).
//
// The bridge is what makes the LoggerProvider non-inert. Before it existed, Init
// built a full OTLP LoggerProvider that nothing ever wrote to — ZapCore had zero
// callers repo-wide — so signoz_logs held 0 rows all-time while traces and
// metrics flowed fine. A "logs are wired" claim needs a test, not a code read.
//
// Installation is observed via aohlog.Get() identity: SetExtraCoreFactory
// rebuilds the default logger, so wiring the bridge necessarily replaces the
// *zap.Logger. That avoids exporting test-only introspection from aohlog.

import (
	"context"
	"testing"
	"time"

	aohlog "github.com/mssfoobar/ops-hub/packages/aoh-golib/logger"
)

// initWithEndpoint runs Init against a (dead) endpoint — the OTLP gRPC exporters
// connect lazily, so Init succeeds. Shutdown is bounded because a flush with no
// live collector will time out; that is not what these tests are asserting.
func initWithEndpoint(t *testing.T) {
	t.Helper()
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")

	shutdown, err := Init(context.Background(), Config{ServiceName: "test-svc"})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		_ = shutdown(ctx)
		aohlog.SetExtraCoreFactory(nil)
	})
}

// TestInit_InstallsLogBridge is the load-bearing assertion: Init must attach the
// bridge to aohlog, not merely construct a LoggerProvider.
func TestInit_InstallsLogBridge(t *testing.T) {
	before := aohlog.Get()
	initWithEndpoint(t)

	if aohlog.Get() == before {
		t.Fatal("Init did not install the aohlog log bridge: the default logger " +
			"was not rebuilt, so aohlog records still go to stdout only and " +
			"signoz_logs stays empty")
	}
}

// TestInit_LogsExporterNone_SkipsBridge pins the documented off-switch. It uses
// the spec'd OTEL_LOGS_EXPORTER rather than an AOH-specific flag.
func TestInit_LogsExporterNone_SkipsBridge(t *testing.T) {
	t.Setenv("OTEL_LOGS_EXPORTER", "none")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")

	before := aohlog.Get()
	shutdown, err := Init(context.Background(), Config{ServiceName: "test-svc"})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		_ = shutdown(ctx)
		aohlog.SetExtraCoreFactory(nil)
	})

	if aohlog.Get() != before {
		t.Error("OTEL_LOGS_EXPORTER=none must not install the log bridge")
	}
}

// TestInit_NoEndpoint_SkipsBridge: export disabled entirely means no bridge, so
// local/CI runs keep plain stdout logging with no OTEL machinery attached.
func TestInit_NoEndpoint_SkipsBridge(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")

	before := aohlog.Get()
	shutdown, err := Init(context.Background(), Config{ServiceName: "test-svc"})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(func() { _ = shutdown(context.Background()) })

	if aohlog.Get() != before {
		t.Error("with no OTLP endpoint the default logger must be left untouched")
	}
}

// TestInit_LoggingAfterBridgeDoesNotPanic exercises the real call path: an
// aohlog write while the bridge is attached but the collector is unreachable
// must degrade to stdout, never panic or block the caller.
func TestInit_LoggingAfterBridgeDoesNotPanic(t *testing.T) {
	initWithEndpoint(t)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("logging through the bridge panicked: %v", r)
		}
	}()
	aohlog.Info("bridge smoke test")
	aohlog.Error("bridge smoke test at error level")
}

// TestShutdown_DetachesBridge pins the teardown half of the bridge contract.
//
// Init registers two shutdown functions for logs: detach the aohlog bridge, and
// shut the LoggerProvider down. `shutdown` runs them in REGISTRATION order (a
// forward loop, not the reverse order teardown usually implies), so the detach
// is appended first — otherwise the provider dies while aohlog is still handing
// it records, and anything logged during shutdown goes to a dead exporter.
//
// The order of the two appends is not observable from outside Init, but the
// detach happening at all is: SetExtraCoreFactory(nil) rebuilds the default
// logger, so a shutdown that detaches necessarily replaces the *zap.Logger.
// Dropping or skipping the detach append fails here.
func TestShutdown_DetachesBridge(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")

	shutdown, err := Init(context.Background(), Config{ServiceName: "test-svc"})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(func() { aohlog.SetExtraCoreFactory(nil) })

	bridged := aohlog.Get()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_ = shutdown(ctx) // error ignored: no live collector, so the flush times out

	if aohlog.Get() == bridged {
		t.Fatal("shutdown did not detach the log bridge: aohlog still holds the " +
			"bridged logger, so log calls during and after teardown are handed " +
			"to a LoggerProvider that has already been shut down")
	}
}
