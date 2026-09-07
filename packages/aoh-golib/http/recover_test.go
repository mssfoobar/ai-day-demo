package aohhttp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	aohhttp "github.com/mssfoobar/ops-hub/packages/aoh-golib/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/zap/zapcore"
)

var hex32 = regexp.MustCompile(`^[0-9a-f]{32}$`)

const panicValue = "widget store exploded at /srv/app/internal/repo/widget.go:88"

func panickingHandler() http.Handler {
	return http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(panicValue)
	})
}

// TestRecoverer_PanicRendersConformant500 replaces chi middleware.Recoverer,
// whose recovery writes a bare 500 with a zero-length body.
func TestRecoverer_PanicRendersConformant500(t *testing.T) {
	logger, logs := observed()

	res := serve(t, aohhttp.RecovererWithLogger(logger)(panickingHandler()),
		httptest.NewRequest(http.MethodGet, "/widgets", nil))

	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
	keys, body := jsonBody(t, res)
	assert.Equal(t, "INTERNAL_ERROR", strValue(t, keys, "errorCode"))
	assert.NotEmpty(t, strValue(t, keys, "errorMessage"))
	assert.NotEmpty(t, strValue(t, keys, "timestamp"))

	// Neither the panic value nor any stack frame may reach the client.
	assert.NotContains(t, body, panicValue)
	assert.NotContains(t, body, "/srv/app")
	assert.NotContains(t, body, "goroutine")
	assert.NotContains(t, body, "recover_test.go")

	require.Len(t, logs.All(), 1, "the panic path obeys the one-record rule")
	entry := logs.All()[0]
	assert.Equal(t, zapcore.ErrorLevel, entry.Level)
	fields := entry.ContextMap()
	assert.Contains(t, fields["error"], panicValue, "the panic value belongs in the record")
	assert.Contains(t, fields, "stack")
	assert.Contains(t, fields["stack"], "recover_test.go",
		"the stack must locate the panicking code, not the render site")
}

// TestRecoverer_PanicIsRecordedOnTheSpanAndCorrelated ties the response the
// client got to the trace an operator will open.
func TestRecoverer_PanicIsRecordedOnTheSpanAndCorrelated(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	ctx, span := tp.Tracer("test").Start(context.Background(), "GET /widgets")
	logger, logs := observed()
	req := httptest.NewRequest(http.MethodGet, "/widgets", nil).WithContext(ctx)

	res := serve(t, aohhttp.RecovererWithLogger(logger)(panickingHandler()), req)
	span.End()

	keys, _ := jsonBody(t, res)
	traceID := strValue(t, keys, "trace_id")
	assert.Regexp(t, hex32, traceID)
	assert.Equal(t, span.SpanContext().TraceID().String(), traceID)

	require.Len(t, logs.All(), 1)
	assert.Equal(t, traceID, logs.All()[0].ContextMap()["trace_id"],
		"the id handed to the client must resolve to the record")

	ended := recorder.Ended()
	require.Len(t, ended, 1)
	assert.Equal(t, codes.Error, ended[0].Status().Code)
	require.NotEmpty(t, ended[0].Events(), "the panic must be recorded as a span exception")
	assert.Equal(t, "exception", ended[0].Events()[0].Name)
}

func TestRecoverer_PanicWithNoActiveSpanOmitsTraceID(t *testing.T) {
	logger, logs := observed()

	res := serve(t, aohhttp.RecovererWithLogger(logger)(panickingHandler()),
		httptest.NewRequest(http.MethodGet, "/widgets", nil))

	keys, body := jsonBody(t, res)
	assert.NotContains(t, keys, "trace_id")
	assert.NotContains(t, body, "unknown", "no substitute identifier is minted")

	require.Len(t, logs.All(), 1, "an untraced panic is still logged")
	fields := logs.All()[0].ContextMap()
	assert.Equal(t, zapcore.ErrorLevel, logs.All()[0].Level)
	assert.Contains(t, fields, "stack")
	assert.NotContains(t, fields, "trace_id")
}

// TestRecoverer_ErrAbortHandlerIsRepanicked keeps net/http's documented abort
// signal working — swallowing it would turn a deliberate connection abort into
// a 500.
func TestRecoverer_ErrAbortHandlerIsRepanicked(t *testing.T) {
	logger, logs := observed()
	handler := aohhttp.RecovererWithLogger(logger)(http.HandlerFunc(
		func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) },
	))

	assert.PanicsWithError(t, http.ErrAbortHandler.Error(), func() {
		handler.ServeHTTP(httptest.NewRecorder(),
			httptest.NewRequest(http.MethodGet, "/widgets", nil))
	})
	assert.Empty(t, logs.All(), "an abort is not a failure to report")
}

func TestRecoverer_PanicWithAnErrorValueKeepsTheChain(t *testing.T) {
	logger, logs := observed()
	cause := assertableError("db handle closed")
	handler := aohhttp.RecovererWithLogger(logger)(http.HandlerFunc(
		func(http.ResponseWriter, *http.Request) { panic(cause) },
	))

	res := serve(t, handler, httptest.NewRequest(http.MethodGet, "/widgets", nil))

	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
	require.Len(t, logs.All(), 1)
	assert.Contains(t, logs.All()[0].ContextMap()["error"], "db handle closed")
}

func TestRecoverer_NoPanicIsTransparent(t *testing.T) {
	logger, logs := observed()
	handler := aohhttp.RecovererWithLogger(logger)(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"data":"ok"}`))
		},
	))

	res := serve(t, handler, httptest.NewRequest(http.MethodPost, "/widgets", nil))

	assert.Equal(t, http.StatusCreated, res.StatusCode)
	assert.Empty(t, logs.All())
}

// TestRecoverer_IsAChiDropInReplacement pins the signature the five mounting
// services use: r.Use(aohhttp.Recoverer), exactly as chi's Recoverer.
func TestRecoverer_IsAChiDropInReplacement(t *testing.T) {
	var mw func(http.Handler) http.Handler = aohhttp.Recoverer

	res := serve(t, mw(panickingHandler()), httptest.NewRequest(http.MethodGet, "/widgets", nil))

	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
	keys, _ := jsonBody(t, res)
	assert.Equal(t, "INTERNAL_ERROR", strValue(t, keys, "errorCode"))
}

type assertableError string

func (e assertableError) Error() string { return string(e) }
