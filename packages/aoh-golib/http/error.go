package aohhttp

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"
)

type errorMsg struct {
	Message string `json:"message"`
}

// ErrorResponse is the legacy AOH error envelope, emitted at 363 call sites and
// parsed by five published SDKs. It is deprecated for new code — use aoherr,
// which renders the platform error contract — and its existing fields are
// frozen: renaming, removing or retyping any of them is a breaking change for
// every consumer.
//
// The one permitted change is additive: TraceID, stamped at render time under the
// same rules as the contract body, so the unmigrated call sites get correlation
// now instead of after five migration PRs.
type ErrorResponse struct {
	HttpStatusCode int        `json:"-"`
	Data           any        `json:"data,omitempty"`
	Message        string     `json:"message,omitempty"`
	SentAt         string     `json:"sent_at,omitempty"`
	Errors         []errorMsg `json:"errors,omitempty"`
	// TraceID is the active span's trace id, omitted when no span is active.
	// It is never minted or forwarded by hand; an id that correlates to nothing
	// is worse than none.
	TraceID string `json:"trace_id,omitempty"`
}

func (e *ErrorResponse) Error() string {
	return fmt.Sprint(e.Errors)
}

func (e *ErrorResponse) Render(w http.ResponseWriter, r *http.Request) error {
	e.TraceID = aoherr.TraceID(r.Context())
	render.Status(r, e.HttpStatusCode)
	return nil
}

func ErrResponse(code int, message string, errors []error) render.Renderer {
	var errMsg []errorMsg
	for _, v := range errors {
		errMsg = append(errMsg, errorMsg{Message: v.Error()})
	}
	return &ErrorResponse{
		HttpStatusCode: code,
		Message:        message,
		SentAt:         time.Now().UTC().Format(time.RFC3339),
		Errors:         errMsg,
	}
}

func ErrResponseWithData(code int, message string, data any, errors []error) render.Renderer {
	var errMsg []errorMsg
	for _, v := range errors {
		errMsg = append(errMsg, errorMsg{Message: v.Error()})
	}
	return &ErrorResponse{
		HttpStatusCode: code,
		Data:           data,
		Message:        message,
		SentAt:         time.Now().UTC().Format(time.RFC3339),
		Errors:         errMsg,
	}
}
