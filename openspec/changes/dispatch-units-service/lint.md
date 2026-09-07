# Lint

Mechanical pre-apply lint over `proposal.md`, `design.md`, `specs/**/*.md`, and
`tasks.md` for `dispatch-units-service`.

**Result: apply-ready.** Every item is PASS or N/A.

## 1. Authorization model

- [x] **N/A** — `iams-keycloak` / `iams-aas` as separate runtime dependencies: no auth
      surface. The workshop is unauthenticated by decision; `baseline-dispatch-console`
      removed the auth layer and this change does not reinstate it (proposal.md,
      design.md D1 of the prior change).
- [x] **PASS** — No "Keycloak role", "Keycloak permission", "realm role granting X", or
      per-app OIDC client registration anywhere. IAMS is named only as the thing
      deliberately excluded.
- [x] **PASS** — No spec scenario asserts `active_tenant.permissions`, or any JWT claim;
      there is no token in this change. `tenant_id` is carried on rows per DB convention
      but nothing derives it from a token, which proposal.md states plainly.

## 2. Session storage

- [x] **N/A** — `sds-server` + `valkey`: no auth surface, therefore no session and no
      token to store.
- [x] **PASS** — No browser-side token storage is described; there are no tokens.
- [x] **N/A** — Cookie-prefix / rtus `cookienames` contract: no auth cookie exists and no
      RTUS is composed.

## 2a. Real-time subscriptions

- [x] **N/A** — No RTUS subscription. proposal.md rules RTUS out with a reason (no write
      path, so no event to publish) and names it the natural next step; design.md's Open
      Questions records that nothing polls in its place.

## 3. API contract

- [x] **PASS** — Every endpoint uses `/v{N}/<resource>` with no `/api` prefix:
      `GET /v1/units`, `GET /v1/units/{unit_code}` (design.md, API surface). Verified
      against the running service.
- [x] **N/A** — Fronting gateway proxy routes: the gateway was removed with the auth
      layer, and it attaches a bearer token this service would not read. The SvelteKit
      server calls the service directly (design.md D3).
- [x] **PASS** — No spec scenario asserts a non-envelope response shape. The
      `dispatch-units-api` spec requires the `{data, message, sent_at}` envelope
      explicitly and forbids a bare array; the handler test asserts it.
- [x] **PASS** — `/livez` and `/readyz` are on the API surface, mounted at the root and
      unauthenticated, with a spec requirement that `readyz` fails when the database is
      unreachable.

## 4. UI surfaces

- [x] **PASS** — `design/units-console-mock.html` exists and covers every state this
      surface has: loaded with a selection, unassigned unit, no-capabilities unit, the
      empty pre-selection state, and the service-unavailable error state, plus a
      light/dark toggle.
- [x] **PASS** — design.md's UI / Design System §2 enumerates every primitive by subpath,
      verified against the installed `@mssfoobar/ui@1.1.0`.
- [x] **N/A** — Role-gated page routes: no route is role-gated; there are no roles.
- [x] **PASS** — No hand-rolling of a shipped primitive. The one composite that could
      tempt it — the error banner — is explicitly composed from `Card` +
      `border-destructive`, because `@mssfoobar/ui` ships no inline `Alert` (design.md §2).
- [x] **PASS** — Design and tasks agree on the UI-builder question: DASH is ruled out in
      proposal.md and restated in `tasks.md` 5.2; the summary tiles are inline metrics on
      a list/detail page, the catalogue's stated exclusion.

## 5. Cross-artifact consistency

- [x] **PASS** — The new capability in proposal.md (`dispatch-units-api`) has a matching
      spec directory, and both modified capabilities (`field-unit-roster`,
      `dispatch-console`) have delta specs under the same names.
- [x] **PASS** — Every requirement has at least one scenario; every scenario header uses
      exactly 4 hashtags. Verified by search: dispatch-units-api 5 requirements / 12
      scenarios, field-unit-roster 3 modified + 2 removed / 7 scenarios, dispatch-console
      4 requirements / 9 scenarios. Zero 3-hashtag headers.
- [x] **PASS** — Every endpoint in design.md's API surface is exercised by a spec
      scenario, and each was verified against the running service (200 / 404 / 405 /
      livez / readyz).
- [x] **PASS** — Every runtime dependency has an owner: `postgres:16` → "This change";
      the two apps → this change / existing-modified.
- [x] **PASS** — The REMOVED requirements in `field-unit-roster` each carry a **Reason**
      and a **Migration**, as the delta format requires.

## 6. Verification feasibility

- [x] **PASS** — Every THEN asserts something observable: an HTTP status, an envelope key,
      a rendered string, a row count, a database constraint rejecting a write. Each was
      exercised — including the seed-idempotency and status-vocabulary scenarios.
- [x] **N/A** — Token acquisition in authentication scenarios: there are none.

## 7. Tasks structure

- [x] **PASS** — Every numbered section ends with a concrete verification naming exact
      commands: 1.4 `go build ./...`; 2.2 compose up + ps; 3.4 a `psql` count run twice;
      4.9 `go build && go vet && go test`; 5.11 `pnpm build && check && lint && test:unit`;
      6.1–6.3 curl probes and the reproducibility gate.
- [x] **PASS** — Every compose command uses the dual `podman compose <args>` (or
      `docker compose <args>`) form — tasks 2.2, 6.1, 6.3, and `compose/compose.yml`'s
      header.
- [x] **PASS** — No task runs `compose up` against a custom app. `dispatch-svc` runs via
      `go run ./cmd/server` and `dispatch-web` via `pnpm dev`, both natively against the
      composed database.
- [x] **PASS** — Section order is Scaffold (1) → Compose (2) → Seed (3) → Implement
      `dispatch-svc` (4, the producer) → Implement `dispatch-web` (5, the consumer) →
      End-to-end verification (6). Backend precedes frontend, and nothing exercises infra
      that a later section composes.

## 8. Skill triggering

- [x] **PASS** — The Scaffold section names its scaffolding skill in backticks
      (`aoh-go-init`, task 1.2) — including, explicitly, why it could not be executed.
- [x] **PASS** — The Compose section's first task (2.1) names `aoh-compose` in backticks.
- [x] **PASS** — The first task of each Implement section names `aoh-conventions` in
      backticks: 4.1 (Go) and 5.1 (web). 1.1 does too.
- [x] **N/A** — `aoh-knowledge` on IAMS authz tasks: no task edits `roles.yaml`,
      `realm-import.json`, or AAS state. The seed touches only this service's own schema.
- [x] **PASS** — `tasks.md` 5.1 names `aoh-design` in backticks AND the mockup path.
- [x] **N/A** — UI-builder skill: the surface matches none; `aoh-dashboard` is ruled out
      with a reason in proposal.md and `tasks.md` 5.2.

## 9. Native dev env documentation

- [x] **PASS** — `tasks.md` 4.8 requires the Go env block, and `apps/dispatch-svc/README.md`
      documents every variable (`HTTP_PORT`, the `SQL_*` set) with defaults, plus the
      `GOPRIVATE` prerequisite for the private `aoh-golib` module. `tasks.md` 5.6 covers
      the web side's `DISPATCH_SVC_URL`, which is checked into `.env.development` and
      `.env.template`.
- [x] **PASS** — `compose/compose.yml` documents the exposed port
      (`${POSTGRES_PORT:-5432}` on localhost), which is what lets `go run` reach the
      database natively.

## 10. Reproducibility gate

- [x] **PASS** — A reproducibility-gate task exists as the LAST task of the E2E section
      (6.3), not as a section of its own.
- [x] **PASS** — It uses `down -v` + `up -d`; the `-v` is what forces the seed to re-apply
      to an empty volume, which is the whole point.
- [x] **PASS** — It restarts the native processes for both apps the change touches.
- [x] **PASS** — It re-runs the same curl checks as 6.1, not an abbreviated variant.

## Resolution

All items **PASS** or **N/A**. The change is apply-ready.

Three things a reviewer should consciously accept:

- **The service was hand-written, not scaffolded** (design.md D1). Python is not installed
  on this machine, so `aoh-go-init` could not run. The architecture matches; the
  scaffold's extras (mockery, swag, its Makefile/Dockerfile) are absent.
- **The end-to-end suite was deleted** (design.md D6), so the console's behaviour has no
  automated guard beyond unit tests and type-checking. This was a facilitator decision.
- **The workshop is no longer container-free.** The console now needs the service, which
  needs Postgres. `pnpm start` reduces that to one command, but it is a real increase in
  setup cost over the baseline.
