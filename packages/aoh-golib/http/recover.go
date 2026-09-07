package aohhttp

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Recoverer recovers a panicking handler, records it on the active span, logs it
// once at ERROR with the stack, and renders the conformant AOH 500.
//
// It is a drop-in replacement for chi's middleware.Recoverer — mount it the same
// way, r.Use(aohhttp.Recoverer) — whose recovery writes a bare 500 with a
// zero-length body, the largest empty-body hole in the platform.
//
// http.ErrAbortHandler is re-panicked, as net/http documents: it is a deliberate
// abort, not a failure to report.
func Recoverer(next http.Handler) http.Handler {
	return RecovererWithLogger(nil)(next)
}

// RecovererWithLogger is Recoverer for services that hold their own logger; nil
// means the package default, resolved per request.
func RecovererWithLogger(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}
				if recovered == http.ErrAbortHandler {
					panic(recovered)
				}
				// Captured inside the deferred call, so the stack still holds
				// the panicking frames rather than this middleware's.
				renderPanic(logger, w, r, recovered, debug.Stack())
			}()

			next.ServeHTTP(w, r)
		})
	}
}

func renderPanic(logger *zap.Logger, w http.ResponseWriter, r *http.Request, recovered any, stack []byte) {
	cause := panicCause(recovered)

	if span := trace.SpanFromContext(r.Context()); span.SpanContext().IsValid() {
		span.RecordError(cause, trace.WithStackTrace(true))
		span.SetStatus(codes.Error, "panic recovered")
	}

	// The renderer is the single logging point: it emits the one ERROR record —
	// with the panic value and this stack — and then writes the response, whose
	// body carries neither.
	aoherr.RenderWithLogger(logger, w, r, aoherr.Wrap(
		aoherr.ClassSystem, aoherr.CodeInternal, "unhandled panic", cause).WithStack(stack))
}

func panicCause(recovered any) error {
	if err, ok := recovered.(error); ok {
		return err
	}
	return fmt.Errorf("panic: %v", recovered)
}
