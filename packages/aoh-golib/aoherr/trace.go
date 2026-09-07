package aoherr

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

// TraceID returns the hex trace id of the span active in ctx, or "" when ctx
// carries no span or an invalid span context (an all-zero id, for instance).
//
// "" means absent: callers omit the field rather than substituting a
// placeholder, a request id or a freshly minted UUID. An identifier that
// correlates to nothing is worse than none, because it looks actionable.
func TraceID(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return ""
	}
	return sc.TraceID().String()
}
