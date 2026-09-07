// Package service holds the domain layer: it turns repository results and failures into
// the shapes the handler renders.
package service

import (
	"context"
	"errors"

	"github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"

	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/domain"
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/repo"
)

// UnitReader is the persistence surface this service needs. Declared here, at the
// consumer, so tests can substitute a fake without a database or a mock framework.
type UnitReader interface {
	List(ctx context.Context) ([]domain.Unit, error)
	Get(ctx context.Context, unitCode string) (domain.Unit, error)
}

type UnitService struct {
	units UnitReader
}

func NewUnitService(units UnitReader) *UnitService {
	return &UnitService{units: units}
}

func (s *UnitService) List(ctx context.Context) ([]domain.Unit, error) {
	units, err := s.units.List(ctx)
	if err != nil {
		return nil, aoherr.Wrap(aoherr.ClassSystem, CodeUnitReadFailed,
			"could not read units", err)
	}
	return units, nil
}

func (s *UnitService) Get(ctx context.Context, unitCode string) (domain.Unit, error) {
	if unitCode == "" {
		return domain.Unit{}, aoherr.New(aoherr.ClassValidation, CodeUnitCodeRequired,
			"unit code must not be empty")
	}

	unit, err := s.units.Get(ctx, unitCode)
	if err != nil {
		// A missing row is a not-found, not a system failure — without this the driver
		// error would surface as a 500.
		if errors.Is(err, repo.ErrNotFound) {
			return domain.Unit{}, aoherr.New(aoherr.ClassNotFound, CodeUnitNotFound,
				"no unit with that code")
		}
		return domain.Unit{}, aoherr.Wrap(aoherr.ClassSystem, CodeUnitReadFailed,
			"could not read unit", err)
	}
	return unit, nil
}
