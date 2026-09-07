package aohhttp_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/render"
	aohhttp "github.com/mssfoobar/ops-hub/packages/aoh-golib/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// legacyBody is the legacy envelope as an SDK parses it, plus the one additive
// field. Decoding into it proves nothing was renamed or retyped.
type legacyBody struct {
	Data    any    `json:"data"`
	Message string `json:"message"`
	SentAt  string `json:"sent_at"`
	Errors  []struct {
		Message string `json:"message"`
	} `json:"errors"`
	TraceID string `json:"trace_id"`
}

// renderLegacy renders a legacy envelope exactly as the 363 unmigrated call
// sites do — through go-chi/render — and returns the status, the body's key set
// and the decoded body.
func renderLegacy(t *testing.T, ctx context.Context, payload render.Renderer) (int, []string, legacyBody) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/widgets", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	require.NoError(t, render.Render(rec, req, payload))

	res := rec.Result()
	raw, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())

	keys := map[string]json.RawMessage{}
	require.NoError(t, json.Unmarshal(raw, &keys), "body must be a JSON object: %s", raw)

	var body legacyBody
	require.NoError(t, json.Unmarshal(raw, &body))

	names := make([]string, 0, len(keys))
	for name := range keys {
		names = append(names, name)
	}
	return res.StatusCode, names, body
}

// TestErrResponse_LegacyShapeIsUnchanged is the compatibility guard for the five
// published SDKs: the only permitted change to this envelope is the additive
// trace_id, absent here because no span is active.
func TestErrResponse_LegacyShapeIsUnchanged(t *testing.T) {
	status, keys, body := renderLegacy(t, context.Background(),
		aohhttp.ErrResponse(http.StatusBadRequest, "invalid request",
			[]error{errors.New("name is required"), errors.New("body is too long")}))

	assert.Equal(t, http.StatusBadRequest, status)
	assert.ElementsMatch(t, []string{"message", "sent_at", "errors"}, keys,
		"no key added, renamed, removed or retyped")

	assert.Equal(t, "invalid request", body.Message)
	require.Len(t, body.Errors, 2)
	assert.Equal(t, "name is required", body.Errors[0].Message)
	assert.Equal(t, "body is too long", body.Errors[1].Message)
	_, err := time.Parse(time.RFC3339, body.SentAt)
	assert.NoError(t, err, "sent_at keeps its RFC3339 second-precision format")
	assert.Empty(t, body.TraceID)
}

func TestErrResponseWithData_LegacyShapeIsUnchanged(t *testing.T) {
	_, keys, body := renderLegacy(t, context.Background(),
		aohhttp.ErrResponseWithData(http.StatusConflict, "conflict",
			map[string]any{"id": "42"}, []error{errors.New("stale occ_lock")}))

	assert.ElementsMatch(t, []string{"data", "message", "sent_at", "errors"}, keys)
	assert.Equal(t, map[string]any{"id": "42"}, body.Data)
}

// TestErrResponse_GainsTraceIDWhenTraced is the one additive change: correlation
// for every unmigrated call site, without touching any of them.
func TestErrResponse_GainsTraceIDWhenTraced(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()

	_, keys, body := renderLegacy(t, ctx,
		aohhttp.ErrResponse(http.StatusInternalServerError, "database error", nil))

	assert.ElementsMatch(t, []string{"message", "sent_at", "trace_id"}, keys)
	assert.Equal(t, span.SpanContext().TraceID().String(), body.TraceID)
	assert.Regexp(t, hex32, body.TraceID)
}

// TestErrResponse_OmitsTraceIDForAnInvalidSpanContext is the invalid partition:
// an all-zero span context must not produce a zero-filled id.
func TestErrResponse_OmitsTraceIDForAnInvalidSpanContext(t *testing.T) {
	ctx := trace.ContextWithSpanContext(context.Background(),
		trace.NewSpanContext(trace.SpanContextConfig{}))

	_, keys, _ := renderLegacy(t, ctx, aohhttp.ErrResponse(http.StatusNotFound, "not found", nil))

	assert.NotContains(t, keys, "trace_id")
}

// TestErrorResponse_ErrorStringIsUnchanged pins the Error() behaviour the call
// sites that pass the envelope around as an error depend on.
func TestErrorResponse_ErrorStringIsUnchanged(t *testing.T) {
	payload := aohhttp.ErrResponse(http.StatusBadRequest, "invalid", []error{errors.New("boom")})

	asError, ok := payload.(error)
	require.True(t, ok, "ErrorResponse must keep satisfying error")
	assert.Contains(t, asError.Error(), "boom")
}

// TestResponse_SuccessEnvelopeIsUntouched: the success payload is out of scope
// for this change and must not have grown a trace_id.
func TestResponse_SuccessEnvelopeIsUntouched(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()

	req := httptest.NewRequest(http.MethodGet, "/widgets", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	require.NoError(t, render.Render(rec, req,
		aohhttp.Response(http.StatusOK, "ok", map[string]any{"id": "42"})))

	keys := map[string]json.RawMessage{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &keys))
	assert.ElementsMatch(t, []string{"data", "message", "sent_at"}, mapKeys(keys))
}

func mapKeys(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
