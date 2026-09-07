package aohlog

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Field keys for the trace correlation pair. They match
// aohotel.TraceContextFields and the `trace_id` the AOH error envelope hands
// the client, so one value joins a response, its logs and its trace.
const (
	TraceIDKey = "trace_id"
	SpanIDKey  = "span_id"
)

// TraceFields returns the correlation fields for the span active in ctx, or nil
// when ctx carries no valid span. Nil is deliberate: absent correlation is
// rendered as absent keys, never as an empty, placeholder or generated value.
//
// This is the same pair aohotel.TraceContextFields produces; it lives here too
// so a service can correlate its logs without linking the OTEL exporters.
func TraceFields(ctx context.Context) []zap.Field {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return nil
	}
	return []zap.Field{
		zap.String(TraceIDKey, sc.TraceID().String()),
		zap.String(SpanIDKey, sc.SpanID().String()),
	}
}

// Ctx returns the default logger with the active span's trace context already
// attached, so a call site that adds no correlation fields still emits a
// correlated record:
//
//	aohlog.Ctx(ctx).Error("could not store template", zap.Error(err))
//
// Prefer this (or the *Ctx helpers below) over hand-splatting trace fields — a
// convention that depends on remembering is why no Go error record in this
// repo carried a trace id before AOH-3985.
// Note it returns the SKIPPED logger (the one the package-level emitters use),
// because the intended use is `aohlog.Ctx(ctx).Error(...)` — the caller adds no
// frame of its own, so the stored logger's skip would misattribute the record.
func Ctx(ctx context.Context) *zap.Logger {
	return WithCtx(ctx, Get())
}

// WithCtx attaches the active span's trace context to an arbitrary logger, for
// the middlewares and helpers that are handed a *zap.Logger rather than using
// the package default.
func WithCtx(ctx context.Context, l *zap.Logger) *zap.Logger {
	if l == nil {
		l = Get()
	}
	fields := TraceFields(ctx)
	if len(fields) == 0 {
		return l
	}
	return l.With(fields...)
}

// The *Ctx helpers are the package-level emitters with automatic correlation.
// They use the stored logger directly — it already carries AddCallerSkip(1), and
// each helper adds exactly the one frame that skip accounts for, so `caller`
// points at the call site just as it does for Debug/Info/…

func DebugCtx(ctx context.Context, msg string, fields ...zap.Field) {
	WithCtx(ctx, defaultLogger.Load()).Debug(msg, fields...)
}

func InfoCtx(ctx context.Context, msg string, fields ...zap.Field) {
	WithCtx(ctx, defaultLogger.Load()).Info(msg, fields...)
}

func WarnCtx(ctx context.Context, msg string, fields ...zap.Field) {
	WithCtx(ctx, defaultLogger.Load()).Warn(msg, fields...)
}

func ErrorCtx(ctx context.Context, msg string, fields ...zap.Field) {
	WithCtx(ctx, defaultLogger.Load()).Error(msg, fields...)
}
