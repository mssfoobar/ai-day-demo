package handler_test

import (
	"context"
	"encoding/json"
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

type fakeUnits struct {
	units []domain.Unit
	err   error
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

func seed() []domain.Unit {
	return []domain.Unit{
		{
			UnitCode: "FU-101", CallSign: "Alpha-1", Status: domain.StatusAvailable,
			UnitType: "Ambulance", Station: "Marina Bay Station 4",
			Capabilities: []string{"ALS"}, Crew: []domain.Crew{{Name: "J. Tan", Role: "Paramedic"}},
			LastContact: time.Now().UTC(),
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

func do(t *testing.T, srv http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestListUnits_UsesTheAOHSuccessEnvelope(t *testing.T) {
	rec := do(t, newServer(&fakeUnits{units: seed()}), http.MethodGet, "/v1/units")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	// The contract is an envelope, never a bare array.
	for _, key := range []string{"data", "sent_at"} {
		if _, ok := body[key]; !ok {
			t.Errorf("envelope is missing %q: %s", key, rec.Body.String())
		}
	}
	data, ok := body["data"].([]any)
	if !ok {
		t.Fatalf("data is not an array: %T", body["data"])
	}
	if len(data) != 2 {
		t.Fatalf("expected 2 units, got %d", len(data))
	}
}

func TestListUnits_OmitsAssignmentWhenUnassigned(t *testing.T) {
	rec := do(t, newServer(&fakeUnits{units: seed()}), http.MethodGet, "/v1/units")

	var body struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}

	// An unassigned unit must omit the key entirely, not serialise an empty object.
	if _, present := body.Data[0]["assignment"]; present {
		t.Error("unassigned unit should omit `assignment`")
	}
	if _, present := body.Data[1]["assignment"]; !present {
		t.Error("assigned unit should carry `assignment`")
	}
}

func TestGetUnit_ReturnsTheUnit(t *testing.T) {
	rec := do(t, newServer(&fakeUnits{units: seed()}), http.MethodGet, "/v1/units/FU-101")

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Alpha-1") {
		t.Errorf("expected the unit in the body, got %s", rec.Body.String())
	}
}

func TestGetUnit_UnknownCodeIs404WithABody(t *testing.T) {
	rec := do(t, newServer(&fakeUnits{units: seed()}), http.MethodGet, "/v1/units/FU-999")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	// The AOH contract forbids a zero-length error body.
	if rec.Body.Len() == 0 {
		t.Fatal("404 must not have an empty body")
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("404 body is not JSON: %q", rec.Body.String())
	}
	if _, ok := body["errorCode"]; !ok {
		t.Errorf("404 body should carry errorCode, got %s", rec.Body.String())
	}
}

func TestWriteVerbsAreNotAllowed(t *testing.T) {
	srv := newServer(&fakeUnits{units: seed()})

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		rec := do(t, srv, method, "/v1/units")
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s /v1/units: expected 405, got %d", method, rec.Code)
		}
	}
}

func TestHealthProbes(t *testing.T) {
	srv := newServer(&fakeUnits{units: seed()})

	for _, path := range []string{"/livez", "/readyz"} {
		if rec := do(t, srv, http.MethodGet, path); rec.Code != http.StatusOK {
			t.Errorf("%s: expected 200, got %d", path, rec.Code)
		}
	}
}

func TestReadyz_FailsWhenTheDatabaseIsUnreachable(t *testing.T) {
	srv := handler.Router(
		handler.NewUnitHandler(&fakeUnits{units: seed()}),
		func(context.Context) error { return context.DeadlineExceeded },
	)

	rec := do(t, srv, http.MethodGet, "/readyz")
	if rec.Code == http.StatusOK {
		t.Error("readyz must not report 200 when the database is unreachable")
	}
	// Liveness is unaffected — the process is still running.
	if rec := do(t, srv, http.MethodGet, "/livez"); rec.Code != http.StatusOK {
		t.Errorf("livez should stay 200, got %d", rec.Code)
	}
}
