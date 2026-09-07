package aohlog

import (
	"sync"
	"sync/atomic"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// defaultLogger is swapped, not mutated: the setters below build a fresh
// *zap.Logger and store it. It is held in an atomic.Pointer because those swaps
// are no longer confined to start-up — the OTEL distribution calls
// SetExtraCoreFactory(nil) from its shutdown hook (see aoh-golib/otel), which
// runs while request goroutines are still logging. A plain package variable
// races there: `go test -race` reports the write below against the read in
// Info/Error/etc. The load is a single atomic read on the hot path.
//
// Get() hands out the *zap.Logger itself, so a caller that holds the returned
// pointer across a Set* call keeps logging through the OLD logger. That is
// pre-existing behaviour and unchanged here — callers wanting to follow the
// swap should call Get() per use, or use the package-level functions.
var defaultLogger atomic.Pointer[zap.Logger]

// directLogger is the same logger with the caller skip cancelled, for Get()
// (AOH-3985). Every logger this package builds carries AddCallerSkip(1) so the
// package-level helpers report their caller; a holder that logs through the
// returned pointer adds no such frame, so that skip would blame its caller's
// caller — which misreported the ~6 HTTP middlewares that log this way.
//
// It is pre-built and stored rather than derived per call for two reasons: Get()
// is on a request hot path and would otherwise allocate a clone every time, and
// a fresh clone per call breaks pointer identity, which the OTEL log-bridge
// tests legitimately compare to prove Init left the logger untouched.
//
// Always written through storeLogger so the pair cannot drift apart.
var directLogger atomic.Pointer[zap.Logger]

// storeLogger publishes l as the package logger and derives the Get() view.
// Callers must hold rebuildMu (except in init, which cannot race).
func storeLogger(l *zap.Logger) {
	defaultLogger.Store(l)
	directLogger.Store(l.WithOptions(zap.AddCallerSkip(-1)))
}

// default to info level logger
func init() {
	storeLogger(zap.Must(newZapLogger(zap.InfoLevel)))
}

func Debug(msg string, fields ...zap.Field) { defaultLogger.Load().Debug(msg, fields...) }
func Info(msg string, fields ...zap.Field)  { defaultLogger.Load().Info(msg, fields...) }
func Warn(msg string, fields ...zap.Field)  { defaultLogger.Load().Warn(msg, fields...) }
func Error(msg string, fields ...zap.Field) { defaultLogger.Load().Error(msg, fields...) }
func Panic(msg string, fields ...zap.Field) { defaultLogger.Load().Panic(msg, fields...) }
func Fatal(msg string, fields ...zap.Field) { defaultLogger.Load().Fatal(msg, fields...) }

func Debugf(template string, args ...interface{}) {
	defaultLogger.Load().Sugar().Debugf(template, args...)
}
func Infof(template string, args ...interface{}) {
	defaultLogger.Load().Sugar().Infof(template, args...)
}
func Warnf(template string, args ...interface{}) {
	defaultLogger.Load().Sugar().Warnf(template, args...)
}
func Errorf(template string, args ...interface{}) {
	defaultLogger.Load().Sugar().Errorf(template, args...)
}
func Panicf(template string, args ...interface{}) {
	defaultLogger.Load().Sugar().Panicf(template, args...)
}
func Fatalf(template string, args ...interface{}) {
	defaultLogger.Load().Sugar().Fatalf(template, args...)
}

// SetDevelopment set logger to development mode
func SetDevelopment() {
	rebuildMu.Lock()
	defer rebuildMu.Unlock()
	storeLogger(zap.Must(newZapLogger(zap.DebugLevel)))
}

// SetProduction set logger to production mode
func SetProduction() {
	rebuildMu.Lock()
	defer rebuildMu.Unlock()
	storeLogger(zap.Must(newZapLogger(zap.InfoLevel)))
}

// Get returns the logger for callers that hold it and log through it directly
// (`aohlog.Get().Error(...)`), as the ~6 HTTP middlewares in this repo do.
//
// It returns the caller-skip-cancelled view (see directLogger) so `caller` points
// at the holder, not at the holder's caller.
func Get() *zap.Logger {
	return directLogger.Load()
}

// WithOptions override the default logger with options parameter
func WithOptions(options ...zap.Option) {
	rebuildMu.Lock()
	defer rebuildMu.Unlock()
	storeLogger(defaultLogger.Load().WithOptions(options...))
}

// extraCoreFactory, when non-nil, builds an additional zapcore.Core that
// newZapLogger tees alongside the stdout core. It is the seam the OTEL
// distribution uses to bridge aohlog records into OTLP log export WITHOUT this
// package taking an OpenTelemetry dependency — the factory is just a func, so
// `logger` stays zap-only and the wiring lives in `aoh-golib/otel`.
//
// The factory receives the level the logger is being built at, so the extra core
// can be gated to that same level (see newZapLogger).
//
// Atomic for the same reason as defaultLogger: newZapLogger reads it on every
// build, and the OTEL shutdown hook writes it concurrently with in-flight
// logging. Keeping the read lock-free is also what lets rebuildMu below be held
// across newZapLogger without any re-entrancy hazard.
var extraCoreFactory atomic.Pointer[func(zapcore.Level) zapcore.Core]

// rebuildMu serialises the setters against each other. The atomics above make
// every individual access race-free, but a rebuild is a read-then-write pair
// (read the current level, store a new logger); without this, two concurrent
// setters can interleave so the later store is built from the earlier's level
// and one caller's change is silently lost. Readers never take this lock.
var rebuildMu sync.Mutex

// SetExtraCoreFactory registers f as the source of an additional core teed into
// every logger this package builds, and rebuilds the current default logger so
// the change takes effect immediately. Pass nil to remove it.
//
// f is consulted by newZapLogger on EVERY build rather than applied once. That
// is load-bearing, not incidental: SetDevelopment/SetProduction *discard and
// rebuild* defaultLogger, and at least one service (wfe-engine) calls
// SetDevelopment AFTER initialising OTEL. A one-shot tee would be silently
// dropped there — that service's logs would go missing from the backend with no
// error raised anywhere.
func SetExtraCoreFactory(f func(zapcore.Level) zapcore.Core) {
	rebuildMu.Lock()
	defer rebuildMu.Unlock()
	if f == nil {
		extraCoreFactory.Store(nil)
	} else {
		extraCoreFactory.Store(&f)
	}
	storeLogger(zap.Must(newZapLogger(defaultLogger.Load().Level())))
}

// newZapLogger init preconfigured zap logger.
// For development, set l to zapcore.DebugLevel.
func newZapLogger(l zapcore.Level) (*zap.Logger, error) {
	// Default to production
	isDev := false
	encoding := "json"
	encodeLevel := zapcore.CapitalLevelEncoder

	// Set development to TRUE if its DebugLevel
	if l == zapcore.DebugLevel {
		isDev = true
		encoding = "console"
		encodeLevel = zapcore.CapitalColorLevelEncoder
	}
	encodeConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    encodeLevel,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(l),
		Development:      isDev,
		Sampling:         nil,
		Encoding:         encoding,
		EncoderConfig:    encodeConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	opts := []zap.Option{zap.AddCallerSkip(1), zap.AddStacktrace(zap.DPanicLevel)}

	// Tee in the registered extra core (OTLP log export, when the OTEL
	// distribution has wired it). Gated with NewIncreaseLevelCore at the same
	// level as the stdout core: bridge cores such as otelzap's report
	// Enabled()==true for every level, so an ungated tee would ship DEBUG records
	// to the backend even while stdout is at Info — silently multiplying log
	// volume and cost. On the dev path (l == DebugLevel) both cores are at Debug,
	// so the two sinks always agree.
	if fp := extraCoreFactory.Load(); fp != nil {
		f := *fp
		opts = append(opts, zap.WrapCore(func(stdout zapcore.Core) zapcore.Core {
			extra := f(l)
			if extra == nil {
				return stdout
			}
			gated, err := zapcore.NewIncreaseLevelCore(extra, zap.NewAtomicLevelAt(l))
			if err != nil {
				// Only returned when the target level is BELOW the core's own
				// level, i.e. the core is already stricter than we asked for.
				// Teeing it unchanged is then correct, not a failure.
				gated = extra
			}
			return zapcore.NewTee(stdout, gated)
		}))
	}

	logger, err := config.Build(opts...)
	return logger, err
}
