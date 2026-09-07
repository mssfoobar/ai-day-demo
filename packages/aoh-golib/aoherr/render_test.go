package aoherr_test

import (
	"context"
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"sort"
	"testing"
	"time"

	"github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// contractKeys is the complete, closed key set of the AOH error body.
var contractKeys = []string{"timestamp", "trace_id", "errorCode", "errorMessage", "details"}

func TestRender_CarriesTheContractKeys(t *testing.T) {
	ctx, _ := tracedContext(t)

	got := renderErr(t, ctx, aoherr.New(aoherr.ClassValidation,
		aoherr.MustCode("UNH_INVALID_TEMPLATE"), "name is required").
		WithDetails(aoherr.FieldDetail("name", "is required")))

	mediaType, _, err := mime.ParseMediaType(got.header.Get("Content-Type"))
	require.NoError(t, err)
	assert.Equal(t, "application/json", mediaType)

	assert.Subset(t, got.keyNames(), []string{"timestamp", "errorCode", "errorMessage"},
		"timestamp, errorCode and errorMessage are required")
	assert.Subset(t, contractKeys, got.keyNames(),
		"body must carry no key outside the contract set, got %v", got.keyNames())
}

// TestRender_OptionalFieldsAreAbsentNotEmpty is the omitempty partition: no
// detail and no span must produce absent keys, not null/""/[].
func TestRender_OptionalFieldsAreAbsentNotEmpty(t *testing.T) {
	got := renderErr(t, context.Background(),
		aoherr.New(aoherr.ClassNotFound, aoherr.MustCode("UNH_TEMPLATE_NOT_FOUND"), "gone"))

	assert.NotContains(t, got.keys, "details")
	assert.NotContains(t, got.keys, "trace_id")
	assert.NotContains(t, got.body, "null")
	assert.Equal(t, []string{"errorCode", "errorMessage", "timestamp"}, sortedKeys(got))
}

// TestRender_EmptyDetailsSliceIsOmitted covers the boundary between "no
// details" and "one detail": a zero-length slice must not serialize as [].
func TestRender_EmptyDetailsSliceIsOmitted(t *testing.T) {
	got := renderErr(t, context.Background(),
		aoherr.New(aoherr.ClassValidation, aoherr.MustCode("UNH_BAD_REQUEST"), "bad").
			WithDetails())

	assert.NotContains(t, got.keys, "details")
}

func TestRender_ErrorMessageIsNonEmptyWithoutACallSiteMessage(t *testing.T) {
	code := aoherr.MustCode("UNH_TEMPLATE_NOT_FOUND")

	got := renderErr(t, context.Background(), aoherr.New(aoherr.ClassNotFound, code, ""))

	msg := got.str(t, "errorMessage")
	assert.NotEmpty(t, msg)
	assert.NotEqual(t, code.String(), msg, "the default message must not be the code")
	assert.NotEqual(t, "null", msg)
}

func TestRender_CarriesNoLegacyOrRetiredKeys(t *testing.T) {
	ctx, _ := tracedContext(t)

	got := renderErr(t, ctx, aoherr.New(aoherr.ClassConflict, aoherr.MustCode("UNH_CONFLICT"), "clash"))

	for _, banned := range []string{"data", "message", "sent_at", "errors", "correlationId"} {
		assert.NotContains(t, got.keys, banned)
	}
}

// TestRender_PresentationFieldsAreNeverEmitted holds the D2 split: a service
// renders the envelope only; userMessage/isRetryable belong to the BFF.
func TestRender_PresentationFieldsAreNeverEmitted(t *testing.T) {
	got := renderErr(t, context.Background(),
		aoherr.New(aoherr.ClassUpstreamUnavailable, aoherr.MustCode("UNH_UPSTREAM_DOWN"), "down").
			WithDetails(aoherr.FieldDetail("upstream", "unreachable")))

	assert.NotContains(t, got.keys, "userMessage")
	assert.NotContains(t, got.keys, "isRetryable")
	assert.NotContains(t, got.body, "userMessage")
	assert.NotContains(t, got.body, "isRetryable")
}

func TestRender_TimestampIsAnRFC3339MillisecondUTCInstant(t *testing.T) {
	got := renderErr(t, context.Background(),
		aoherr.New(aoherr.ClassValidation, aoherr.MustCode("UNH_BAD_REQUEST"), "bad"))

	ts := got.str(t, "timestamp")
	assert.Regexp(t, `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$`, ts)

	parsed, err := time.Parse(time.RFC3339, ts)
	require.NoError(t, err)
	assert.Equal(t, time.UTC, parsed.Location(), "the instant must be UTC, not local or offset")
	assert.WithinDuration(t, time.Now().UTC(), parsed, time.Minute)
}

func TestRender_TraceIDIsThe32HexIDOfTheActiveSpan(t *testing.T) {
	ctx, sc := tracedContext(t)

	got := renderErr(t, ctx, aoherr.New(aoherr.ClassSystem, aoherr.CodeInternal, "boom"))

	traceID := got.str(t, "trace_id")
	assert.Equal(t, sc.TraceID().String(), traceID)
	assert.Regexp(t, hex32, traceID)
	assert.NotContains(t, traceID, "-", "a trace id is not a UUID")
}

func TestRender_TraceIDOmittedWithNoSpanAndNoSubstituteMinted(t *testing.T) {
	got := renderErr(t, context.Background(),
		aoherr.New(aoherr.ClassSystem, aoherr.CodeInternal, "boom"))

	assert.NotContains(t, got.keys, "trace_id")
	for _, banned := range []string{"unknown", "null", "correlation"} {
		assert.NotContains(t, got.body, banned)
	}
}

// TestRender_DetailsIsAnArrayOfObjects pins the D4 shape decision: an array,
// not msr's map keyed by field name and not an array of strings.
func TestRender_DetailsIsAnArrayOfObjects(t *testing.T) {
	got := renderErr(t, context.Background(),
		aoherr.New(aoherr.ClassValidation, aoherr.MustCode("UNH_INVALID_TEMPLATE"), "2 fields failed").
			WithDetails(
				aoherr.FieldDetail("name", "is required"),
				aoherr.FieldDetail("body", "exceeds 4000 characters"),
			))

	var details []map[string]any
	require.NoError(t, json.Unmarshal(got.keys["details"], &details),
		"details must decode as an array of objects: %s", got.body)
	require.Len(t, details, 2)
	assert.Equal(t, "name", details[0]["field"])
	assert.Equal(t, "body", details[1]["field"])

	var asObject map[string]any
	assert.Error(t, json.Unmarshal(got.keys["details"], &asObject),
		"details must not be an object keyed by field name")
}

// TestRender_DetailElementKeySetIsExactlyFieldAndMessage is the regression test
// for the defect that made Detail a struct. While Detail was a map[string]any a
// call site could serialize any key it liked into a 4xx body, and a review probe
// produced a rendered 400 carrying `{"submitted":"hunter2"}` alongside a driver
// string with an absolute source path. The element schema is now structural, so
// the assertion here is the exact key set — if someone reintroduces a map, or
// adds a third member, this fails.
func TestRender_DetailElementKeySetIsExactlyFieldAndMessage(t *testing.T) {
	got := renderErr(t, context.Background(),
		aoherr.New(aoherr.ClassValidation, aoherr.MustCode("UNH_VALIDATION_FAILED"), "2 fields failed").
			WithDetails(
				aoherr.FieldDetail("password", "must be at least 12 characters"),
				aoherr.FieldDetail("port", "out of range"),
			))

	var details []map[string]any
	require.NoError(t, json.Unmarshal(got.keys["details"], &details))
	require.Len(t, details, 2)
	for i, element := range details {
		keys := make([]string, 0, len(element))
		for key := range element {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		assert.Equal(t, []string{"field", "message"}, keys,
			"element %d must carry exactly field and message", i)
	}

	// The specific values the probe leaked must not be expressible at all now.
	for _, banned := range []string{"hunter2", "submitted", "pq:", ".go:", "/srv/"} {
		assert.NotContains(t, got.body, banned)
	}
}

// TestRender_HalfPopulatedDetailIsDropped is the boundary: the contract requires
// both members non-empty, so an element missing either is dropped rather than
// emitted as a non-conformant `{"field":""}` or an empty object.
func TestRender_HalfPopulatedDetailIsDropped(t *testing.T) {
	got := renderErr(t, context.Background(),
		aoherr.New(aoherr.ClassConflict, aoherr.MustCode("UNH_CONFLICT"), "clash").
			WithDetails(
				aoherr.FieldDetail("", "message with no field"),
				aoherr.FieldDetail("field with no message", ""),
				aoherr.FieldDetail("", ""),
			))

	assert.NotContains(t, got.keys, "details")
	assert.NotContains(t, got.body, "{}")
	assert.NotContains(t, got.body, `"field":""`)
}

// TestRender_StatusComesFromTheClass renders one error per class and asserts
// the response status, so the table is proven on the wire and not only on the
// Class type.
func TestRender_StatusComesFromTheClass(t *testing.T) {
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
			got := renderErr(t, context.Background(), aoherr.New(class, aoherr.CodeInternal, "x"))
			assert.Equal(t, status, got.status)
		})
	}
}

func TestRender_UnclassifiedErrorIsA500(t *testing.T) {
	got := renderErr(t, context.Background(), errors.New("something fell over"))

	assert.Equal(t, http.StatusInternalServerError, got.status)
	assert.Equal(t, aoherr.CodeInternal.String(), got.str(t, "errorCode"))
	assert.NotContains(t, got.body, "something fell over")
}

// TestRender_SuppressesInternalDetailOn500 is the security property: the
// renderer, not the call site, decides what a 5xx body may contain.
func TestRender_SuppressesInternalDetailOn500(t *testing.T) {
	cause := errors.New(
		`pq: duplicate key value violates unique constraint "widgets_pkey" at /srv/app/internal/repo/widget.go:88`)

	got := renderErr(t, context.Background(),
		aoherr.Wrap(aoherr.ClassSystem, aoherr.MustCode("UNH_TEMPLATE_STORE_FAILED"),
			"insert template: "+cause.Error(), cause).
			// A careless call site putting the cause in the one free-text member
			// it has. Suppression must still hold.
			WithDetails(aoherr.FieldDetail("template", cause.Error())))

	assert.Equal(t, http.StatusInternalServerError, got.status)
	assert.Equal(t, "UNH_TEMPLATE_STORE_FAILED", got.str(t, "errorCode"),
		"suppression keeps the specific code")
	for _, leak := range []string{"pq:", "duplicate key", "widgets_pkey", "/srv/app", ".go:88"} {
		assert.NotContains(t, got.body, leak)
	}
	assert.NotContains(t, got.keys, "details", "details cannot survive suppression")
	assert.NotEmpty(t, got.str(t, "errorMessage"))
}

func TestRender_SuppressesInternalDetailOn503(t *testing.T) {
	cause := errors.New("dial tcp 10.0.3.14:5432: connect: connection refused")

	got := renderErr(t, context.Background(),
		aoherr.Wrap(aoherr.ClassUpstreamUnavailable, aoherr.MustCode("UNH_IAMS_UNAVAILABLE"),
			cause.Error(), cause))

	assert.Equal(t, http.StatusServiceUnavailable, got.status)
	assert.NotContains(t, got.body, "10.0.3.14")
	assert.NotContains(t, got.body, "connection refused")
}

// TestRender_4xxKeepsTheCallSiteMessage is the other side of suppression: a
// client-fault body stays actionable.
func TestRender_4xxKeepsTheCallSiteMessage(t *testing.T) {
	got := renderErr(t, context.Background(),
		aoherr.New(aoherr.ClassValidation, aoherr.MustCode("UNH_INVALID_TEMPLATE"),
			"field \"name\" is required").
			WithDetails(aoherr.FieldDetail("name", "is required")))

	assert.Contains(t, got.str(t, "errorMessage"), "name")
	assert.Contains(t, got.body, "is required")
}

// TestRender_4xxBodyLeaksNoCredential proves a validation body identifies the
// offending field without echoing the submitted secret.
func TestRender_4xxBodyLeaksNoCredential(t *testing.T) {
	got := renderErr(t, context.Background(),
		aoherr.New(aoherr.ClassValidation, aoherr.MustCode("UNH_INVALID_TEMPLATE"), "password is too short").
			WithDetails(aoherr.FieldDetail("password", "must be at least 12 characters")))

	assert.Contains(t, got.body, "password")
	assert.NotContains(t, got.body, "hunter2")
	assert.NotContains(t, got.body, "Bearer ")
}

func TestRender_NotFoundFallbackIsAConformant404(t *testing.T) {
	got := renderErr(t, context.Background(), aoherr.NotFoundError())

	assert.Equal(t, http.StatusNotFound, got.status)
	assert.Equal(t, aoherr.CodeNotFound.String(), got.str(t, "errorCode"))
	assert.NotEmpty(t, got.str(t, "errorMessage"))
}

func TestRender_MethodNotAllowedFallbackIsAConformant405(t *testing.T) {
	got := renderErr(t, context.Background(), aoherr.MethodNotAllowedError())

	assert.Equal(t, http.StatusMethodNotAllowed, got.status)
	assert.Equal(t, aoherr.CodeMethodNotAllowed.String(), got.str(t, "errorCode"))
	assert.NotEmpty(t, got.str(t, "errorMessage"))
}

func sortedKeys(r rendered) []string {
	names := r.keyNames()
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && names[j] < names[j-1]; j-- {
			names[j], names[j-1] = names[j-1], names[j]
		}
	}
	return names
}
