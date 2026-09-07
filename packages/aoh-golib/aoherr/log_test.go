package aoherr_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// TestRender_LogsExactlyOnce is the one-record rule: the rendering layer is the
// single logging point for a rendered failure.
func TestRender_LogsExactlyOnce(t *testing.T) {
	got := renderErr(t, context.Background(),
		aoherr.New(aoherr.ClassValidation, aoherr.MustCode("UNH_INVALID_TEMPLATE"), "bad"))

	entry := got.only(t)
	assert.Equal(t, "UNH_INVALID_TEMPLATE", fields(entry)["errorCode"])
}

// TestRender_LevelIsDerivedFromStatus walks the status-class boundaries: 4xx
// WARN, 5xx ERROR, never a 4xx at ERROR.
func TestRender_LevelIsDerivedFromStatus(t *testing.T) {
	cases := map[aoherr.Class]zapcore.Level{
		aoherr.ClassValidation:          zapcore.WarnLevel,  // 400
		aoherr.ClassAuthentication:      zapcore.WarnLevel,  // 401
		aoherr.ClassAuthorization:       zapcore.WarnLevel,  // 403
		aoherr.ClassNotFound:            zapcore.WarnLevel,  // 404
		aoherr.ClassConflict:            zapcore.WarnLevel,  // 409
		aoherr.ClassSystem:              zapcore.ErrorLevel, // 500
		aoherr.ClassUpstreamUnavailable: zapcore.ErrorLevel, // 503
	}

	for class, level := range cases {
		t.Run(string(class), func(t *testing.T) {
			got := renderErr(t, context.Background(), aoherr.New(class, aoherr.CodeInternal, "x"))

			entry := got.only(t)
			assert.Equal(t, level, entry.Level)
			if got.status < 500 {
				assert.Less(t, entry.Level, zapcore.ErrorLevel,
					"a %d must never be logged at ERROR or above", got.status)
			}
		})
	}
}

func TestRender_MethodNotAllowedLogsAtWarn(t *testing.T) {
	got := renderErr(t, context.Background(), aoherr.MethodNotAllowedError())

	assert.Equal(t, zapcore.WarnLevel, got.only(t).Level)
}

// TestRender_RecordIsStructuredAndQueryableWithoutATrace backs the fallback
// reportable tuple: errorCode + timestamp must locate the record.
func TestRender_RecordIsStructuredAndQueryableWithoutATrace(t *testing.T) {
	got := renderErr(t, context.Background(),
		aoherr.New(aoherr.ClassNotFound, aoherr.MustCode("UNH_TEMPLATE_NOT_FOUND"), "gone"))

	entry := got.only(t)
	f := fields(entry)
	assert.Equal(t, "UNH_TEMPLATE_NOT_FOUND", f["errorCode"])
	assert.EqualValues(t, http.StatusNotFound, f["status"])
	assert.Equal(t, http.MethodPost, f["method"])
	assert.Equal(t, got.str(t, "timestamp"), f["timestamp"],
		"the record's timestamp must be the one the client was given")
	assert.NotContains(t, entry.Message, "UNH_TEMPLATE_NOT_FOUND",
		"identity belongs in fields, not interpolated into the message")
}

// TestRender_RecordCarriesTheRouteWhenTheRouterMatched exercises the chi route
// context; the unmatched case is covered by the test below it.
func TestRender_RecordCarriesTheRouteWhenTheRouterMatched(t *testing.T) {
	core, recorded := observer.New(zapcore.DebugLevel)

	r := chi.NewRouter()
	r.Get("/widgets/{id}", func(w http.ResponseWriter, req *http.Request) {
		aoherr.RenderWithLogger(zap.New(core), w, req,
			aoherr.New(aoherr.ClassNotFound, aoherr.MustCode("UNH_WIDGET_NOT_FOUND"), "gone"))
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/widgets/42", nil))

	require.Equal(t, http.StatusNotFound, rec.Code)
	entries := recorded.All()
	require.Len(t, entries, 1)
	assert.Equal(t, "/widgets/{id}", fields(entries[0])["route"],
		"the route pattern must be low-cardinality, not the concrete path")
}

func TestRender_RecordOmitsTheRouteWhenNothingMatched(t *testing.T) {
	got := renderErr(t, context.Background(), aoherr.NotFoundError())

	f := fields(got.only(t))
	assert.NotContains(t, f, "route")
	assert.Contains(t, f, "errorCode", "an unmatched route still yields a structured record")
	assert.Contains(t, f, "status")
	assert.Contains(t, f, "method")
	assert.Contains(t, f, "timestamp")
}

// TestRender_RecordCarriesTheActiveTraceContextAutomatically is the D9
// guarantee: the call site passes no correlation fields at all.
func TestRender_RecordCarriesTheActiveTraceContextAutomatically(t *testing.T) {
	ctx, sc := tracedContext(t)

	got := renderErr(t, ctx, aoherr.New(aoherr.ClassSystem, aoherr.CodeInternal, "boom"))

	f := fields(got.only(t))
	assert.Equal(t, sc.TraceID().String(), f["trace_id"])
	assert.Equal(t, sc.SpanID().String(), f["span_id"])
	assert.Equal(t, got.str(t, "trace_id"), f["trace_id"],
		"the id handed to the client must resolve to this record")
}

func TestRender_RecordOmitsTraceContextWithNoSpan(t *testing.T) {
	got := renderErr(t, context.Background(),
		aoherr.New(aoherr.ClassSystem, aoherr.CodeInternal, "boom"))

	f := fields(got.only(t))
	assert.NotContains(t, f, "trace_id")
	assert.NotContains(t, f, "span_id")
	assert.Contains(t, f, "errorCode", "an untraced error is still logged")
}

// TestRender_5xxRecordCarriesTheCauseAndStack keeps the detail suppressed from
// the body available for diagnosis.
func TestRender_5xxRecordCarriesTheCauseAndStack(t *testing.T) {
	cause := errors.New(`pq: relation "widgets" does not exist`)

	got := renderErr(t, context.Background(),
		aoherr.Wrap(aoherr.ClassSystem, aoherr.MustCode("UNH_STORE_FAILED"), "insert template", cause))

	f := fields(got.only(t))
	require.Contains(t, f, "error")
	assert.Contains(t, f["error"], cause.Error())
	require.Contains(t, f, "stack")
	assert.NotEmpty(t, f["stack"])
	assert.NotContains(t, got.body, cause.Error(), "the cause stays out of the response")
}

func TestRender_4xxRecordCarriesNoStackOrCause(t *testing.T) {
	cause := errors.New("token signature invalid for kid=abc")

	got := renderErr(t, context.Background(),
		aoherr.Wrap(aoherr.ClassAuthentication, aoherr.CodeUnauthenticated, "bad token", cause))

	f := fields(got.only(t))
	assert.NotContains(t, f, "stack")
	assert.NotContains(t, f, "error",
		"a 4xx record must not carry the cause, which can embed the credential")
	assert.Equal(t, aoherr.CodeUnauthenticated.String(), f["errorCode"])
}

// TestRender_RecordCarriesNoSubmittedValues keeps raw detail values (which can
// hold credentials or PII) out of the record; the count is enough for triage.
func TestRender_RecordCarriesNoSubmittedValues(t *testing.T) {
	got := renderErr(t, context.Background(),
		aoherr.New(aoherr.ClassValidation, aoherr.MustCode("UNH_INVALID_TEMPLATE"), "invalid").
			WithDetails(aoherr.FieldDetail("password", "must be at least 12 characters")))

	entry := got.only(t)
	f := fields(entry)
	assert.EqualValues(t, 1, f["details_count"])
	assert.NotContains(t, entry.ContextMap(), "details")
	for k, v := range f {
		if s, ok := v.(string); ok {
			assert.NotContains(t, s, "hunter2", "field %q leaked a submitted value", k)
		}
	}
}
