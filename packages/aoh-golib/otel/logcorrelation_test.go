package aohotel

// Pins how trace correlation ACTUALLY behaves on the OTLP log path (AOH-7540).
//
// This test exists because the repo documents the opposite. Several docs assert
// "the otelzap bridge injects trace_id/span_id into every aohlog record — do not
// hand-roll log correlation". Measured, that is wrong twice over: the bridge does
// NOT auto-correlate, and every AOH service *does* hand-roll via
// TraceContextFields. otelzap defaults its emit context to context.Background()
// and only adopts a span when a field of type context.Context is present
// (otelzap/core.go: `ctx: context.Background()`, overridden in convertField).
//
// Consequence, pinned below: records emitted the way services log today carry
// trace_id/span_id as ordinary string ATTRIBUTES but leave the record's NATIVE
// TraceID zero. Backends join logs to traces on the native field, so those
// records are not natively correlated. Getting native correlation requires
// passing the context as a field.
//
// If a future change makes correlation automatic, these assertions should be
// updated deliberately — not deleted because they "look wrong".

import (
	"context"
	"testing"

	aohlog "github.com/mssfoobar/ops-hub/packages/aoh-golib/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	otellog "go.opentelemetry.io/otel/log"
	otellogglobal "go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// captureProcessor records every emitted log record for inspection.
type captureProcessor struct{ records []sdklog.Record }

func (c *captureProcessor) OnEmit(_ context.Context, r *sdklog.Record) error {
	c.records = append(c.records, *r)
	return nil
}
func (c *captureProcessor) Enabled(context.Context, sdklog.EnabledParameters) bool { return true }
func (c *captureProcessor) Shutdown(context.Context) error                         { return nil }
func (c *captureProcessor) ForceFlush(context.Context) error                       { return nil }

func newCorrelationHarness(t *testing.T) (*captureProcessor, context.Context, string) {
	t.Helper()

	cap := &captureProcessor{}
	lp := sdklog.NewLoggerProvider(sdklog.WithProcessor(cap))
	tp := sdktrace.NewTracerProvider()

	prevLP := otellogglobal.GetLoggerProvider()
	prevTP := otel.GetTracerProvider()
	otellogglobal.SetLoggerProvider(lp)
	otel.SetTracerProvider(tp)

	aohlog.SetExtraCoreFactory(func(zapcore.Level) zapcore.Core { return ZapCore("test") })
	t.Cleanup(func() {
		aohlog.SetExtraCoreFactory(nil)
		otellogglobal.SetLoggerProvider(prevLP)
		otel.SetTracerProvider(prevTP)
	})

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	t.Cleanup(func() { span.End() })
	return cap, ctx, span.SpanContext().TraceID().String()
}

func attrKeys(r sdklog.Record) map[string]string {
	out := map[string]string{}
	r.WalkAttributes(func(kv otellog.KeyValue) bool {
		out[kv.Key] = kv.Value.String()
		return true
	})
	return out
}

// TestLogRecord_TraceContextFields_LeavesNativeTraceIDEmpty pins the CURRENT
// behaviour of every AOH service: trace ids arrive as attributes only.
func TestLogRecord_TraceContextFields_LeavesNativeTraceIDEmpty(t *testing.T) {
	cap, ctx, wantTraceID := newCorrelationHarness(t)

	// Exactly how dash/unh/form/wfe/msr log today.
	aohlog.Info("saved", TraceContextFields(ctx)...)

	require.Len(t, cap.records, 1)
	rec := cap.records[0]

	assert.False(t, rec.TraceID().IsValid(),
		"native TraceID is NOT populated by TraceContextFields — the bridge never "+
			"sees the context, so backend logs<->traces joins do not fire")

	attrs := attrKeys(rec)
	assert.Equal(t, wantTraceID, attrs["trace_id"],
		"trace_id is present, but only as an ordinary string attribute")
	assert.Contains(t, attrs, "span_id")
}

// TestLogRecord_ContextField_PopulatesNativeTraceID shows the path that DOES
// produce a natively correlated record.
func TestLogRecord_ContextField_PopulatesNativeTraceID(t *testing.T) {
	cap, ctx, wantTraceID := newCorrelationHarness(t)

	aohlog.Info("saved", zap.Any("ctx", ctx))

	require.Len(t, cap.records, 1)
	rec := cap.records[0]

	require.True(t, rec.TraceID().IsValid(),
		"passing the context as a field must populate the native TraceID")
	assert.Equal(t, wantTraceID, rec.TraceID().String())
}

// TestLogRecord_NoContext_HasNoTraceID is the baseline: a plain log call outside
// any span is exported, just uncorrelated.
func TestLogRecord_NoContext_HasNoTraceID(t *testing.T) {
	cap, _, _ := newCorrelationHarness(t)

	aohlog.Info("no context")

	require.Len(t, cap.records, 1, "the record must still be exported")
	assert.False(t, cap.records[0].TraceID().IsValid())
}

// TestLogRecord_IsExported guards the headline behaviour the bridge exists for:
// aohlog records reach the LoggerProvider at all. This is what was broken —
// signoz_logs held 0 rows all-time because nothing wrote to the provider.
func TestLogRecord_IsExported(t *testing.T) {
	cap, _, _ := newCorrelationHarness(t)

	aohlog.Info("first")
	aohlog.Warn("second")
	aohlog.Error("third")

	require.Len(t, cap.records, 3, "every aohlog record must reach the OTLP provider")
	assert.Equal(t, "first", cap.records[0].Body().AsString())
	assert.Equal(t, otellog.SeverityWarn, cap.records[1].Severity())
	assert.Equal(t, otellog.SeverityError, cap.records[2].Severity())
}
