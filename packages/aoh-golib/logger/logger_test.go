package aohlog

// Tests for the extra-core tee seam (AOH-7540 logs export).
//
// Two of these encode bugs that were found by inspection BEFORE they shipped, so
// they exist specifically to keep those bugs from coming back:
//
//   - SurvivesRebuild: SetDevelopment/SetProduction discard and rebuild
//     defaultLogger. wfe-engine calls SetDevelopment *after* OTEL init, so a
//     tee applied once at init time is silently discarded there.
//   - RespectsLoggerLevel: bridge cores (otelzap) report Enabled()==true at every
//     level, so an ungated tee ships DEBUG to the backend while stdout is at Info.
//
// Both failure modes are SILENT — no error, no panic, just missing or excess
// telemetry — which is exactly why they need tests rather than review.

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// withObserverFactory registers a factory returning an observer core that, like
// otelzap's, is enabled at Debug and below — i.e. it accepts everything and does
// no gating of its own. Restores prior global state on cleanup.
func withObserverFactory(t *testing.T) *observer.ObservedLogs {
	t.Helper()
	core, logs := observer.New(zapcore.DebugLevel)

	prevLogger := defaultLogger.Load()
	prevFactory := extraCoreFactory.Load()
	t.Cleanup(func() {
		extraCoreFactory.Store(prevFactory)
		defaultLogger.Store(prevLogger)
	})

	SetExtraCoreFactory(func(zapcore.Level) zapcore.Core { return core })
	return logs
}

func TestExtraCoreFactory_TeesRecords(t *testing.T) {
	logs := withObserverFactory(t)

	Info("hello", zap.String("k", "v"))

	require.Equal(t, 1, logs.Len(), "record must reach the teed core")
	entry := logs.All()[0]
	assert.Equal(t, "hello", entry.Message)
	assert.Equal(t, zapcore.InfoLevel, entry.Level)
	assert.Equal(t, "v", entry.ContextMap()["k"], "fields must survive the tee")
}

// TestExtraCoreFactory_SurvivesRebuild is TRAP 1. SetDevelopment replaces
// defaultLogger wholesale; the factory must be re-consulted so the tee persists.
func TestExtraCoreFactory_SurvivesRebuild(t *testing.T) {
	logs := withObserverFactory(t)

	SetDevelopment() // what wfe-engine does AFTER aohotel.Init
	Info("after SetDevelopment")
	require.Equal(t, 1, logs.Len(),
		"tee must survive SetDevelopment — wfe-engine calls it after OTEL init")

	SetProduction()
	Info("after SetProduction")
	assert.Equal(t, 2, logs.Len(), "tee must survive SetProduction too")
}

// TestExtraCoreFactory_RespectsLoggerLevel is TRAP 2. The teed core accepts every
// level on its own, so without gating a Debug call at Info level would still be
// exported — silently multiplying backend log volume.
func TestExtraCoreFactory_RespectsLoggerLevel(t *testing.T) {
	logs := withObserverFactory(t)
	SetProduction() // Info level

	Debug("should not be exported")
	assert.Equal(t, 0, logs.Len(),
		"DEBUG must not reach the teed core while the logger is at Info")

	Info("should be exported")
	assert.Equal(t, 1, logs.Len(), "INFO must still reach the teed core")
}

// TestExtraCoreFactory_DevLevelExportsDebug is the other half of TRAP 2: gating
// must track the configured level, not hardcode Info, so dev mode exports Debug.
func TestExtraCoreFactory_DevLevelExportsDebug(t *testing.T) {
	logs := withObserverFactory(t)
	SetDevelopment() // Debug level

	Debug("dev debug")
	assert.Equal(t, 1, logs.Len(),
		"at Debug level both sinks must agree and export DEBUG")
}

func TestExtraCoreFactory_NilDetaches(t *testing.T) {
	logs := withObserverFactory(t)

	Info("before detach")
	require.Equal(t, 1, logs.Len())

	SetExtraCoreFactory(nil)
	Info("after detach")
	assert.Equal(t, 1, logs.Len(), "nil factory must remove the tee")
}

// TestExtraCoreFactory_NilCoreIsSafe covers a factory that returns nil (e.g. the
// OTEL global LoggerProvider not being installed yet) — must not panic.
func TestExtraCoreFactory_NilCoreIsSafe(t *testing.T) {
	prevLogger := defaultLogger.Load()
	prevFactory := extraCoreFactory.Load()
	t.Cleanup(func() {
		extraCoreFactory.Store(prevFactory)
		defaultLogger.Store(prevLogger)
	})

	SetExtraCoreFactory(func(zapcore.Level) zapcore.Core { return nil })
	assert.NotPanics(t, func() { Info("nil core must be tolerated") })
}

// TestStdoutStillWorks guards the obvious regression: teeing must not replace the
// primary sink. Asserted structurally, since stdout itself is awkward to capture.
func TestStdoutStillWorks(t *testing.T) {
	withObserverFactory(t)
	assert.True(t, defaultLogger.Load().Core().Enabled(zapcore.InfoLevel),
		"logger must remain enabled at Info after the tee is installed")
	assert.NotNil(t, Get())
}

// TestConcurrentRebuildAndLog is a race regression test, and only means anything
// under `go test -race`.
//
// Before this package held defaultLogger atomically, the write in
// SetExtraCoreFactory raced the reads in Info/Error/etc. That race was latent
// for as long as the factory was only ever set once during start-up — the OTLP
// log distribution is what made it reachable, because its shutdown hook calls
// SetExtraCoreFactory(nil) to detach the bridge while request goroutines are
// still logging. The detector reported logger.go's store against its load.
//
// The assertion is the detector itself: if the guarantee regresses, this test
// fails with DATA RACE rather than a bad value.
func TestConcurrentRebuildAndLog(t *testing.T) {
	withObserverFactory(t)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				Info("concurrent log")
				Errorf("concurrent %d", j)
				_ = Get()
			}
		}()
	}

	// Mirrors the real shutdown path: detach and re-attach under load.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 50; j++ {
			SetExtraCoreFactory(nil)
			SetExtraCoreFactory(func(zapcore.Level) zapcore.Core { return nil })
		}
	}()

	// The other rebuild path: wfe-engine calls SetDevelopment after OTEL init.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 50; j++ {
			SetDevelopment()
			SetProduction()
		}
	}()

	wg.Wait()
}
