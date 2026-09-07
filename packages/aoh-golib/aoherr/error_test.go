package aoherr_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errCause = errors.New("pq: relation \"widgets\" does not exist")

func TestError_UnwrapReachesTheCause(t *testing.T) {
	e := aoherr.Wrap(aoherr.ClassSystem, aoherr.CodeInternal, "load widget", errCause)

	assert.ErrorIs(t, e, errCause, "the wrapped cause stays reachable via errors.Is")
	assert.Equal(t, errCause, errors.Unwrap(e))
}

// TestError_Is_MatchesClassAndCode lets services declare sentinels and compare
// with errors.Is without exporting comparison helpers.
func TestError_Is_MatchesClassAndCode(t *testing.T) {
	sentinel := aoherr.New(aoherr.ClassNotFound, aoherr.MustCode("UNH_TEMPLATE_NOT_FOUND"), "")
	same := aoherr.Wrap(aoherr.ClassNotFound, aoherr.MustCode("UNH_TEMPLATE_NOT_FOUND"),
		"template 7 is gone", errCause)
	otherCode := aoherr.New(aoherr.ClassNotFound, aoherr.MustCode("UNH_CHANNEL_NOT_FOUND"), "")
	otherClass := aoherr.New(aoherr.ClassConflict, aoherr.MustCode("UNH_TEMPLATE_NOT_FOUND"), "")

	assert.ErrorIs(t, same, sentinel)
	assert.NotErrorIs(t, otherCode, sentinel)
	assert.NotErrorIs(t, otherClass, sentinel)
	assert.NotErrorIs(t, errCause, sentinel)

	// Reachable through a non-aoherr wrapper too.
	assert.ErrorIs(t, fmt.Errorf("handler: %w", same), sentinel)
}

func TestError_As(t *testing.T) {
	wrapped := fmt.Errorf("service: %w",
		aoherr.Wrap(aoherr.ClassConflict, aoherr.MustCode("UNH_STALE_WRITE"), "occ", errCause))

	var target *aoherr.Error
	require.ErrorAs(t, wrapped, &target)
	assert.Equal(t, aoherr.ClassConflict, target.Class())
	assert.Equal(t, http.StatusConflict, target.Status())
}

func TestError_Error_IsDeveloperFacing(t *testing.T) {
	e := aoherr.Wrap(aoherr.ClassSystem, aoherr.CodeInternal, "load widget", errCause)

	msg := e.Error()
	assert.Contains(t, msg, aoherr.CodeInternal.String())
	assert.Contains(t, msg, "load widget")
	assert.Contains(t, msg, errCause.Error(), "the cause belongs in logs, not on the wire")
}

func TestError_Status_DerivedFromClass(t *testing.T) {
	for _, c := range aoherr.Classes() {
		assert.Equal(t, c.Status(), aoherr.New(c, aoherr.CodeInternal, "x").Status())
	}
}

// TestError_WithDetails_DoesNotMutateReceiver keeps package-level sentinels
// safe: decorating one must not accumulate details across requests.
func TestError_WithDetails_DoesNotMutateReceiver(t *testing.T) {
	sentinel := aoherr.New(aoherr.ClassValidation, aoherr.MustCode("UNH_INVALID_TEMPLATE"), "invalid")

	first := sentinel.WithDetails(aoherr.FieldDetail("name", "is required"))
	second := sentinel.WithDetails(aoherr.FieldDetail("body", "is too long"))

	assert.Empty(t, sentinel.Details(), "the sentinel must stay pristine")
	assert.Len(t, first.Details(), 1)
	assert.Len(t, second.Details(), 1)
	assert.Equal(t, "name", first.Details()[0].Field)
	assert.Equal(t, "body", second.Details()[0].Field)
}

func TestError_WithCause_DoesNotMutateReceiver(t *testing.T) {
	sentinel := aoherr.New(aoherr.ClassSystem, aoherr.CodeInternal, "boom")

	decorated := sentinel.WithCause(errCause)

	assert.NoError(t, errors.Unwrap(sentinel))
	assert.ErrorIs(t, decorated, errCause)
}

// TestFrom_PlainErrorBecomesSystemClass is the fallback for an unclassified
// error reaching the renderer: it must become a 500, never a 200 or a panic.
func TestFrom_PlainErrorBecomesSystemClass(t *testing.T) {
	e := aoherr.From(errCause)

	require.NotNil(t, e)
	assert.Equal(t, aoherr.ClassSystem, e.Class())
	assert.Equal(t, aoherr.CodeInternal, e.Code())
	assert.Equal(t, http.StatusInternalServerError, e.Status())
	assert.ErrorIs(t, e, errCause)
}

func TestFrom_FindsTheErrorInAWrappedChain(t *testing.T) {
	inner := aoherr.New(aoherr.ClassAuthorization, aoherr.CodeForbidden, "no")

	e := aoherr.From(fmt.Errorf("handler: %w", fmt.Errorf("service: %w", inner)))

	assert.Same(t, inner, e)
}

func TestFrom_NilIsInternal(t *testing.T) {
	e := aoherr.From(nil)

	require.NotNil(t, e)
	assert.Equal(t, http.StatusInternalServerError, e.Status())
}

func TestFieldDetail(t *testing.T) {
	d := aoherr.FieldDetail("email", "must be a valid address")

	assert.Equal(t, "email", d.Field)
	assert.Equal(t, "must be a valid address", d.Message)
}
