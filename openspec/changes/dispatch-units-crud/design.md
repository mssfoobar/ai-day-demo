## Context

`dispatch-svc` is read-only and the console renders it. The schema already carries every
AOH mandatory column a write needs (`occ_lock`, `created_by`, `updated_by`, `tenant_id`,
the `updated_at` trigger) and the CHECK constraints that make invalid writes fail at the
database. So this change adds no migration — it adds the code paths and the UI that use
what is there.

Constraints that shape it (all from `aoh-conventions`): `occ_lock` MUST be echoed on every
mutation and mismatch is 409; errors use the `aoherr` contract with module-prefixed codes;
success bodies stay in the envelope; the browser does not call backends directly.

## Goals / Non-Goals

**Goals:** create / replace / delete a unit from the console; prove persistence across a
service restart; correct AOH write semantics (validation 400, conflict 409, optimistic
concurrency).

**Non-Goals:** crew and assignment editing, real-time fan-out (RTUS), authentication,
`PATCH` partial updates, pagination, undo.

## Runtime dependencies

| Dependency | Role | Owned by |
|---|---|---|
| `postgres:16` | Unchanged; already composed. | Existing — unchanged |
| `apps/dispatch-svc` | Gains write endpoints. | Existing — modified |
| `apps/dispatch-web` | Gains form actions and forms. | Existing — modified |

## API surface

| Method | Path | Auth posture | Description | Fronting gateway route |
|---|---|---|---|---|
| POST | /v1/units | Unauthenticated | Create. 201 + envelope. 400 validation, 409 duplicate code. | n/a — called by SvelteKit server |
| PUT | /v1/units/{unit_code} | Unauthenticated | Replace editable fields; body carries `occ_lock`. 200 + envelope. 400 / 404 / 409 stale. | n/a |
| DELETE | /v1/units/{unit_code}?occ_lock=N | Unauthenticated | Delete unit + crew. 204. 400 missing / 404 / 409 stale. | n/a |
| GET | /v1/units, /v1/units/{unit_code} | Unauthenticated | Unchanged, now include `occ_lock`. | n/a |

Request body (POST; PUT is the same minus `unit_code`, plus `occ_lock`):

```jsonc
{ "unit_code": "FU-401", "call_sign": "Delta-1", "status": "Available",
  "unit_type": "Ambulance", "station": "Bedok Station 3", "sector": "Sector 9 · Bedok",
  "radio_channel": "TAC-3", "shift": "Day · 07:00-19:00", "capabilities": ["ALS"] }
```

New error codes (in `internal/service/errors.go`): `DISPATCH_UNIT_INVALID` (400),
`DISPATCH_UNIT_CODE_TAKEN` (409), `DISPATCH_UNIT_STALE` (409),
`DISPATCH_UNIT_WRITE_FAILED` (500).

## UI / Design System

No new mockup: the surfaces are the two overlay primitives used exactly as the design
system ships them. Composition:

| Element | Primitive | Subpath |
|---|---|---|
| Add / Edit form container | `Sheet` (side panel) | `@mssfoobar/ui/sheet` |
| Fields | `Label` + `Input`; status via `Select` | `/label`, `/input`, `/select` |
| Field error | sibling `<p class="text-destructive text-xs">` | (documented composition — no `FormMessage` in the package) |
| Delete confirm | `AlertDialog`, action `variant="destructive"` | `/alert-dialog` |
| Outcome feedback | `toast` + `<Toaster />` mounted once in the root layout | `/toast` |
| Actions | `Button` — Add in the header; Edit / Delete in the detail header | `/button` |

Copy: `Add unit`, `Edit`, `Delete`, `Delete Alpha-1?` / `This removes the unit and its crew.
It cannot be undone.`, `Save`, `Cancel`, toasts `Alpha-1 added` / `Alpha-1 saved` /
`Alpha-1 deleted`, conflict `Alpha-1 was changed by someone else. Reload and try again.`

## Decisions

**D1 — Form actions, not client fetch.** `+page.server.ts` gets `create` / `update` /
`delete` actions; the page uses `use:enhance`. Same-origin is preserved, validation errors
come back via `fail(400, …)` and render beside the field, and the roster reloads through
SvelteKit's normal invalidation. Selection is preserved across the reload (and cleared on
delete) by keeping `selectedId` and re-resolving it against the new `units`.

**D2 — Client-supplied `unit_code`.** Dispatchers already speak in codes (`FU-401`), the
seed uses them, and the uniqueness constraint exists. Generating them server-side would
need a numbering scheme this workshop has no basis for.

**D3 — `occ_lock` in the body for PUT, in the query for DELETE.** DELETE has no body by
convention; the query string is the idiomatic carrier.

**D4 — Any write bumps `last_contact`.** Simplest honest semantics for a workshop: a
change made at the console is a contact. Documented so it can be revisited.

**D5 — Map Postgres errors at the repo boundary.** `23505` (unique) → `ErrConflict`,
`23514` (check) → `ErrInvalid` with the constraint name, `0 rows affected` on an
`occ_lock`-guarded UPDATE/DELETE → `ErrStale`. The service classifies these into `aoherr`
classes; the handler never sees a driver error.

**D6 — Validate in the service, then let the database be the last word.** The service
checks required fields and the status vocabulary so the user gets field-level messages;
the CHECK constraints remain as the backstop for anything that bypasses the service.

**D7 — Crew and assignment are deliberately untouchable here.** Create leaves crew empty
and assignment absent; PUT does not touch either. Both need their own form design.

## Risks / Trade-offs

- **[A stale second screen]** → 409 on stale `occ_lock` prevents silent overwrite; the
  screen still only refreshes on the user's own action. RTUS is the fix, recorded as next.
- **[`last_contact` bump surprises someone editing a typo]** → D4 is explicit; revisit if
  it misleads.
- **[No authentication means `created_by` is a placeholder]** → same posture as
  `tenant_id`; a value change later, not a schema change.

## Migration Plan

None — no schema change, no data change. Restart the service to pick up the new routes.

## Open Questions

- Whether `unit_code` should be validated to a pattern (`FU-\d{3}`) or left free-form.
  Left free-form (non-blank, ≤ 32 chars) pending a real numbering convention.
