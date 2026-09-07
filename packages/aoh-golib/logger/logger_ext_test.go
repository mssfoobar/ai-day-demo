package aohlog_test

// External-package tests (AOH-3985): caller attribution and the context-aware
// emitters. Kept separate from logger_test.go, which is in `package aohlog`
// because it reaches the unexported extra-core seam — one directory can hold both
// packages, one file cannot.

import (
	"context"
	"path/filepath"
	"testing"

	aohlog "github.com/mssfoobar/ops-hub/packages/aoh-golib/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// observeDefault redirects the package-default logger into an in-memory sink for
// the duration of one test, so both emitter paths can be inspected.
func observeDefault(t *testing.T) *observer.ObservedLogs {
	t.Helper()

	core, logs := observer.New(zapcore.DebugLevel)
	aohlog.WithOptions(zap.WrapCore(func(zapcore.Core) zapcore.Core { return core }))
	t.Cleanup(aohlog.SetProduction)
	return logs
}

// thisFile is the file every caller assertion below expects. Named rather than
// literal so a rename cannot silently turn these into vacuous passes.
const thisFile = "logger_ext_test.go"

func tracedContext(t *testing.T) (context.Context, trace.SpanContext) {
	t.Helper()

	tp := sdktrace.NewTracerProvider()
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	ctx, span := tp.Tracer("aohlog-test").Start(context.Background(), "op")
	t.Cleanup(func() { span.End() })
	return ctx, span.SpanContext()
}

// TestPackageLevelEmitter_AttributesTheCaller: aohlog.Info adds exactly one
// frame, so `caller` must be this file.
func TestPackageLevelEmitter_AttributesTheCaller(t *testing.T) {
	logs := observeDefault(t)

	aohlog.Info("saved")

	entries := logs.All()
	require.Len(t, entries, 1)
	assert.Equal(t, thisFile, filepath.Base(entries[0].Caller.File))
}

// TestGet_AttributesTheCaller is the AddCallerSkip(1) fix: a caller holding the
// logger directly (as the ~6 aohlog.Get() middleware users do) adds no frame, so
// the single skip baked into the logger misattributed every one of their records
// to their own caller.
func TestGet_AttributesTheCaller(t *testing.T) {
	logs := observeDefault(t)

	aohlog.Get().Error("could not store template")

	entries := logs.All()
	require.Len(t, entries, 1)
	assert.Equal(t, thisFile, filepath.Base(entries[0].Caller.File),
		"aohlog.Get() must report its immediate caller")
}

func TestSugaredEmitter_AttributesTheCaller(t *testing.T) {
	logs := observeDefault(t)

	aohlog.Warnf("retrying %s", "publish")

	entries := logs.All()
	require.Len(t, entries, 1)
	assert.Equal(t, thisFile, filepath.Base(entries[0].Caller.File))
	assert.Equal(t, "retrying publish", entries[0].Message)
}

func TestTraceFields_NoSpan(t *testing.T) {
	assert.Nil(t, aohlog.TraceFields(context.Background()),
		"absent correlation is absent fields, never an empty or generated value")
}

func TestTraceFields_ActiveSpan(t *testing.T) {
	ctx, sc := tracedContext(t)

	fields := aohlog.TraceFields(ctx)

	require.Len(t, fields, 2)
	assert.Equal(t, aohlog.TraceIDKey, fields[0].Key)
	assert.Equal(t, sc.TraceID().String(), fields[0].String)
	assert.Equal(t, aohlog.SpanIDKey, fields[1].Key)
	assert.Equal(t, sc.SpanID().String(), fields[1].String)
}

func TestTraceFields_InvalidSpanContext(t *testing.T) {
	ctx := trace.ContextWithSpanContext(context.Background(),
		trace.NewSpanContext(trace.SpanContextConfig{}))

	assert.Nil(t, aohlog.TraceFields(ctx))
}

// TestCtx_CorrelatesWithoutAnyCallSiteFields is the point of the whole helper: a
// call site that passes no correlation field still emits a correlated record.
func TestCtx_CorrelatesWithoutAnyCallSiteFields(t *testing.T) {
	logs := observeDefault(t)
	ctx, sc := tracedContext(t)

	aohlog.Ctx(ctx).Error("could not store template")

	entries := logs.All()
	require.Len(t, entries, 1)
	fields := entries[0].ContextMap()
	assert.Equal(t, sc.TraceID().String(), fields["trace_id"])
	assert.Equal(t, sc.SpanID().String(), fields["span_id"])
	assert.Equal(t, thisFile, filepath.Base(entries[0].Caller.File))
}

func TestCtxEmitters_CorrelateAndAttributeTheCaller(t *testing.T) {
	ctx, sc := tracedContext(t)

	emitters := map[string]struct {
		emit  func(context.Context, string, ...zap.Field)
		level zapcore.Level
	}{
		"DebugCtx": {aohlog.DebugCtx, zapcore.DebugLevel},
		"InfoCtx":  {aohlog.InfoCtx, zapcore.InfoLevel},
		"WarnCtx":  {aohlog.WarnCtx, zapcore.WarnLevel},
		"ErrorCtx": {aohlog.ErrorCtx, zapcore.ErrorLevel},
	}

	for name, emitter := range emitters {
		t.Run(name, func(t *testing.T) {
			logs := observeDefault(t)

			emitter.emit(ctx, "record")

			entries := logs.All()
			require.Len(t, entries, 1)
			assert.Equal(t, emitter.level, entries[0].Level)
			assert.Equal(t, sc.TraceID().String(), entries[0].ContextMap()["trace_id"])
			assert.Equal(t, thisFile, filepath.Base(entries[0].Caller.File))
		})
	}
}

func TestCtxEmitters_OmitCorrelationWithNoSpan(t *testing.T) {
	logs := observeDefault(t)

	aohlog.ErrorCtx(context.Background(), "could not store template")

	entries := logs.All()
	require.Len(t, entries, 1, "an untraced failure is still logged")
	assert.NotContains(t, entries[0].ContextMap(), "trace_id")
	assert.NotContains(t, entries[0].ContextMap(), "span_id")
}

func TestWithCtx_NilLoggerFallsBackToTheDefault(t *testing.T) {
	logs := observeDefault(t)
	ctx, sc := tracedContext(t)

	aohlog.WithCtx(ctx, nil).Info("record")

	entries := logs.All()
	require.Len(t, entries, 1)
	assert.Equal(t, sc.TraceID().String(), entries[0].ContextMap()["trace_id"])
}

func TestWithCtx_NoSpanReturnsTheLoggerUnchanged(t *testing.T) {
	core, _ := observer.New(zapcore.DebugLevel)
	logger := zap.New(core)

	assert.Same(t, logger, aohlog.WithCtx(context.Background(), logger),
		"no span means no wrapping, so no allocation on the hot path")
}
