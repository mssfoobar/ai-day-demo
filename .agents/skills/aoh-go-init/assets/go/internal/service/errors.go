package service

import "errors"

type ErrorCode string

const (
	ErrNotFound   ErrorCode = "NOT_FOUND"
	ErrValidation ErrorCode = "VALIDATION"
	ErrForbidden  ErrorCode = "FORBIDDEN"
	ErrConflict   ErrorCode = "CONFLICT"
)

// ValidationDetail describes a single field-level validation failure.
// Handlers map this into a 400 response body shaped like
// {"error":"validation","details":[{"field":"...","message":"..."}]}.
type ValidationDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type Error struct {
	Code    ErrorCode
	Message string
	Details []ValidationDetail
	Err     error
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Err }

func NewError(code ErrorCode, msg string, err error) *Error {
	return &Error{Code: code, Message: msg, Err: err}
}

// NewValidationError returns an ErrValidation with field-level details.
// Use this when rejecting input so handlers can surface per-field messages
// to API consumers and UIs.
func NewValidationError(msg string, details []ValidationDetail) *Error {
	return &Error{Code: ErrValidation, Message: msg, Details: details}
}

func GetErrorCode(err error) ErrorCode {
	var svcErr *Error
	if errors.As(err, &svcErr) {
		return svcErr.Code
	}
	return ""
}

// GetValidationDetails extracts the per-field details from a validation error,
// or returns nil for any other error.
func GetValidationDetails(err error) []ValidationDetail {
	var svcErr *Error
	if errors.As(err, &svcErr) {
		return svcErr.Details
	}
	return nil
}
