package aohhttp_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	aohhttp "github.com/mssfoobar/ops-hub/packages/aoh-golib/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// healthRouter wires the health endpoints the way a service does.
func healthRouter(t *testing.T, readiness map[string]aohhttp.Check) *chi.Mux {
	t.Helper()
	r := chi.NewRouter()
	h := aohhttp.NewHealthCheck(r)
	for name, check := range readiness {
		h.AddReadinessCheck(name, check)
	}
	return r
}

// TestReadyEndpoint_PassingProbeKeepsItsEmpty200 pins the unchanged happy path:
// probes match on the status, so a 200 is deliberately left bodiless.
func TestReadyEndpoint_PassingProbeKeepsItsEmpty200(t *testing.T) {
	r := healthRouter(t, map[string]aohhttp.Check{
		"db": func() error { return nil },
	})

	res := serve(t, r, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	assert.Equal(t, http.StatusOK, res.StatusCode)
	_, body := jsonBodyOrEmpty(t, res)
	assert.Empty(t, body)
}

// TestReadyEndpoint_FailingProbeRendersTheContractShape is the regression: a
// failing readiness probe used to be a bare 503 — no body for the caller, and the
// per-check causes were collected and then discarded, so nothing recorded which
// dependency was down.
func TestReadyEndpoint_FailingProbeRendersTheContractShape(t *testing.T) {
	r := healthRouter(t, map[string]aohhttp.Check{
		"db":   func() error { return errors.New("dial tcp 10.0.3.14:5432: connect: connection refused") },
		"nats": func() error { return nil },
	})

	res := serve(t, r, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	assert.Equal(t, http.StatusServiceUnavailable, res.StatusCode)
	keys, body := jsonBody(t, res)
	require.NotEmpty(t, body, "a 503 must not have a zero-length body")

	assert.Equal(t, "UPSTREAM_UNAVAILABLE", strValue(t, keys, "errorCode"))
	assert.NotEmpty(t, strValue(t, keys, "errorMessage"))
	assert.NotEmpty(t, strValue(t, keys, "timestamp"))

	// Suppression still applies at 503: the internal host and port the check
	// reported must reach the log record, never the response body.
	for _, leak := range []string{"10.0.3.14", "5432", "connection refused", "dial tcp"} {
		assert.NotContains(t, body, leak)
	}
	assert.NotContains(t, keys, "details", "details cannot survive suppression at 5xx")
}

// jsonBodyOrEmpty tolerates an empty body, which the 200 path deliberately has.
func jsonBodyOrEmpty(t *testing.T, res *http.Response) (map[string]json.RawMessage, string) {
	t.Helper()
	defer func() { _ = res.Body.Close() }()
	raw, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	if len(raw) == 0 {
		return nil, ""
	}
	var keys map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &keys))
	return keys, string(raw)
}
