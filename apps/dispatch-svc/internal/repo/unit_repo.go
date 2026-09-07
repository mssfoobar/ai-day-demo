// Package repo is the persistence layer for field units.
package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/domain"
)

// ErrNotFound is returned when a unit code matches no row. The service layer translates
// it into the AOH not-found error class; callers above never see a driver error.
var ErrNotFound = errors.New("unit not found")

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
	radio_channel, shift, capabilities, last_contact,
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
	query := fmt.Sprintf(`SELECT %s FROM %s.unit WHERE unit_code = $1`, unitColumns, r.schema)
	if err := r.db.GetContext(ctx, &row, query, unitCode); err != nil {
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
