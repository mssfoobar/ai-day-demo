// Package projection mirrors field units into gis-service as geo-entities.
//
// The mirror is written through a transactional outbox: a unit write records a row in
// the same transaction, and a worker drains it (design.md D6). gis-service republishes
// each change to the RTUS map named `gis`, which is what puts a moving unit on another
// dispatcher's screen — this service never publishes to RTUS itself.
package projection

import (
	"encoding/json"
	"fmt"

	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/domain"
)

// Intent is what the outbox row asks the worker to do.
type Intent string

const (
	// IntentUpsert — write or move the unit's geo-entity.
	IntentUpsert Intent = "upsert"
	// IntentDelete — remove it. A delete for an entity that is not there is a no-op,
	// which is what lets the rule below stay this simple.
	IntentDelete Intent = "delete"
)

// EntityType is GIS's reserved type for entities with real-time position updates.
// `static` would be wrong the moment a unit moves (design.md D8).
const EntityType = "track"

// Kind is the app-level grouping gis-service filters and styles by, and the value the
// map's MapEntityProvider keys on. Note this is a `kind`, not an `entity_type`.
const Kind = "field-unit"

// IntentFor derives the intent from the unit's position AFTER the write, not from the
// HTTP verb that produced it (design.md D6a).
//
// A geo-entity is a Point, so a unit with no coordinates has nothing to project. Without
// this rule a cleared position would leave a stale marker at the unit's last known
// location — the worst failure a dispatch surface has, because it looks like data rather
// than an absence.
func IntentFor(u domain.Unit) Intent {
	if u.Position == nil {
		return IntentDelete
	}
	return IntentUpsert
}

// geoEntity is the gis-service upsert body. `entity_id` is the unit_code (design.md D8),
// which is unique per tenant — safe because GIS is itself tenant-partitioned and the
// entity is written in the token's tenant.
type geoEntity struct {
	EntityID   string  `json:"entity_id"`
	EntityType string  `json:"entity_type"`
	GeoJSON    feature `json:"geojson"`
}

type feature struct {
	Type       string         `json:"type"`
	Geometry   geometry       `json:"geometry"`
	Properties map[string]any `json:"properties"`
}

type geometry struct {
	Type        string     `json:"type"`
	Coordinates [2]float64 `json:"coordinates"`
}

// Payload renders the body the worker sends for an upsert.
//
// For a delete intent the body is never sent (the worker calls DELETE by entity_id), but
// the row still carries one so an operator reading the outbox can see which unit it was
// about without joining anything.
func Payload(u domain.Unit) ([]byte, error) {
	entity := geoEntity{
		EntityID:   u.UnitCode,
		EntityType: EntityType,
		GeoJSON: feature{
			Type: "Feature",
			Properties: map[string]any{
				"kind":      Kind,
				"call_sign": u.CallSign,
				"status":    u.Status,
			},
		},
	}
	if u.Position != nil {
		entity.GeoJSON.Geometry = geometry{
			Type:        "Point",
			Coordinates: [2]float64{u.Position.Lon, u.Position.Lat},
		}
	}

	body, err := json.Marshal(entity)
	if err != nil {
		return nil, fmt.Errorf("marshal geo-entity for %s: %w", u.UnitCode, err)
	}
	return body, nil
}
