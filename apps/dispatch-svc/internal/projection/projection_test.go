package projection_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/domain"
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/projection"
)

func positioned() domain.Unit {
	return domain.Unit{
		UnitCode: "FU-101",
		CallSign: "Alpha-1",
		Status:   domain.StatusAvailable,
		Position: &domain.Position{Lon: 103.8607, Lat: 1.2830, At: time.Now().UTC()},
	}
}

// The intent is derived from the unit's position AFTER the write, never from the verb
// that produced it. Without that rule, "mirror every unit" and "an un-positioned unit has
// no entity" cannot both hold, and a cleared position leaves a stale marker on the map.
func TestIntentFor_IsDerivedFromPositionNotFromTheVerb(t *testing.T) {
	t.Run("a unit with a position upserts", func(t *testing.T) {
		assert.Equal(t, projection.IntentUpsert, projection.IntentFor(positioned()))
	})

	t.Run("a unit that never had a position deletes", func(t *testing.T) {
		u := positioned()
		u.Position = nil
		assert.Equal(t, projection.IntentDelete, projection.IntentFor(u))
	})

	t.Run("a unit whose position was cleared deletes", func(t *testing.T) {
		// Indistinguishable from "never had one" by design — the row after the write is
		// all the rule looks at.
		cleared := positioned()
		cleared.Position = nil
		assert.Equal(t, projection.IntentDelete, projection.IntentFor(cleared))
	})

	t.Run("the zero origin is a real position, not an absent one", func(t *testing.T) {
		u := positioned()
		u.Position = &domain.Position{Lon: 0, Lat: 0}
		assert.Equal(t, projection.IntentUpsert, projection.IntentFor(u),
			"null island is a location; absence is a nil pointer")
	})
}

func TestPayload_MatchesTheGeoEntityContract(t *testing.T) {
	body, err := projection.Payload(positioned())
	require.NoError(t, err)

	var got struct {
		EntityID   string `json:"entity_id"`
		EntityType string `json:"entity_type"`
		GeoJSON    struct {
			Type     string `json:"type"`
			Geometry struct {
				Type        string    `json:"type"`
				Coordinates []float64 `json:"coordinates"`
			} `json:"geometry"`
			Properties map[string]any `json:"properties"`
		} `json:"geojson"`
	}
	require.NoError(t, json.Unmarshal(body, &got))

	assert.Equal(t, "FU-101", got.EntityID, "entity_id is the unit_code")
	assert.Equal(t, "track", got.EntityType, "static would be wrong the moment a unit moves")
	assert.Equal(t, "Feature", got.GeoJSON.Type)
	assert.Equal(t, "Point", got.GeoJSON.Geometry.Type)
	assert.Equal(t, []float64{103.8607, 1.2830}, got.GeoJSON.Geometry.Coordinates,
		"GeoJSON is lon,lat — not lat,lon")
	assert.Equal(t, "field-unit", got.GeoJSON.Properties["kind"],
		"field-unit is the kind the map's MapEntityProvider keys on, not the entity_type")
	assert.Equal(t, "Alpha-1", got.GeoJSON.Properties["call_sign"])
	assert.Equal(t, domain.StatusAvailable, got.GeoJSON.Properties["status"])
}

// The row carries no credential. The projection travels on the writing operator's own
// bearer, held in memory for the life of that delivery — a token here would be a
// persisted credential, and would expire anyway.
func TestPayload_CarriesNoCredential(t *testing.T) {
	body, err := projection.Payload(positioned())
	require.NoError(t, err)

	var fields map[string]any
	require.NoError(t, json.Unmarshal(body, &fields))
	for _, forbidden := range []string{"token", "access_token", "refresh_token", "authorization", "client_secret"} {
		assert.NotContains(t, fields, forbidden)
	}
}
