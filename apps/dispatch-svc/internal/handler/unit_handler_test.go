package handler_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"

	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/domain"
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/handler"
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/service"
)

// fakeUnits stands in for the service layer and records what the handler passed through.
type fakeUnits struct {
	units       []domain.Unit
	err         error
	gotInput    domain.UnitInput
	gotOccLock  int
	gotUnitCode string
}

func (f *fakeUnits) List(context.Context) ([]domain.Unit, error) { return f.units, f.err }

func (f *fakeUnits) Get(_ context.Context, code string) (domain.Unit, error) {
	if f.err != nil {
		return domain.Unit{}, f.err
	}
	for _, u := range f.units {
		if u.UnitCode == code {
			return u, nil
		}
	}
	return domain.Unit{}, aoherr.New(aoherr.ClassNotFound, service.CodeUnitNotFound, "no unit")
}

func (f *fakeUnits) Create(_ context.Context, in domain.UnitInput) (domain.Unit, error) {
	f.gotInput = in
	if f.err != nil {
		return domain.Unit{}, f.err
	}
	return domain.Unit{UnitCode: in.UnitCode, CallSign: in.CallSign, Status: in.Status,
		Capabilities: in.Capabilities, Crew: []domain.Crew{}}, nil
}

func (f *fakeUnits) Update(_ context.Context, code string, occLock int, in domain.UnitInput) (domain.Unit, error) {
	f.gotUnitCode, f.gotOccLock, f.gotInput = code, occLock, in
	if f.err != nil {
		return domain.Unit{}, f.err
	}
	return domain.Unit{UnitCode: code, CallSign: in.CallSign, Status: in.Status, OccLock: occLock + 1,
		Capabilities: []string{}, Crew: []domain.Crew{}}, nil
}

func (f *fakeUnits) Delete(_ context.Context, code string, occLock int) error {
	f.gotUnitCode, f.gotOccLock = code, occLock
	return f.err
}

func seed() []domain.Unit {
	return []domain.Unit{
		{
			UnitCode: "FU-101", CallSign: "Alpha-1", Status: domain.StatusAvailable,
			UnitType: "Ambulance", Station: "Marina Bay Station 4",
			Capabilities: []string{"ALS"}, Crew: []domain.Crew{{Name: "J. Tan", Role: "Paramedic"}},
			LastContact: time.Now().UTC(), OccLock: 2,
		},
		{
			UnitCode: "FU-102", CallSign: "Alpha-2", Status: domain.StatusEnRoute,
			UnitType: "Ambulance", Capabilities: []string{}, Crew: []domain.Crew{},
			Assignment: &domain.Assignment{
				IncidentCode: "INC-2841", Title: "Cardiac arrest", Priority: "P1",
				Location: "12 Raffles Quay", Since: time.Now().UTC(),
			},
			LastContact: time.Now().UTC(),
		},
	}
}

func newServer(units *fakeUnits) http.Handler {
	return handler.Router(handler.NewUnitHandler(units), func(context.Context) error { return nil })
}

func do(t *testing.T, srv http.Handler, method, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON (%d): %q", rec.Code, rec.Body.String())
	}
	return body
}

const validCreate = `{"unit_code":"FU-401","call_sign":"Delta-1","status":"Available",
	"unit_type":"Ambulance","station":"Bedok Station 3","sector":"Sector 9",
	"radio_channel":"TAC-3","shift":"Day","capabilities":["ALS"]}`

const validReplace = `{"call_sign":"Alpha-1 (renamed)","status":"Idle","unit_type":"Ambulance",
	"station":"Marina Bay Station 4","sector":"Sector 4","radio_channel":"TAC-2",
	"shift":"Day","capabilities":[],"occ_lock":2}`

// --- reads -----------------------------------------------------------------------------

func TestListUnits_UsesTheAOHEnvelopeAndCarriesOccLock(t *testing.T) {
	rec := do(t, newServer(&fakeUnits{units: seed()}), http.MethodGet, "/v1/units", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	body := decodeBody(t, rec)
	for _, key := range []string{"data", "sent_at"} {
		if _, ok := body[key]; !ok {
			t.Errorf("envelope is missing %q", key)
		}
	}
	data := body["data"].([]any)
	first := data[0].(map[string]any)
	if _, ok := first["occ_lock"]; !ok {
		t.Error("unit must carry occ_lock so a client can echo it")
	}
	if _, present := first["assignment"]; present {
		t.Error("unassigned unit should omit `assignment`")
	}
}

func TestGetUnit_UnknownCodeIs404WithABody(t *testing.T) {
	rec := do(t, newServer(&fakeUnits{units: seed()}), http.MethodGet, "/v1/units/FU-999", "")
	if rec.Code != http.StatusNotFound || rec.Body.Len() == 0 {
		t.Fatalf("expected 404 with a body, got %d %q", rec.Code, rec.Body.String())
	}
	if _, ok := decodeBody(t, rec)["errorCode"]; !ok {
		t.Error("404 body should carry errorCode")
	}
}

// --- create ----------------------------------------------------------------------------

func TestCreate_ValidBodyIs201WithEnvelope(t *testing.T) {
	f := &fakeUnits{}
	rec := do(t, newServer(f), http.MethodPost, "/v1/units", validCreate)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	data := decodeBody(t, rec)["data"].(map[string]any)
	if data["unit_code"] != "FU-401" {
		t.Errorf("expected the created unit back, got %v", data)
	}
	if f.gotInput.CallSign != "Delta-1" || len(f.gotInput.Capabilities) != 1 {
		t.Errorf("handler did not pass the body through: %+v", f.gotInput)
	}
}

func TestCreate_ValidationErrorIs400WithFieldDetails(t *testing.T) {
	svcErr := aoherr.New(aoherr.ClassValidation, service.CodeUnitInvalid, "invalid").
		WithDetails(aoherr.FieldDetail("call_sign", "must not be empty"))
	rec := do(t, newServer(&fakeUnits{err: svcErr}), http.MethodPost, "/v1/units", validCreate)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	body := decodeBody(t, rec)
	if body["errorCode"] != "DISPATCH_UNIT_INVALID" {
		t.Errorf("errorCode = %v", body["errorCode"])
	}
	if !strings.Contains(rec.Body.String(), "call_sign") {
		t.Errorf("details should name the field: %s", rec.Body.String())
	}
}

func TestCreate_DuplicateIs409(t *testing.T) {
	svcErr := aoherr.New(aoherr.ClassConflict, service.CodeUnitCodeTaken, "taken")
	rec := do(t, newServer(&fakeUnits{err: svcErr}), http.MethodPost, "/v1/units", validCreate)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestCreate_MalformedJSONIs400(t *testing.T) {
	rec := do(t, newServer(&fakeUnits{}), http.MethodPost, "/v1/units", `{"unit_code": `)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCreate_UnknownFieldIs400(t *testing.T) {
	// A typo'd field must fail loudly, not be silently dropped.
	rec := do(t, newServer(&fakeUnits{}), http.MethodPost, "/v1/units",
		strings.Replace(validCreate, `"call_sign"`, `"callsign"`, 1))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown field, got %d (%s)", rec.Code, rec.Body.String())
	}
}

// --- replace ---------------------------------------------------------------------------

func TestUpdate_PassesCodeFromURLAndOccLockFromBody(t *testing.T) {
	f := &fakeUnits{}
	rec := do(t, newServer(f), http.MethodPut, "/v1/units/FU-101", validReplace)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if f.gotUnitCode != "FU-101" || f.gotOccLock != 2 {
		t.Errorf("got code=%q occ=%d", f.gotUnitCode, f.gotOccLock)
	}
	data := decodeBody(t, rec)["data"].(map[string]any)
	if data["occ_lock"] != float64(3) {
		t.Errorf("expected bumped occ_lock 3, got %v", data["occ_lock"])
	}
}

func TestUpdate_MissingOccLockIs400(t *testing.T) {
	body := strings.Replace(validReplace, `,"occ_lock":2`, ``, 1)
	rec := do(t, newServer(&fakeUnits{}), http.MethodPut, "/v1/units/FU-101", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (%s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "occ_lock") {
		t.Errorf("400 should name occ_lock: %s", rec.Body.String())
	}
}

func TestUpdate_ZeroOccLockIsAccepted(t *testing.T) {
	// A pointer distinguishes "sent 0" from "not sent": version 0 is a real version.
	f := &fakeUnits{}
	body := strings.Replace(validReplace, `"occ_lock":2`, `"occ_lock":0`, 1)
	rec := do(t, newServer(f), http.MethodPut, "/v1/units/FU-101", body)
	if rec.Code != http.StatusOK || f.gotOccLock != 0 {
		t.Fatalf("expected 200 with occ_lock 0 passed through, got %d / %d", rec.Code, f.gotOccLock)
	}
}

func TestUpdate_StaleIs409(t *testing.T) {
	svcErr := aoherr.New(aoherr.ClassConflict, service.CodeUnitStale, "stale")
	rec := do(t, newServer(&fakeUnits{err: svcErr}), http.MethodPut, "/v1/units/FU-101", validReplace)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
	if decodeBody(t, rec)["errorCode"] != "DISPATCH_UNIT_STALE" {
		t.Errorf("errorCode = %v", decodeBody(t, rec)["errorCode"])
	}
}

// --- delete ----------------------------------------------------------------------------

func TestDelete_Is204AndPassesOccLockFromQuery(t *testing.T) {
	f := &fakeUnits{}
	rec := do(t, newServer(f), http.MethodDelete, "/v1/units/FU-101?occ_lock=2", "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d (%s)", rec.Code, rec.Body.String())
	}
	if rec.Body.Len() != 0 {
		t.Errorf("204 must have no body, got %q", rec.Body.String())
	}
	if f.gotUnitCode != "FU-101" || f.gotOccLock != 2 {
		t.Errorf("got code=%q occ=%d", f.gotUnitCode, f.gotOccLock)
	}
}

func TestDelete_MissingOccLockIs400(t *testing.T) {
	for _, q := range []string{"", "?occ_lock=", "?occ_lock=abc"} {
		rec := do(t, newServer(&fakeUnits{}), http.MethodDelete, "/v1/units/FU-101"+q, "")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%q: expected 400, got %d", q, rec.Code)
		}
	}
}

func TestDelete_StaleIs409(t *testing.T) {
	svcErr := aoherr.New(aoherr.ClassConflict, service.CodeUnitStale, "stale")
	rec := do(t, newServer(&fakeUnits{err: svcErr}), http.MethodDelete, "/v1/units/FU-101?occ_lock=1", "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

// --- undefined verbs -------------------------------------------------------------------

func TestUndefinedVerbsAre405(t *testing.T) {
	srv := newServer(&fakeUnits{units: seed()})
	cases := []struct{ method, path string }{
		{http.MethodPatch, "/v1/units/FU-101"},
		{http.MethodPut, "/v1/units"},
		{http.MethodDelete, "/v1/units"},
	}
	for _, c := range cases {
		if rec := do(t, srv, c.method, c.path, `{}`); rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s %s: expected 405, got %d", c.method, c.path, rec.Code)
		}
	}
}

func TestHealthProbes(t *testing.T) {
	srv := newServer(&fakeUnits{units: seed()})
	for _, path := range []string{"/livez", "/readyz"} {
		if rec := do(t, srv, http.MethodGet, path, ""); rec.Code != http.StatusOK {
			t.Errorf("%s: expected 200, got %d", path, rec.Code)
		}
	}
}

func TestReadyz_FailsWhenTheDatabaseIsUnreachable(t *testing.T) {
	srv := handler.Router(
		handler.NewUnitHandler(&fakeUnits{units: seed()}),
		func(context.Context) error { return context.DeadlineExceeded },
	)
	if rec := do(t, srv, http.MethodGet, "/readyz", ""); rec.Code == http.StatusOK {
		t.Error("readyz must not report 200 when the database is unreachable")
	}
}
