package aohhttp_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// observed returns a logger writing into an in-memory sink, so a test can
// assert on the records a middleware emitted — level and fields both.
func observed() (*zap.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zapcore.DebugLevel)
	return zap.New(core), logs
}

// jsonBody reads the response body and decodes it as a bag of raw keys, so a
// test asserts the marshalled JSON rather than a Go struct.
func jsonBody(t *testing.T, res *http.Response) (map[string]json.RawMessage, string) {
	t.Helper()

	raw, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
	require.NotEmpty(t, raw, "a 4xx/5xx must never have a zero-length body")

	keys := map[string]json.RawMessage{}
	require.NoError(t, json.Unmarshal(raw, &keys),
		"body must be a single JSON object, got: %s", raw)
	return keys, string(raw)
}

// serve runs one request through h and returns the response.
func serve(t *testing.T, h http.Handler, req *http.Request) *http.Response {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Result()
}

// strValue decodes a string-valued key from a decoded body.
func strValue(t *testing.T, keys map[string]json.RawMessage, key string) string {
	t.Helper()
	raw, ok := keys[key]
	require.True(t, ok, "body key %q absent", key)
	var s string
	require.NoError(t, json.Unmarshal(raw, &s))
	return s
}
