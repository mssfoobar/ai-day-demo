# Workshop exercises

Three features are **stubbed but not built**. Each is visible in the running console and in
the service's API, so you can see exactly where it lands before you write a line. Pick one,
build it end to end, remove its placeholder.

| # | Feature | Where the placeholder is |
|---|---|---|
| 1 | Dispatch a unit to an incident / stand it down | **Dispatch** button in the detail-pane header (reads **Stand down** when the unit is assigned); `POST` / `DELETE /v1/units/{unit_code}/assignment` answer 501 |
| 2 | Unit activity timeline | dashed **Activity** section in the detail pane; `GET /v1/units/{unit_code}/events` answers 501 |
| 3 | Manage crew | **Manage** button beside the **Crew** heading; `PUT /v1/units/{unit_code}/crew` answers 501 |

Clicking a placeholder button raises a toast naming the exercise. The 501s carry the AOH
error contract with `errorCode: DISPATCH_NOT_IMPLEMENTED`:

```sh
curl -i -X POST http://localhost:8081/v1/units/FU-101/assignment
```

## Ground rules

- **No new libraries.** Everything you need is already here: `@mssfoobar/ui` primitives,
  SvelteKit form actions with `use:enhance`, chi + sqlx + `aoh-golib` on the Go side, and a
  PostgreSQL schema that already anticipates two of the three features.
- **Copy the existing slice.** Create / edit / delete is the worked example: a form
  component (`UnitForm.svelte`), a form action in `+page.server.ts`, a client call in
  `units.server.ts`, then handler → service → repo in `apps/dispatch-svc`. Follow that shape.
- **Keep optimistic concurrency.** Every write echoes the unit's `occ_lock`; a stale write
  is a 409 `DISPATCH_UNIT_STALE`, never a silent overwrite.
- **Errors follow the contract.** Validation failures list every bad field in `details`
  via `aoherr.FieldDetail`; new error codes are `DISPATCH_*` in `internal/service/errors.go`.
- **Placeholders go when the feature arrives.** Delete the `notImplemented` route, the
  `Workshop exercise N` markup, and the toast wiring for the exercise you built.
- `pnpm verify` must stay green.

Each exercise is written as a user story with acceptance criteria. Anything not listed is
out of scope — resist adding it.

---

## Exercise 1 — Dispatch a unit

> **As a** dispatcher, **I want to** dispatch an available unit to an incident and stand it
> down when the job is done, **so that** the console shows who is working what and the
> fleet's status is real rather than seeded.

### Acceptance criteria

1. With an unassigned unit selected, **Dispatch** opens a side Sheet with four fields:
   incident code, title, priority (`P1` / `P2` / `P3`), location. All are required.
2. Submitting sets the unit's assignment and changes its status to **En route**. The
   detail pane's Assignment section, the unit's row, the status tiles and the header count
   all update without a page reload.
3. With an assigned unit selected, the same button reads **Stand down**. Confirming clears
   the assignment and returns the status to **Available**.
4. A stale `occ_lock` is refused with a 409 and the console shows the conflict message in
   the form, exactly as edit does today.
5. Both changes survive a service restart.

### What already exists

- The `unit` table already has `assignment_incident_code`, `assignment_title`,
  `assignment_priority`, `assignment_location`, `assignment_since` and a CHECK constraint
  that they are set all-or-nothing. **No migration needed.**
- `domain.Assignment` and the `assignment` field in the wire model are already read and
  rendered. You only need to write them.
- `UnitForm.svelte` is a Sheet form with field errors, `use:enhance`, a hidden `occLock`
  and an `onoutcome` callback. Copy it to `DispatchForm.svelte` and cut it down.
- `+page.server.ts` shows how a form action parses, calls the service, maps a 400's
  `details` onto field errors and returns `fail()`.
- `repo.Update` shows the `WHERE ... AND occ_lock = $n` write with `occ_lock = occ_lock + 1`,
  and `staleOrMissing` for telling 404 from 409.

### Suggested API

| Method | Path | Body | Success |
|---|---|---|---|
| POST | `/v1/units/{unit_code}/assignment` | `{ "incident_code", "title", "priority", "location", "occ_lock" }` | 200, the updated unit |
| DELETE | `/v1/units/{unit_code}/assignment?occ_lock=N` | — | 200, the updated unit |

The service layer owns the status rule: dispatch ⇒ `En route`, stand down ⇒ `Available`.
Keep the handler thin.

### Out of scope

Choosing from a list of incidents, a map, more than one unit per incident, notifying anyone.

---

## Exercise 2 — Unit activity timeline

> **As a** dispatcher, **I want to** see a unit's recent status and assignment changes with
> timestamps, **so that** I can tell what happened to it during the shift without asking
> over the radio.

### Acceptance criteria

1. The **Activity** section lists the unit's most recent events, newest first, at most 20.
   Each line shows when it happened (use the existing `sinceLabel`) and what changed, for
   example `Status Available → En route`, `Dispatched to INC-2841`, `Details edited`.
2. Every write to a unit records an event: create, edit, delete, and — if Exercise 1 is
   built — dispatch and stand down. The event is written in the **same transaction** as
   the unit change, so the two can never disagree.
3. A unit with no history shows `No activity yet.`
4. The seed data includes a few events for the seeded units, so the section is not empty
   on first boot.
5. Events survive a service restart.

### What already exists

- `migrations/0001_init.up.sql` shows the mandatory columns every table carries (`id`,
  `created_at`, `updated_at`, `created_by`, `updated_by`, `tenant_id`, `occ_lock`) and the
  `set_updated_at` trigger. `0002_seed.up.sql` shows idempotent seeding with
  `ON CONFLICT ... DO UPDATE`. Add `0003_unit_event.up.sql` in the same style; the migrator
  picks it up by filename.
- `repo.crewFor` shows how a child table is read per unit and folded into the domain
  object. `eventsFor` is the same shape.
- `UnitDetail.svelte` already has the section title snippet and the dashed placeholder box
  to replace.

### Suggested API

Two acceptable designs. Pick the simpler one for you and delete the other's stub:

- **Embed** the last 20 events on the unit, next to `crew`: `unit.events[]`. No new route,
  no new client call, and the console gets them with the roster it already loads. Remove
  the `GET .../events` stub.
- **Separate resource**: `GET /v1/units/{unit_code}/events` returning the envelope
  `{ "data": [ ... ] }`, fetched in the page `load` for the selected unit.

Event shape: `{ "at", "kind", "summary" }` where `kind` is one of `created`, `updated`,
`status_changed`, `dispatched`, `stood_down`, `deleted`.

### Out of scope

Filtering, paging past 20, a fleet-wide feed, who did it (there is no login).

---

## Exercise 3 — Manage crew

> **As a** dispatcher, **I want to** add and remove the crew on a unit, **so that** the
> roster matches who is actually on the vehicle this shift.

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

### What already exists

- The `unit_crew` table exists with `UNIQUE (unit_id, name)` and `ON DELETE CASCADE`.
  **No migration needed.**
- `repo.mapWriteError` already turns Postgres `23505` into `ErrConflict`; the service maps
  that to a 409 today. For this exercise you may prefer to validate duplicates in the
  service and return a 400 with a `FieldDetail` per duplicate — your call, state it.
- `UnitForm.svelte` for the Sheet, `use:enhance` and error plumbing. The crew rows are a
  small `{#each}` over `crew[]` with an index in each input's `name` (`crew[0].name`).
  `parseUnitForm` in `forms.ts` shows how form data is parsed and validated server-side.

### Suggested API

| Method | Path | Body | Success |
|---|---|---|---|
| PUT | `/v1/units/{unit_code}/crew` | `{ "occ_lock": N, "crew": [ { "name", "role" } ] }` | 200, the updated unit |

Replace-all is deliberate: one call, one transaction (`DELETE` then `INSERT`), and the
unit's `occ_lock` bumps once so the concurrency story stays the same as edit.

### Out of scope

A people directory, roles as a fixed vocabulary, crew history, assigning one person to
several units.

---

## Done looks like

- The placeholder for your exercise is gone from the console, the page, and `Routes`.
- `pnpm verify` is green.
- `pnpm reset-db && pnpm start`, then walk the acceptance criteria against a clean
  database — including the restart in the last criterion.
