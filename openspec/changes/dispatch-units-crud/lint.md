# Lint

Mechanical pre-apply lint over `proposal.md`, `design.md`, `specs/**/*.md`, and
`tasks.md` for `dispatch-units-crud`.

**Result: apply-ready.** Every item is PASS or N/A.

## 1. Authorization model

- [x] **N/A** — `iams-keycloak` / `iams-aas`: no auth surface. Writes carry a fixed
      `created_by` / `updated_by` placeholder, stated in proposal.md and design.md.
- [x] **PASS** — No Keycloak role/permission or per-app OIDC client anywhere.
- [x] **PASS** — No scenario asserts `active_tenant.permissions` or any JWT claim.

## 2. Session storage

- [x] **N/A** — `sds-server` + `valkey`: no auth surface, no session.
- [x] **PASS** — No browser-side token storage described.
- [x] **N/A** — Cookie-prefix contract: no auth cookie; no RTUS composed.

## 2a. Real-time subscriptions

- [x] **N/A** — No RTUS subscription. proposal.md names RTUS as the next step now that
      writes exist, and design.md's Risks records that a second screen goes stale until then.

## 3. API contract

- [x] **PASS** — Every endpoint is `/v1/units[...]`, no `/api` prefix (design.md, API
      surface). Verified against the running service.
- [x] **N/A** — Gateway routes: none; the SvelteKit server calls the service directly.
- [x] **PASS** — Success bodies are the envelope (201/200 via `aohhttp.Response`; 204 has
      no body by definition). No spec scenario asserts a non-envelope shape; failures use
      the `aoherr` contract with `details`.
- [x] **N/A** — `/livez` + `/readyz` on *new* backend services: no new service; the existing
      probes are unchanged and still tested.

## 4. UI surfaces

- [x] **PASS** — No new mockup was drawn, and design.md says why: the surfaces are two
      overlay primitives (`Sheet`, `AlertDialog`) used as shipped, with the field layout
      and copy enumerated in design.md's UI section. The states are: form empty, form with
      field errors, form with a form-level (conflict) error, confirm dialog, and toasts —
      each named there.
- [x] **PASS** — design.md's UI table enumerates every primitive by subpath (`sheet`,
      `label`, `input`, `select`, `alert-dialog`, `toast`, `button`), all present in the
      installed `@mssfoobar/ui@1.1.0` `exports`.
- [x] **N/A** — Role-gated routes: none.
- [x] **PASS** — No hand-rolled primitive. The one gap in the package — no `FormMessage` —
      is filled with the composition `aoh-conventions/web.md` documents (a sibling
      `<p class="text-destructive">`), and design.md says so.
- [x] **PASS** — No UI-builder SDK is mandated, so no task hand-rolls one.

## 5. Cross-artifact consistency

- [x] **PASS** — proposal.md lists no new capabilities and two modified ones;
      `specs/dispatch-units-api/` and `specs/dispatch-console/` exist with matching names.
- [x] **PASS** — Every requirement has at least one `#### Scenario:`; all headers use four
      hashtags. dispatch-units-api: 5 ADDED + 1 MODIFIED requirements / 15 scenarios;
      dispatch-console: 4 ADDED / 8 scenarios. REMOVED requirements each carry **Reason**
      and **Migration**.
- [x] **PASS** — Every endpoint in design.md's API surface is exercised by a spec scenario
      (create, replace, delete, reads, 405s) and was exercised live in the smoke.
- [x] **PASS** — Every runtime dependency has an owner (`Existing — unchanged` /
      `Existing — modified`).
- [x] **PASS** — The MODIFIED `Read API for field units` requirement carries its full
      updated content, not a partial edit.

## 6. Verification feasibility

- [x] **PASS** — Every THEN asserts an observable: an HTTP status, an `errorCode`, a
      `details` field name, a DB row, a rendered element, a toast. The restart scenario
      is realised by stopping and starting the process and reading the unit back.
- [x] **N/A** — Token acquisition: no authentication scenarios.

## 7. Tasks structure

- [x] **PASS** — Each section ends with concrete commands: 1.7 `go build && go vet && go
      test`; 2.8 `pnpm build && check && lint && test:unit`; 3.1 curl probes; 3.3 the gate.
- [x] **PASS** — Compose commands appear in the dual `podman compose` (or `docker compose`)
      form (3.3).
- [x] **PASS** — No task runs `compose up` against a custom app; both run natively.
- [x] **PASS** — Order is Implement `dispatch-svc` (producer) → Implement `dispatch-web`
      (consumer) → E2E verification. Scaffold, Compose and Seed are omitted: nothing new is
      scaffolded, composed or seeded (no migration — design.md, Context).

## 8. Skill triggering

- [x] **N/A** — Scaffold: no new service or app.
- [x] **N/A** — `aoh-compose`: no Compose section.
- [x] **PASS** — First task of each Implement section names `aoh-conventions` in backticks
      (1.1, 2.1); 1.1 also names `aoh-error-handling`, which owns the error contract.
- [x] **N/A** — `aoh-knowledge` on IAMS authz: no such task.
- [x] **PASS** — UI task 2.1 names `aoh-design` in backticks. There is no mockup path to
      name because design.md deliberately ships none for this change (see §4).
- [x] **N/A** — UI-builder skill: not applicable.

## 9. Native dev env documentation

- [x] **PASS** — No new environment variable is introduced. The existing blocks in both
      apps' READMEs remain accurate; `DISPATCH_SVC_URL` already covers the write calls.
- [x] **N/A** — Compose port contract: no Compose section.

## 10. Reproducibility gate

- [x] **PASS** — Task 3.3 is the last task of the E2E section, uses `down -v` + `up -d`,
      restarts both native apps, and re-runs the same probes as 3.1. It additionally
      asserts the right thing for a *write* change: the created unit is gone with the
      volume and the seed converges to its 5 units.

## Resolution

All items **PASS** or **N/A**. Apply-ready.

Two things a reviewer should consciously accept:

- **No mockup for the forms** — a deliberate call, argued in design.md §UI: the surfaces
  are stock `Sheet` / `AlertDialog` compositions and drawing them would document the
  package rather than the design.
- **Writes exist but do not propagate to other sessions.** `occ_lock` prevents the
  dangerous outcome (silent overwrite); it does not prevent the confusing one (a stale
  screen). RTUS is the AOH answer and is recorded as the next change.
