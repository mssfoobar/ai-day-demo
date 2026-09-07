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
| **roster** | The full set of field units the console knows about, in a defined order. Currently hardcoded; read through the `listUnits()` accessor. | fleet, list, inventory, units *(plural alone)* |
| **console** | The operator-facing dispatch screen: the roster on the left, the selected unit's detail on the right. | dashboard *(a DASH surface in AOH — see Cautions)*, page, screen, panel |
| **selection** | Which single field unit the operator is currently inspecting. At most one at a time, client-side only, and it changes nothing about the unit. | active unit, current unit, focus |

## Mappings to platform vocabulary

- Our **field unit** ↔ a GIS **`geo-entity`** (`entity_type=field_unit`), *once units carry
  positions*. They do not today: this baseline stores no coordinates and renders no map,
  so no `geo-entity` exists yet. Keep `FieldUnit`'s shape compatible with that mapping so
  adopting GIS stays a widening rather than a rewrite.
- Our **unit ID** ↔ the `entity_id` of that future `geo-entity`.
- Our **status** is app-level, closest to a GIS **`kind`** (the `geojson.properties.kind`
  app-level classifier) rather than to the `entity_type` DB column.
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
- **"dashboard"** carries a specific platform meaning; do not use it for the console.
