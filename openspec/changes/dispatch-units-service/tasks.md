## 1. Scaffold `dispatch-svc`

- [x] 1.1 Consult the `aoh-conventions` skill (Go conventions, API conventions, DB schema conventions) before implementing.
- [x] 1.2 The `aoh-go-init` skill's scaffold is a Python script and Python is NOT installed on this machine (`python`/`python3` resolve to the Windows Store stub). Hand-write the service to the same architecture per design.md D1 — `cmd/server`, `internal/{config,handler,service,repo}`, chi, sqlx, Viper, `aoh-golib` — and record the deviation in the service README. Do not silently skip the layering.
- [x] 1.3 Create `apps/dispatch-svc` with `go.mod` (module `github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc`, Go 1.25) requiring `github.com/mssfoobar/ops-hub/packages/aoh-golib v0.3.0`. Register the workspace in `go.work` and add a package-scoped entry to root `turbo.json`.
- [x] 1.4 Verify: `cd apps/dispatch-svc && go build ./...` exits 0.

## 2. Compose runtime dependencies

- [x] 2.1 Add `compose/compose.yml` with a single `postgres:16` service using the `aoh-compose` skill's conventions where they apply (named volume, healthcheck, `.env`-driven credentials). No Traefik, no Keycloak, no IAMS, no SDS — the workshop is unauthenticated (design.md, Runtime dependencies).
- [x] 2.2 Verify: `podman compose up -d postgres` (or `docker compose up -d postgres`) followed by `podman compose ps` (or `docker compose ps`) shows `postgres` healthy.

## 3. Seed / resource creation

- [x] 3.1 Write `migrations/0001_init.up.sql` creating schema `dispatch`, table `unit` (AOH mandatory columns + the unit fields, `UNIQUE (unit_code, tenant_id)`, `CHECK` on `status`, and a `CHECK` keeping the assignment columns all-null or all-non-null) and table `unit_crew` (mandatory columns + `unit_id` FK ON DELETE CASCADE, `name`, `role`, `sort_order`). Add the `updated_at` trigger.
- [x] 3.2 Write `migrations/0002_seed.up.sql` seeding the five baseline units with type, station, sector, radio channel, shift, capabilities, crew, assignments and last-contact times. It MUST be idempotent — `INSERT … ON CONFLICT (unit_code, tenant_id) DO UPDATE` — so re-running converges rather than duplicating.
- [x] 3.3 Apply migrations on service start (embedded, in order, recorded in a `schema_migration` table) so no external migration tool is a prerequisite.
- [x] 3.4 Verify (state present): `docker compose exec -T postgres psql -U dispatch -d dispatch -c "SELECT count(*) FROM dispatch.unit"` reports 5, and re-running the service reports 5 again, not 10.

## 4. Implement `dispatch-svc`

- [x] 4.1 Consult the `aoh-conventions` skill (Go conventions) before implementing.
- [x] 4.2 `internal/config`: Viper config (HTTP port, `SQL_*` connection settings, schema name) with env-var binding and sane local defaults.
- [x] 4.3 `internal/repo`: sqlx queries returning units with their crew, ordered deterministically (`unit_code`). Assignment columns map to a nullable embedded struct.
- [x] 4.4 `internal/service`: the domain layer, plus `errors.go` declaring the service's `errorCode` namespace; classify "unit not found" as a not-found error rather than returning a bare `sql.ErrNoRows`.
- [x] 4.5 `internal/handler`: chi routes `GET /v1/units`, `GET /v1/units/{unit_code}` rendering the AOH envelope via `aohhttp.Response`, and errors via `aoherr`/`aohhttp.Render`. Mount `/livez` and `/readyz` at the root, with `readyz` pinging the database.
- [x] 4.6 `cmd/server`: wire config → db → repo → service → handler, run migrations, start the HTTP server with graceful shutdown.
- [x] 4.7 Tests: repo-free service tests with a fake repo (list, get, not-found) and handler tests asserting the envelope shape, the 404 body, and 405 on write verbs. No end-to-end test (design.md D6).
- [x] 4.8 Document the native env block in the service README: `HTTP_PORT`, `SQL_HOST`, `SQL_PORT`, `SQL_USER`, `SQL_PASSWORD`, `SQL_DATABASE_NAME`, `SQL_SCHEMA_NAME`, `SQL_SSL_MODE`, plus the `aoh-golib` private-module prerequisite.
- [x] 4.9 Verify: `cd apps/dispatch-svc && go build ./... && go vet ./... && go test ./... -count=1` all exit 0.

## 5. Implement `dispatch-web`

- [x] 5.1 Consult the `aoh-conventions` skill (web conventions) and the `aoh-design` skill before editing the page. Open `openspec/changes/dispatch-units-service/design/units-console-mock.html` in a browser as the visual reference.
- [x] 5.2 No AOH UI-builder skill applies — the surface is a master-detail list/detail page with inline metrics, not a dashboard, so `aoh-dashboard` and `@mssfoobar/dash-web-sdk` are deliberately NOT used (proposal.md; design.md §1). Build from `@mssfoobar/ui` primitives by subpath.
- [x] 5.3 Remove the Playwright layer: delete `tests/e2e/` and `playwright.config.ts`, drop `@playwright/test` and the `test:integration` / `test` scripts that invoke it, and remove the `test:integration` task from root `turbo.json` and the `test:e2e` root script (design.md D6).
- [x] 5.4 Replace `src/lib/aoh/dispatch/roster.ts` with `src/lib/aoh/dispatch/units.server.ts`: the widened `FieldUnit` / `Crew` / `Assignment` types, an async `listUnits()` that calls `${DISPATCH_SVC_URL}/v1/units`, unwraps the AOH envelope, maps snake_case → camelCase, and rejects a unit whose status is outside the known vocabulary (design.md D4, D7).
- [x] 5.5 Add `src/routes/units/+page.server.ts` whose load calls `listUnits()` and returns either the units or a flag for the error state. It must contain no URL or fetch wiring of its own (specs/field-unit-roster).
- [x] 5.6 Add `DISPATCH_SVC_URL` to `.env.development` and `.env.template` with the local default, and document it in the app README.
- [x] 5.7 Rebuild the console page against the widened model: the status summary tiles, the two-line unit row (assignment or station), and the sectioned detail pane (Overview / Assignment / Crew / Capabilities), with the unassigned and no-capabilities wordings from design.md §6. Keep the selected-row treatment (`aria-current` **plus** `bg-accent text-accent-foreground`) — the attribute alone renders invisibly.
- [x] 5.8 Add the service-unavailable error state, composed from `Card` with `border-destructive` (there is no `Alert` primitive in `@mssfoobar/ui`), with `role="alert"` and the copy from design.md §6.
- [x] 5.9 Unit tests (`vitest`) for the mapping layer: envelope unwrapping, snake→camel mapping, an absent assignment staying absent, crew order preserved, and an unknown status being rejected.
- [x] 5.10 Update `apps/dispatch-web/README.md` and `AGENTS.md`: the roster now comes from `dispatch-svc`, the three commands to run everything, and the fact that there is no longer an E2E suite.
- [x] 5.11 Verify: `cd apps/dispatch-web && pnpm build && pnpm check && pnpm lint && pnpm test:unit` all exit 0.

## 6. End-to-end verification

- [x] 6.1 Whole-stack smoke, by hand rather than by an automated E2E suite (design.md D6):
  ```bash
  docker compose up -d postgres        # or: podman compose up -d postgres
  cd apps/dispatch-svc && go run ./cmd/server &
  curl -s localhost:8081/v1/units | head -c 400        # envelope with 5 units
  curl -s -o /dev/null -w '%{http_code}\n' localhost:8081/v1/units/FU-101   # 200
  curl -s -o /dev/null -w '%{http_code}\n' localhost:8081/v1/units/NOPE     # 404
  cd apps/dispatch-web && pnpm dev &
  curl -s localhost:5173/units | grep -c 'Alpha-1'     # console renders service data
  ```
- [x] 6.2 Error-state check: stop the service, reload `/units`, confirm the console shows `Units unavailable` rather than an empty list, then restart it and confirm recovery.
- [x] 6.3 **Reproducibility gate** — tear and rebuild infra, restart the native processes, and re-run the same checks as 6.1:
  ```bash
  docker compose down -v               # or: podman compose down -v
  docker compose up -d postgres        # or: podman compose up -d postgres
  cd apps/dispatch-svc && go run ./cmd/server &
  cd apps/dispatch-web && pnpm dev &
  curl -s localhost:8081/v1/units | head -c 400
  curl -s localhost:5173/units | grep -c 'Alpha-1'
  ```
  The seed must converge to the same 5 units on the fresh volume.
