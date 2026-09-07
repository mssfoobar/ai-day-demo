package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"

	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/domain"
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/repo"
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/service"
)

// fakeRepo is a hand-written stand-in for the persistence layer — no database, no mock
// framework. It stores units in a map and can be told to fail a given operation.
type fakeRepo struct {
	units     map[string]domain.Unit
	listErr   error
	createErr error
	updateErr error
	deleteErr error
	lastInput domain.UnitInput
}

func newFake(units ...domain.Unit) *fakeRepo {
	f := &fakeRepo{units: map[string]domain.Unit{}}
	for _, u := range units {
		f.units[u.UnitCode] = u
	}
	return f
}

func (f *fakeRepo) List(context.Context) ([]domain.Unit, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]domain.Unit, 0, len(f.units))
	for _, u := range f.units {
		out = append(out, u)
	}
	return out, nil
}

func (f *fakeRepo) Get(_ context.Context, code string) (domain.Unit, error) {
	if u, ok := f.units[code]; ok {
		return u, nil
	}
	return domain.Unit{}, repo.ErrNotFound
}

func (f *fakeRepo) Create(_ context.Context, in domain.UnitInput) (domain.Unit, error) {
	f.lastInput = in
	if f.createErr != nil {
		return domain.Unit{}, f.createErr
	}
	if _, exists := f.units[in.UnitCode]; exists {
		return domain.Unit{}, repo.ErrConflict
	}
	u := domain.Unit{UnitCode: in.UnitCode, CallSign: in.CallSign, Status: in.Status,
		Capabilities: in.Capabilities, Crew: []domain.Crew{}}
	f.units[in.UnitCode] = u
	return u, nil
}

func (f *fakeRepo) Update(_ context.Context, code string, occLock int, in domain.UnitInput) (domain.Unit, error) {
	f.lastInput = in
	if f.updateErr != nil {
		return domain.Unit{}, f.updateErr
	}
	u, ok := f.units[code]
	if !ok {
		return domain.Unit{}, repo.ErrNotFound
	}
	if u.OccLock != occLock {
		return domain.Unit{}, repo.ErrStale
	}
	u.CallSign, u.Status, u.OccLock = in.CallSign, in.Status, u.OccLock+1
	f.units[code] = u
	return u, nil
}

func (f *fakeRepo) Delete(_ context.Context, code string, occLock int) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	u, ok := f.units[code]
	if !ok {
		return repo.ErrNotFound
	}
	if u.OccLock != occLock {
		return repo.ErrStale
	}
	delete(f.units, code)
	return nil
}

func validInput() domain.UnitInput {
	return domain.UnitInput{
		UnitCode: "FU-401", CallSign: "Delta-1", Status: domain.StatusAvailable,
		UnitType: "Ambulance", Station: "Bedok Station 3", Sector: "Sector 9",
		RadioChannel: "TAC-3", Shift: "Day", Capabilities: []string{"ALS"},
	}
}

func alpha1() domain.Unit {
	return domain.Unit{UnitCode: "FU-101", CallSign: "Alpha-1", Status: domain.StatusAvailable, OccLock: 2}
}

func asAOH(t *testing.T, err error) *aoherr.Error {
	t.Helper()
	var aohErr *aoherr.Error
	if !errors.As(err, &aohErr) {
		t.Fatalf("expected an *aoherr.Error, got %T (%v)", err, err)
	}
	return aohErr
}

func detailFields(e *aoherr.Error) []string {
	var fields []string
	for _, d := range e.Details() {
		fields = append(fields, d.Field)
	}
	return fields
}

// --- reads (unchanged behaviour) -------------------------------------------------------

func TestGet_UnknownCodeIsNotFoundNotSystem(t *testing.T) {
	svc := service.NewUnitService(newFake(alpha1()))
	_, err := svc.Get(context.Background(), "FU-999")
	e := asAOH(t, err)
	if e.Class() != aoherr.ClassNotFound || e.Code() != service.CodeUnitNotFound {
		t.Errorf("got %s/%s", e.Class(), e.Code())
	}
}

func TestList_RepositoryFailureIsASystemError(t *testing.T) {
	svc := service.NewUnitService(&fakeRepo{listErr: errors.New("connection refused")})
	_, err := svc.List(context.Background())
	if asAOH(t, err).Class() != aoherr.ClassSystem {
		t.Errorf("expected system class")
	}
}

// --- create ----------------------------------------------------------------------------

func TestCreate_ValidInputIsStored(t *testing.T) {
	f := newFake()
	svc := service.NewUnitService(f)

	unit, err := svc.Create(context.Background(), validInput())
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if unit.UnitCode != "FU-401" || f.units["FU-401"].CallSign != "Delta-1" {
		t.Errorf("unit not stored as given: %+v", unit)
	}
}

func TestCreate_TrimsAndDropsBlankCapabilities(t *testing.T) {
	f := newFake()
	svc := service.NewUnitService(f)
	in := validInput()
	in.CallSign = "  Delta-1  "
	in.Capabilities = []string{" ALS ", "", "  "}

	if _, err := svc.Create(context.Background(), in); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if f.lastInput.CallSign != "Delta-1" {
		t.Errorf("call sign not trimmed: %q", f.lastInput.CallSign)
	}
	if len(f.lastInput.Capabilities) != 1 || f.lastInput.Capabilities[0] != "ALS" {
		t.Errorf("capabilities not normalised: %v", f.lastInput.Capabilities)
	}
}

func TestCreate_ReportsEveryMissingFieldAtOnce(t *testing.T) {
	svc := service.NewUnitService(newFake())
	in := validInput()
	in.CallSign, in.Station, in.Status = "", "   ", ""

	_, err := svc.Create(context.Background(), in)
	e := asAOH(t, err)
	if e.Class() != aoherr.ClassValidation || e.Code() != service.CodeUnitInvalid {
		t.Fatalf("got %s/%s", e.Class(), e.Code())
	}
	got := strings.Join(detailFields(e), ",")
	for _, want := range []string{"call_sign", "station", "status"} {
		if !strings.Contains(got, want) {
			t.Errorf("details %q missing %q", got, want)
		}
	}
}

func TestCreate_StatusOutsideVocabularyIsAValidationError(t *testing.T) {
	svc := service.NewUnitService(newFake())
	in := validInput()
	in.Status = "Out of service"

	_, err := svc.Create(context.Background(), in)
	e := asAOH(t, err)
	if e.Class() != aoherr.ClassValidation {
		t.Fatalf("expected validation, got %s", e.Class())
	}
	if fields := detailFields(e); len(fields) != 1 || fields[0] != "status" {
		t.Errorf("expected only status to fail, got %v", fields)
	}
}

func TestCreate_DuplicateCodeIsAConflict(t *testing.T) {
	svc := service.NewUnitService(newFake(alpha1()))
	in := validInput()
	in.UnitCode = "FU-101"

	_, err := svc.Create(context.Background(), in)
	e := asAOH(t, err)
	if e.Class() != aoherr.ClassConflict || e.Code() != service.CodeUnitCodeTaken {
		t.Errorf("got %s/%s", e.Class(), e.Code())
	}
}

func TestCreate_DatabaseCheckViolationIsValidationNotSystem(t *testing.T) {
	// The DB CHECK is the backstop; when it fires the client should still see a 400.
	svc := service.NewUnitService(&fakeRepo{units: map[string]domain.Unit{}, createErr: repo.ErrInvalid})
	_, err := svc.Create(context.Background(), validInput())
	if asAOH(t, err).Class() != aoherr.ClassValidation {
		t.Errorf("expected validation class")
	}
}

// --- update ----------------------------------------------------------------------------

func TestUpdate_MatchingOccLockAppliesAndBumps(t *testing.T) {
	f := newFake(alpha1())
	svc := service.NewUnitService(f)
	in := validInput()
	in.CallSign = "Alpha-1 (renamed)"

	unit, err := svc.Update(context.Background(), "FU-101", 2, in)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if unit.CallSign != "Alpha-1 (renamed)" || unit.OccLock != 3 {
		t.Errorf("unexpected result %+v", unit)
	}
	// The URL's code wins over anything in the body.
	if f.lastInput.UnitCode != "FU-101" {
		t.Errorf("body unit_code should be overridden by the URL, got %q", f.lastInput.UnitCode)
	}
}

func TestUpdate_StaleOccLockIsAConflictAndChangesNothing(t *testing.T) {
	f := newFake(alpha1())
	svc := service.NewUnitService(f)
	in := validInput()
	in.CallSign = "should not land"

	_, err := svc.Update(context.Background(), "FU-101", 1, in)
	e := asAOH(t, err)
	if e.Class() != aoherr.ClassConflict || e.Code() != service.CodeUnitStale {
		t.Errorf("got %s/%s", e.Class(), e.Code())
	}
	if f.units["FU-101"].CallSign != "Alpha-1" {
		t.Error("stale write must not be applied")
	}
}

func TestUpdate_UnknownCodeIsNotFound(t *testing.T) {
	svc := service.NewUnitService(newFake())
	_, err := svc.Update(context.Background(), "FU-999", 0, validInput())
	if asAOH(t, err).Class() != aoherr.ClassNotFound {
		t.Errorf("expected not-found")
	}
}

func TestUpdate_ValidatesLikeCreateButNotTheCode(t *testing.T) {
	svc := service.NewUnitService(newFake(alpha1()))
	in := validInput()
	in.UnitCode = "" // absent from a replace body — must not be a validation failure
	in.CallSign = ""

	_, err := svc.Update(context.Background(), "FU-101", 2, in)
	fields := detailFields(asAOH(t, err))
	if strings.Contains(strings.Join(fields, ","), "unit_code") {
		t.Errorf("unit_code must not be validated on replace, got %v", fields)
	}
	if len(fields) != 1 || fields[0] != "call_sign" {
		t.Errorf("expected only call_sign, got %v", fields)
	}
}

// --- delete ----------------------------------------------------------------------------

func TestDelete_MatchingOccLockRemoves(t *testing.T) {
	f := newFake(alpha1())
	svc := service.NewUnitService(f)
	if err := svc.Delete(context.Background(), "FU-101", 2); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok := f.units["FU-101"]; ok {
		t.Error("unit should be gone")
	}
}

func TestDelete_StaleOccLockIsAConflictAndKeepsTheUnit(t *testing.T) {
	f := newFake(alpha1())
	svc := service.NewUnitService(f)
	err := svc.Delete(context.Background(), "FU-101", 0)
	if asAOH(t, err).Code() != service.CodeUnitStale {
		t.Errorf("expected stale")
	}
	if _, ok := f.units["FU-101"]; !ok {
		t.Error("unit must survive a stale delete")
	}
}
