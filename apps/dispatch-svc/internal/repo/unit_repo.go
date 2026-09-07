// Package repo is the persistence layer for field units.
package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/domain"
)

// Sentinel errors the service layer classifies. Callers above never see a driver error.
var (
	// ErrNotFound — no unit with that code.
	ErrNotFound = errors.New("unit not found")
	// ErrConflict — a unique constraint was violated (duplicate unit_code for the tenant).
	ErrConflict = errors.New("unit code already exists")
	// ErrStale — the caller's occ_lock did not match the stored row (Postgres said 0 rows).
	ErrStale = errors.New("unit was modified by someone else")
	// ErrInvalid — a CHECK constraint rejected the row; the database is the last word on
	// the status vocabulary and the assignment all-or-nothing rule.
	ErrInvalid = errors.New("unit violates a database constraint")
)

// No authentication in this workshop, so the audit columns carry a fixed marker rather
// than a caller identity. Same posture as tenant_id.
const (
	actor  = "workshop"
	tenant = "workshop"
)

type UnitRepo struct {
	db     *sqlx.DB
	schema string
}

func NewUnitRepo(db *sqlx.DB, schema string) *UnitRepo {
	return &UnitRepo{db: db, schema: schema}
}

// unitRow mirrors one row of <schema>.unit. Assignment columns are nullable as a group.
type unitRow struct {
	ID           string         `db:"id"`
	UnitCode     string         `db:"unit_code"`
	CallSign     string         `db:"call_sign"`
	Status       string         `db:"status"`
	UnitType     string         `db:"unit_type"`
	Station      string         `db:"station"`
	Sector       string         `db:"sector"`
	RadioChannel string         `db:"radio_channel"`
	Shift        string         `db:"shift"`
	Capabilities pq.StringArray `db:"capabilities"`
	LastContact  time.Time      `db:"last_contact"`
	OccLock      int            `db:"occ_lock"`

	AssignmentIncidentCode sql.NullString `db:"assignment_incident_code"`
	AssignmentTitle        sql.NullString `db:"assignment_title"`
	AssignmentPriority     sql.NullString `db:"assignment_priority"`
	AssignmentLocation     sql.NullString `db:"assignment_location"`
	AssignmentSince        sql.NullTime   `db:"assignment_since"`
}

type crewRow struct {
	UnitID string `db:"unit_id"`
	Name   string `db:"name"`
	Role   string `db:"role"`
}

const unitColumns = `id, unit_code, call_sign, status, unit_type, station, sector,
	radio_channel, shift, capabilities, last_contact, occ_lock,
	assignment_incident_code, assignment_title, assignment_priority,
	assignment_location, assignment_since`

// List returns every unit ordered by unit_code, each with its crew in display order.
//
// Two queries rather than one join: a join would repeat every unit column per crew member
// and need de-duplicating in Go. With a fleet this size the round trip costs nothing and
// the mapping stays obvious.
func (r *UnitRepo) List(ctx context.Context) ([]domain.Unit, error) {
	var rows []unitRow
	query := fmt.Sprintf(`SELECT %s FROM %s.unit ORDER BY unit_code`, unitColumns, r.schema)
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, fmt.Errorf("select units: %w", err)
	}

	crewByUnit, err := r.crewFor(ctx, nil)
	if err != nil {
		return nil, err
	}

	units := make([]domain.Unit, 0, len(rows))
	for _, row := range rows {
		units = append(units, toDomain(row, crewByUnit[row.ID]))
	}
	return units, nil
}

// Get returns one unit by its human-readable code, or ErrNotFound.
func (r *UnitRepo) Get(ctx context.Context, unitCode string) (domain.Unit, error) {
	var row unitRow
	query := fmt.Sprintf(`SELECT %s FROM %s.unit WHERE unit_code = $1 AND tenant_id = $2`,
		unitColumns, r.schema)
	if err := r.db.GetContext(ctx, &row, query, unitCode, tenant); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Unit{}, ErrNotFound
		}
		return domain.Unit{}, fmt.Errorf("select unit %s: %w", unitCode, err)
	}

	crewByUnit, err := r.crewFor(ctx, []string{row.ID})
	if err != nil {
		return domain.Unit{}, err
	}
	return toDomain(row, crewByUnit[row.ID]), nil
}

// Create inserts a unit with no crew and no assignment, and returns it as stored.
func (r *UnitRepo) Create(ctx context.Context, in domain.UnitInput) (domain.Unit, error) {
	query := fmt.Sprintf(`
		INSERT INTO %s.unit
			(unit_code, call_sign, status, unit_type, station, sector, radio_channel, shift,
			 capabilities, last_contact, created_by, updated_by, tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now(), $10, $10, $11)
		RETURNING %s`, r.schema, unitColumns)

	var row unitRow
	err := r.db.GetContext(ctx, &row, query,
		in.UnitCode, in.CallSign, in.Status, in.UnitType, in.Station, in.Sector,
		in.RadioChannel, in.Shift, pq.Array(capsOrEmpty(in.Capabilities)), actor, tenant)
	if err != nil {
		return domain.Unit{}, mapWriteError("insert unit", err)
	}
	return toDomain(row, nil), nil
}

// Update replaces the editable fields of the unit identified by unitCode, guarded by
// occLock. It bumps occ_lock and last_contact (a write from the console counts as a
// contact — design.md D4). ErrStale when the guard matched nothing; ErrNotFound when the
// code does not exist at all, so the two are distinguishable to the caller.
func (r *UnitRepo) Update(ctx context.Context, unitCode string, occLock int, in domain.UnitInput) (domain.Unit, error) {
	query := fmt.Sprintf(`
		UPDATE %s.unit SET
			call_sign = $3, status = $4, unit_type = $5, station = $6, sector = $7,
			radio_channel = $8, shift = $9, capabilities = $10,
			last_contact = now(), updated_by = $11, occ_lock = occ_lock + 1
		WHERE unit_code = $1 AND tenant_id = $12 AND occ_lock = $2
		RETURNING %s`, r.schema, unitColumns)

	var row unitRow
	err := r.db.GetContext(ctx, &row, query,
		unitCode, occLock, in.CallSign, in.Status, in.UnitType, in.Station, in.Sector,
		in.RadioChannel, in.Shift, pq.Array(capsOrEmpty(in.Capabilities)), actor, tenant)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Unit{}, r.staleOrMissing(ctx, unitCode)
		}
		return domain.Unit{}, mapWriteError("update unit "+unitCode, err)
	}

	crewByUnit, err := r.crewFor(ctx, []string{row.ID})
	if err != nil {
		return domain.Unit{}, err
	}
	return toDomain(row, crewByUnit[row.ID]), nil
}

// Delete removes the unit (crew cascades) guarded by occLock.
func (r *UnitRepo) Delete(ctx context.Context, unitCode string, occLock int) error {
	query := fmt.Sprintf(`DELETE FROM %s.unit WHERE unit_code = $1 AND tenant_id = $2 AND occ_lock = $3`, r.schema)
	res, err := r.db.ExecContext(ctx, query, unitCode, tenant, occLock)
	if err != nil {
		return mapWriteError("delete unit "+unitCode, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete unit %s: rows affected: %w", unitCode, err)
	}
	if n == 0 {
		return r.staleOrMissing(ctx, unitCode)
	}
	return nil
}

// staleOrMissing tells a guarded write that matched nothing apart: does the unit exist
// with a different occ_lock (stale), or not at all (not found)?
func (r *UnitRepo) staleOrMissing(ctx context.Context, unitCode string) error {
	var exists bool
	query := fmt.Sprintf(`SELECT EXISTS (SELECT 1 FROM %s.unit WHERE unit_code = $1 AND tenant_id = $2)`, r.schema)
	if err := r.db.GetContext(ctx, &exists, query, unitCode, tenant); err != nil {
		return fmt.Errorf("check unit %s: %w", unitCode, err)
	}
	if exists {
		return ErrStale
	}
	return ErrNotFound
}

// mapWriteError turns Postgres constraint violations into the repo's sentinel errors so
// the service can classify them. Anything else stays a wrapped driver error (→ 500).
func mapWriteError(op string, err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return fmt.Errorf("%s: %w", op, ErrConflict)
		case "23514": // check_violation — name the constraint so the message is actionable
			return fmt.Errorf("%s: %s: %w", op, pgErr.ConstraintName, ErrInvalid)
		}
	}
	return fmt.Errorf("%s: %w", op, err)
}

func capsOrEmpty(caps []string) []string {
	if caps == nil {
		return []string{}
	}
	return caps
}

// crewFor loads crew for the given unit ids, or for every unit when ids is nil.
func (r *UnitRepo) crewFor(ctx context.Context, unitIDs []string) (map[string][]domain.Crew, error) {
	var (
		rows  []crewRow
		query string
		args  []any
	)
	if unitIDs == nil {
		query = fmt.Sprintf(
			`SELECT unit_id, name, role FROM %s.unit_crew ORDER BY unit_id, sort_order`, r.schema)
	} else {
		query = fmt.Sprintf(
			`SELECT unit_id, name, role FROM %s.unit_crew WHERE unit_id = ANY($1) ORDER BY unit_id, sort_order`,
			r.schema)
		args = append(args, pq.Array(unitIDs))
	}
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("select crew: %w", err)
	}

	byUnit := make(map[string][]domain.Crew, len(rows))
	for _, row := range rows {
		byUnit[row.UnitID] = append(byUnit[row.UnitID], domain.Crew{Name: row.Name, Role: row.Role})
	}
	return byUnit, nil
}

func toDomain(row unitRow, crew []domain.Crew) domain.Unit {
	unit := domain.Unit{
		UnitCode:     row.UnitCode,
		CallSign:     row.CallSign,
		Status:       row.Status,
		UnitType:     row.UnitType,
		Station:      row.Station,
		Sector:       row.Sector,
		RadioChannel: row.RadioChannel,
		Shift:        row.Shift,
		// Never nil: the wire should carry [] rather than null for an empty list.
		Capabilities: []string(row.Capabilities),
		Crew:         crew,
		LastContact:  row.LastContact,
		OccLock:      row.OccLock,
	}
	if unit.Capabilities == nil {
		unit.Capabilities = []string{}
	}
	if unit.Crew == nil {
		unit.Crew = []domain.Crew{}
	}

	// The schema's CHECK keeps these all-null or all-non-null, so one probe is enough.
	if row.AssignmentIncidentCode.Valid {
		unit.Assignment = &domain.Assignment{
			IncidentCode: row.AssignmentIncidentCode.String,
			Title:        row.AssignmentTitle.String,
			Priority:     row.AssignmentPriority.String,
			Location:     row.AssignmentLocation.String,
			Since:        row.AssignmentSince.Time,
		}
	}
	return unit
}
