## Why

`dispatch-units-service` made the console read from Postgres, but nothing ever writes to
it — persistence is only demonstrated by the seed. Basic create / edit / delete on field
units exercises the real path (form → SvelteKit action → service → database → reload) and
gives the workshop its first mutation to reason about.

## What Changes

- **`dispatch-svc` gains write endpoints**: `POST /v1/units`, `PUT /v1/units/{unit_code}`,
  `DELETE /v1/units/{unit_code}`. Validation failures render as 400 with field details, a
  duplicate `unit_code` as 409, and every update/delete is guarded by **`occ_lock`**
  optimistic concurrency — a stale edit gets 409 rather than silently overwriting.
- **The unit resource exposes `occ_lock`** so clients can echo it back.
- **Any write bumps `last_contact`** to now. A change made from the console is, for this
  workshop, a contact with the unit.
- **The console gains** an *Add unit* action (a `Sheet` form), and on a selected unit
  *Edit* (same form, prefilled) and *Delete* (an `AlertDialog` confirm). Writes go through
  SvelteKit **form actions** in `+page.server.ts`, so the browser still only talks to its
  own origin; success and failure surface as toasts.
- **BREAKING (spec)**: `dispatch-units-api`'s "no write verbs" requirement is replaced. The
  `405` for unsupported methods remains for `PATCH` and for verbs on the collection that
  have no meaning.

Not in this pass: editing crew or assignments (schema supports both; the forms are the
work), real-time propagation of writes to other sessions (that is RTUS, and would be the
right next step once writes exist), authentication (`created_by` / `updated_by` are a
fixed placeholder, as `tenant_id` already is).

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `dispatch-units-api`: adds create / replace / delete with validation, conflict and
  optimistic-concurrency semantics; `occ_lock` joins the resource; removes the
  "read-only" requirement.
- `dispatch-console`: adds the add / edit / delete interactions and their outcome
  feedback; removes the "no command actions" requirement.

> Both are specified by `dispatch-units-service`, which is complete but not archived.
> Archive `baseline-dispatch-console` then `dispatch-units-service` before this one.

## Existing AOH services considered

- `rtus` (Real-time Update Service): **ruled out for this pass, now clearly next.** With
  writes in place, a second dispatcher's screen goes stale the moment the first one edits.
  RTUS is how AOH pushes that change; polling is the wrong answer. Kept out here to keep
  the CRUD change small and container count at one — recorded as the follow-up.
- `iams` (Keycloak + AAS): **ruled out — deliberately.** Writes normally carry the caller
  into `created_by` / `updated_by` from the token. With no auth these are a fixed
  placeholder, matching `tenant_id`. The columns are populated, so wiring identity later is
  a value change, not a schema change.
- `form` (Form Builder): **ruled out.** The unit form is a fixed, eight-field record edit,
  not an authored, versioned form with a submission lifecycle. Composing `Label` + `Input`
  + `Select` inside a `Sheet` is the right size.

## Impact

- `apps/dispatch-svc`: repo (insert / update-with-occ / delete), service (validation,
  conflict classification), handler (three routes, request decoding), domain (`occ_lock`),
  tests. No migration — the schema already has every column.
- `apps/dispatch-web`: `units.server.ts` gains `createUnit` / `updateUnit` / `deleteUnit`
  and error-envelope decoding; `+page.server.ts` gains three actions; a `UnitForm`
  component; `Toaster` mounted in the root layout; unit tests for form parsing.
- Runtime dependencies unchanged: the one Postgres container.
