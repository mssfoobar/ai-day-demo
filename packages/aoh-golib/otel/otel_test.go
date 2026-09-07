package aohotel

import (
	"context"
	"slices"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
)

// With no OTLP endpoint, Init is a no-op for export but still installs the
// W3C propagators and returns a working no-op shutdown.
func TestInit_NoEndpoint_InstallsPropagatorsOnly(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")

	shutdown, err := Init(context.Background(), Config{ServiceName: "test-svc"})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if shutdown == nil {
		t.Fatal("Init returned a nil shutdown func")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	fields := otel.GetTextMapPropagator().Fields()
	for _, want := range []string{"traceparent", "baggage"} {
		if !slices.Contains(fields, want) {
			t.Errorf("propagator missing %q header; got fields %v", want, fields)
		}
	}
}

// With an endpoint set, Init wires the providers without error (the OTLP gRPC
// exporters connect lazily). Shutdown does a best-effort flush; with no live
// collector the exporters time out, so we bound it with a short context and
// don't treat that as a failure — we only assert Init wired things cleanly.
func TestInit_WithEndpoint_WiresProviders(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4317")

	shutdown, err := Init(context.Background(), Config{
		ServiceName:    "test-svc",
		ServiceVersion: "0.0.0",
	})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if shutdown == nil {
		t.Fatal("Init returned a nil shutdown func")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = shutdown(ctx) // best-effort flush; no collector in unit tests
}

func TestNewResource_HasStandardAttributes(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "")
	res, err := newResource(context.Background(), Config{ServiceName: "msr-service", ServiceVersion: "1.2.3"})
	if err != nil {
		t.Fatalf("newResource: %v", err)
	}

	got := map[string]string{}
	for _, kv := range res.Attributes() {
		got[string(kv.Key)] = kv.Value.AsString()
	}
	for k, want := range map[string]string{
		"service.namespace": "aoh",
		"service.name":      "msr-service",
		"service.version":   "1.2.3",
	} {
		if got[k] != want {
			t.Errorf("resource %q = %q, want %q", k, got[k], want)
		}
	}
	if got["service.instance.id"] == "" {
		t.Error("resource missing service.instance.id")
	}
}
