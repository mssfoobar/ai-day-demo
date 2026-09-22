package service

import (
	"time"

	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/domain"
)

// baselineRoster is the five units a tenant is seeded with on its first dispatcher
// request (design.md D12).
//
// The data is the one that used to live in `migrations/0002_seed.up.sql` — the same call
// sign ↔ unit_code pairings, statuses, stations, sectors, channels, shifts, capabilities,
// crew and the two assignments. It moved here because a committed migration cannot know
// the tenant id (AAS assigns it at stack-up) and because rows written in SQL bypass the
// GIS outbox, so they would exist in the console and be permanently absent from the map.
// Migration 0004 deletes the rows the SQL seed left behind.
//
// Deliberate variety, because each case is a UI state something asserts:
//   - FU-311 has NO capabilities — the "none recorded" path in the detail pane.
//   - FU-204 has NO position — the "no position reported" path, and the map's
//     "not shown" count.
//   - FU-102 and FU-311 carry assignments — the console's assigned-unit state, and the
//     unit workshop Exercise 1 starts from.
//   - The statuses cover the whole vocabulary: Available, En route and Idle.
//
// Times are relative to the seed, so a roster seeded today never looks weeks stale.
func baselineRoster() []domain.SeedUnit {
	// Coordinates are the units' working sectors in Singapore. Fix times sit a little
	// behind last contact: a position report is a contact, but not every contact is a
	// position report (design.md D11).
	at := func(minutes int) *time.Time {
		t := time.Now().UTC().Add(-time.Duration(minutes) * time.Minute)
		return &t
	}
	since := func(minutes int) time.Time {
		return time.Now().UTC().Add(-time.Duration(minutes) * time.Minute)
	}

	return []domain.SeedUnit{
		{
			UnitInput: domain.UnitInput{
				UnitCode:     "FU-101",
				CallSign:     "Alpha-1",
				Status:       domain.StatusAvailable,
				UnitType:     "Ambulance",
				Station:      "Marina Bay Station 4",
				Sector:       "Sector 4 · Marina Bay",
				RadioChannel: "TAC-2",
				Shift:        "Day · 07:00-19:00",
				Capabilities: []string{"ALS", "Water rescue"},
				Position:     &domain.PositionInput{Lon: 103.8607, Lat: 1.2830, At: at(4)},
			},
			LastContactAgo: 3 * time.Minute,
			Crew: []domain.Crew{
				{Name: "J. Tan", Role: "Paramedic"},
				{Name: "M. Lim", Role: "EMT"},
			},
		},
		{
			UnitInput: domain.UnitInput{
				UnitCode:     "FU-102",
				CallSign:     "Alpha-2",
				Status:       domain.StatusEnRoute,
				UnitType:     "Ambulance",
				Station:      "Marina Bay Station 4",
				Sector:       "Sector 2 · Raffles Place",
				RadioChannel: "TAC-2",
				Shift:        "Day · 07:00-19:00",
				Capabilities: []string{"ALS"},
				Position:     &domain.PositionInput{Lon: 103.8514, Lat: 1.2839, At: at(1)},
			},
			LastContactAgo: 1 * time.Minute,
			Crew: []domain.Crew{
				{Name: "S. Wong", Role: "Paramedic"},
				{Name: "K. Chua", Role: "EMT"},
			},
			Assignment: &domain.Assignment{
				IncidentCode: "INC-2841",
				Title:        "Cardiac arrest",
				Priority:     "P1",
				Location:     "12 Raffles Quay",
				Since:        since(14),
			},
		},
		{
			// Deliberately has NO position: exercises the map's "not shown" count and the
			// detail pane's "no position reported" line.
			UnitInput: domain.UnitInput{
				UnitCode:     "FU-204",
				CallSign:     "Bravo-1",
				Status:       domain.StatusIdle,
				UnitType:     "Fire engine",
				Station:      "Tanjong Pagar Station 2",
				Sector:       "Sector 7 · Tanjong Pagar",
				RadioChannel: "TAC-4",
				Shift:        "Night · 19:00-07:00",
				Capabilities: []string{"Hazmat", "Ladder 30m"},
			},
			LastContactAgo: 22 * time.Minute,
			Crew: []domain.Crew{
				{Name: "R. Kumar", Role: "Officer"},
				{Name: "D. Ng", Role: "Driver"},
				{Name: "A. Rahman", Role: "Firefighter"},
			},
		},
		{
			UnitInput: domain.UnitInput{
				UnitCode:     "FU-205",
				CallSign:     "Bravo-2",
				Status:       domain.StatusAvailable,
				UnitType:     "Rescue tender",
				Station:      "Tanjong Pagar Station 2",
				Sector:       "Sector 7 · Tanjong Pagar",
				RadioChannel: "TAC-4",
				Shift:        "Day · 07:00-19:00",
				Capabilities: []string{"Extrication", "Rope rescue"},
				Position:     &domain.PositionInput{Lon: 103.8434, Lat: 1.2765, At: at(9)},
			},
			LastContactAgo: 8 * time.Minute,
			Crew: []domain.Crew{
				{Name: "P. Goh", Role: "Officer"},
				{Name: "Y. Lee", Role: "Firefighter"},
			},
		},
		{
			// Deliberately has NO capabilities: exercises the "none recorded" path.
			UnitInput: domain.UnitInput{
				UnitCode:     "FU-311",
				CallSign:     "Charlie-1",
				Status:       domain.StatusEnRoute,
				UnitType:     "Patrol car",
				Station:      "Central Division",
				Sector:       "Sector 1 · City Hall",
				RadioChannel: "TAC-1",
				Shift:        "Day · 07:00-19:00",
				Capabilities: []string{},
				Position:     &domain.PositionInput{Lon: 103.8520, Lat: 1.2931, At: at(2)},
			},
			LastContactAgo: 2 * time.Minute,
			Crew: []domain.Crew{
				{Name: "L. Fernandez", Role: "Sergeant"},
			},
			Assignment: &domain.Assignment{
				IncidentCode: "INC-2839",
				Title:        "Traffic obstruction",
				Priority:     "P3",
				Location:     "Nicoll Highway / Republic Ave",
				Since:        since(35),
			},
		},
	}
}
