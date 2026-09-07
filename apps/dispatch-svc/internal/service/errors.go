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
)
