# Ubiquitous language

This project's own domain vocabulary. Use these terms in specs, tests, and code.

**Platform** vocabulary — `geo-entity`, `active_tenant`, `BFF`, the response envelope,
`occ_lock`, `trace_id`, and so on — is defined by the AOH glossary in the
`aoh-knowledge` skill. Those terms are **referenced, never redefined** here. This file
holds only terms this project owns, plus how they map onto platform ones.

Built against **`aoh-knowledge` @ `@mssfoobar/agent-skills@v0.9.0`** (see
`skills-lock.json`). When that pin moves, re-read the platform glossary and reconcile the
mappings below — a skill bump is the moment vocabulary drift becomes visible.

## Terms

| Term | Definition | Aliases to avoid |
| --- | --- | --- |
| **field unit** | A deployable resource a dispatcher can task — a vehicle and its crew, treated as one addressable thing. The core noun of this project. In code: the `FieldUnit` type in `apps/dispatch-web/src/lib/aoh/dispatch/roster.ts`. | unit *(ambiguous alone — see Cautions)*, resource, asset, vehicle, responder |
| **call sign** | The human-readable name an operator uses to address a field unit over the radio (`Alpha-1`). Display identity, not a key: it is not guaranteed unique or stable across a shift. | name, label, callsign *(one word)* |
| **unit ID** | The stable, unique machine key for a field unit (`FU-101`). What code joins on. Distinct from the call sign. | id *(unqualified)*, code, reference |
| **status** | A field unit's current operational state, drawn from a closed vocabulary: `Available`, `En route`, `Idle`. Not free text — adding a value is a deliberate, type-checked change. | state, availability, condition |
| **Available** | The unit is ready to be tasked. | free, idle *(means something else here)*, ready |
| **En route** | The unit is actively moving to a tasking. Two words, sentence case. | enroute, en-route, responding, active |
| **Idle** | The unit is on duty but not tasked and not ready to be tasked. Neutral, **not** a fault condition — this is why it renders in a neutral colour rather than a warning one. | offline, inactive, unavailable, down |
| **roster** | The full set of field units **in the caller's tenant**, in a defined order. Read through the `listUnits()` accessor. Seeded once per tenant, on its first dispatcher request; an empty roster is a legitimate state, not a failure. | fleet, list, inventory, units *(plural alone)* |
| **console** | The operator-facing dispatch screen: the roster on the left, the selected unit's detail on the right. | dashboard *(a DASH surface in AOH — see Cautions)*, page, screen, panel |
| **selection** | Which single field unit the operator is currently inspecting. At most one at a time, and it changes nothing about the unit. Shared between the console and the map: selecting on one carries to the other. | active unit, current unit, focus |
| **position** | Where a field unit was last located — a longitude, a latitude, and the fix time. A unit either has all three or none; there is no partial position and no zero-coordinate placeholder. Only positioned units can appear on the map. | location *(too broad — a station and a sector are also locations)*, coordinates *(only the pair, not the fix time)*, GPS |
| **fix time** | When a unit's position was reported. Deliberately **not** last contact: a unit can be heard from without reporting a location, so the two advance independently. Editing a unit's radio channel is a contact, not a fix. | position time, updated at, timestamp |
| **last contact** | When the unit was last heard from at all. Every write counts as a contact. | last seen, last update, heartbeat |
| **map** | The operator-facing geospatial surface at `/aoh/dispatch/map`: the tenant's positioned field units on a live canvas, alongside the console rather than inside it. | GIS *(the platform module, not this page)*, tracking view, situational display |
| **dispatcher** | An operator who may change the roster — add, edit and delete field units. The `dispatch-dispatcher` application role. | admin, editor, operator *(covers both roles)* |
| **viewer** | An operator who may read the roster and the map but change nothing. The `dispatch-viewer` application role. | guest, read-only user, observer |

## Mappings to platform vocabulary

- A **positioned** field unit ↔ a GIS **`geo-entity`** with `entity_type` **`track`** and
  `geojson.properties.kind` **`field-unit`**. This mapping is real now, not conditional:
  `dispatch-svc` mirrors every positioned unit into `gis-service` through a transactional
  outbox. `track` is GIS's reserved type for entities whose position updates in real time —
  `static` would be wrong the moment a unit moves. A unit with **no** position has **no**
  geo-entity at all, which is what keeps a cleared position from leaving a stale marker.
- Our **unit ID** ↔ the geo-entity's `entity_id`. It is unique per *tenant*, not globally;
  that is safe only because GIS is itself tenant-partitioned.
- Our **position** ↔ the geo-entity's `geojson.geometry`, a Point of `[lon, lat]`. Note the
  order: GeoJSON is longitude first, which is the reverse of how an operator reads a
  coordinate aloud.
- Our **status** is an ordinary GeoJSON **property** (`geojson.properties.status`), *not* a
  GIS `kind`. An earlier version of this file said otherwise; `kind` is the fixed literal
  `field-unit`, because it is what the map's `MapEntityProvider` filters on and every field
  unit is the same kind of thing whatever its status.
- Our **dispatcher** / **viewer** ↔ AAS **tenant roles** (`dispatch-dispatcher`,
  `dispatch-viewer`) in the `active_tenant.roles` claim. They are *not* Keycloak realm
  roles, and there is no `active_tenant.permissions` claim to read — each app projects
  role names onto permissions in-process.
- Our **console** is **not** an AOH **dashboard**. In AOH a dashboard is a DASH surface
  (`@mssfoobar/dash-web-sdk`, widgets via `defineWidget`). The console is a master–detail
  list/detail page built from `@mssfoobar/ui` primitives. Calling it a dashboard invites
  exactly the wrong implementation — see `openspec/changes/baseline-dispatch-console/`
  (proposal.md, design.md D4).

## Cautions

- **"unit" alone is ambiguous** — it also reads as a unit of measure and as "unit test".
  Say **field unit** in prose, specs, and comments. `unit` as a variable name inside
  `roster.ts` or the console page is fine, because the type makes it unambiguous.
- **"status" is overloaded** — this project's `status` is a *field unit's* operational
  state. It is unrelated to an HTTP status code or a Playwright test status. Qualify the
  others (`HTTP status`, `test status`) rather than this one.
- **"position" and "last contact" are not interchangeable**, and conflating them is the
  most likely way to make the map lie. A unit's marker can be minutes old while its last
  contact is seconds old, and that difference is the operationally interesting part. Always
  say **fix time** for the position's timestamp.
- **"map" is this page, not the module.** GIS is the AOH platform module; the map is the
  surface this project builds on it. "The map is down" and "GIS is down" are different
  claims.
- **"dashboard"** carries a specific platform meaning; do not use it for the console.
