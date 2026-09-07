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

// Unit is a deployable resource a dispatcher can task.
//
// Assignment is a pointer so an unassigned unit omits the key entirely rather than
// serialising an empty object — the spec calls for absence, not a placeholder.
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
	LastContact  time.Time   `json:"last_contact"`
}
