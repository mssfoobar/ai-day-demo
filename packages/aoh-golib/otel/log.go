package aohotel

import (
	"context"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	otellogglobal "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// TraceContextFields returns zap fields (trace_id, span_id) for the span active
// in ctx, so a log line can be joined to its trace in the backend. It returns
// nil when ctx carries no valid span, so it's safe to splat unconditionally:
//
//	aohlog.Info("saved", append(aohotel.TraceContextFields(ctx),
//		zap.String("id", id))...)
//
// This puts the ids in the stdout JSON itself, so `kubectl logs` and any log
// scraper can correlate without depending on the OTEL logs SDK.
//
// CORRECTION (AOH-7540): this comment used to say the collector's `filelog`
// receiver joins these lines to traces, and to prefer this over ZapCore for
// correlation. **No filelog receiver was ever deployed** — the AOH gateway
// collector has only an `otlp` receiver — so nothing ever consumed stdout and
// signoz_logs.distributed_logs_v2 held 0 rows all-time while traces and metrics
// flowed normally. Backend correlation now comes from the OTLP log bridge that
// Init installs (see otel.go). Keep using this helper for stdout readability; it
// is not what gets logs into the backend.
//
// This function is fully supported and is what the six services' access loggers
// use. For NEW code that has a ctx in hand, aohlog.Ctx / aohlog.ErrorCtx are
// usually easier — they derive the same pair from the request context, so the
// fields cannot be forgotten (AOH-3985; no Go error record in this repo carried a
// trace id while attaching them was left to each author). Reach for this one when
// you hold a plain *zap.Logger, or when you are assembling a field slice by hand.
//
// Either way the fields must be attached explicitly by someone: note that the
// concern here is orthogonal to the correction above. ZapCore governs whether a
// record reaches the backend at all; this governs whether the record carries the
// ids that make it joinable once there.
func TraceContextFields(ctx context.Context) []zap.Field {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return nil
	}
	return []zap.Field{
		zap.String("trace_id", sc.TraceID().String()),
		zap.String("span_id", sc.SpanID().String()),
	}
}

// ZapCore returns an otelzap bridge core that emits zap records to the global
// OTEL LoggerProvider (OTLP log export). Tee it into an existing zap logger so
// logs keep flowing to their normal sink too:
//
//	logger = zap.New(zapcore.NewTee(consoleCore, aohotel.ZapCore("aoh")))
//
// To correlate an exported record with a trace on this path, pass the context
// as a field — zap.Any("ctx", ctx) — which the bridge reads as the emit context.
//
// Services do NOT need to call this themselves: as of AOH-7540, Init registers it
// with aohlog automatically (unless OTEL_LOGS_EXPORTER=none), so every aohlog
// record is exported. It stays exported for services that build their own zap
// logger outside aohlog. The previous advice here — "reach for ZapCore only when
// you want OTLP-native log export", implying the default path was filelog — was
// wrong: it left ZapCore with zero callers repo-wide and logs entirely dead.
//
// Note (AOH-3985) that this handles EXPORT, not the record's contents: the bridge
// reads a trace only from an explicitly-passed ctx field, so a record still needs
// aohlog.Ctx (or TraceContextFields) to carry trace_id/span_id. The two are
// complementary — ZapCore gets the record to the backend, aohlog.Ctx makes it
// joinable once there.
func ZapCore(name string) zapcore.Core {
	return otelzap.NewCore(name, otelzap.WithLoggerProvider(otellogglobal.GetLoggerProvider()))
}
