package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/render"
	"github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"
)

// notImplemented is the server-side placeholder for a workshop exercise — see WORKSHOP.md
// at the repo root. It answers 501 in the AOH error contract shape, so the console and
// curl get a real, parseable failure instead of the router's 404, and the exercise is
// visible in the API surface before anyone has written it.
//
// aoherr has no class that maps to 501: its classes describe the platform's *runtime*
// failures, and "not built yet" is not one of them. So the body is written directly with
// the contract's fields. When you implement an exercise, replace its notImplemented call
// in Routes with a real handler — do not extend this.
func notImplemented(exercise string) http.HandlerFunc {
	type body struct {
		Timestamp    string `json:"timestamp"`
		TraceID      string `json:"trace_id,omitempty"`
		ErrorCode    string `json:"errorCode"`
		ErrorMessage string `json:"errorMessage"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		render.Status(r, http.StatusNotImplemented)
		render.JSON(w, r, body{
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
			TraceID:      aoherr.TraceID(r.Context()),
			ErrorCode:    "DISPATCH_NOT_IMPLEMENTED",
			ErrorMessage: "Workshop " + exercise + " is not implemented yet. See WORKSHOP.md.",
		})
	}
}
