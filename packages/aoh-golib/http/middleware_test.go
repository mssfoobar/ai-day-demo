package aohhttp_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	aohhttp "github.com/mssfoobar/ops-hub/packages/aoh-golib/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// echoStatus writes body at the given status. The raw write is deliberate: these
// tests assert that ResponseLogger streams a response through byte-for-byte, so
// the fixture needs exact control of the bytes and cannot route through a
// template or an encoder.
func echoStatus(status int, body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		// Test fixture writing a constant JSON literal declared in this file; never
		// request-derived, and the response is application/json, so no XSS surface.
		_, _ = w.Write([]byte(body)) // nosemgrep
	})
}

// TestResponseLogger_StreamsTheBodyByteForByte guards the rewrite: the previous
// implementation buffered every response through httptest.NewRecorder.
func TestResponseLogger_StreamsTheBodyByteForByte(t *testing.T) {
	logger, _ := observed()
	payload := `{"errorCode":"UNH_TEMPLATE_NOT_FOUND","details":[{"field":"id"}]}`

	res := serve(t, aohhttp.ResponseLogger(logger)(echoStatus(http.StatusNotFound, payload)),
		httptest.NewRequest(http.MethodGet, "/templates/7", nil))

	assert.Equal(t, http.StatusNotFound, res.StatusCode)
	got, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Equal(t, payload, string(got))
	assert.Equal(t, "application/json", res.Header.Get("Content-Type"),
		"headers written by the handler must survive")
}

// TestResponseLogger_LargeErrorResponseIsLoggedWithoutItsBody is the bounded
// requirement: no field may grow with the response body.
func TestResponseLogger_LargeErrorResponseIsLoggedWithoutItsBody(t *testing.T) {
	logger, logs := observed()
	needle := strings.Repeat("SECRET-PAYLOAD", 4096)

	res := serve(t, aohhttp.ResponseLogger(logger)(echoStatus(http.StatusInternalServerError, needle)),
		httptest.NewRequest(http.MethodGet, "/widgets", nil))

	got, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Len(t, got, len(needle), "the response reaches the client unchanged")

	entry := onlyRecord(t, logs)
	assert.NotContains(t, entry.Message, "SECRET-PAYLOAD")
	for key, value := range entry.ContextMap() {
		if s, ok := value.(string); ok {
			assert.NotContains(t, s, "SECRET-PAYLOAD", "field %q carries body content", key)
			assert.Less(t, len(s), 256, "field %q grows with the body", key)
		}
	}
	assert.NotContains(t, entry.ContextMap(), "body", "the body field is gone for good")
}

// TestResponseLogger_ClientErrorsAreNotEscalated is the inverted-level fix: the
// previous implementation logged all of 400-599 at ERROR.
func TestResponseLogger_ClientErrorsAreNotEscalated(t *testing.T) {
	for _, status := range []int{
		http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden,
		http.StatusNotFound, http.StatusConflict, 499,
	} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			logger, logs := observed()

			serve(t, aohhttp.ResponseLogger(logger)(echoStatus(status, `{}`)),
				httptest.NewRequest(http.MethodGet, "/widgets", nil))

			entry := onlyRecord(t, logs)
			assert.Equal(t, zapcore.WarnLevel, entry.Level)
			assert.Less(t, entry.Level, zapcore.ErrorLevel)
		})
	}
}

func TestResponseLogger_ServerErrorsAreRecordedAtError(t *testing.T) {
	for _, status := range []int{http.StatusInternalServerError, http.StatusServiceUnavailable, 599} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			logger, logs := observed()

			serve(t, aohhttp.ResponseLogger(logger)(echoStatus(status, `{}`)),
				httptest.NewRequest(http.MethodGet, "/widgets", nil))

			assert.Equal(t, zapcore.ErrorLevel, onlyRecord(t, logs).Level)
		})
	}
}

func TestResponseLogger_SuccessStaysAtDebug(t *testing.T) {
	logger, logs := observed()

	serve(t, aohhttp.ResponseLogger(logger)(echoStatus(http.StatusOK, `{"data":1}`)),
		httptest.NewRequest(http.MethodGet, "/widgets", nil))

	assert.Equal(t, zapcore.DebugLevel, onlyRecord(t, logs).Level)
}

// TestResponseLogger_RecordIsBoundedAccessMetadata pins the field set: enough to
// see traffic, never enough to double-report a failure the renderer logged.
func TestResponseLogger_RecordIsBoundedAccessMetadata(t *testing.T) {
	logger, logs := observed()
	payload := `{"errorCode":"UNH_TEMPLATE_NOT_FOUND"}`
	r := chi.NewRouter()
	r.Use(aohhttp.ResponseLogger(logger))
	r.Get("/templates/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, payload)
	})

	serve(t, r, httptest.NewRequest(http.MethodGet, "/templates/7", nil))

	fields := onlyRecord(t, logs).ContextMap()
	assert.EqualValues(t, http.StatusNotFound, fields["status"])
	assert.Equal(t, http.MethodGet, fields["method"])
	assert.Equal(t, "/templates/{id}", fields["route"])
	assert.EqualValues(t, len(payload), fields["bytes"])
	assert.Contains(t, fields, "duration")
	assert.NotContains(t, fields, "errorCode",
		"the access record must not report the failure the renderer already logged")
	assert.NotContains(t, fields, "error")
	assert.NotContains(t, fields, "stack")
}

// TestResponseLogger_UnwrittenStatusIsRecordedAs200 covers the boundary where a
// handler returns without calling WriteHeader — net/http implies 200.
func TestResponseLogger_UnwrittenStatusIsRecordedAs200(t *testing.T) {
	logger, logs := observed()

	serve(t, aohhttp.ResponseLogger(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})),
		httptest.NewRequest(http.MethodGet, "/widgets", nil))

	fields := onlyRecord(t, logs).ContextMap()
	assert.EqualValues(t, http.StatusOK, fields["status"])
	assert.EqualValues(t, 0, fields["bytes"])
}

func TestResponseLogger_RecordCarriesTheActiveTraceContext(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()

	logger, logs := observed()
	req := httptest.NewRequest(http.MethodGet, "/widgets", nil).WithContext(ctx)

	serve(t, aohhttp.ResponseLogger(logger)(echoStatus(http.StatusInternalServerError, `{}`)), req)

	fields := onlyRecord(t, logs).ContextMap()
	assert.Equal(t, span.SpanContext().TraceID().String(), fields["trace_id"])
	assert.Equal(t, span.SpanContext().SpanID().String(), fields["span_id"])
}

// TestResponseLogger_LegacyEnvelope500StillYieldsALevelledRecord is why the
// access record survives at all: for the unmigrated call sites it is the only
// signal there is.
func TestResponseLogger_LegacyEnvelope500StillYieldsALevelledRecord(t *testing.T) {
	logger, logs := observed()
	legacy := `{"message":"database error","sent_at":"2026-07-28T00:00:00Z","errors":[{"message":"boom"}]}`

	serve(t, aohhttp.ResponseLogger(logger)(echoStatus(http.StatusInternalServerError, legacy)),
		httptest.NewRequest(http.MethodGet, "/widgets", nil))

	entry := onlyRecord(t, logs)
	assert.Equal(t, zapcore.ErrorLevel, entry.Level)
	assert.EqualValues(t, http.StatusInternalServerError, entry.ContextMap()["status"])
	for _, value := range entry.ContextMap() {
		if s, ok := value.(string); ok {
			assert.NotContains(t, s, "database error")
		}
	}
}

func TestRequestLogger_StillLogsAtDebug(t *testing.T) {
	logger, logs := observed()

	serve(t, aohhttp.RequestLogger(logger)(echoStatus(http.StatusOK, `{}`)),
		httptest.NewRequest(http.MethodGet, "/widgets", nil))

	assert.Equal(t, zapcore.DebugLevel, onlyRecord(t, logs).Level)
}

func onlyRecord(t *testing.T, logs *observer.ObservedLogs) observer.LoggedEntry {
	t.Helper()
	all := logs.All()
	require.Len(t, all, 1, "expected exactly one record, got %d", len(all))
	return all[0]
}
