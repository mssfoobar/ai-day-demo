package aoherr_test

import (
	"context"
	"regexp"
	"testing"

	"github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

var hex32 = regexp.MustCompile(`^[0-9a-f]{32}$`)

func TestTraceID_NoSpan(t *testing.T) {
	assert.Empty(t, aoherr.TraceID(context.Background()),
		"no span must yield the empty string, never a placeholder or a minted id")
}

func TestTraceID_ActiveSpan(t *testing.T) {
	ctx, sc := tracedContext(t)

	got := aoherr.TraceID(ctx)
	assert.Equal(t, sc.TraceID().String(), got)
	assert.Regexp(t, hex32, got)
}

// TestTraceID_RemoteTraceparent proves the id is the *incoming* trace's, i.e.
// the same value the caller put in its traceparent header.
func TestTraceID_RemoteTraceparent(t *testing.T) {
	tid, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	require.NoError(t, err)
	sid, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	require.NoError(t, err)

	ctx := trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(
		trace.SpanContextConfig{TraceID: tid, SpanID: sid, TraceFlags: trace.FlagsSampled, Remote: true},
	))

	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", aoherr.TraceID(ctx))
}

// TestTraceID_InvalidSpanContext covers the invalid partition: an all-zero
// trace id must be treated as absent, not rendered.
func TestTraceID_InvalidSpanContext(t *testing.T) {
	ctx := trace.ContextWithSpanContext(context.Background(),
		trace.NewSpanContext(trace.SpanContextConfig{}))

	assert.Empty(t, aoherr.TraceID(ctx))
}
