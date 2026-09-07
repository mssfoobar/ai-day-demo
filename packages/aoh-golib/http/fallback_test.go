package aohhttp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	aohhttp "github.com/mssfoobar/ops-hub/packages/aoh-golib/http"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

// routerWithFallbacks mirrors how a service mounts the shared fallbacks.
func routerWithFallbacks() chi.Router {
	r := chi.NewRouter()
	r.NotFound(aohhttp.NotFoundHandler())
	r.MethodNotAllowed(aohhttp.MethodNotAllowedHandler())
	r.Get("/widgets/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	return r
}

// TestNotFoundHandler_RendersConformantJSON replaces chi's default, which writes
// the plain-text body "404 page not found".
func TestNotFoundHandler_RendersConformantJSON(t *testing.T) {
	res := serve(t, routerWithFallbacks(),
		httptest.NewRequest(http.MethodGet, "/no/such/path", nil))

	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
	keys, body := jsonBody(t, res)
	assert.Equal(t, "NOT_FOUND", strValue(t, keys, "errorCode"))
	assert.NotEmpty(t, strValue(t, keys, "errorMessage"))
	assert.NotEmpty(t, strValue(t, keys, "timestamp"))
	assert.NotContains(t, body, "404 page not found", "chi's plain-text default is gone")
}

// TestMethodNotAllowedHandler_RendersConformantJSON replaces chi's plain-text
// "405 method not allowed".
func TestMethodNotAllowedHandler_RendersConformantJSON(t *testing.T) {
	res := serve(t, routerWithFallbacks(),
		httptest.NewRequest(http.MethodDelete, "/widgets/42", nil))

	assert.Equal(t, http.StatusMethodNotAllowed, res.StatusCode)
	keys, body := jsonBody(t, res)
	assert.Equal(t, "METHOD_NOT_ALLOWED", strValue(t, keys, "errorCode"))
	assert.NotContains(t, body, "405 method not allowed")
}

func TestFallbacks_DoNotShadowAMatchedRoute(t *testing.T) {
	res := serve(t, routerWithFallbacks(),
		httptest.NewRequest(http.MethodGet, "/widgets/42", nil))

	assert.Equal(t, http.StatusNoContent, res.StatusCode)
}

// TestFallbacks_LogAtWarnWithStructuredFields keeps an unmatched route
// queryable: no matched route must not mean an unstructured record.
func TestFallbacks_LogAtWarnWithStructuredFields(t *testing.T) {
	logger, logs := observed()
	r := chi.NewRouter()
	r.NotFound(aohhttp.NotFoundHandlerWithLogger(logger))

	serve(t, r, httptest.NewRequest(http.MethodPatch, "/no/such/path", nil))

	entry := onlyRecord(t, logs)
	assert.Equal(t, zapcore.WarnLevel, entry.Level)
	fields := entry.ContextMap()
	assert.Equal(t, "NOT_FOUND", fields["errorCode"])
	assert.EqualValues(t, http.StatusNotFound, fields["status"])
	assert.Equal(t, http.MethodPatch, fields["method"])
	assert.Contains(t, fields, "timestamp")
	assert.NotContains(t, fields, "route", "nothing matched, so there is no route")
}

func TestMethodNotAllowedHandler_LogsAtWarn(t *testing.T) {
	logger, logs := observed()
	r := chi.NewRouter()
	r.MethodNotAllowed(aohhttp.MethodNotAllowedHandlerWithLogger(logger))
	r.Get("/widgets", func(w http.ResponseWriter, _ *http.Request) {})

	serve(t, r, httptest.NewRequest(http.MethodPost, "/widgets", nil))

	entry := onlyRecord(t, logs)
	assert.Equal(t, zapcore.WarnLevel, entry.Level)
	assert.Equal(t, "METHOD_NOT_ALLOWED", entry.ContextMap()["errorCode"])
}
