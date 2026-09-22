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
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/projection"
)

// Sentinel errors the service layer classifies. Callers above never see a driver error.
var (
	// ErrNotFound — no unit with that code in the caller's tenant. A unit that exists
	// only under another tenant is indistinguishable from one that does not exist.
	ErrNotFound = errors.New("unit not found")
	// ErrConflict — a unique constraint was violated (duplicate unit_code for the tenant).
	ErrConflict = errors.New("unit code already exists")
	// ErrStale — the caller's occ_lock did not match the stored row (Postgres said 0 rows).
	ErrStale = errors.New("unit was modified by someone else")
	// ErrInvalid — a CHECK constraint rejected the row; the database is the last word on
	// the status vocabulary, the assignment all-or-nothing rule and the coordinate ranges.
	ErrInvalid = errors.New("unit violates a database constraint")
)

// Caller is the identity every write is attributed to and every read is scoped by. It
// comes from the bearer token — never from a request body, and never a placeholder: the
// pre-auth 'workshop' / 'system' literals are gone, and migration 0004 deletes the rows
// that carried them.
type Caller struct {
	TenantID string
	Subject  string
}

type UnitRepo struct {
	db     *sqlx.DB
	schema string
}

func NewUnitRepo(db *sqlx.DB, schema string) *UnitRepo {
	return &UnitRepo{db: db, schema: schema}
}

// unitRow mirrors one row of <schema>.unit. Assignment and position columns are each
// nullable as a group.
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

	AssignmentIncidentCode sql.NullString  `db:"assignment_incident_code"`
	AssignmentTitle        sql.NullString  `db:"assignment_title"`
	AssignmentDescription  sql.NullString  `db:"assignment_description"`
	AssignmentPriority     sql.NullString  `db:"assignment_priority"`
	AssignmentLocation     sql.NullString  `db:"assignment_location"`
	AssignmentLon          sql.NullFloat64 `db:"assignment_lon"`
	AssignmentLat          sql.NullFloat64 `db:"assignment_lat"`
	AssignmentSince        sql.NullTime    `db:"assignment_since"`

	PositionLon sql.NullFloat64 `db:"position_lon"`
	PositionLat sql.NullFloat64 `db:"position_lat"`
	PositionAt  sql.NullTime    `db:"position_at"`
}

type crewRow struct {
	UnitID string `db:"unit_id"`
	Name   string `db:"name"`
	Role   string `db:"role"`
}

const unitColumns = `id, unit_code, call_sign, status, unit_type, station, sector,
	radio_channel, shift, capabilities, last_contact, occ_lock,
	assignment_incident_code, assignment_title, assignment_description,
	assignment_priority, assignment_location, assignment_lon, assignment_lat,
	assignment_since,
	position_lon, position_lat, position_at`

// --------------------------------------------------------------------------- //
// Reads
// --------------------------------------------------------------------------- //

// List returns the caller's tenant's units ordered by unit_code, each with its crew in
// display order.
//
// Two queries rather than one join: a join would repeat every unit column per crew member
// and need de-duplicating in Go. With a fleet this size the round trip costs nothing and
// the mapping stays obvious.
func (r *UnitRepo) List(ctx context.Context, caller Caller) ([]domain.Unit, error) {
	var rows []unitRow
	query := fmt.Sprintf(
		`SELECT %s FROM %s.unit WHERE tenant_id = $1 ORDER BY unit_code`, unitColumns, r.schema)
	if err := r.db.SelectContext(ctx, &rows, query, caller.TenantID); err != nil {
		return nil, fmt.Errorf("select units: %w", err)
	}

	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	crewByUnit, err := r.crewFor(ctx, r.db, ids)
	if err != nil {
		return nil, err
	}

	units := make([]domain.Unit, 0, len(rows))
	for _, row := range rows {
		units = append(units, toDomain(row, crewByUnit[row.ID]))
	}
	return units, nil
}

// Get returns one unit by its human-readable code within the caller's tenant, or
// ErrNotFound.
func (r *UnitRepo) Get(ctx context.Context, caller Caller, unitCode string) (domain.Unit, error) {
	row, err := r.getRow(ctx, r.db, caller, unitCode)
	if err != nil {
		return domain.Unit{}, err
	}
	crewByUnit, err := r.crewFor(ctx, r.db, []string{row.ID})
	if err != nil {
		return domain.Unit{}, err
	}
	return toDomain(row, crewByUnit[row.ID]), nil
}

// --------------------------------------------------------------------------- //
// Writes — each one enqueues its GIS projection in the same transaction
// --------------------------------------------------------------------------- //

// Create inserts a unit with no crew and no assignment, and returns it as stored,
// alongside the outbox row its position state produced.
func (r *UnitRepo) Create(ctx context.Context, caller Caller, in domain.UnitInput) (domain.Unit, []projection.Row, error) {
	var (
		unit domain.Unit
		rows []projection.Row
	)
	err := r.inTx(ctx, func(tx *sqlx.Tx) error {
		query := fmt.Sprintf(`
			INSERT INTO %s.unit
				(unit_code, call_sign, status, unit_type, station, sector, radio_channel, shift,
				 capabilities, last_contact, position_lon, position_lat, position_at,
				 created_by, updated_by, tenant_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now(), $10, $11, $12, $13, $13, $14)
			RETURNING %s`, r.schema, unitColumns)

		lon, lat, at := positionArgs(in.Position)
		var row unitRow
		if err := tx.GetContext(ctx, &row, query,
			in.UnitCode, in.CallSign, in.Status, in.UnitType, in.Station, in.Sector,
			in.RadioChannel, in.Shift, pq.Array(capsOrEmpty(in.Capabilities)),
			lon, lat, at, caller.Subject, caller.TenantID); err != nil {
			return mapWriteError("insert unit", err)
		}

		unit = toDomain(row, nil)
		out, err := r.enqueue(ctx, tx, caller, unit)
		if err != nil {
			return err
		}
		rows = append(rows, out)
		return nil
	})
	if err != nil {
		return domain.Unit{}, nil, err
	}
	return unit, rows, nil
}

// Update replaces the editable fields of the unit identified by unitCode, guarded by
// occLock, within the caller's tenant.
//
// It bumps occ_lock and last_contact — a write from the console counts as a contact
// (dispatch-units-crud D4) — and deliberately does NOT carry last_contact into
// position_at. position_at moves only when the write supplies a position, so an operator
// editing a radio channel does not make the map's fix age a lie (design.md D11).
//
// ErrStale when the guard matched nothing; ErrNotFound when the code does not exist in
// this tenant at all, so the two are distinguishable to the caller.
func (r *UnitRepo) Update(ctx context.Context, caller Caller, unitCode string, occLock int, in domain.UnitInput) (domain.Unit, []projection.Row, error) {
	var (
		unit domain.Unit
		rows []projection.Row
	)
	err := r.inTx(ctx, func(tx *sqlx.Tx) error {
		query := fmt.Sprintf(`
			UPDATE %s.unit SET
				call_sign = $3, status = $4, unit_type = $5, station = $6, sector = $7,
				radio_channel = $8, shift = $9, capabilities = $10,
				position_lon = $11, position_lat = $12, position_at = $13,
				last_contact = now(), updated_by = $14, occ_lock = occ_lock + 1
			WHERE unit_code = $1 AND tenant_id = $15 AND occ_lock = $2
			RETURNING %s`, r.schema, unitColumns)

		lon, lat, at := positionArgs(in.Position)
		var row unitRow
		err := tx.GetContext(ctx, &row, query,
			unitCode, occLock, in.CallSign, in.Status, in.UnitType, in.Station, in.Sector,
			in.RadioChannel, in.Shift, pq.Array(capsOrEmpty(in.Capabilities)),
			lon, lat, at, caller.Subject, caller.TenantID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return r.staleOrMissing(ctx, tx, caller, unitCode)
			}
			return mapWriteError("update unit "+unitCode, err)
		}

		crewByUnit, err := r.crewFor(ctx, tx, []string{row.ID})
		if err != nil {
			return err
		}
		unit = toDomain(row, crewByUnit[row.ID])

		out, err := r.enqueue(ctx, tx, caller, unit)
		if err != nil {
			return err
		}
		rows = append(rows, out)
		return nil
	})
	if err != nil {
		return domain.Unit{}, nil, err
	}
	return unit, rows, nil
}

// Delete removes the unit (crew cascades) guarded by occLock, and enqueues the removal of
// its geo-entity.
func (r *UnitRepo) Delete(ctx context.Context, caller Caller, unitCode string, occLock int) ([]projection.Row, error) {
	var rows []projection.Row
	err := r.inTx(ctx, func(tx *sqlx.Tx) error {
		// Read before deleting: the outbox row names the unit the projection is about,
		// and after the DELETE there is nothing left to name it from.
		row, err := r.getRow(ctx, tx, caller, unitCode)
		if err != nil {
			return err
		}

		query := fmt.Sprintf(
			`DELETE FROM %s.unit WHERE unit_code = $1 AND tenant_id = $2 AND occ_lock = $3`, r.schema)
		res, err := tx.ExecContext(ctx, query, unitCode, caller.TenantID, occLock)
		if err != nil {
			return mapWriteError("delete unit "+unitCode, err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("delete unit %s: rows affected: %w", unitCode, err)
		}
		if n == 0 {
			return r.staleOrMissing(ctx, tx, caller, unitCode)
		}

		// A deleted unit has no position, so IntentFor yields a delete.
		deleted := toDomain(row, nil)
		deleted.Position = nil
		out, err := r.enqueue(ctx, tx, caller, deleted)
		if err != nil {
			return err
		}
		rows = append(rows, out)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// --------------------------------------------------------------------------- //
// Seed
// --------------------------------------------------------------------------- //

// IsSeeded reports whether this tenant's marker row exists.
//
// A cheap pre-check so an ordinary request does not open a transaction it will
// immediately roll back. It is NOT the guard: SeedTenant's ON CONFLICT is, because two
// requests can pass this check at the same moment.
func (r *UnitRepo) IsSeeded(ctx context.Context, tenantID string) (bool, error) {
	var seeded bool
	query := fmt.Sprintf(
		`SELECT EXISTS (SELECT 1 FROM %s.tenant_seed WHERE tenant_id = $1)`, r.schema)
	if err := r.db.GetContext(ctx, &seeded, query, tenantID); err != nil {
		return false, fmt.Errorf("check tenant seed: %w", err)
	}
	return seeded, nil
}

// SeedTenant writes the baseline roster for a tenant that has never been seeded, in one
// transaction, and reports whether it did.
//
// The marker insert is the transaction's FIRST statement and reports whether it inserted.
// ON CONFLICT DO NOTHING raises nothing and returns no error, so a guard that ran the
// insert and carried on would let both racers write the whole roster, bump every occ_lock
// and enqueue two full sets of outbox rows on two different operators' tokens — with row
// counts that still looked correct.
//
// The roster rows are plain inserts with no ON CONFLICT clause: the marker makes the seed
// run at most once, so a conflict branch is unreachable by design and would only convert
// a lost race into a silent double write instead of a loud failure.
func (r *UnitRepo) SeedTenant(ctx context.Context, caller Caller, roster []domain.SeedUnit) (bool, []projection.Row, error) {
	var (
		seeded bool
		rows   []projection.Row
	)
	err := r.inTx(ctx, func(tx *sqlx.Tx) error {
		var marker string
		err := tx.GetContext(ctx, &marker, fmt.Sprintf(
			`INSERT INTO %s.tenant_seed (tenant_id, seeded_by) VALUES ($1, $2)
			 ON CONFLICT (tenant_id) DO NOTHING
			 RETURNING tenant_id`, r.schema), caller.TenantID, caller.Subject)
		if errors.Is(err, sql.ErrNoRows) {
			// Someone else got there first. Abort before any roster row is written.
			return nil
		}
		if err != nil {
			return fmt.Errorf("claim tenant seed: %w", err)
		}

		for _, unit := range roster {
			out, err := r.seedOne(ctx, tx, caller, unit)
			if err != nil {
				return err
			}
			rows = append(rows, out)
		}
		seeded = true
		return nil
	})
	if err != nil {
		return false, nil, err
	}
	return seeded, rows, nil
}

func (r *UnitRepo) seedOne(ctx context.Context, tx *sqlx.Tx, caller Caller, seed domain.SeedUnit) (projection.Row, error) {
	query := fmt.Sprintf(`
		INSERT INTO %s.unit
			(unit_code, call_sign, status, unit_type, station, sector, radio_channel, shift,
			 capabilities, last_contact, position_lon, position_lat, position_at,
			 assignment_incident_code, assignment_title, assignment_description,
			 assignment_priority, assignment_location, assignment_lon, assignment_lat,
			 assignment_since,
			 created_by, updated_by, tenant_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9,
		        now() - make_interval(secs => $10), $11, $12, $13,
		        $14, $15, $16, $17, $18, $19, $20, $21, $22, $22, $23)
		RETURNING %s`, r.schema, unitColumns)

	lon, lat, at := positionArgs(seed.Position)
	// The described columns move together: the all-or-nothing CHECK rejects a partial
	// assignment. The point is its own pair and stays null when unresolved.
	var incident, title, description, priority, location, since any
	var incidentLon, incidentLat any
	if seed.Assignment != nil {
		incident, title = seed.Assignment.IncidentCode, seed.Assignment.Title
		description, priority = seed.Assignment.Description, seed.Assignment.Priority
		location, since = seed.Assignment.Location, seed.Assignment.Since
		if seed.Assignment.Point != nil {
			incidentLon, incidentLat = seed.Assignment.Point.Lon, seed.Assignment.Point.Lat
		}
	}

	var row unitRow
	if err := tx.GetContext(ctx, &row, query,
		seed.UnitCode, seed.CallSign, seed.Status, seed.UnitType, seed.Station, seed.Sector,
		seed.RadioChannel, seed.Shift, pq.Array(capsOrEmpty(seed.Capabilities)),
		seed.LastContactAgo.Seconds(), lon, lat, at,
		incident, title, description, priority, location, incidentLon, incidentLat, since,
		caller.Subject, caller.TenantID); err != nil {
		return projection.Row{}, mapWriteError("seed unit "+seed.UnitCode, err)
	}

	for i, member := range seed.Crew {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
			INSERT INTO %s.unit_crew (unit_id, name, role, sort_order, created_by, updated_by, tenant_id)
			VALUES ($1, $2, $3, $4, $5, $5, $6)`, r.schema),
			row.ID, member.Name, member.Role, i, caller.Subject, caller.TenantID); err != nil {
			return projection.Row{}, mapWriteError("seed crew for "+seed.UnitCode, err)
		}
	}

	// The projection travels the same path, on the same token, as any other write.
	return r.enqueue(ctx, tx, caller, toDomain(row, seed.Crew))
}

// --------------------------------------------------------------------------- //
// Outbox
// --------------------------------------------------------------------------- //

// enqueue writes the unit's projection row. It runs inside the caller's transaction, so
// the unit row and its pending projection commit or roll back together.
func (r *UnitRepo) enqueue(ctx context.Context, tx *sqlx.Tx, caller Caller, unit domain.Unit) (projection.Row, error) {
	intent := projection.IntentFor(unit)
	payload, err := projection.Payload(unit)
	if err != nil {
		return projection.Row{}, err
	}

	var id string
	query := fmt.Sprintf(`
		INSERT INTO %s.gis_outbox (unit_code, intent, payload, created_by, updated_by, tenant_id)
		VALUES ($1, $2, $3, $4, $4, $5)
		RETURNING id`, r.schema)
	if err := tx.GetContext(ctx, &id, query,
		unit.UnitCode, string(intent), payload, caller.Subject, caller.TenantID); err != nil {
		return projection.Row{}, fmt.Errorf("enqueue projection for %s: %w", unit.UnitCode, err)
	}

	return projection.Row{ID: id, UnitCode: unit.UnitCode, Intent: intent, Payload: payload}, nil
}

// MarkDelivered records that gis-service accepted the projection.
func (r *UnitRepo) MarkDelivered(ctx context.Context, id string) error {
	query := fmt.Sprintf(
		`UPDATE %s.gis_outbox SET delivered_at = now(), attempts = attempts + 1, last_error = NULL,
		 occ_lock = occ_lock + 1 WHERE id = $1`, r.schema)
	if _, err := r.db.ExecContext(ctx, query, id); err != nil {
		return fmt.Errorf("mark projection %s delivered: %w", id, err)
	}
	return nil
}

// RecordAttempt leaves the row pending with its attempt count and last error, so a
// stranded projection is inspectable rather than silent.
func (r *UnitRepo) RecordAttempt(ctx context.Context, id string, lastError string) error {
	query := fmt.Sprintf(
		`UPDATE %s.gis_outbox SET attempts = attempts + 1, last_error = $2, occ_lock = occ_lock + 1
		 WHERE id = $1`, r.schema)
	if _, err := r.db.ExecContext(ctx, query, id, lastError); err != nil {
		return fmt.Errorf("record projection attempt %s: %w", id, err)
	}
	return nil
}

// --------------------------------------------------------------------------- //
// Helpers
// --------------------------------------------------------------------------- //

// queryer is the read surface shared by *sqlx.DB and *sqlx.Tx, so the same helpers serve
// a bare read and one inside a write transaction.
type queryer interface {
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
}

// inTx runs fn in a transaction, rolling back on error or panic.
func (r *UnitRepo) inTx(ctx context.Context, fn func(tx *sqlx.Tx) error) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // no-op once committed

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (r *UnitRepo) getRow(ctx context.Context, q queryer, caller Caller, unitCode string) (unitRow, error) {
	var row unitRow
	query := fmt.Sprintf(`SELECT %s FROM %s.unit WHERE unit_code = $1 AND tenant_id = $2`,
		unitColumns, r.schema)
	if err := q.GetContext(ctx, &row, query, unitCode, caller.TenantID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return unitRow{}, ErrNotFound
		}
		return unitRow{}, fmt.Errorf("select unit %s: %w", unitCode, err)
	}
	return row, nil
}

// staleOrMissing tells a guarded write that matched nothing apart: does the unit exist in
// this tenant with a different occ_lock (stale), or not at all (not found)? Another
// tenant's row answers "not at all", which is the point.
// Assign commits a unit to an incident and moves it to status.
//
// The whole described group is written in one statement, so the all-or-nothing CHECK
// cannot see a partial assignment. `assignment_since` is stamped here rather than taken
// from the client: re-dispatching to a different incident starts a new commitment.
func (r *UnitRepo) Assign(ctx context.Context, caller Caller, unitCode string, occLock int, in domain.AssignmentInput, status string) (domain.Unit, []projection.Row, error) {
	var (
		unit domain.Unit
		rows []projection.Row
	)
	err := r.inTx(ctx, func(tx *sqlx.Tx) error {
		query := fmt.Sprintf(`
			UPDATE %s.unit SET
				assignment_incident_code = $3, assignment_title = $4,
				assignment_description = $5, assignment_priority = $6,
				assignment_location = $7, assignment_lon = $8, assignment_lat = $9,
				assignment_since = now(), status = $10,
				last_contact = now(), updated_by = $11, occ_lock = occ_lock + 1
			WHERE unit_code = $1 AND tenant_id = $12 AND occ_lock = $2
			RETURNING %s`, r.schema, unitColumns)

		var lon, lat any
		if in.Point != nil {
			lon, lat = in.Point.Lon, in.Point.Lat
		}

		var row unitRow
		err := tx.GetContext(ctx, &row, query,
			unitCode, occLock, in.IncidentCode, in.Title, in.Description, in.Priority,
			in.Location, lon, lat, status, caller.Subject, caller.TenantID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return r.staleOrMissing(ctx, tx, caller, unitCode)
			}
			return mapWriteError("assign unit "+unitCode, err)
		}
		return r.finishWrite(ctx, tx, caller, row, &unit, &rows)
	})
	if err != nil {
		return domain.Unit{}, nil, err
	}
	return unit, rows, nil
}

// ClearAssignment stands a unit down and moves it to status.
//
// Every described column and the point go to NULL together, which is the only shape the
// all-or-nothing CHECKs accept for an unassigned unit.
func (r *UnitRepo) ClearAssignment(ctx context.Context, caller Caller, unitCode string, occLock int, status string) (domain.Unit, []projection.Row, error) {
	var (
		unit domain.Unit
		rows []projection.Row
	)
	err := r.inTx(ctx, func(tx *sqlx.Tx) error {
		query := fmt.Sprintf(`
			UPDATE %s.unit SET
				assignment_incident_code = NULL, assignment_title = NULL,
				assignment_description = NULL, assignment_priority = NULL,
				assignment_location = NULL, assignment_lon = NULL, assignment_lat = NULL,
				assignment_since = NULL, status = $3,
				last_contact = now(), updated_by = $4, occ_lock = occ_lock + 1
			WHERE unit_code = $1 AND tenant_id = $5 AND occ_lock = $2
			RETURNING %s`, r.schema, unitColumns)

		var row unitRow
		err := tx.GetContext(ctx, &row, query,
			unitCode, occLock, status, caller.Subject, caller.TenantID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return r.staleOrMissing(ctx, tx, caller, unitCode)
			}
			return mapWriteError("stand down unit "+unitCode, err)
		}
		return r.finishWrite(ctx, tx, caller, row, &unit, &rows)
	})
	if err != nil {
		return domain.Unit{}, nil, err
	}
	return unit, rows, nil
}

// finishWrite loads the written unit's crew and enqueues its projection.
func (r *UnitRepo) finishWrite(ctx context.Context, tx *sqlx.Tx, caller Caller, row unitRow, unit *domain.Unit, rows *[]projection.Row) error {
	crewByUnit, err := r.crewFor(ctx, tx, []string{row.ID})
	if err != nil {
		return err
	}
	*unit = toDomain(row, crewByUnit[row.ID])

	out, err := r.enqueue(ctx, tx, caller, *unit)
	if err != nil {
		return err
	}
	*rows = append(*rows, out)
	return nil
}

func (r *UnitRepo) staleOrMissing(ctx context.Context, q queryer, caller Caller, unitCode string) error {
	var exists bool
	query := fmt.Sprintf(
		`SELECT EXISTS (SELECT 1 FROM %s.unit WHERE unit_code = $1 AND tenant_id = $2)`, r.schema)
	if err := q.GetContext(ctx, &exists, query, unitCode, caller.TenantID); err != nil {
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

// positionArgs renders a position as the three column values, all nil when there is none.
// The database's all-or-nothing CHECK is the backstop.
func positionArgs(p *domain.PositionInput) (lon, lat, at any) {
	if p == nil {
		return nil, nil, nil
	}
	fix := time.Now().UTC()
	if p.At != nil {
		fix = *p.At
	}
	return p.Lon, p.Lat, fix
}

// crewFor loads crew for the given unit ids. An empty slice means no units, so no query.
func (r *UnitRepo) crewFor(ctx context.Context, q queryer, unitIDs []string) (map[string][]domain.Crew, error) {
	if len(unitIDs) == 0 {
		return map[string][]domain.Crew{}, nil
	}
	var rows []crewRow
	query := fmt.Sprintf(
		`SELECT unit_id, name, role FROM %s.unit_crew WHERE unit_id = ANY($1) ORDER BY unit_id, sort_order`,
		r.schema)
	if err := q.SelectContext(ctx, &rows, query, pq.Array(unitIDs)); err != nil {
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

	// The schema's CHECKs keep each group all-null or all-non-null, so one probe is
	// enough per group.
	if row.AssignmentIncidentCode.Valid {
		unit.Assignment = &domain.Assignment{
			IncidentCode: row.AssignmentIncidentCode.String,
			Title:        row.AssignmentTitle.String,
			Description:  row.AssignmentDescription.String,
			Priority:     row.AssignmentPriority.String,
			Location:     row.AssignmentLocation.String,
			Since:        row.AssignmentSince.Time,
		}
		// The point is its own pair, optional within an assignment.
		if row.AssignmentLon.Valid {
			unit.Assignment.Point = &domain.Point{
				Lon: row.AssignmentLon.Float64,
				Lat: row.AssignmentLat.Float64,
			}
		}
	}
	if row.PositionLon.Valid {
		unit.Position = &domain.Position{
			Lon: row.PositionLon.Float64,
			Lat: row.PositionLat.Float64,
			At:  row.PositionAt.Time,
		}
	}
	return unit
}
