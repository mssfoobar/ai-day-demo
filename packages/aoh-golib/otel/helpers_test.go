package aohotel

import (
	"context"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap/zapcore"
)

func TestTeamID_RoundTrip(t *testing.T) {
	if got := TeamID(context.Background()); got != "" {
		t.Errorf("empty ctx: TeamID = %q, want empty", got)
	}

	ctx := ContextWithTeamID(context.Background(), "alpha")
	if got := TeamID(ctx); got != "alpha" {
		t.Errorf("TeamID = %q, want %q", got, "alpha")
	}

	f := TeamIDField(ctx)
	if f.Key != TeamIDKey || f.String != "alpha" {
		t.Errorf("TeamIDField = {%q: %q}, want {%q: alpha}", f.Key, f.String, TeamIDKey)
	}

	// Empty id is a no-op (no baggage member added).
	if got := TeamID(ContextWithTeamID(context.Background(), "")); got != "" {
		t.Errorf("empty team id: TeamID = %q, want empty", got)
	}
}

func TestTeamIDField_AbsentIsSkip(t *testing.T) {
	if f := TeamIDField(context.Background()); f.Type != zapcore.SkipType {
		t.Errorf("absent team_id: field type = %v, want Skip", f.Type)
	}
}

func TestTraceContextFields_NoSpan(t *testing.T) {
	if f := TraceContextFields(context.Background()); f != nil {
		t.Errorf("no span: want nil, got %v", f)
	}
}

func TestTraceContextFields_WithSpan(t *testing.T) {
	tp := sdktrace.NewTracerProvider() // records spans; no exporter needed
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()

	fields := TraceContextFields(ctx)
	if len(fields) != 2 {
		t.Fatalf("want 2 fields (trace_id, span_id), got %d: %v", len(fields), fields)
	}
	got := map[string]string{fields[0].Key: fields[0].String, fields[1].Key: fields[1].String}
	if got["trace_id"] == "" || got["span_id"] == "" {
		t.Errorf("missing trace_id/span_id: %v", got)
	}
}
