package aoherr

import (
	"errors"
	"net/http"
	"strings"
)

// Detail is one descriptive element of a failure: the offending field and why it
// failed.
//
// The wire contract pins an element to exactly two members, `field` and
// `message`, so this is a STRUCT rather than a map — the element schema is then
// enforced by the type system instead of by a render-time key filter. That
// distinction is load-bearing, not stylistic: while this was a `map[string]any`
// a call site could put any key it liked into a 400 body, and the reviewers
// demonstrated a rendered 400 carrying both `{"submitted":"hunter2"}` and a
// driver string with an absolute source path. A struct makes that unrepresentable.
//
// It is also why retryability and user-facing wording are not expressible here:
// they are resolved once by the presentation layer, and with a struct a call
// site cannot smuggle them in under any spelling.
type Detail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// FieldDetail is the constructor: one failing field and why it failed. message
// must be safe to show a user — it is the one part of `details` the presentation
// layer may render inline, next to the control named by field.
func FieldDetail(field, message string) Detail {
	return Detail{Field: field, Message: message}
}

// sanitizeDetails drops any element that cannot be serialized conformantly.
// The contract requires both members to be non-empty, so a half-populated
// element is dropped rather than emitted as a non-conformant `{"field":""}` —
// the renderer owns wire conformance, and emitting a bad element would push that
// obligation onto every call site.
func sanitizeDetails(details []Detail) []Detail {
	var out []Detail
	for _, d := range details {
		if d.Field == "" || d.Message == "" {
			continue
		}
		out = append(out, d)
	}
	return out
}

// Error is a classified AOH failure. Its fields are unexported so that the
// class, the code and the details can only be set through the constructors —
// which is what lets the renderer guarantee the wire shape.
type Error struct {
	class   Class
	code    Code
	message string
	cause   error
	details []Detail
	stack   []byte
	// status pins the HTTP status when the failure has no member in the
	// canonical class table. Unexported and set only inside this package: a
	// call site cannot choose a status.
	status int
}

// New returns a classified error. message is developer-facing — it is never
// shown to a user, and on a 5xx it is never sent to the client at all.
func New(class Class, code Code, message string) *Error {
	return &Error{class: class, code: code, message: message}
}

// Wrap is New with the underlying cause attached. The cause reaches the log
// record on a 5xx and never reaches the response body.
func Wrap(class Class, code Code, message string, cause error) *Error {
	return &Error{class: class, code: code, message: message, cause: cause}
}

// NotFoundError is the failure the shared router fallback renders for a path no
// route matched.
func NotFoundError() *Error {
	return New(ClassNotFound, CodeNotFound, "No route matches the requested path.")
}

// MethodNotAllowedError is the failure the shared router fallback renders when
// the path matched a route that does not allow the request's method.
//
// 405 has no member in the canonical class table (which covers the eight
// application error classes), so this is the one place in the library that pins
// a status directly. The class is recorded as validation because the fault is
// the client's.
func MethodNotAllowedError() *Error {
	e := New(ClassValidation, CodeMethodNotAllowed,
		"The HTTP method is not allowed for this resource.")
	e.status = http.StatusMethodNotAllowed
	return e
}

// From returns the first *Error in err's chain. An error carrying none becomes
// a system-class failure, so an unclassified error renders as a suppressed 500
// instead of leaking or crashing.
func From(err error) *Error {
	var target *Error
	if errors.As(err, &target) {
		return target
	}
	return Wrap(ClassSystem, CodeInternal, "unclassified error", err)
}

func (e *Error) Error() string {
	var b strings.Builder
	b.WriteString(string(e.effectiveCode()))
	if e.message != "" {
		b.WriteString(": ")
		b.WriteString(e.message)
	}
	if e.cause != nil {
		b.WriteString(": ")
		b.WriteString(e.cause.Error())
	}
	return b.String()
}

// Unwrap exposes the cause to errors.Is/errors.As.
func (e *Error) Unwrap() error { return e.cause }

// Is matches on identity — class and code — so a service can declare package
// level sentinels and compare with errors.Is regardless of the message, the
// cause or the details a particular render attached.
func (e *Error) Is(target error) bool {
	other, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.class == other.class && e.code == other.code
}

func (e *Error) Class() Class    { return e.class }
func (e *Error) Code() Code      { return e.code }
func (e *Error) Message() string { return e.message }
func (e *Error) Cause() error    { return e.cause }
func (e *Error) Stack() []byte   { return e.stack }

// Status is the HTTP status this error renders as.
func (e *Error) Status() int {
	if e.status != 0 {
		return e.status
	}
	return e.class.Status()
}

// Details returns the sanitized details — what the wire would carry, control
// keys already removed.
func (e *Error) Details() []Detail {
	return sanitizeDetails(e.details)
}

// WithDetails returns a copy carrying details. It copies rather than mutates so
// that a package-level sentinel can be decorated per request without
// accumulating another request's details.
func (e *Error) WithDetails(details ...Detail) *Error {
	clone := e.clone()
	clone.details = append(clone.details, details...)
	return clone
}

// WithCause returns a copy with cause attached.
func (e *Error) WithCause(cause error) *Error {
	clone := e.clone()
	clone.cause = cause
	return clone
}

// WithStack returns a copy carrying an already-captured stack — the panic
// recovery path uses it so the record shows where the panic happened rather
// than where it was rendered.
func (e *Error) WithStack(stack []byte) *Error {
	clone := e.clone()
	clone.stack = stack
	return clone
}

func (e *Error) clone() *Error {
	out := *e
	out.details = append([]Detail(nil), e.details...)
	return &out
}

// effectiveCode is the code that reaches the wire: the call site's when it
// conforms, the class fallback otherwise. A code built by conversion instead of
// NewCode is the only way a non-conforming value gets this far.
func (e *Error) effectiveCode() Code {
	if e.code.Valid() {
		return e.code
	}
	if fallback, ok := classCode[e.class]; ok {
		return fallback
	}
	return CodeInternal
}

// publicMessage is the errorMessage the client receives. It is always a
// non-empty, developer-facing string that is never the code itself, and for a
// 5xx it is always the class's generic message — the call site's message can
// embed the cause, so suppression cannot trust it.
func (e *Error) publicMessage() string {
	if e.Status() >= http.StatusInternalServerError {
		return e.classMessage()
	}
	if e.message != "" {
		return e.message
	}
	return e.classMessage()
}

func (e *Error) classMessage() string {
	if msg, ok := classMessage[e.class]; ok {
		return msg
	}
	return classMessage[ClassSystem]
}
