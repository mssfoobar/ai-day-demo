package service

import "github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"

// This service's errorCode namespace.
//
// Codes are UPPER_SNAKE_CASE and prefixed with the module so a code read from a log or a
// support ticket names its origin. MustCode validates the format at init, so a malformed
// code panics on start rather than reaching a client.
var (
	// CodeUnitNotFound — the requested unit_code matches no unit.
	CodeUnitNotFound = aoherr.MustCode("DISPATCH_UNIT_NOT_FOUND")
	// CodeUnitCodeRequired — the request omitted the unit code.
	CodeUnitCodeRequired = aoherr.MustCode("DISPATCH_UNIT_CODE_REQUIRED")
	// CodeUnitReadFailed — the unit store could not be read.
	CodeUnitReadFailed = aoherr.MustCode("DISPATCH_UNIT_READ_FAILED")

	// CodeUnitInvalid — a write failed validation; details name the fields.
	CodeUnitInvalid = aoherr.MustCode("DISPATCH_UNIT_INVALID")
	// CodeUnitCodeTaken — create collided with an existing unit_code.
	CodeUnitCodeTaken = aoherr.MustCode("DISPATCH_UNIT_CODE_TAKEN")
	// CodeUnitStale — the caller's occ_lock no longer matches; someone else wrote first.
	CodeUnitStale = aoherr.MustCode("DISPATCH_UNIT_STALE")
	// CodeUnitWriteFailed — the unit store could not be written.
	CodeUnitWriteFailed = aoherr.MustCode("DISPATCH_UNIT_WRITE_FAILED")
)
