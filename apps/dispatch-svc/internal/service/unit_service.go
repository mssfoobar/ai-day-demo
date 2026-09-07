// Package service holds the domain layer: it turns repository results and failures into
// the shapes the handler renders.
package service

import (
	"context"
	"errors"
	"strings"

	"github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"

	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/domain"
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/repo"
)

// UnitStore is the persistence surface this service needs. Declared here, at the
// consumer, so tests can substitute a fake without a database or a mock framework.
type UnitStore interface {
	List(ctx context.Context) ([]domain.Unit, error)
	Get(ctx context.Context, unitCode string) (domain.Unit, error)
	Create(ctx context.Context, in domain.UnitInput) (domain.Unit, error)
	Update(ctx context.Context, unitCode string, occLock int, in domain.UnitInput) (domain.Unit, error)
	Delete(ctx context.Context, unitCode string, occLock int) error
}

// UnitReader is kept for callers that only read.
type UnitReader interface {
	List(ctx context.Context) ([]domain.Unit, error)
	Get(ctx context.Context, unitCode string) (domain.Unit, error)
}

type UnitService struct {
	units UnitStore
}

func NewUnitService(units UnitStore) *UnitService {
	return &UnitService{units: units}
}

func (s *UnitService) List(ctx context.Context) ([]domain.Unit, error) {
	units, err := s.units.List(ctx)
	if err != nil {
		return nil, aoherr.Wrap(aoherr.ClassSystem, CodeUnitReadFailed, "could not read units", err)
	}
	return units, nil
}

func (s *UnitService) Get(ctx context.Context, unitCode string) (domain.Unit, error) {
	if strings.TrimSpace(unitCode) == "" {
		return domain.Unit{}, aoherr.New(aoherr.ClassValidation, CodeUnitCodeRequired,
			"unit code must not be empty")
	}

	unit, err := s.units.Get(ctx, unitCode)
	if err != nil {
		// A missing row is a not-found, not a system failure — without this the driver
		// error would surface as a 500.
		if errors.Is(err, repo.ErrNotFound) {
			return domain.Unit{}, notFound()
		}
		return domain.Unit{}, aoherr.Wrap(aoherr.ClassSystem, CodeUnitReadFailed, "could not read unit", err)
	}
	return unit, nil
}

// Create validates and inserts a new unit.
func (s *UnitService) Create(ctx context.Context, in domain.UnitInput) (domain.Unit, error) {
	in = normalise(in)
	if err := validate(in, true); err != nil {
		return domain.Unit{}, err
	}
	unit, err := s.units.Create(ctx, in)
	if err != nil {
		return domain.Unit{}, classifyWrite(err)
	}
	return unit, nil
}

// Update validates and replaces the unit's editable fields, guarded by occLock.
func (s *UnitService) Update(ctx context.Context, unitCode string, occLock int, in domain.UnitInput) (domain.Unit, error) {
	if strings.TrimSpace(unitCode) == "" {
		return domain.Unit{}, aoherr.New(aoherr.ClassValidation, CodeUnitCodeRequired,
			"unit code must not be empty")
	}
	in = normalise(in)
	// unit_code comes from the URL, not the body, on replace.
	in.UnitCode = unitCode
	if err := validate(in, false); err != nil {
		return domain.Unit{}, err
	}
	unit, err := s.units.Update(ctx, unitCode, occLock, in)
	if err != nil {
		return domain.Unit{}, classifyWrite(err)
	}
	return unit, nil
}

// Delete removes the unit, guarded by occLock.
func (s *UnitService) Delete(ctx context.Context, unitCode string, occLock int) error {
	if strings.TrimSpace(unitCode) == "" {
		return aoherr.New(aoherr.ClassValidation, CodeUnitCodeRequired, "unit code must not be empty")
	}
	if err := s.units.Delete(ctx, unitCode, occLock); err != nil {
		return classifyWrite(err)
	}
	return nil
}

// normalise trims every string field and drops blank capabilities, so validation and
// storage see what the user meant rather than what the form sent.
func normalise(in domain.UnitInput) domain.UnitInput {
	in.UnitCode = strings.TrimSpace(in.UnitCode)
	in.CallSign = strings.TrimSpace(in.CallSign)
	in.Status = strings.TrimSpace(in.Status)
	in.UnitType = strings.TrimSpace(in.UnitType)
	in.Station = strings.TrimSpace(in.Station)
	in.Sector = strings.TrimSpace(in.Sector)
	in.RadioChannel = strings.TrimSpace(in.RadioChannel)
	in.Shift = strings.TrimSpace(in.Shift)
	caps := make([]string, 0, len(in.Capabilities))
	for _, c := range in.Capabilities {
		if c = strings.TrimSpace(c); c != "" {
			caps = append(caps, c)
		}
	}
	in.Capabilities = caps
	return in
}

// validate returns one validation error naming every failing field, so a form can show
// them all at once rather than one per round trip. The database CHECK constraints remain
// the backstop (design.md D6).
func validate(in domain.UnitInput, requireCode bool) error {
	var details []aoherr.Detail
	require := func(field, value string) {
		if value == "" {
			details = append(details, aoherr.FieldDetail(field, "must not be empty"))
		} else if len(value) > 120 {
			details = append(details, aoherr.FieldDetail(field, "must be 120 characters or fewer"))
		}
	}
	if requireCode {
		require("unit_code", in.UnitCode)
		if len(in.UnitCode) > 32 {
			details = append(details, aoherr.FieldDetail("unit_code", "must be 32 characters or fewer"))
		}
	}
	require("call_sign", in.CallSign)
	require("unit_type", in.UnitType)
	require("station", in.Station)
	require("sector", in.Sector)
	require("radio_channel", in.RadioChannel)
	require("shift", in.Shift)
	if in.Status == "" {
		details = append(details, aoherr.FieldDetail("status", "must not be empty"))
	} else if !domain.ValidStatus(in.Status) {
		details = append(details, aoherr.FieldDetail("status",
			"must be one of "+strings.Join(domain.Statuses, ", ")))
	}

	if len(details) == 0 {
		return nil
	}
	return aoherr.New(aoherr.ClassValidation, CodeUnitInvalid, "unit failed validation").
		WithDetails(details...)
}

// classifyWrite maps the repo's sentinel errors onto the AOH error classes.
func classifyWrite(err error) error {
	switch {
	case errors.Is(err, repo.ErrNotFound):
		return notFound()
	case errors.Is(err, repo.ErrConflict):
		return aoherr.Wrap(aoherr.ClassConflict, CodeUnitCodeTaken,
			"a unit with that code already exists", err).
			WithDetails(aoherr.FieldDetail("unit_code", "already exists"))
	case errors.Is(err, repo.ErrStale):
		return aoherr.Wrap(aoherr.ClassConflict, CodeUnitStale,
			"the unit was modified by someone else; reload and try again", err)
	case errors.Is(err, repo.ErrInvalid):
		return aoherr.Wrap(aoherr.ClassValidation, CodeUnitInvalid,
			"the unit violates a database constraint", err)
	default:
		return aoherr.Wrap(aoherr.ClassSystem, CodeUnitWriteFailed, "could not write unit", err)
	}
}

func notFound() *aoherr.Error {
	return aoherr.New(aoherr.ClassNotFound, CodeUnitNotFound, "no unit with that code")
}
