// Package aoherr is the AOH error vocabulary and the renderer for the platform
// error contract.
//
// An error is described by a Class (what kind of failure it is) and a Code (the
// machine-readable identity a client may branch on). The renderer turns that
// into exactly one wire shape —
//
//	{timestamp, trace_id, errorCode, errorMessage, details}
//
// — derives the HTTP status from the class, stamps the active span's trace id,
// suppresses internal detail on 5xx, and emits exactly one log record at the
// level the status implies. Call sites choose *what* failed; the renderer owns
// *how* it is presented and reported.
//
// Two things deliberately do not exist here:
//
//   - userMessage and isRetryable. They are presentation fields added by the
//     BFF (see @mssfoobar/errors); a service never emits either.
//   - the transport and timeout failure kinds. They have no Go representation
//     because they describe a client that never reached a service.
//
// The legacy aohhttp envelope ({data, message, sent_at, errors}) is unaffected
// and remains supported for the call sites that have not migrated.
package aoherr

import "net/http"

// Class is the kind of failure, and the single input to the HTTP status the
// renderer emits. Statuses are never chosen per call site.
type Class string

const (
	// ClassValidation is a malformed or invalid request — 400.
	ClassValidation Class = "validation"
	// ClassBusiness is a well-formed request that violates a domain rule — 400.
	// A failure that must present as 409 is a ClassConflict, not a business
	// rule with an overridden status.
	ClassBusiness Class = "business"
	// ClassConflict is a clash with the current state of a resource, such as a
	// stale OCC lock or a duplicate key — 409.
	ClassConflict Class = "conflict"
	// ClassAuthentication is a missing, malformed or expired credential — 401.
	ClassAuthentication Class = "authentication"
	// ClassAuthorization is an authenticated caller lacking permission — 403.
	ClassAuthorization Class = "authorization"
	// ClassNotFound is a resource that does not exist, or that the caller's
	// tenant cannot see — 404.
	ClassNotFound Class = "not-found"
	// ClassSystem is an internal failure: a database error, a bug, a panic —
	// 500. Never 503; an unreachable dependency is ClassUpstreamUnavailable.
	ClassSystem Class = "system"
	// ClassUpstreamUnavailable is a dependency that is unreachable or timed
	// out — 503.
	ClassUpstreamUnavailable Class = "upstream-unavailable"
)

// classStatus is the canonical class -> HTTP status map. It is the whole of the
// mapping: there is no per-call-site override.
var classStatus = map[Class]int{
	ClassValidation:          http.StatusBadRequest,
	ClassBusiness:            http.StatusBadRequest,
	ClassConflict:            http.StatusConflict,
	ClassAuthentication:      http.StatusUnauthorized,
	ClassAuthorization:       http.StatusForbidden,
	ClassNotFound:            http.StatusNotFound,
	ClassSystem:              http.StatusInternalServerError,
	ClassUpstreamUnavailable: http.StatusServiceUnavailable,
}

// classMessage is the developer-facing message used when a call site supplied
// none, and — for the 5xx classes — the only message a client is ever given.
var classMessage = map[Class]string{
	ClassValidation:          "The request is not valid.",
	ClassBusiness:            "The request could not be completed because it violates a business rule.",
	ClassConflict:            "The request conflicts with the current state of the resource.",
	ClassAuthentication:      "The request is not authenticated.",
	ClassAuthorization:       "The caller is not permitted to perform this operation.",
	ClassNotFound:            "The requested resource does not exist.",
	ClassSystem:              "The request could not be completed because of an internal error.",
	ClassUpstreamUnavailable: "A required dependency is currently unavailable.",
}

// Status is the canonical HTTP status for the class. An unrecognised class —
// which no constructor can produce, but a zero value or a bad conversion can —
// degrades to 500 rather than emitting a 200 for a failure.
func (c Class) Status() int {
	if status, ok := classStatus[c]; ok {
		return status
	}
	return http.StatusInternalServerError
}

// Valid reports whether c is one of the declared classes.
func (c Class) Valid() bool {
	_, ok := classStatus[c]
	return ok
}

func (c Class) String() string { return string(c) }

// Classes returns the declared classes, for tests and for tooling that has to
// enumerate the vocabulary.
func Classes() []Class {
	return []Class{
		ClassValidation,
		ClassBusiness,
		ClassConflict,
		ClassAuthentication,
		ClassAuthorization,
		ClassNotFound,
		ClassSystem,
		ClassUpstreamUnavailable,
	}
}
