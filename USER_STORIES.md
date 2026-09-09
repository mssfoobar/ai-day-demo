# Workshop user stories

Paste-ready briefs for the three workshop exercises. Each block is self-contained: copy the
whole exercise, story through out-of-scope, so the acceptance criteria and the scope boundary
travel with it. The full brief, including hints on what already exists in the codebase, is in
`WORKSHOP.md`.

---

## Exercise 1 — Dispatch a unit

**As a** dispatcher,
**I want to** dispatch an available unit to an incident and stand it down when the job is done,
**so that** the console shows who is working what and the fleet's status is real rather than seeded.

### Acceptance criteria

1. With an unassigned unit selected, **Dispatch** opens a side Sheet with four fields:
   incident code, title, priority (`P1` / `P2` / `P3`), location. All are required.
2. Submitting sets the unit's assignment and changes its status to **En route**. The detail
   pane's Assignment section, the unit's row, the status tiles and the header count all
   update without a page reload.
3. With an assigned unit selected, the same button reads **Stand down**. Confirming clears
   the assignment and returns the status to **Available**.
4. A stale `occ_lock` is refused with a 409 and the console shows the conflict message in
   the form, exactly as edit does today.
5. Both changes survive a service restart.

### API

| Method | Path | Body | Success |
|---|---|---|---|
| POST | `/v1/units/{unit_code}/assignment` | `{ "incident_code", "title", "priority", "location", "occ_lock" }` | 200, the updated unit |
| DELETE | `/v1/units/{unit_code}/assignment?occ_lock=N` | — | 200, the updated unit |

The service layer owns the status rule: dispatch ⇒ `En route`, stand down ⇒ `Available`.

### Out of scope

Choosing from a list of incidents, a map, more than one unit per incident, notifying anyone.

---

## Exercise 2 — Unit activity timeline

**As a** dispatcher,
**I want to** see a unit's recent status and assignment changes with timestamps,
**so that** I can tell what happened to it during the shift without asking over the radio.

### Acceptance criteria

1. The **Activity** section lists the unit's most recent events, newest first, at most 20.
   Each line shows when it happened (use the existing `sinceLabel`) and what changed, for
   example `Status Available → En route`, `Dispatched to INC-2841`, `Details edited`.
2. Every write to a unit records an event: create, edit, delete, and, if Exercise 1 is
   built, dispatch and stand down. The event is written in the **same transaction** as the
   unit change, so the two can never disagree.
3. A unit with no history shows `No activity yet.`
4. The seed data includes a few events for the seeded units, so the section is not empty
   on first boot.
5. Events survive a service restart.

### API

Pick one design and delete the other's stub:

- **Embed** the last 20 events on the unit, next to `crew`, as `unit.events[]`. No new
  route. Remove the `GET .../events` stub.
- **Separate resource**: `GET /v1/units/{unit_code}/events` returning `{ "data": [ ... ] }`,
  fetched in the page `load` for the selected unit.

Event shape: `{ "at", "kind", "summary" }` where `kind` is one of `created`, `updated`,
`status_changed`, `dispatched`, `stood_down`, `deleted`.

### Out of scope

Filtering, paging past 20, a fleet-wide feed, who did it (there is no login).

---

## Exercise 3 — Manage crew

**As a** dispatcher,
**I want to** add and remove the crew on a unit,
**so that** the roster matches who is actually on the vehicle this shift.

### Acceptance criteria

1. **Manage** beside the Crew heading opens a side Sheet listing the current crew, each
   with a remove control, and one empty row to add a member (name, role). Both fields are
   required for an added row.
2. Saving replaces the unit's crew with what is in the Sheet, in one request. The Crew
   section and its `(n)` count update without a page reload.
3. Two members with the same name on one unit is refused; the error appears beside the
   offending row.
4. A stale `occ_lock` is refused with a 409, as everywhere else.
5. The change survives a service restart.

### API

| Method | Path | Body | Success |
|---|---|---|---|
| PUT | `/v1/units/{unit_code}/crew` | `{ "occ_lock": N, "crew": [ { "name", "role" } ] }` | 200, the updated unit |

Replace-all is deliberate: one call, one transaction (`DELETE` then `INSERT`), and the
unit's `occ_lock` bumps once.

### Out of scope

A people directory, roles as a fixed vocabulary, crew history, assigning one person to
several units.

---

## Ground rules that apply to every exercise

- No new libraries.
- Copy the existing create / edit / delete slice: form component, form action, client call,
  then handler → service → repo.
- Keep optimistic concurrency: every write echoes `occ_lock`; a stale write is a 409
  `DISPATCH_UNIT_STALE`.
- Errors follow the AOH contract; new codes are `DISPATCH_*` in `internal/service/errors.go`.
- Remove the placeholder for the exercise you built and flip its `complete` flag in
  `apps/dispatch-web/src/lib/aoh/dispatch/workshop.ts`.
- `pnpm verify` must stay green.
