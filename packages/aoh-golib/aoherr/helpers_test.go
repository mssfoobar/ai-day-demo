package aoherr_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// rendered is what one aoherr render produced: the HTTP response, the body
// decoded as a bag of raw keys (so a test can assert on key *presence* and not
// only on unmarshalled values), and the log records the render emitted.
type rendered struct {
	status int
	header http.Header
	body   string
	keys   map[string]json.RawMessage
	logs   []observer.LoggedEntry
}

// renderErr renders err through aoherr with an observed logger, on a request
// carrying ctx. It is the single seam every wire-shape assertion goes through —
// tests assert the marshalled JSON, never the Go struct.
func renderErr(t *testing.T, ctx context.Context, err error) rendered {
	t.Helper()

	core, recorded := observer.New(zapcore.DebugLevel)
	req := httptest.NewRequest(http.MethodPost, "/widgets/42", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	aoherr.RenderWithLogger(zap.New(core), rec, req, err)

	res := rec.Result()
	t.Cleanup(func() { _ = res.Body.Close() })
	raw, readErr := io.ReadAll(res.Body)
	require.NoError(t, readErr)

	keys := map[string]json.RawMessage{}
	require.NoError(t, json.Unmarshal(raw, &keys),
		"body must unmarshal as a single JSON object, got: %s", raw)

	return rendered{
		status: res.StatusCode,
		header: res.Header,
		body:   string(raw),
		keys:   keys,
		logs:   recorded.All(),
	}
}

// str decodes a string-valued body key, failing the test when it is absent.
func (r rendered) str(t *testing.T, key string) string {
	t.Helper()
	raw, ok := r.keys[key]
	require.True(t, ok, "body key %q absent: %s", key, r.body)
	var s string
	require.NoError(t, json.Unmarshal(raw, &s))
	return s
}

// keyNames lists the body's top-level keys.
func (r rendered) keyNames() []string {
	names := make([]string, 0, len(r.keys))
	for k := range r.keys {
		names = append(names, k)
	}
	return names
}

// only asserts exactly one record was emitted and returns it.
func (r rendered) only(t *testing.T) observer.LoggedEntry {
	t.Helper()
	require.Len(t, r.logs, 1, "expected exactly one log record, got %d", len(r.logs))
	return r.logs[0]
}

// tracedContext returns a context carrying a live, valid span.
func tracedContext(t *testing.T) (context.Context, trace.SpanContext) {
	t.Helper()
	tp := sdktrace.NewTracerProvider() // records spans; no exporter needed
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	ctx, span := tp.Tracer("aoherr-test").Start(context.Background(), "op")
	t.Cleanup(func() { span.End() })
	return ctx, span.SpanContext()
}

// fields flattens a log entry's context into a map for assertion.
func fields(e observer.LoggedEntry) map[string]any {
	return e.ContextMap()
}
