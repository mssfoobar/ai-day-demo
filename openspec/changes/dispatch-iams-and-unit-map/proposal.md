## Why

The workshop console is the only AOH app in the repo that nobody logs into and that shows
no map. Both gaps were deliberate — `baseline-dispatch-console` (design.md D1) stripped the
scaffold's auth layer, and the same change recorded GIS as "the natural successor once units
carry positions". Those two deferrals have now come due together: a dispatcher
who cannot see where a unit *is* cannot dispatch it well, and a console with no caller has
nowhere to put `created_by`, no tenant, and no way to say who may write.

They land as one change because they are not independent. GIS depends on IAMS **and** RTUS
(`aoh-knowledge` → `service-catalogue.md` dependency graph); the map page cannot exist
before the `(private)` route group and the SDS-backed session that IAMS restores; and the
browser's SSE subscription to rtus-seh is authorised by the very session cookie IAMS
mints. Splitting them would mean shipping an auth change whose only consumer
arrives in the next PR.

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
  `aohhttp.BearerAuth` middleware. `/livez` and `/readyz` stay unauthenticated. Its
  *outbound* calls to `gis-service` carry the **operator's** bearer, captured by value before
  the post-commit goroutine detaches — a service-account token would not work, because
  client-credentials tokens carry no `active_tenant` claim and `gis-service` is tenant-aware
  (design.md D2a).
- The `aoh` realm gains exactly one thing: a **second seed user** in `realm-import.json`, so
  the viewer/dispatcher split has an account to test against. The shipped realm has only one
  interactive user, and `roles.yaml` cannot create one — `project-aas-init` requires the user
  to exist in Keycloak already. No OIDC client, realm role or claim mapper is added.
- **BREAKING**: `tenant_id`, `created_by` and `updated_by` stop being the literals
  `'workshop'` / `'system'` and come from the token's `active_tenant.tenant_id` and `sub`.
  Reads are tenant-scoped, so a seeded row no longer appears for an arbitrary caller.
- **BREAKING**: the roster seed moves out of the SQL migration and into an authenticated
  script (`scripts/seed-roster.mjs`). A committed migration cannot know the tenant id — AAS
  assigns it at stack-up — and rows written in SQL bypass the GIS projection, so they would
  never appear on the map. A migration deletes the pre-auth rows instead (design.md D12).
- Two application roles — `dispatch-viewer` (read) and `dispatch-dispatcher` (write) — are
  declared as **AAS tenant roles** in `compose/iams/init/project-aas/roles.yaml`, not as
  Keycloak realm roles. The JWT carries `active_tenant.roles`; both console and service
  project roles onto permissions in-process (`aoh-knowledge` → `services/iams.md`, option 1).

**Map.**

- The `unit` table and the wire model gain a position: `position_lon`, `position_lat`,
  `position_at`, surfaced as an optional `position` object. A unit without a fix omits the
  key, exactly as `assignment` does.
- `dispatch-svc` mirrors every **positioned** field unit into `gis-service` as a
  **geo-entity** (`entity_id` = `unit_code`, `entity_type` = `track`, `properties.kind` =
  `field-unit`), written through a transactional outbox so the mirror can never disagree with
  the unit row. A unit with no position — including one whose position is cleared — has its
  entity removed instead, so the map never shows a stale marker (design.md D6a). GIS
  republishes each change to the RTUS map `gis`.
- `dispatch-web` gains a **Map** page at `/aoh/dispatch/map` mounting
  `@mssfoobar/gis-web-sdk` (Cesium engine, 2D, OSM tiles), subscribed to the `gis` RTUS map
  over SSE via the session cookie. Selecting a unit on the map opens it in the console;
  selecting one in the console flies the map to it.
- The console's detail pane shows the selected unit's last known position and fix time.

**Runtime.**

- **BREAKING**: the compose stack grows from one container (Postgres) to the platform set —
  Traefik, `iams-db`, `iams-keycloak`, `iams-aas`, `iams-web`, `iams-init`,
  `project-aas-init`, `sds-server`, `valkey`, `rtus-db`, `rtus-pms`, `rtus-seh`, `gis-db`,
  `gis-service`, plus the `otel-collector` gateway — fifteen alongside the dispatch
  PostgreSQL. `pnpm start`'s "only PostgreSQL runs in a container" promise
  ends with this change.

Not in this pass: editing a unit's position from the console (positions are seeded and
mirrored, not authored), drawing/annotation tools, GIS bookmarks, geofences, replay (that is
MSR), and the three stubbed workshop exercises, which stay stubbed.

## Capabilities

### New Capabilities

- `dispatch-access-control`: who may reach the console and the service — sign-in, the
  server-side session, the role→permission projection that gates writes, sign-out, and the
  denied states (401 unauthenticated, 403 wrong role).
- `field-unit-geo-projection`: the unit's position data and its mirroring into GIS as a
  `geo-entity` — what is written, when, transactionally, and what happens on delete.
- `dispatch-map`: the operator map surface — live rendering of field units, selection
  cross-linked to the console, layers, and the states when the live feed is absent.

### Modified Capabilities

- `dispatch-console`: moves under the `(private)` route group at `/aoh/dispatch/units`,
  requires a session, gates add / edit / delete on the `dispatch-dispatcher` role, shows the
  signed-in user, gains a nav entry and a link to the map, and renders position in the
  detail pane.
- `dispatch-units-api`: reads and writes are scoped to the caller's tenant; `tenant_id`,
  `created_by` and `updated_by` derive from the token; `position` joins the resource; and the
  persisted model gains the position columns and the projection outbox table. (The bearer
  requirement itself and its 401 / 403 behaviour are specified once, under
  `dispatch-access-control`.)
- `field-unit-roster`: a field unit's shape widens — units carry a last-known **position**
  and its **fix time**, distinct from last contact. (Tenant scoping of what the roster
  returns is specified once, under `dispatch-units-api`, rather than restated here.)

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
- `gis` (Geospatial Information System): **selected.** The selection criteria say to use GIS
  when "a map display is a product requirement" and to skip it when coordinates are never
  visualised — this change is precisely the former. `UBIQUITOUS_LANGUAGE.md` has mapped our
  *field unit* onto a GIS `geo-entity` since the baseline; this is that mapping becoming real
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
- **Building positions into `dispatch-svc` without GIS**: **ruled out.** It would mean
  hand-rolling a tile/entity renderer and a live-position feed that GIS + RTUS already
  provide, and it would strand the `geo-entity` mapping the project has documented since day
  one.

## Impact

- **`apps/dispatch-svc`**: bearer-auth middleware, tenant + identity threading through
  service and repo, a migration adding position columns and the GIS outbox table, an outbox
  worker publishing to `gis-service`, config for the Keycloak and GIS hosts. Existing
  handlers keep their shape; the `WHERE tenant_id = $n` predicate is new on every query.
- **`apps/dispatch-web`**: the restored auth surface (hooks, auth routes, `(private)` group,
  `AuthProvider`, `nav.ts`, Sidebar/Headerbar — but no gateway proxy), the map page and its
  `+page.ts` (`ssr = false`), the Cesium static-asset vite plugin, role-gated controls on the
  console, `App.Locals` regaining `authResult`. New deps: `@mssfoobar/gis-web-sdk`,
  `@mssfoobar/auth-sdk`, `@cesium/engine`, `@cesium/widgets`. `@mssfoobar/sse-client` is
  already a dependency.
- **`compose/`**: regenerated by `aoh-compose` with `rtus` + `gis`; `compose.yml`'s "there is
  deliberately no Traefik, no Keycloak, no iams-aas, no sds-server and no valkey" comment is
  now false and goes. `compose/iams/init/project-aas/roles.yaml` carries the two roles and the
  `compose/iams/init/project-aas/roles.yaml` carries the two roles and their assignments;
  `compose/iams/keycloak/realm-import.json` gains one seed user and nothing else. Because
  Keycloak's realm import is skip-if-exists, that edit only takes effect on a stack brought
  up from empty volumes.
- **Docs that assert the opposite of this change and must move with it**:
  `apps/dispatch-web/AGENTS.md`'s "🛑 This app has NO authentication" table,
  `apps/dispatch-svc/AGENTS.md`'s "No authentication" bullet, the root `README.md`'s no-auth
  and one-container claims, `apps/dispatch-svc/README.md`'s "no authentication" summary line,
  `apps/dispatch-web/README.md`'s "This app has no authentication" section, `SETUP.md`, and
  `UBIQUITOUS_LANGUAGE.md` (the *field unit* ↔
  `geo-entity` mapping stops being conditional; *position* and *fix time* join the table).
- **Workshop**: the three stubbed exercises keep their stubs, but their routes now sit behind
  a token — `WORKSHOP.md`'s bare `curl -i -X POST http://localhost:8081/v1/units/FU-101/assignment`
  stops working as written and needs a bearer. This is the change's biggest cost to the
  workshop and is carried as a risk in `design.md`.
- **APIs**: no new `dispatch-svc` route. The surface changes are the auth posture on existing
  routes and the `position` field on the unit resource.
