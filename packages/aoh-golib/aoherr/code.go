package aoherr

import (
	"errors"
	"fmt"
	"regexp"
)

// CodePattern is the required errorCode format: UPPER_SNAKE_CASE, starting with
// a letter, single underscores between non-empty segments.
const CodePattern = `^[A-Z][A-Z0-9]*(_[A-Z0-9]+)*$`

var codeRE = regexp.MustCompile(CodePattern)

// ErrInvalidCode is returned by NewCode for any string that does not match
// CodePattern. Validating at construction is what keeps a non-conforming code
// from ever reaching the wire.
var ErrInvalidCode = errors.New("aoherr: error code must match " + CodePattern)

// Code is the machine-readable identity of a failure. It is part of the API:
// once a client branches on a code, renaming it is a breaking change, and a
// retired value is never reassigned to a different failure.
//
// Codes are module-namespaced by convention (UNH_TEMPLATE_NOT_FOUND). The only
// exceptions are the shared codes below, which exist because the failure is
// genuinely platform-wide and a module prefix would be noise.
type Code string

// The shared, cross-cutting codes. Each is either a failure raised outside any
// module's domain (an unparseable request, an unmatched route, a rejected
// token), or the platform's fallback identity for a class when a call site
// supplies no code of its own. Modules must not use them for domain failures —
// use a namespaced code, so a client can tell one module's 404 from another's.
const (
	// CodeUnauthenticated — the credential is missing, malformed or expired.
	CodeUnauthenticated Code = "UNAUTHENTICATED"
	// CodeForbidden — authenticated, but not permitted.
	CodeForbidden Code = "FORBIDDEN"
	// CodeMalformedRequest — the request body or parameters could not be
	// decoded into the shape the endpoint expects.
	CodeMalformedRequest Code = "MALFORMED_REQUEST"
	// CodeInternal — an unhandled internal failure, including a recovered panic.
	CodeInternal Code = "INTERNAL_ERROR"
	// CodeNotFound — no route matched, or no resource-specific code applies.
	// A module's own "not found" failure should be namespaced instead.
	CodeNotFound Code = "NOT_FOUND"
	// CodeMethodNotAllowed — the path matched a route that does not allow the
	// request's method.
	CodeMethodNotAllowed Code = "METHOD_NOT_ALLOWED"
	// CodeValidationFailed — the platform's fallback for a validation failure
	// a call site did not name.
	CodeValidationFailed Code = "VALIDATION_FAILED"
	// CodeBusinessRuleViolation — fallback for an unnamed business-rule failure.
	CodeBusinessRuleViolation Code = "BUSINESS_RULE_VIOLATION"
	// CodeConflict — fallback for an unnamed state conflict.
	CodeConflict Code = "CONFLICT"
	// CodeUpstreamUnavailable — fallback for an unreachable dependency.
	CodeUpstreamUnavailable Code = "UPSTREAM_UNAVAILABLE"
)

// classCode is the platform fallback code per class, used when a call site's
// code is absent or (by a conversion that bypassed NewCode) non-conforming.
// Falling back per class keeps a rendered code accurate — a validation failure
// never gets labelled INTERNAL_ERROR — and guarantees the wire always carries a
// code that matches CodePattern.
var classCode = map[Class]Code{
	ClassValidation:          CodeValidationFailed,
	ClassBusiness:            CodeBusinessRuleViolation,
	ClassConflict:            CodeConflict,
	ClassAuthentication:      CodeUnauthenticated,
	ClassAuthorization:       CodeForbidden,
	ClassNotFound:            CodeNotFound,
	ClassSystem:              CodeInternal,
	ClassUpstreamUnavailable: CodeUpstreamUnavailable,
}

// NewCode validates s and returns it as a Code. Construct codes this way (or
// with MustCode for package-level constants) so a malformed code fails in the
// service's own tests rather than on a client's screen.
func NewCode(s string) (Code, error) {
	if !codeRE.MatchString(s) {
		return "", fmt.Errorf("%w, got %q", ErrInvalidCode, s)
	}
	return Code(s), nil
}

// MustCode is NewCode for package-level declarations, panicking on a
// non-conforming value so the failure surfaces at init rather than at render.
func MustCode(s string) Code {
	code, err := NewCode(s)
	if err != nil {
		panic(err)
	}
	return code
}

// Valid reports whether c matches CodePattern.
func (c Code) Valid() bool { return codeRE.MatchString(string(c)) }

func (c Code) String() string { return string(c) }

// SharedCodes returns the platform's cross-cutting codes, for tests and tooling
// that enumerate the vocabulary.
func SharedCodes() []Code {
	return []Code{
		CodeUnauthenticated,
		CodeForbidden,
		CodeMalformedRequest,
		CodeInternal,
		CodeNotFound,
		CodeMethodNotAllowed,
		CodeValidationFailed,
		CodeBusinessRuleViolation,
		CodeConflict,
		CodeUpstreamUnavailable,
	}
}
