## Context

The console is an `aoh-web-init` scaffold with its auth layer amputated
(`baseline-dispatch-console` D1) and a hand-written Go service behind it
(`dispatch-units-service` D1). Reads happen in a SvelteKit server `load`, writes in form
actions; the browser only ever talks to its own origin. `UBIQUITOUS_LANGUAGE.md` has mapped
our *field unit* onto a GIS `geo-entity` since the baseline and recorded that the mapping is
dormant "until units carry positions."

Three platform constraints shape everything below.

1. **Auth here is all-or-nothing.** The scaffold ran OIDC discovery in a top-level `await`
   in `hooks.server.ts`, so with no reachable identity provider *every* route returned 500
   — a `(public)` group did not help. That is why the baseline deleted the layer rather
   than disabling it, and why it returns whole.
2. **AAS owns application roles; Keycloak issues tokens.** The JWT carries
   `active_tenant.roles` and **no** resolved `active_tenant.permissions` array
   (`aoh-knowledge` → `services/iams.md`, verified against iams-aas v1.3.7). Middleware that
   reads a permissions claim rejects every legitimate user.
3. **GIS does not stand alone.** `gis` depends on `iams` and `rtus`; `sds` ships with
   `iams` and is mandatory for any AOH web app, not least because `rtus-seh` authorises the
   browser's SSE stream by resolving the SDS session cookie.

The consumers are workshop attendees on laptops, running the two apps natively against
composed-up infrastructure. Nothing here targets production.

## Goals / Non-Goals

**Goals:**

- Restore the console's authentication layer whole, on the platform's terms: OIDC/PKCE,
  SDS-held tokens, AAS-owned roles.
- Give `dispatch-svc` a real caller — tenant scoping and audit identity from the token
  instead of the `'workshop'` / `'system'` placeholders.
- Put field units on a map that moves without a reload, by mirroring them into GIS and
  letting GIS's existing RTUS publication do the fan-out.
- Keep the change reversible in review: every new behaviour is additive to existing code
  paths, and the console's data-flow architecture (server-side reads, form-action writes)
  is untouched.

**Non-Goals:**

- Authoring positions from the console. Positions are seeded and settable through the API;
  no drag-to-move, no simulator.
- GIS bookmarks, drawing, measurement, geofences, or a layer-management UI beyond what is
  needed to see the fleet.
- Replay of historical movement (that is MSR) and any second map surface.
- Production authorization. `roles.yaml` is development-only, as `aoh-knowledge` states.
- Building the three stubbed workshop exercises. They stay stubbed.

## Runtime dependencies

| Dependency | Role | Owned by |
|---|---|---|
| `traefik` | Dev routing on `${DEV_DOMAIN}`, plus the TCP entrypoint SDS is reached on | Platform — added by `aoh-compose` |
| `iams-db` | Keycloak's backing PostgreSQL | Platform — added by `aoh-compose` |
| `iams-keycloak` | OIDC provider; hosts the `aoh` realm, its `web` PKCE client and the seeded users; issues and signs the JWTs | Platform — added by `aoh-compose` |
| `iams-aas` | Authorization control plane; owns the `dispatch-viewer` / `dispatch-dispatcher` tenant roles and surfaces them as `active_tenant.roles` | Platform — added by `aoh-compose` |
| `iams-init` | One-shot Newman runner; creates the `development` tenant and the admin membership | Platform — added by `aoh-compose` |
| `project-aas-init` | One-shot idempotent reconciler that applies `roles.yaml` to AAS | Platform — added by `aoh-compose`; its `roles.yaml` is owned by this change |
| `sds-server` | Server-side session store; holds the access + refresh tokens and mints an access token for `rtus-seh` | Platform — added by `aoh-compose` |
| `valkey` | SDS's backing store | Platform — added by `aoh-compose` |
| `rtus-db` | RTUS metadata PostgreSQL | Platform — added by `aoh-compose` |
| `rtus-pms` | Pub/sub control plane; holds the `gis` JSON map that `gis-service` publishes to | Platform — added by `aoh-compose` |
| `rtus-seh` | SSE delivery to the browser; authorises the subscription from the `web_auth_session_id` cookie via SDS | Platform — added by `aoh-compose` |
| `gis-db` | Geospatial PostgreSQL | Platform — added by `aoh-compose` |
| `gis-service` | Geo-entity REST API; publishes entity changes to the `gis` RTUS map through its own transactional outbox | Platform — added by `aoh-compose` |
| `otel` collector | OTLP gateway both apps already export to | Platform — added by `aoh-compose` |
| `postgres` (dispatch) | `dispatch-svc`'s own database — gains position columns and the projection outbox table | Existing — modified by this change |
| `dispatch-svc` | Go field-unit service; runs natively (`go run`), not composed | Existing — modified by this change |
| `dispatch-web` | SvelteKit console; runs natively (`pnpm dev`), not composed | Existing — modified by this change |

`rtus-pms` and `rtus-seh` must share `rtus.clustername` (`aoh_rtus` in the shipped
template) or SEH forms its own one-node Hazelcast cluster and every subscription silently
receives nothing.

## API surface

### Exposed by `dispatch-svc` (this change modifies the auth posture; it adds no route)

| Method | Path | Auth posture | Description | Fronting gateway route |
|---|---|---|---|---|
| GET | `/v1/units` | BearerAuth, role `dispatch-viewer` or `dispatch-dispatcher` | List the caller's tenant's units, each with `position` when it has one | n/a — see D5 |
| GET | `/v1/units/{unit_code}` | BearerAuth, role `dispatch-viewer` or `dispatch-dispatcher` | One unit in the caller's tenant; 404 for another tenant's | n/a — see D5 |
| POST | `/v1/units` | BearerAuth, role `dispatch-dispatcher` | Create, now accepting `position`; `tenant_id` / `created_by` from the token | n/a — see D5 |
| PUT | `/v1/units/{unit_code}` | BearerAuth, role `dispatch-dispatcher` | Replace, now accepting and clearing `position`; `occ_lock` unchanged | n/a — see D5 |
| DELETE | `/v1/units/{unit_code}` | BearerAuth, role `dispatch-dispatcher` | Delete, and enqueue the geo-entity delete | n/a — see D5 |
| GET | `/livez` | Unauthenticated | Liveness — unchanged | n/a |
| GET | `/readyz` | Unauthenticated | Readiness — unchanged; still fails when the database is unreachable | n/a |

Unchanged from `dispatch-units-crud`: `PATCH /v1/units/{unit_code}` and collection-level
`PUT` / `DELETE` stay unmounted and answer 405. Every 2xx body remains the AOH success
envelope (`data`, `message`, `sent_at`); errors remain the AOH error contract.

### Exposed by `dispatch-web`

| Route | Auth posture | Description |
|---|---|---|
| `/(public)/aoh/api/auth/login` · `/callback` · `/logout` · `/refresh` | PKCE (the flow itself) | The restored OIDC endpoints from the `aoh-web-init` scaffold |
| `/(private)/aoh/dispatch/units` | Session required | The console, moved from `/units` |
| `/(private)/aoh/dispatch/map` | Session required | The map, `ssr = false` |
| `/livez` · `/readyz` | Unauthenticated | Unchanged |

### Consumed, not exposed

These are `gis-service`'s own endpoints, called server-to-server by `dispatch-svc`'s outbox
worker. They are listed for completeness and are **not** part of this change's API surface —
their paths are `gis-service`'s to define, not ours.

| Method | Path (on `gis-service`) | Called by | Purpose |
|---|---|---|---|
| PUT | `/geoentity` | `dispatch-svc` outbox worker | Upsert the geo-entity mirroring a unit |
| DELETE | `/geoentity/entity_id/{entity_id}` | `dispatch-svc` outbox worker | Remove a deleted unit's entity |
| GET | `/geoentity/entity_id/{entity_id}` | Verification only | Assert the projection landed |

The browser's only cross-origin call is the SSE subscription to `rtus-seh`, carrying the
session cookie and no token.

## UI / Design System

Two surfaces change. Clickable mockups, each with a state switcher covering every state the
specs name:

- `openspec/changes/dispatch-iams-and-unit-map/design/dispatch-map-mock.html` — the map:
  loaded with a live feed, live feed unavailable, no positioned units, un-positioned count
  shown, a unit selected, and the viewer (read-only) variant.
- `openspec/changes/dispatch-iams-and-unit-map/design/dispatch-console-auth-mock.html` —
  the console's new states: signed in as a dispatcher (write controls present), signed in as
  a viewer (write controls absent), permission denied after a 403, the position section for
  a positioned unit, and the position section for an un-positioned unit.

**Primitives.** Everything outside the map canvas composes from `@mssfoobar/ui` subpaths —
`Button`, `Card`/`CardHeader`/`CardContent`/`CardTitle`, `Badge`, `Separator`, `ScrollArea`,
`Sheet`, `AlertDialog`, `Select`, `Input`, `Toaster`, plus `Sidebar` and `Navbar` returning
with the layout. The permission-denied state is a `Card` + `Badge` + `Button`, per
`aoh-design`'s empty/denied pattern; the signed-in operator sits in the `Navbar`. Nothing
here is hand-rolled — `@mssfoobar/ui` ships every one of these, and hand-rolling silently
bypasses the theme tokens and dark mode.

**The map itself is SDK-owned.** GIS is a platform module with its own SDK, so the map is
composed from `@mssfoobar/gis-web-sdk` — `GisProvider` (at the layout root, fed the app's
existing `ThemeProvider` dark-mode store), `CesiumMapEngineProvider`, `Map`,
`MapBaseLayerProvider` + `MapXyzSourceProvider` (OpenStreetMap tiles), `MapEntityLayerProvider`
+ `MapEntityProvider kind="field-unit"` with a typed snippet rendering the call-sign marker,
and `MapLayerManager`. No tile client, entity layer, or marker renderer is written by hand.
`MapControlPanel` and `MapBookmarkManager` are deliberately left out — drawing and bookmarks
are non-goals.

**Copy and tone.** Sentence case, operational register, no emoji, per `aoh-design`. "Live
positions are unavailable." — not "Oops! We couldn't connect." The permission-denied copy
follows the platform line: "Access to this action is restricted. For assistance with access,
please contact your administrator."

**Dark mode and tenant theming.** The map repaints with the host's theme because
`GisProvider` takes the app's dark-mode store; the console's tokens are unchanged. Both mock
files carry the light/dark toggle the baseline mock established.

## Decisions

**D1 — One change, not two.** IAMS and the map ship together. *Alternative considered:* two
sequential changes, `dispatch-iams-auth` then `dispatch-unit-map`, which is what the repo's
"one logical change per PR" rule would normally suggest. Rejected because the map has no
independent existence: GIS requires IAMS and RTUS, the map route requires the `(private)`
group, and `rtus-seh` authorises the browser's SSE stream with the SDS cookie IAMS mints.
Splitting them means an auth change whose only consumer is the next PR, and a map change
that cannot be demonstrated until both have landed. The cost is a large blast radius,
carried as R1.

**D2 — Reuse the bundled `web` client for the console; register exactly one confidential
client for the outbox worker.** The console authenticates with the `aoh` realm's existing
public `web` PKCE client, in the `development` tenant `iams-init` creates — no frontend
client, no realm role, no claim mapper is added. `dispatch-svc` validates incoming bearer
tokens offline against the realm JWKS, which needs no client either.

The one exception is **outbound**: the outbox worker calls `gis-service` asynchronously,
after the originating request has returned, so it has no user token to carry and cannot be
given one — a retried delivery may happen minutes later, long after the operator's
five-minute access token has expired. *Alternatives considered:* (a) propagating the
operator's access token into the outbox row — breaks on the first retry past expiry, and
persists a bearer token in the database; (b) holding the operator's refresh token — worse,
and it makes a background worker able to act as a user indefinitely. So a **confidential
`dispatch-svc` client** with `serviceAccountsEnabled` is added to `realm-import.json` and the
worker uses the client-credentials grant. This is precisely the case `aoh-knowledge` →
`services/iams.md` names as warranting a new client ("Register a new client only for a
*confidential* (client-credentials / introspection) backend"), and `aoh-compose` documents
`realm-import.json` as the right home for it. Its service account is made a member of the
`development` tenant through `roles.yaml` so `gis-service` accepts its writes. Consequence:
editing `realm-import.json` means the Keycloak import must be re-run with
`down -v iams-db` — carried as R9.

**D3 — Roles in AAS; permissions projected in-process.** `dispatch-viewer` and
`dispatch-dispatcher` are AAS tenant roles declared in
`compose/iams/init/project-aas/roles.yaml`. Both apps map role names to permissions with a
static table that mirrors that file — option 1 of the three `services/iams.md` lists.
*Alternatives considered:* (a) `POST` to AAS `/evaluate` per request — authoritative, but a
network hop on every call for a two-role matrix, and it would put AAS on `dispatch-svc`'s
request path; (b) gating on role names inline at each handler — the same thing without a
named projection, fine for coarse checks but it scatters the matrix. The projection is
duplicated in two places (Go and TypeScript), which R3 covers. What is *not* on the table is
reading `active_tenant.permissions`: AAS does not emit it, and code that expects it 403s
every legitimate user.

**D4 — SDS holds the tokens; the browser holds a session id.** `PUBLIC_COOKIE_PREFIX` stays
at the platform default `web`, so the cookie is `web_auth_session_id` and matches the
`rtus.session-id.cookienames` value already seeded on `rtus-seh` — no `compose/rtus` edit.
*Alternative considered:* the cookie-only fallback in `auth.ts`. Rejected: it is a
diagnostic mode, it puts a usable bearer token in the browser, and `rtus-seh` would have to
fall back to reading an access-token cookie — the legacy path.

**D5 — No gateway proxy; the browser still never calls `dispatch-svc`.** The console keeps
its architecture: reads in a server `load`, writes in form actions, with the access token
fetched from SDS server-side and sent as `Authorization: Bearer`. *Alternative considered:*
restoring `gateway.config.ts` and the `(private)/aoh/gateway/[...path]` proxy along with the
rest of the auth layer. Rejected as speculative — this change introduces no browser-side
call to `dispatch-svc`, and `aoh-conventions/web.md` is explicit that a server `load` must
*not* route through the gateway anyway (the proxy reads a bearer that a server sub-request
does not carry, so it 401s before forwarding). The map's cross-link to the console is a
SvelteKit navigation, not a fetch. When a browser-side call first appears, the gateway
arrives with it.

**D6 — Mirror units into GIS through a transactional outbox; never call GIS inline.** Each
unit write records an outbox row in the same transaction; a worker drains it into
`gis-service` with retry. *Alternatives considered:* (a) calling `gis-service` inline in the
handler — couples a unit write to GIS's availability and can half-succeed; (b) logical
replication / CDC — far more machinery than five units need. The outbox is also the pattern
`gis-service` itself uses to reach RTUS, so the shape is already in the platform's
vocabulary. Delivery is at-least-once, so the projection must be idempotent — `PUT
/geoentity` is an upsert keyed on `entity_id`, which makes replay a no-op.

**D7 — `dispatch-svc` owns the position; GIS is a projection, not the source of truth.** The
`unit` row carries `position_lon` / `position_lat` / `position_at`; GIS holds a mirror.
*Alternative considered:* storing positions only in GIS and reading them back for the
console. Rejected — it would put a second service on the console's read path for a field the
detail pane always shows, and it would make `dispatch-svc` unable to answer "where is this
unit" without a network hop. The cost is that the two can diverge while the outbox is
draining; R4 covers it.

**D8 — `entity_id` = `unit_code`, `entity_type` = `track`, `kind` = `field-unit`.**
`unit_code` is already the stable per-tenant key and is what `UBIQUITOUS_LANGUAGE.md` maps
onto `entity_id`. `track` is GIS's reserved type for entities with real-time position
updates — `static` would be wrong the moment a unit moves. `kind` is the app-level grouping
GIS filters and styles by, and `MapEntityProvider` keys on it. Note this corrects the
language file, which currently guesses `entity_type=field_unit`; `field-unit` is a `kind`,
not an `entity_type`.

**D9 — The map has one source of entity state.** The page does not fetch an entity list of
its own; the SDK's subscription (`init=true`) is the only source. *Alternative considered:*
an initial server-side list plus live updates, the usual AOH pattern. Rejected here because
it requires a dedup step at the prepend site to survive the fetch/SSE race, and there is
nothing to gain — the SDK already replays current state on connect. The console keeps its
own server-side roster read; that is unit data, not entity data.

**D10 — Native dev runs on a `${DEV_DOMAIN}` hostname, not `localhost`.** The console is
served at `http://dispatch.${DEV_DOMAIN}:5173` with `ORIGIN` matching and
`PUBLIC_DOMAIN=${DEV_DOMAIN}`, so the session cookie is issued on the parent domain and
travels to `rtus-seh.${DEV_DOMAIN}` on the SSE request. *Alternative considered:* staying on
`localhost:5173`. Rejected — `localhost` and `*.aoh.localhost` do not share a cookie parent,
so the subscription would 401 with nothing in either log to explain it. `*.localhost`
resolves to the loopback without a hosts-file edit on macOS and Linux; Windows is R5.

## Risks / Trade-offs

**R1 — The workshop's "one container, no auth" promise ends.** The stack goes from one
container to roughly fourteen, several pulled from `ghcr.io/mssfoobar`, and every attendee
now needs a working login before they see a unit. → The reproducibility gate in `tasks.md`
proves a cold `down -v` / `up -d` converges, `SETUP.md` and `README.md` are rewritten in the
same change rather than left stale, and the seeded dispatcher account is documented where
the old curl example lived. This is the single largest cost of the change and it is worth
re-confirming before apply.

**R2 — `WORKSHOP.md`'s bare `curl` stops working.** The 501 stubs are now behind a bearer,
so the documented `curl -i -X POST http://localhost:8081/v1/units/FU-101/assignment` answers
401. → `WORKSHOP.md` gains a one-line token fetch (password grant against the bundled `web`
client) that the curl examples reuse. The three exercises' acceptance criteria are otherwise
untouched.

**R3 — The role→permission matrix is duplicated in Go, TypeScript and `roles.yaml`.** Three
copies drift. → Keep the matrix to two roles and one permission boundary (read vs write),
name the file each copy mirrors in a comment, and make the verification step decode a real
token rather than asserting the map in isolation. If it grows past a handful of roles, that
is the signal to move to AAS `/evaluate`.

**R4 — The unit row and its geo-entity diverge while the outbox drains.** A dispatcher can
see a unit in the console that is not yet on the map. → The window is a worker interval, the
projection is idempotent and retried, and the map's un-positioned count is computed from
unit data rather than entity data, so the page never silently under-reports the fleet.

**R5 — Windows attendees and `*.localhost`.** Windows does not resolve `*.localhost` to the
loopback by default, so D10's hostname may not resolve. → Document the one-line hosts-file
entry in `SETUP.md`; the devcontainer path is unaffected because the container joins the
compose network. Confirm on a Windows laptop before the workshop.

**R6 — A black map canvas.** Cesium fetches its workers and textures at runtime; without the
asset-copy step the canvas renders entirely black with only 404s in the console to explain
it. → The vite plugin runs on both `buildStart` and `configureServer` so dev and build are
covered, and the specs assert "no 404 under the Cesium base URL" as an observable outcome
rather than trusting the plugin.

**R7 — Cesium is a heavy dependency on a workshop laptop.** `@cesium/engine` plus
`@cesium/widgets` add a large install and a WebGL requirement. → Accepted: it is the only
working engine in the SDK (`MapLibreEngineProvider` is an empty stub), and the map is a 2D
scene with Ion disabled, so no external asset quota is consumed.

**R8 — `@mssfoobar` image pulls.** The npm token checked into `.npmrc` covers packages, not
`ghcr.io` container images. → Confirm image pull access (or a preloaded image tarball, as
the repo already does for the devcontainer) before the workshop; this is a prerequisite of
apply, not a discovery for the day.

**R9 — The `realm-import.json` edit is skip-if-exists.** Keycloak's `start --import-realm`
imports only into an empty database, so adding the confidential client to an already-running
stack silently does nothing and the worker then fails to get a token, with no error at
import time to explain it. → The compose task pairs the edit with
`podman compose down -v iams-db` (or `docker compose down -v iams-db`) followed by
`up -d`, and the seed verification obtains a client-credentials token rather than assuming
the client exists. The reproducibility gate tears the whole stack with `-v` anyway, so a
fresh run always re-imports.

## Migration Plan

1. Bring the platform stack up (`aoh-compose` output), let `iams-init` then
   `project-aas-init` complete, and confirm a token for the seeded dispatcher carries
   `dispatch-dispatcher` in `active_tenant.roles`.
2. Apply the `dispatch-svc` migration: position columns, their constraints, the outbox
   table. Additive only — every column is nullable or defaulted, so the existing rows and the
   pre-change binary keep working.
3. Re-seed. Seeded rows move from `tenant_id = 'workshop'` to the `development` tenant id and
   gain positions. Because the seed is `ON CONFLICT … DO UPDATE` and keyed on
   `(unit_code, tenant_id)`, the old-tenant rows are **not** rewritten in place — the reset
   path (`pnpm reset-db`) is the supported route, and the gate exercises it.
4. Deploy `dispatch-svc` with bearer auth on. Anything still calling it unauthenticated —
   including a stale `pnpm dev` console — starts receiving 401, which is the intended signal.
5. Deploy `dispatch-web` with the restored auth layer and the map.

**Rollback:** revert both apps and re-seed. The migration is additive, so a rolled-back
binary ignores the new columns and the outbox table without error; the orphaned geo-entities
in `gis-service` are harmless and are cleaned by tearing the GIS volume. There is no
irreversible step.

## Open Questions

1. **Should `/units` redirect to `/aoh/dispatch/units`?** A 301 would spare anyone with a
   bookmark or a slide deck pointing at the old path. Against: it keeps a route alive that
   the convention says should not exist, and the console's audience is one workshop cohort.
   Currently specified as *not* serving the console; flip it if the slides reference `/units`.
2. **Does the workshop need moving units?** With positions seeded and never updated, the map
   is live in mechanism but static in appearance, and the RTUS path is never visibly
   exercised. A tiny position-nudging loop (behind a flag, off by default) would make the
   live feed self-evident in a demo. Deliberately out of scope here — raise it as its own
   change if the deck needs movement.
3. **Which seeded realm users get which role?** The realm ships its own seed users; this
   change needs one that is dispatcher and one that is viewer-only. Whether to reuse two
   existing accounts or add a viewer-only account to `roles.yaml`'s `assignments` is settled
   at apply time against the realm as shipped.
4. **Is `last_contact` still bumped on every write now that a fix time exists?**
   `dispatch-units-crud` D4 made every write bump `last_contact`. With `position_at` alongside
   it, that rule is worth re-confirming rather than inheriting silently — the spec keeps them
   independent, but the existing UPDATE does not.
