package aoherr_test

import (
	"context"
	"testing"

	"github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewCode_ConformingPartitions covers the valid partitions of
// ^[A-Z][A-Z0-9]*(_[A-Z0-9]+)*$ plus its boundaries: the shortest legal value
// (one character), a digit-bearing segment, and a multi-segment namespaced code.
func TestNewCode_ConformingPartitions(t *testing.T) {
	for _, in := range []string{
		"A",                      // boundary: minimum length
		"CONFLICT",               // single segment
		"UNH_TEMPLATE_NOT_FOUND", // module-namespaced, three separators
		"DB_TIMEOUT_2",           // trailing digit segment
		"A0",                     // digit inside the first segment
		"A_0",                    // segment that is only digits
	} {
		t.Run(in, func(t *testing.T) {
			code, err := aoherr.NewCode(in)
			require.NoError(t, err)
			assert.Equal(t, in, code.String(), "code must serialize byte-for-byte as given")
			assert.True(t, code.Valid())
		})
	}
}

// TestNewCode_NonConformingPartitions covers every invalid partition named in
// the error-envelope capability plus the separator boundaries.
func TestNewCode_NonConformingPartitions(t *testing.T) {
	for name, in := range map[string]string{
		"empty":               "",
		"lowercase":           "unh_not_found",
		"mixed case":          "Unh_Not_Found",
		"leading digit":       "1_BAD_CODE",
		"leading underscore":  "_LEADING",
		"bare underscore":     "_",
		"trailing underscore": "TRAILING_",
		"double underscore":   "DOUBLE__UNDERSCORE",
		"hyphen":              "HAS-HYPHEN",
		"space":               "HAS SPACE",
		"prose":               "database error",
		"newline":             "CODE\n",
		"non-ascii":           "CÓDIGO",
	} {
		t.Run(name, func(t *testing.T) {
			code, err := aoherr.NewCode(in)
			require.Error(t, err)
			assert.ErrorIs(t, err, aoherr.ErrInvalidCode)
			assert.Empty(t, code.String(), "a rejected code must not be returned")
			assert.False(t, aoherr.Code(in).Valid())
		})
	}
}

func TestMustCode_PanicsOnNonConforming(t *testing.T) {
	assert.Panics(t, func() { aoherr.MustCode("bad code") })
	assert.NotPanics(t, func() { aoherr.MustCode("GOOD_CODE") })
}

// TestSharedCodes_AreConforming guards the platform's own vocabulary against
// the format gate — these are typed constants, so nothing else checks them.
func TestSharedCodes_AreConforming(t *testing.T) {
	for _, c := range aoherr.SharedCodes() {
		assert.True(t, c.Valid(), "shared code %q must match the required format", c)
	}
	assert.NotEmpty(t, aoherr.SharedCodes())
}

// TestRender_NonConformingCodeNeverSerialized is the invalid-partition end of
// the format rule: a code smuggled in by direct conversion (bypassing the
// constructor) must not reach the wire.
func TestRender_NonConformingCodeNeverSerialized(t *testing.T) {
	got := renderErr(t, context.Background(),
		aoherr.New(aoherr.ClassValidation, aoherr.Code("database error"), "boom"))

	code := got.str(t, "errorCode")
	assert.NotEqual(t, "database error", code)
	assert.True(t, aoherr.Code(code).Valid(), "rendered code %q must conform", code)
}

// TestRender_CrossCuttingFailuresUseOneSharedCode is the namespacing rule's
// other half: two services rejecting a request for the same cross-cutting reason
// must emit the identical value, so a client can branch on it once.
func TestRender_CrossCuttingFailuresUseOneSharedCode(t *testing.T) {
	shared := map[aoherr.Class]aoherr.Code{
		aoherr.ClassAuthentication: aoherr.CodeUnauthenticated,
		aoherr.ClassAuthorization:  aoherr.CodeForbidden,
	}

	for class, want := range shared {
		t.Run(string(class), func(t *testing.T) {
			serviceA := renderErr(t, context.Background(), aoherr.New(class, "", "rejected"))
			serviceB := renderErr(t, context.Background(), aoherr.New(class, want, "rejected"))

			assert.Equal(t, want.String(), serviceA.str(t, "errorCode"))
			assert.Equal(t, serviceA.str(t, "errorCode"), serviceB.str(t, "errorCode"))
		})
	}
}

func TestRender_EmptyCodeFallsBackToClassCode(t *testing.T) {
	got := renderErr(t, context.Background(),
		aoherr.New(aoherr.ClassNotFound, aoherr.Code(""), "nope"))

	assert.Equal(t, aoherr.CodeNotFound.String(), got.str(t, "errorCode"))
}
