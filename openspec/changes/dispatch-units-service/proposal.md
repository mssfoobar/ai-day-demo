## Why

`baseline-dispatch-console` shipped a console backed by a hardcoded, five-field roster
read through `listUnits()`. That accessor exists precisely so the data can come from a
real service without touching the page. This change makes it real, and fills the console
out with the information a dispatcher would actually need — the current screen is too thin
to be a credible operations surface.

## What Changes

- Add a Go microservice at `apps/dispatch-svc` (chi, layered handler → service → repo,
  sqlx, Viper, `aoh-golib` for the response envelope and logging), exposing
  `GET /v1/units`, `GET /v1/units/{unit_code}`, `GET /livez` and `GET /readyz`.
- Add a PostgreSQL database (one container — **no Traefik, no Keycloak, no IAMS, no SDS**)
  with a `dispatch` schema, migrations, and idempotent seed data.
- **BREAKING (internal)**: widen the field-unit model well beyond call sign / id / status.
  A unit now also carries its type, station, sector, radio channel, shift, capabilities,
  crew, current assignment, and last-contact time.
- Point the console at the service through a SvelteKit **server-side load**
  (`+page.server.ts`). The browser only ever talks to its own origin, so there is no CORS
  configuration and no service URL in the browser bundle. `listUnits()` moves server-side
  and becomes an HTTP client call.
- Enrich the console: status summary tiles, a richer unit row (type, assignment or
  station on a second line), and a detail pane with Overview / Assignment / Crew /
  Capabilities sections.
- **Remove the Playwright end-to-end layer** from `dispatch-web` — `tests/e2e/`,
  `playwright.config.ts`, the `@playwright/test` dependency, and the `test:integration`
  turbo task. Verification is `lint`, `check-types`, `build`, and unit tests on both apps.

Non-goals: authentication of any kind, real-time updates, a map, write operations
(the API is read-only), and pagination.

The console remains an AOH design-system surface built from `@mssfoobar/ui` primitives.

## Capabilities

### New Capabilities

- `dispatch-units-api`: the service's read API — resource paths, response envelope,
  health probes, the persisted unit model, and its seed data.

### Modified Capabilities

- `field-unit-roster`: the roster's shape widens (type, station, sector, radio channel,
  shift, capabilities, crew, assignment, last contact) and its provenance changes from
  hardcoded in-app data to the `dispatch-svc` API read server-side.
- `dispatch-console`: the console gains status summary tiles, a two-line unit row, and a
  sectioned detail pane; and it now renders data fetched on the server rather than
  imported at build time, so it gains a load-failure state.

> Both modified capabilities are specified by `baseline-dispatch-console`, which is
> complete but not yet archived. Archive that change before this one so the deltas apply
> to a populated `openspec/specs/`.

## Existing AOH services considered

- `gis` (Geospatial Information System): **ruled out, but now the closest call.** Units
  gain a `sector` and a station, and GIS is the platform's home for anything with a
  position — a field unit maps onto a `geo-entity`. It stays out because this change still
  stores no coordinates and renders no map, which is the catalogue's stated skip
  condition. `dispatch-svc` owns the unit's operational attributes; if the workshop later
  puts units on a map, GIS holds the geometry and references the unit by `unit_code`
  rather than this service being replaced.
- `rtus` (Real-time Update Service): **ruled out for now, and the natural next step.**
  A dispatch console genuinely wants live status changes, and RTUS is how AOH pushes
  them. This change has no write path, so nothing mutates and there is no event to
  publish. Adding writes should pull RTUS in with them rather than introducing polling.
- `iams` (Keycloak + AAS): **ruled out — deliberately.** The workshop is unauthenticated
  by decision; `baseline-dispatch-console` removed the auth layer entirely and this change
  does not reinstate it. `tenant_id` is still carried on every row so multi-tenancy is not
  designed out, but nothing populates it from a token.
- `dash` (Dashboard Service): **ruled out.** The status summary tiles are inline metrics
  on a list/detail page, which is the catalogue's explicit exclusion from the
  "dashboards belong to DASH" rule — not a widget grid.

## Impact

- **New code**: `apps/dispatch-svc/` (Go service + migrations + seed), `compose/` (one
  Postgres service).
- **Modified**: `apps/dispatch-web` — `roster.ts` becomes an HTTP client, the console page
  gains a `+page.server.ts`, and the page renders the widened model.
- **Removed**: the Playwright layer in `dispatch-web` (see What Changes). This drops the
  automated guard on selection behaviour and the no-auth posture; unit tests and
  type-checking remain.
- **Runtime dependencies**: PostgreSQL. The frontend alone no longer renders data — it
  needs the service, which needs the database. That is a real regression in setup cost
  versus the baseline and is accepted deliberately.
- **APIs**: `GET /v1/units`, `GET /v1/units/{unit_code}`, `GET /livez`, `GET /readyz` —
  all unauthenticated.
- **Tooling**: `go.work` and `turbo.json` gain the new workspace. Go 1.25 and Docker are
  required in addition to Node and pnpm.
