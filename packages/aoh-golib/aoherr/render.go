package aoherr

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	aohlog "github.com/mssfoobar/ops-hub/packages/aoh-golib/logger"
	"go.uber.org/zap"
)

// timestampLayout is RFC 3339 at millisecond precision, in UTC. Milliseconds
// are fixed-width so the field is directly comparable and sortable as a string.
const timestampLayout = "2006-01-02T15:04:05.000Z"

// contentType is set explicitly rather than left to the client to sniff.
const contentType = "application/json"

// logMessage is a constant so the record's identity lives in its fields. Never
// interpolate the code or the path into it — that is what makes a record
// unqueryable.
const logMessage = "error response"

// wireError is the AOH error contract. These five keys are the whole of it:
// nothing is added, and trace_id/details are omitted rather than emitted empty.
//
// It is unexported on purpose. The only way to produce the shape is to render an
// Error, so no call site can hand-assemble a near-conformant body.
type wireError struct {
	Timestamp    string   `json:"timestamp"`
	TraceID      string   `json:"trace_id,omitempty"`
	ErrorCode    string   `json:"errorCode"`
	ErrorMessage string   `json:"errorMessage"`
	Details      []Detail `json:"details,omitempty"`
}

// Render writes err as the AOH error contract using the package-default logger.
//
// It is the single point where a failure becomes a response *and* a log record:
// the status comes from the class, the trace id from the active span, internal
// detail is suppressed on 5xx, and exactly one record is emitted at the level
// the status implies (4xx WARN, 5xx ERROR). Layers below — repositories,
// services, wrappers — must not log the failure they return.
func Render(w http.ResponseWriter, r *http.Request, err error) {
	RenderWithLogger(nil, w, r, err)
}

// RenderWithLogger is Render for the middlewares and handlers that hold their
// own *zap.Logger. A nil logger means the package default, resolved per request
// — so a middleware built before aohlog.SetDevelopment() still logs to the
// current sink. The trace context is attached from the request either way.
func RenderWithLogger(logger *zap.Logger, w http.ResponseWriter, r *http.Request, err error) {
	e := From(err)
	status := e.Status()
	payload := e.wire(r.Context(), time.Now())

	// Log before writing, so the record exists even if the client is gone.
	logRendered(logger, r, e, status, payload)
	writeWire(w, status, payload)
}

// wire builds the response payload. Suppression lives here, not at the call
// site: on a 5xx the message is the class's generic one and details are dropped
// entirely, because a call site can put the cause in either.
func (e *Error) wire(ctx context.Context, now time.Time) wireError {
	out := wireError{
		Timestamp:    now.UTC().Format(timestampLayout),
		TraceID:      TraceID(ctx),
		ErrorCode:    e.effectiveCode().String(),
		ErrorMessage: e.publicMessage(),
	}
	if e.Status() < http.StatusInternalServerError {
		out.Details = sanitizeDetails(e.details)
	}
	return out
}

func writeWire(w http.ResponseWriter, status int, payload wireError) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	// The client being gone is the only realistic failure here, and there is
	// nowhere left to report it.
	_ = json.NewEncoder(w).Encode(payload)
}

// logRendered emits the one record that reports this failure. It carries the
// identity a query needs (code, status, method, route, timestamp) plus the
// automatic trace context, and on a 5xx the cause and stack that the response
// body is not allowed to contain.
//
// Deliberately absent: the request body, any header, and the detail *values* —
// which can hold submitted credentials or PII. The detail count is what triage
// actually needs. On a 4xx the cause is omitted too: for a rejected credential
// it can embed the token.
func logRendered(logger *zap.Logger, r *http.Request, e *Error, status int, payload wireError) {
	fields := []zap.Field{
		zap.String("errorCode", payload.ErrorCode),
		zap.Int("status", status),
		zap.String("method", r.Method),
		zap.String("timestamp", payload.Timestamp),
		zap.String("class", e.class.String()),
		zap.String("errorMessage", payload.ErrorMessage),
	}
	if route := routePattern(r); route != "" {
		fields = append(fields, zap.String("route", route))
	}
	if count := len(e.details); count > 0 {
		fields = append(fields, zap.Int("details_count", count))
	}

	l := aohlog.WithCtx(r.Context(), logger)
	if status < http.StatusInternalServerError {
		l.Warn(logMessage, fields...)
		return
	}
	l.Error(logMessage, append(fields, zap.Error(e), stackField(e))...)
}

// stackField prefers a stack captured where the failure happened (the panic
// path attaches one) over the render site's.
func stackField(e *Error) zap.Field {
	if len(e.stack) > 0 {
		return zap.ByteString("stack", e.stack)
	}
	return zap.StackSkip("stack", 2)
}

// routePattern is the low-cardinality route the router matched, or "" when
// nothing matched — an unmatched route still yields a structured record, it just
// has no route field.
func routePattern(r *http.Request) string {
	rc := chi.RouteContext(r.Context())
	if rc == nil {
		return ""
	}
	return rc.RoutePattern()
}
