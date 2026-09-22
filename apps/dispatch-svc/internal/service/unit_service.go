// Package service holds the domain layer: it turns repository results and failures into
// the shapes the handler renders, and it is where the caller's roles decide what they may
// do.
package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"

	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/auth"
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/domain"
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/projection"
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/repo"
)

// UnitStore is the persistence surface this service needs. Declared here, at the
// consumer, so tests can substitute a fake without a database or a mock framework.
type UnitStore interface {
	List(ctx context.Context, caller repo.Caller) ([]domain.Unit, error)
	Get(ctx context.Context, caller repo.Caller, unitCode string) (domain.Unit, error)
	Create(ctx context.Context, caller repo.Caller, in domain.UnitInput) (domain.Unit, []projection.Row, error)
	Update(ctx context.Context, caller repo.Caller, unitCode string, occLock int, in domain.UnitInput) (domain.Unit, []projection.Row, error)
	Delete(ctx context.Context, caller repo.Caller, unitCode string, occLock int) ([]projection.Row, error)

	IsSeeded(ctx context.Context, tenantID string) (bool, error)
	SeedTenant(ctx context.Context, caller repo.Caller, roster []domain.SeedUnit) (bool, []projection.Row, error)
}

// UnitReader is kept for callers that only read.
type UnitReader interface {
	List(ctx context.Context, caller repo.Caller) ([]domain.Unit, error)
	Get(ctx context.Context, caller repo.Caller, unitCode string) (domain.Unit, error)
}

// Projector hands committed outbox rows to the thing that delivers them. Declared here so
// the service does not depend on the worker's construction, and so a test can assert what
// was enqueued without a gis-service.
//
// Deliver must not block: a gis-service outage must never fail or slow a unit write.
type Projector interface {
	Deliver(rows []projection.Row, token string, tokenExpiry time.Time)
}

type UnitService struct {
	units     UnitStore
	projector Projector
}

func NewUnitService(units UnitStore, projector Projector) *UnitService {
	return &UnitService{units: units, projector: projector}
}

// --------------------------------------------------------------------------- //
// Authorisation and seeding — the two things every entry point passes through
// --------------------------------------------------------------------------- //

// begin resolves the caller, checks they hold the permission this operation needs, and
// gives an unseeded tenant its baseline roster before the request is answered.
//
// The two checks are deliberately distinct. `need` is the operation's own gate, which
// admits a viewer on a read. Seeding is a *write*, so it has its own gate — a dispatcher
// role — applied inside ensureSeeded. Collapsing them would either stop a dispatcher's
// read from seeding (which is the trigger the design specifies) or let a viewer's read
// perform writes, which design.md D2a forbids elsewhere in this same service.
func (s *UnitService) begin(ctx context.Context, need Permission) (repo.Caller, error) {
	id, err := auth.MustFromContext(ctx)
	if err != nil {
		return repo.Caller{}, err
	}
	if !HasPermission(id.Roles, need) {
		return repo.Caller{}, forbidden(need)
	}
	if err := s.ensureSeeded(ctx, id); err != nil {
		return repo.Caller{}, err
	}
	return repo.Caller{TenantID: id.TenantID, Subject: id.Subject}, nil
}

// ensureSeeded seeds the caller's tenant the first time a dispatcher touches the service.
//
// The trigger is a marker row, not an emptiness check: the console ships a working delete,
// so "this tenant has no units" would silently resurrect the whole roster for a dispatcher
// who had just deleted the last one. Seed once per tenant, ever (design.md D12).
//
// A failure fails the triggering request rather than being logged and answered with an
// empty roster: an empty console with no error is indistinguishable, to the operator, from
// a tenant nobody has seeded and from one whose units were deleted on purpose.
func (s *UnitService) ensureSeeded(ctx context.Context, id auth.Identity) error {
	// Seeding is a write, so only a caller who may write can trigger it. A viewer on a
	// fresh stack sees an empty roster, which the console renders as an explicit state.
	if !HasPermission(id.Roles, PermissionWrite) {
		return nil
	}

	// Cheap pre-check so an ordinary request does not open a transaction it would
	// immediately roll back. It is not the guard — SeedTenant's ON CONFLICT is.
	seeded, err := s.units.IsSeeded(ctx, id.TenantID)
	if err != nil {
		return aoherr.Wrap(aoherr.ClassSystem, CodeTenantSeedFailed,
			"could not determine whether this tenant has been seeded", err)
	}
	if seeded {
		return nil
	}

	caller := repo.Caller{TenantID: id.TenantID, Subject: id.Subject}
	done, rows, err := s.units.SeedTenant(ctx, caller, baselineRoster())
	if err != nil {
		return aoherr.Wrap(aoherr.ClassSystem, CodeTenantSeedFailed,
			"could not seed this tenant's baseline roster", err)
	}
	if done {
		// The seed's projections travel the ordinary path, on the token of the
		// dispatcher who triggered it.
		s.projector.Deliver(rows, id.BearerToken, id.ExpiresAt)
	}
	return nil
}

// deliver hands committed outbox rows to the projector as the caller who wrote them.
func (s *UnitService) deliver(ctx context.Context, rows []projection.Row) {
	if len(rows) == 0 {
		return
	}
	// The identity is already known to be present — begin() resolved it.
	if id, ok := auth.FromContext(ctx); ok {
		s.projector.Deliver(rows, id.BearerToken, id.ExpiresAt)
	}
}

// --------------------------------------------------------------------------- //
// Operations
// --------------------------------------------------------------------------- //

func (s *UnitService) List(ctx context.Context) ([]domain.Unit, error) {
	caller, err := s.begin(ctx, PermissionRead)
	if err != nil {
		return nil, err
	}
	units, err := s.units.List(ctx, caller)
	if err != nil {
		return nil, aoherr.Wrap(aoherr.ClassSystem, CodeUnitReadFailed, "could not read units", err)
	}
	return units, nil
}

func (s *UnitService) Get(ctx context.Context, unitCode string) (domain.Unit, error) {
	caller, err := s.begin(ctx, PermissionRead)
	if err != nil {
		return domain.Unit{}, err
	}
	if strings.TrimSpace(unitCode) == "" {
		return domain.Unit{}, aoherr.New(aoherr.ClassValidation, CodeUnitCodeRequired,
			"unit code must not be empty")
	}

	unit, err := s.units.Get(ctx, caller, unitCode)
	if err != nil {
		// A missing row is a not-found, not a system failure — without this the driver
		// error would surface as a 500. Another tenant's unit arrives here too, and is
		// deliberately indistinguishable from one that does not exist.
		if errors.Is(err, repo.ErrNotFound) {
			return domain.Unit{}, notFound()
		}
		return domain.Unit{}, aoherr.Wrap(aoherr.ClassSystem, CodeUnitReadFailed, "could not read unit", err)
	}
	return unit, nil
}

// Create validates and inserts a new unit in the caller's tenant.
func (s *UnitService) Create(ctx context.Context, in domain.UnitInput) (domain.Unit, error) {
	caller, err := s.begin(ctx, PermissionWrite)
	if err != nil {
		return domain.Unit{}, err
	}
	in = normalise(in)
	if err := validate(in, true); err != nil {
		return domain.Unit{}, err
	}
	unit, rows, err := s.units.Create(ctx, caller, in)
	if err != nil {
		return domain.Unit{}, classifyWrite(err)
	}
	s.deliver(ctx, rows)
	return unit, nil
}

// Update validates and replaces the unit's editable fields, guarded by occLock.
//
// A body with no `position` clears an existing one — a replace replaces, and the
// projection that follows removes the geo-entity rather than leaving a stale marker.
func (s *UnitService) Update(ctx context.Context, unitCode string, occLock int, in domain.UnitInput) (domain.Unit, error) {
	caller, err := s.begin(ctx, PermissionWrite)
	if err != nil {
		return domain.Unit{}, err
	}
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
	unit, rows, err := s.units.Update(ctx, caller, unitCode, occLock, in)
	if err != nil {
		return domain.Unit{}, classifyWrite(err)
	}
	s.deliver(ctx, rows)
	return unit, nil
}

// Delete removes the unit, guarded by occLock, and enqueues the removal of its
// geo-entity.
func (s *UnitService) Delete(ctx context.Context, unitCode string, occLock int) error {
	caller, err := s.begin(ctx, PermissionWrite)
	if err != nil {
		return err
	}
	if strings.TrimSpace(unitCode) == "" {
		return aoherr.New(aoherr.ClassValidation, CodeUnitCodeRequired, "unit code must not be empty")
	}
	rows, err := s.units.Delete(ctx, caller, unitCode, occLock)
	if err != nil {
		return classifyWrite(err)
	}
	s.deliver(ctx, rows)
	return nil
}

// --------------------------------------------------------------------------- //
// Validation
// --------------------------------------------------------------------------- //

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

	// A position is all-or-nothing by shape — omit the object to say there is none — so
	// only the ranges need checking here. The database CHECKs are the backstop.
	if in.Position != nil {
		if in.Position.Lon < -180 || in.Position.Lon > 180 {
			details = append(details, aoherr.FieldDetail("position.lon",
				"must be between -180 and 180"))
		}
		if in.Position.Lat < -90 || in.Position.Lat > 90 {
			details = append(details, aoherr.FieldDetail("position.lat",
				"must be between -90 and 90"))
		}
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

// forbidden is the 403 for an authenticated caller whose roles do not carry the
// permission. It names the permission, not the roles: which roles grant it is this
// service's business, not the caller's.
func forbidden(need Permission) *aoherr.Error {
	return aoherr.New(aoherr.ClassAuthorization, CodeUnitForbidden,
		"the caller's roles do not permit this operation").
		WithDetails(aoherr.FieldDetail("permission", string(need)))
}
