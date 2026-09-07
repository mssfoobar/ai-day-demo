package aohhttp_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/go-chi/render"
	aohhttp "github.com/mssfoobar/ops-hub/packages/aoh-golib/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sentAtField matches the one field that cannot be pinned: sent_at is time.Now().
var sentAtField = regexp.MustCompile(`"sent_at":"[^"]*"`)

const goldenSentAt = `"sent_at":"2026-01-01T00:00:00Z"`

// TestLegacyEnvelope_MatchesGoldenFixture is the byte-comparison the error-envelope
// spec requires, and the only thing in the repo that actually pins the promise the
// whole change rests on: that the legacy envelope — emitted at ~356 call sites across
// five unmigrated services and parsed by four published *HttpError classes — did not
// change shape.
//
// A byte comparison rather than a field-by-field assertion is deliberate. Those call
// sites are not covered by any other test, so the failure mode to catch is a *silent*
// alteration: a renamed json tag, a dropped omitempty, a reordered field. Any of those
// is a breaking change for every SDK consumer, and none of them would fail a test that
// merely checked "message is present".
//
// If this fails, the question is not "update the fixture" — it is whether the change
// that broke it is a deliberate major for the published SDKs.
func TestLegacyEnvelope_MatchesGoldenFixture(t *testing.T) {
	want, err := os.ReadFile("testdata/legacy-envelope.golden.json")
	require.NoError(t, err, "the golden fixture the spec mandates must exist")

	r := httptest.NewRequest(http.MethodPost, "/v1/thing", nil)
	w := httptest.NewRecorder()

	// Every legacy field populated, so the fixture pins all four rather than only
	// the ones a sparser case happens to emit.
	err = render.Render(w, r, aohhttp.ErrResponseWithData(
		http.StatusBadRequest, "validation",
		map[string]any{"id": "abc"},
		[]error{errors.New("name: required"), errors.New("size: out of range")},
	))
	require.NoError(t, err)

	got := sentAtField.ReplaceAllString(w.Body.String(), goldenSentAt)
	assert.Equal(t, string(want), got,
		"the legacy envelope changed shape; this is a breaking change for @mssfoobar/{gis,wfe,unh,dash,form}-* consumers")
}

// TestLegacyEnvelope_TraceIDIsAbsentWithoutASpan pins the other half of the additive
// promise: the field this change added to the legacy envelope is `omitempty`, so an
// untraced legacy response is byte-identical to the pre-change one. Were it emitted as
// `"trace_id":""` the fixture above would fail — this test says why, so a future reader
// does not "fix" it by regenerating the fixture.
func TestLegacyEnvelope_TraceIDIsAbsentWithoutASpan(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/v1/thing", nil)
	w := httptest.NewRecorder()

	require.NoError(t, render.Render(w, r, aohhttp.ErrResponse(http.StatusNotFound, "missing", nil)))

	assert.NotContains(t, w.Body.String(), "trace_id")
	assert.NotContains(t, w.Body.String(), `""`)
}
