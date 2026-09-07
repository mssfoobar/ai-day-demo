package aoherr_test

import (
	"net/http"
	"testing"

	"github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"
	"github.com/stretchr/testify/assert"
)

// TestClass_Status_CanonicalTable pins the class -> HTTP status map from the
// error-envelope capability. system is 500 (wfe's 503 is the bug this fixes)
// and a business-rule violation is 400, never a 5xx.
func TestClass_Status_CanonicalTable(t *testing.T) {
	want := map[aoherr.Class]int{
		aoherr.ClassValidation:          http.StatusBadRequest,
		aoherr.ClassBusiness:            http.StatusBadRequest,
		aoherr.ClassAuthentication:      http.StatusUnauthorized,
		aoherr.ClassAuthorization:       http.StatusForbidden,
		aoherr.ClassNotFound:            http.StatusNotFound,
		aoherr.ClassConflict:            http.StatusConflict,
		aoherr.ClassSystem:              http.StatusInternalServerError,
		aoherr.ClassUpstreamUnavailable: http.StatusServiceUnavailable,
	}

	for class, status := range want {
		t.Run(string(class), func(t *testing.T) {
			assert.Equal(t, status, class.Status())
		})
	}

	assert.Len(t, want, len(aoherr.Classes()), "every declared class needs a status")
}

func TestClass_Status_SystemIsNot503(t *testing.T) {
	assert.NotEqual(t, http.StatusServiceUnavailable, aoherr.ClassSystem.Status())
}

// TestClass_Status_UnknownClassFallsBackTo500 covers the invalid partition: a
// class value no constructor can produce must still render a valid status.
func TestClass_Status_UnknownClassFallsBackTo500(t *testing.T) {
	assert.Equal(t, http.StatusInternalServerError, aoherr.Class("not-a-class").Status())
	assert.Equal(t, http.StatusInternalServerError, aoherr.Class("").Status())
}

func TestClass_Valid(t *testing.T) {
	for _, c := range aoherr.Classes() {
		assert.True(t, c.Valid(), "declared class %q must be valid", c)
	}
	assert.False(t, aoherr.Class("").Valid())
	assert.False(t, aoherr.Class("VALIDATION").Valid(), "classes are lower-case")
	assert.False(t, aoherr.Class("transport").Valid(), "transport is a TS-only failure kind (D4)")
	assert.False(t, aoherr.Class("timeout").Valid(), "timeout is a TS-only failure kind (D4)")
}
