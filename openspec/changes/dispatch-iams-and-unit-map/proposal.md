## Why

The workshop console is the only AOH app in the repo that nobody logs into and that shows
no map. Both gaps were deliberate — `baseline-dispatch-console` (design.md D1) stripped the
scaffold's auth layer, and the same change recorded GIS as "the natural successor once units
carry positions". Both come due together: a console with no caller has nowhere to put
`created_by`, no tenant and no way to say who may write; and a C2 app without a map is
missing the surface its operators reason on.

They land as one change because they are not independent. GIS depends on IAMS **and** RTUS
(`aoh-knowledge` → `service-catalogue.md` dependency graph); the map page cannot exist
before the `(private)` route group and the SDS-backed session that IAMS restores; and the
browser's SSE subscription to rtus-seh is authorised by the very session cookie IAMS mints.

**The map arrives integrated but empty, on purpose.** This change mounts the GIS module,
serves its Cesium assets, gates it behind a session and connects its live feed — and puts no
dispatch data on it. Field units on the map becomes a **workshop exercise**, which is the
point: integrating a platform module against a running stack is the skill the workshop
teaches, and it is a far better exercise when the infrastructure underneath is already proven
to work. `gis-service` and RTUS run from day one so the exercise has a live backend to
integrate against rather than a compose file to write first.

## What Changes

**Identity (reverses `baseline-dispatch-console` D1).**

- `dispatch-web` regains the auth layer the baseline removed: OIDC Authorization Code + PKCE
  against the bundled `aoh` realm's `web` client, the `(public)/aoh/api/auth/*` routes, the
  `(private)` route group and layout, and `AuthProvider`. Tokens live **server-side in SDS**;
  the browser holds only a `web_auth_session_id` cookie. The scaffold's gateway proxy is
  deliberately **not** restored — nothing in this change calls `dispatch-svc` from the
  browser (design.md D5).
- **BREAKING**: the console is no longer reachable unauthenticated, and its route moves from
  `/units` to `/aoh/dispatch/units` under the `(private)` group, per `aoh-conventions`
  `[project]/[module]` routing. `/` redirects to the login flow, not to the console.
- `dispatch-svc` gains bearer-token auth on `/v1/units` via `aoh-golib`'s shipped
  `aohhttp.BearerAuth` middleware. `/livez` and `/readyz` stay unauthenticated. It makes no
  outbound calls to `gis-service` — that is the exercise.
- The `aoh` realm gains exactly one thing: a **second seed user** in `realm-import.json`, so
  the viewer/dispatcher split has an account to test against. The shipped realm has only one
  interactive user, and `roles.yaml` cannot create one — `project-aas-init` requires the user
  to exist in Keycloak already. No OIDC client, realm role or claim mapper is added.
- **BREAKING**: `tenant_id`, `created_by` and `updated_by` stop being the literals
  `'workshop'` / `'system'` and come from the token's `active_tenant.tenant_id` and `sub`.
  Reads are tenant-scoped, so a seeded row no longer appears for an arbitrary caller.
- **BREAKING**: the roster seed moves out of the SQL migration and becomes service
  behaviour — `dispatch-svc` seeds a tenant once, on its first request from a dispatcher,
  before answering it. A committed migration cannot know the tenant id (AAS assigns it at
  stack-up) and SQL cannot carry the caller's identity into `created_by` either; an
  authenticated request is the only place the tenant, the identity, and the crew and
  assignment shapes the write API withholds are all available at once. A marker row makes it
  once-per-tenant, so deleting units does not resurrect them. A migration deletes the pre-auth
  rows (design.md D7). There is no seed
  endpoint, binary or script — nothing for an attendee to run.
- Two application roles — `dispatch-viewer` (read) and `dispatch-dispatcher` (write) — are
  declared as **AAS tenant roles** in `compose/iams/init/project-aas/roles.yaml`, not as
  Keycloak realm roles. The JWT carries `active_tenant.roles`; both console and service
  project roles onto permissions in-process (`aoh-knowledge` → `services/iams.md`, option 1).

**Map.**

- `dispatch-web` gains a **Map** page at `/aoh/dispatch/map` mounting
  `@mssfoobar/gis-web-sdk` (Cesium engine, 2D, OSM tiles), with its runtime assets served
  from the app's own origin and its live feed subscribed to the `gis` RTUS map over SSE via
  the session cookie. Connecting the feed now is deliberate: proving the session cookie
  reaches `rtus-seh` across origins is the fiddliest part of the integration, and it should
  not be discovered during the exercise.
- **No dispatch data reaches the map.** No position on the unit model, no `geo-entity`
  projection, no entity layer, no map↔console selection. The map draws base tiles and says
  that nothing is published to it yet.

**Runtime.**

- **BREAKING**: the compose stack grows from one container (Postgres) to the platform set —
  Traefik, `iams-db`, `iams-keycloak`, `iams-aas`, `iams-web`, `iams-init`,
  `project-aas-init`, `sds-server`, `valkey`, `rtus-db`, `rtus-pms`, `rtus-seh`, `gis-db`,
  `gis-service`, plus the `otel-collector` gateway — fifteen alongside the dispatch
  PostgreSQL. `pnpm start`'s "only PostgreSQL runs in a container" promise
  ends with this change.

Not in this pass: anything that puts dispatch data on the map (that is the new exercise),
drawing/annotation tools, GIS bookmarks, geofences, replay (that is MSR), and the three
existing stubbed exercises, which stay stubbed.

## Capabilities

### New Capabilities

- `dispatch-access-control`: who may reach the console and the service — sign-in, the
  server-side session, the role→permission projection that gates writes, sign-out, and the
  denied states (401 unauthenticated, 403 wrong role).
- `dispatch-map`: the map surface — mounted, themed, auth-gated, assets served, live feed
  connected, and explicitly carrying no dispatch data.

### Modified Capabilities

- `dispatch-console`: moves under the `(private)` route group at `/aoh/dispatch/units`,
  requires a session, gates add / edit / delete on the `dispatch-dispatcher` role, shows the
  signed-in user, and gains a nav entry alongside the map.
- `dispatch-units-api`: reads and writes are scoped to the caller's tenant; `tenant_id`,
  `created_by` and `updated_by` derive from the token; the roster is seeded by the service on
  a tenant's first dispatcher request. (The bearer requirement itself and its 401 / 403
  behaviour are specified once, under `dispatch-access-control`.)

The field unit's shape is **unchanged** — no position, no new fields — which is why
`field-unit-roster` is not among these.

## Existing AOH services considered

- `iams` (Keycloak + AAS): **selected.** The platform's only identity story, and the one the
  `aoh-web-init` scaffold this app was built from already expects. Adopted whole —
  `iams-keycloak` for authentication, `iams-aas` for the tenant-scoped application roles.
  Nothing here is built bespoke; the change *configures* IAMS — one `roles.yaml` edit for the
  application roles, and one seed user in `realm-import.json` — rather than extending it.
- `sds` (Session Data Store): **selected — not optional.** `aoh-knowledge` →
  `services/sds.md` makes SDS mandatory for every AOH web app, and rtus-seh reads the SDS
  session cookie to authorise the map's SSE stream. The cookie-only fallback in `auth.ts` is
  a diagnostic mode, not a posture to ship.
- `gis` (Geospatial Information System): **selected, and integrated only as far as the map
  surface.** The selection criteria say to use GIS when "a map display is a product
  requirement" — it is. What this change does *not* do is populate it:
  `UBIQUITOUS_LANGUAGE.md`'s *field unit* ↔ `geo-entity` mapping stays dormant, and realising
  it is the workshop exercise. The service runs so the exercise has a backend to integrate
  against. This is that mapping still waiting
  rather than a new dependency being invented.
- `rtus` (Real-time Update Service): **selected, transitively.** GIS depends on it, and it is
  what makes a moving unit move on another dispatcher's screen. It was already recorded as
  the next step in `dispatch-units-crud`'s "Existing AOH services considered".
- `dash` (Dashboard Service): **ruled out.** Checked first because the catalogue defaults any
  dashboard surface to DASH. The map is a full-bleed geospatial canvas, not a widget grid —
  the same exclusion `baseline-dispatch-console` recorded for the console still holds.
- `msr` (Multi-Session Replay): **ruled out.** Replaying a shift's movements is the obvious
  follow-on once positions exist, but nothing in this change requires time-travel, and MSR
  ships no renderer — it would need the map to land first anyway.
- **A hand-rolled map instead of the GIS module**: **ruled out.** It would mean building a
  tile client and an entity renderer that `@mssfoobar/gis-web-sdk` already provides, and it
  would teach attendees the opposite of the lesson — the workshop is about integrating
  platform modules, not replacing them.

## Impact

- **`apps/dispatch-svc`**: bearer-auth middleware, tenant + identity threading through
  service and repo, a migration adding the seed-marker table and removing the pre-auth rows,
  the once-per-tenant seed, and config for the Keycloak host. Existing handlers keep their
  shape; the `WHERE tenant_id = $n` predicate is new on every query. No GIS client, no
  outbox, no worker — those belong to the exercise.
- **`apps/dispatch-web`**: the restored auth surface (hooks, auth routes, `(private)` group,
  `AuthProvider`, `nav.ts`, `Sidebar` + `Navbar` — but no gateway proxy), the map page and its
  `+page.ts` (`ssr = false`), the Cesium static-asset vite plugin, role-gated controls on the
  console, `App.Locals` regaining `authResult`. New deps: `@mssfoobar/gis-web-sdk`,
  `@mssfoobar/auth-sdk`, `@cesium/engine`, `@cesium/widgets`. `@mssfoobar/sse-client` is
  already a dependency.
- **`compose/`**: regenerated by `aoh-compose` with `rtus` + `gis`; `compose.yml`'s "there is
  deliberately no Traefik, no Keycloak, no iams-aas, no sds-server and no valkey" comment is
  now false and goes. `compose/iams/init/project-aas/roles.yaml` carries the two roles and
  their assignments;
  `compose/iams/keycloak/realm-import.json` gains one seed user and nothing else. Because
  Keycloak's realm import is skip-if-exists, that edit only takes effect on a stack brought
  up from empty volumes.
- **`scripts/` and the root `package.json`**: one new script (`e2e-smoke.mjs`) and one
  broken one — `dev.mjs` starts only PostgreSQL, waits on `/units`, and serves on
  `localhost`, all three of which stop working here. `stop` and `reset-db` now act on the
  whole stack rather than one container. No seed script: the service seeds itself.
- **Docs that assert the opposite of this change and must move with it**:
  `apps/dispatch-web/AGENTS.md`'s "🛑 This app has NO authentication" table,
  `apps/dispatch-svc/AGENTS.md`'s "No authentication" bullet, the root `README.md`'s no-auth
  and one-container claims, `apps/dispatch-svc/README.md`'s "no authentication" summary line,
  `apps/dispatch-web/README.md`'s "This app has no authentication" section, `SETUP.md`, and
  `UBIQUITOUS_LANGUAGE.md` (the *field unit* ↔
  `geo-entity` mapping stays conditional and is corrected — `field-unit` would be the `kind`
  and `track` the `entity_type` — with a note that realising it is the new exercise).
- **Workshop**: the three stubbed exercises keep their stubs, but their routes now sit behind
  a token — `WORKSHOP.md`'s bare `curl -i -X POST http://localhost:8081/v1/units/FU-101/assignment`
  stops working as written and needs a bearer. This is the change's biggest cost to the
  workshop and is carried as a risk in `design.md`.
- **APIs**: no new `dispatch-svc` route and no new field. The only surface change is the auth
  posture on the existing routes — and that `GET /v1/units` stops being side-effect-free,
  because a dispatcher's first call seeds the tenant.
