package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"

	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/domain"
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/repo"
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/service"
)

// fakeRepo is a hand-written stand-in for the persistence layer — no database, no mock
// framework. The service's only interesting behaviour is how it classifies failures.
type fakeRepo struct {
	units   []domain.Unit
	listErr error
	getErr  error
}

func (f *fakeRepo) List(context.Context) ([]domain.Unit, error) {
	return f.units, f.listErr
}

func (f *fakeRepo) Get(_ context.Context, code string) (domain.Unit, error) {
	if f.getErr != nil {
		return domain.Unit{}, f.getErr
	}
	for _, u := range f.units {
		if u.UnitCode == code {
			return u, nil
		}
	}
	return domain.Unit{}, repo.ErrNotFound
}

func sampleUnits() []domain.Unit {
	return []domain.Unit{
		{UnitCode: "FU-101", CallSign: "Alpha-1", Status: domain.StatusAvailable},
		{UnitCode: "FU-102", CallSign: "Alpha-2", Status: domain.StatusEnRoute},
	}
}

func TestList_ReturnsUnits(t *testing.T) {
	svc := service.NewUnitService(&fakeRepo{units: sampleUnits()})

	units, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(units) != 2 {
		t.Fatalf("expected 2 units, got %d", len(units))
	}
}

func TestList_RepositoryFailureIsASystemError(t *testing.T) {
	svc := service.NewUnitService(&fakeRepo{listErr: errors.New("connection refused")})

	_, err := svc.List(context.Background())
	if err == nil {
		t.Fatal("expected an error")
	}

	var aohErr *aoherr.Error
	if !errors.As(err, &aohErr) {
		t.Fatalf("expected an *aoherr.Error, got %T", err)
	}
	if aohErr.Class() != aoherr.ClassSystem {
		t.Errorf("expected class %q, got %q", aoherr.ClassSystem, aohErr.Class())
	}
	// The cause must stay reachable so the log records what actually broke.
	if !errors.Is(err, aohErr.Unwrap()) {
		t.Error("expected the cause to remain unwrappable")
	}
}

func TestGet_ReturnsTheMatchingUnit(t *testing.T) {
	svc := service.NewUnitService(&fakeRepo{units: sampleUnits()})

	unit, err := svc.Get(context.Background(), "FU-102")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if unit.CallSign != "Alpha-2" {
		t.Errorf("expected Alpha-2, got %q", unit.CallSign)
	}
}

func TestGet_UnknownCodeIsNotFoundNotSystem(t *testing.T) {
	// The regression this guards: a bare sql.ErrNoRows reaching the handler renders as a
	// 500 instead of a 404.
	svc := service.NewUnitService(&fakeRepo{units: sampleUnits()})

	_, err := svc.Get(context.Background(), "FU-999")

	var aohErr *aoherr.Error
	if !errors.As(err, &aohErr) {
		t.Fatalf("expected an *aoherr.Error, got %T (%v)", err, err)
	}
	if aohErr.Class() != aoherr.ClassNotFound {
		t.Errorf("expected class %q, got %q", aoherr.ClassNotFound, aohErr.Class())
	}
	if aohErr.Code() != service.CodeUnitNotFound {
		t.Errorf("expected code %q, got %q", service.CodeUnitNotFound, aohErr.Code())
	}
}

func TestGet_EmptyCodeIsAValidationError(t *testing.T) {
	svc := service.NewUnitService(&fakeRepo{units: sampleUnits()})

	_, err := svc.Get(context.Background(), "")

	var aohErr *aoherr.Error
	if !errors.As(err, &aohErr) {
		t.Fatalf("expected an *aoherr.Error, got %T", err)
	}
	if aohErr.Class() != aoherr.ClassValidation {
		t.Errorf("expected class %q, got %q", aoherr.ClassValidation, aohErr.Class())
	}
}
