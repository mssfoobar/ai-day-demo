// Package domain holds the field-unit model the service serves.
//
// JSON tags are snake_case: that is the AOH wire convention, and the SvelteKit client
// maps to camelCase at its own boundary.
package domain

import "time"

// Status values a unit may hold. The authoritative constraint is the CHECK on
// dispatch.unit.status — these constants exist so Go code and tests do not spell the
// strings by hand.
const (
	StatusAvailable = "Available"
	StatusEnRoute   = "En route"
	StatusIdle      = "Idle"
)

// Statuses is the closed vocabulary, in display order.
var Statuses = []string{StatusAvailable, StatusEnRoute, StatusIdle}

// ValidStatus reports whether s is in the vocabulary.
func ValidStatus(s string) bool {
	for _, v := range Statuses {
		if v == s {
			return true
		}
	}
	return false
}

// Crew is one person aboard a unit.
type Crew struct {
	Name string `json:"name"`
	Role string `json:"role"`
}

// Assignment is the incident a unit is currently committed to.
type Assignment struct {
	IncidentCode string    `json:"incident_code"`
	Title        string    `json:"title"`
	Priority     string    `json:"priority"`
	Location     string    `json:"location"`
	Since        time.Time `json:"since"`
}

// Position is where a unit was last located, and when the fix was taken.
//
// At is the *fix* time, not the last-contact time: a unit can be heard from without
// reporting a location, so the two move independently (design.md D11).
type Position struct {
	Lon float64   `json:"lon"`
	Lat float64   `json:"lat"`
	At  time.Time `json:"at"`
}

// Unit is a deployable resource a dispatcher can task.
//
// Assignment and Position are pointers so a unit without either omits the key entirely
// rather than serialising an empty object — the spec calls for absence, not a placeholder.
//
// OccLock is the optimistic-concurrency version (aoh-conventions/database.md): a client
// echoes it on PUT/DELETE and a mismatch is rejected with 409 rather than overwriting.
type Unit struct {
	UnitCode     string      `json:"unit_code"`
	CallSign     string      `json:"call_sign"`
	Status       string      `json:"status"`
	UnitType     string      `json:"unit_type"`
	Station      string      `json:"station"`
	Sector       string      `json:"sector"`
	RadioChannel string      `json:"radio_channel"`
	Shift        string      `json:"shift"`
	Capabilities []string    `json:"capabilities"`
	Crew         []Crew      `json:"crew"`
	Assignment   *Assignment `json:"assignment,omitempty"`
	Position     *Position   `json:"position,omitempty"`
	LastContact  time.Time   `json:"last_contact"`
	OccLock      int         `json:"occ_lock"`
}

// PositionInput is a position as a client may supply it.
//
// At is optional: a client reporting "here, now" should not have to clock the fix
// itself. When it is absent the service stamps the write time. Lon/Lat are plain
// float64 rather than pointers because a position is all-or-nothing — omit the whole
// object to say "no position", which is also how a replace clears one.
type PositionInput struct {
	Lon float64    `json:"lon"`
	Lat float64    `json:"lat"`
	At  *time.Time `json:"at,omitempty"`
}

// UnitInput is the writable subset of a unit — what create and replace accept.
//
// Crew and assignment are deliberately absent: both need their own form design and are
// not editable in this change. A nil Capabilities is stored as an empty array. A nil
// Position means the unit has none — on a replace that *clears* an existing one.
type UnitInput struct {
	UnitCode     string         `json:"unit_code"`
	CallSign     string         `json:"call_sign"`
	Status       string         `json:"status"`
	UnitType     string         `json:"unit_type"`
	Station      string         `json:"station"`
	Sector       string         `json:"sector"`
	RadioChannel string         `json:"radio_channel"`
	Shift        string         `json:"shift"`
	Capabilities []string       `json:"capabilities"`
	Position     *PositionInput `json:"position,omitempty"`

	// Accepted and ignored. Identity always comes from the token — `tenant_id`,
	// `created_by` and `updated_by` in a body are discarded, never stored. They are
	// declared here so such a body is *ignored* rather than rejected: the handler
	// decodes with DisallowUnknownFields, which would otherwise turn "ignored" into
	// a 400. Nothing in this service ever reads these three fields.
	IgnoredTenantID  string `json:"tenant_id,omitempty"`
	IgnoredCreatedBy string `json:"created_by,omitempty"`
	IgnoredUpdatedBy string `json:"updated_by,omitempty"`
}

// SeedUnit is one unit of the baseline roster, with the crew and assignment the write
// API deliberately withholds. It exists because the roster is seeded through the
// service (design.md D12), not by a SQL migration, and so needs a shape wider than
// UnitInput.
type SeedUnit struct {
	UnitInput
	Crew       []Crew
	Assignment *Assignment
	// LastContactAgo backdates last_contact so a freshly seeded roster reads like a
	// live fleet rather than five units all heard from in the same instant.
	LastContactAgo time.Duration
}
