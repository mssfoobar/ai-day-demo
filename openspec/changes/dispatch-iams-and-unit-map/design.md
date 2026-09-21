## Context

The console is an `aoh-web-init` scaffold with its auth layer amputated
(`baseline-dispatch-console` D1) and a hand-written Go service behind it
(`dispatch-units-service` D1). Reads happen in a SvelteKit server `load`, writes in form
actions; the browser only ever talks to its own origin. `UBIQUITOUS_LANGUAGE.md` has mapped
our *field unit* onto a GIS `geo-entity` since the baseline and recorded that the mapping
becomes real *"once units carry positions. They do not today."*

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
  needed to see the roster on a map.
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
| `iams-web` | User/role management UI; comes up unconditionally with the IAMS fragment. Not used by this change, but it runs and is worth knowing about | Platform — added by `aoh-compose` |
| `iams-init` | One-shot Newman runner; creates the `development` tenant and the admin membership | Platform — added by `aoh-compose` |
| `project-aas-init` | One-shot idempotent reconciler that applies `roles.yaml` to AAS | Platform — added by `aoh-compose`; its `roles.yaml` is owned by this change |
| `sds-server` | Server-side session store; holds the access + refresh tokens and mints an access token for `rtus-seh` | Platform — added by `aoh-compose` |
| `valkey` | SDS's backing store | Platform — added by `aoh-compose` |
| `rtus-db` | RTUS metadata PostgreSQL | Platform — added by `aoh-compose` |
| `rtus-pms` | Pub/sub control plane; holds the `gis` JSON map that `gis-service` publishes to | Platform — added by `aoh-compose` |
| `rtus-seh` | SSE delivery to the browser; authorises the subscription from the `web_auth_session_id` cookie via SDS | Platform — added by `aoh-compose` |
| `gis-db` | Geospatial PostgreSQL | Platform — added by `aoh-compose` |
| `gis-service` | Geo-entity REST API; publishes entity changes to the `gis` RTUS map through its own transactional outbox | Platform — added by `aoh-compose` |
| `otel-collector` | OTLP gateway. Neither app exports to it today — `dispatch-svc` has no OTEL wiring at all and `dispatch-web`'s `instrumentation.server.ts` is a no-op until `OTEL_EXPORTER_OTLP_ENDPOINT` is set. It arrives with the stack; wiring it is not in this change | Platform — added by `aoh-compose` |
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
| POST · DELETE | `/v1/units/{unit_code}/assignment` | BearerAuth, role `dispatch-dispatcher` | Workshop exercise 1 stub. Still answers 501 — but now only to an authorised caller | n/a — see D5 |
| GET | `/v1/units/{unit_code}/events` | BearerAuth, role `dispatch-viewer` or `dispatch-dispatcher` | Workshop exercise 2 stub. Still 501 | n/a — see D5 |
| PUT | `/v1/units/{unit_code}/crew` | BearerAuth, role `dispatch-dispatcher` | Workshop exercise 3 stub. Still 501 | n/a — see D5 |
| GET | `/livez` | Unauthenticated | Liveness — unchanged | n/a |
| GET | `/readyz` | Unauthenticated | Readiness — unchanged; still fails when the database is unreachable | n/a |

The four stub routes are listed because this change alters their posture even though it does
not implement them: they are mounted today and answer 501 to anyone, and after this change an
unauthenticated caller gets 401 before ever reaching the 501. That is what breaks
`WORKSHOP.md`'s `curl` (R2). Unchanged from `dispatch-units-crud`:
`PATCH /v1/units/{unit_code}` and collection-level `PUT` / `DELETE` stay unmounted and
answer 405. Every 2xx body remains the AOH success
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
| GET | `/geoentity` | Verification only | Assert a token is accepted by `gis-service` at all |

The browser's only cross-origin call is the SSE subscription to `rtus-seh`, carrying the
session cookie and no token.

## UI / Design System

Two surfaces change. Clickable mockups, each with a state switcher covering every state the
specs name:

- `openspec/changes/dispatch-iams-and-unit-map/design/dispatch-map-mock.html` — the map:
  live feed; a positioned unit selected; an **un-positioned unit selected** (camera does not
  move); live feed unavailable; no positioned units; a **fully positioned roster** (no
  "not shown" count); and the viewer (read-only) variant.
- `openspec/changes/dispatch-iams-and-unit-map/design/dispatch-console-auth-mock.html` —
  the console's new states: signed in as a dispatcher (write controls present), signed in as
  a viewer (write controls absent), permission denied after a 403, **service unreachable**,
  the position section for a positioned unit, and for an un-positioned unit.

Both mocks render the `Sidebar` + `Navbar` the restored layout brings back, with the console
and map nav entries, and both use the real seeded roster (`0002_seed.up.sql`) rather than
invented units — a mockup that mis-pairs a **call sign** with a **unit ID** is precisely the
drift `UBIQUITOUS_LANGUAGE.md` exists to prevent.

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

**D2 — Reuse the bundled `web` client and add nothing to Keycloak but one seed user.** The
console authenticates with the `aoh` realm's existing public `web` PKCE client, in the
`development` tenant `iams-init` creates. No OIDC client, realm role or claim mapper is
added. `dispatch-svc` validates incoming bearer tokens with the shipped
`aohhttp.BearerAuth` middleware from `aoh-golib`, which calls Keycloak's userinfo endpoint
(`aoh-conventions/go.md`) — so it needs no client either. *Alternative considered:*
hand-rolling offline JWKS validation. Rejected: it re-implements what `aoh-golib` ships,
and it would mask the `scope=openid` requirement locally (userinfo 403s on a token issued
without it) while `gis-service` still rejected the same token.

The realm's `users` array does gain **one** entry. The shipped realm seeds exactly one
interactive user — `${DEV_USER}`, plus the `sds` and `unh` service accounts — so the
viewer/dispatcher split this change specifies has no second account to assign. `roles.yaml`
cannot create one: `bootstrap.py` exits with *"user not found in Keycloak — add to
realm-import.json"*. Adding a seed user is what `aoh-knowledge` →
`keycloak-realm-guide.md` documents `realm-import.json` for, so the viewer account goes
there. Consequence: the Keycloak import must be re-run from empty volumes — carried as R8.

**D2a — The projection carries the operator's bearer, not a service-account token.** This
reverses an earlier draft of this design, and the reason is worth recording because it is
not obvious. A client-credentials token **carries no `active_tenant` claim** at all
(`aoh-knowledge` → `integration-patterns.md`, posture B), and `gis-service` is
tenant-aware (`GIS_ACTIVE_TENANT_CLAIM_KEY=active_tenant`). A service account therefore
cannot write a tenant-scoped geo-entity — the writes would be rejected or land in the wrong
tenant, and no amount of AAS membership fixes it, because the claim is absent rather than
wrong. Posture A is mandatory for any target that validates `active_tenant`, and the
platform's own GIS seeding script uses a password-grant **user** token for exactly this
reason.

So the outbox worker carries the operator's access token, captured **by value** before the
post-commit goroutine detaches from the request context — the precise gotcha
`integration-patterns.md` names. The token is never persisted: it lives in memory for the
life of one delivery attempt. A row that outlives its token stays pending and is drained
opportunistically by the next authenticated request from the same tenant, which supplies a
live token. *Alternatives considered:* (a) storing the bearer in the outbox row — persists
a credential in the database and still expires; (b) storing the refresh token — worse, it
lets a background worker act as a user indefinitely. Both were rejected. The trade-off is
that a tenant with no traffic can hold a pending projection indefinitely; R4 covers the
resulting window.

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

**D6a — Position presence, not the write verb, decides the projection intent.** A geo-entity
is a Point; a unit with no coordinates has nothing to project. So the intent recorded in the
outbox is derived from the unit's position *after* the write, not from whether the write was
a create, a replace or a delete:

| Unit state after the write | Outbox intent | Effect in GIS |
|---|---|---|
| Has a position | `upsert` | `PUT /geoentity` |
| Has no position (never had one) | `delete` | `DELETE /geoentity/entity_id/{unit_code}` — a no-op when nothing is there |
| Position cleared by a replace | `delete` | The entity is removed |
| Unit deleted | `delete` | The entity is removed |

Without this rule the design contradicts itself: "mirror every unit" and "an un-positioned
unit has no entity" cannot both hold under an unconditional upsert, and a cleared position
would silently leave a stale marker on the map at the unit's last known location — the worst
possible failure for a dispatch surface, because it looks like data rather than an absence.
Making `delete` a no-op when the entity is missing is what lets the rule stay this simple.

**D7 — `dispatch-svc` owns the position; GIS is a projection, not the source of truth.** The
`unit` row carries `position_lon` / `position_lat` / `position_at`; GIS holds a mirror.
*Alternative considered:* storing positions only in GIS and reading them back for the
console. Rejected — it would put a second service on the console's read path for a field the
detail pane always shows, and it would make `dispatch-svc` unable to answer "where is this
unit" without a network hop. The cost is that the two can diverge while the outbox is
draining; R4 covers that window.

**D8 — `entity_id` = `unit_code`, `entity_type` = `track`, `kind` = `field-unit`.**
`unit_code` is already the stable per-tenant key and is what `UBIQUITOUS_LANGUAGE.md` maps
onto `entity_id`. `track` is GIS's reserved type for entities with real-time position
updates — `static` would be wrong the moment a unit moves. `kind` is the app-level grouping
GIS filters and styles by, and `MapEntityProvider` keys on it. Note this corrects the
language file, which currently guesses `entity_type=field_unit`; `field-unit` is a `kind`,
not an `entity_type`.

`unit_code` is unique per *tenant*, not globally, so two tenants may legitimately hold the
same code. That is safe here only because GIS is itself tenant-partitioned — entities are
written in the token's tenant and RTUS json-maps are keyed per tenant
(`aoh-gis-integration/references/entities.md`), so `entity_id` need only be unique within a
tenant. If GIS's partitioning ever turns out to be weaker than that, the fix is to key on
`{tenant_id}:{unit_code}`; it is called out here so the assumption is visible rather than
discovered by two tenants overwriting each other's markers.

**D9 — The map has one source of entity state.** The page does not fetch an entity list of
its own; the SDK's subscription (`init=true`) is the only source. *Alternative considered:*
an initial server-side list plus live updates, the usual AOH pattern. Rejected here because
it requires a dedup step at the prepend site to survive the fetch/SSE race, and there is
nothing to gain — the SDK already replays current state on connect. The console keeps its
own server-side roster read; that is unit data, not entity data.

**D10 — Native dev runs on the platform's own dev origin, `http://${DEV_DOMAIN}:5173`.**
`DEV_DOMAIN` keeps its shipped default `127.0.0.1.nip.io`, so the console is served at
`http://127.0.0.1.nip.io:5173` with `ORIGIN` matching and `PUBLIC_DOMAIN=${DEV_DOMAIN}`.
Three things fall out of that one choice, and each would otherwise be a separate bug:

1. **The cookie reaches `rtus-seh`.** The session cookie is issued on `${DEV_DOMAIN}`, which
   is the parent of `rtus-seh.${DEV_DOMAIN}`, so the SSE request carries it.
2. **CORS already permits it.** `rtus-seh`'s Traefik middleware seeds
   `accesscontrolalloworiginlist` with `http://${DEV_DOMAIN}:5173` among others. A
   credentialed SSE request from any *other* origin — including a
   `dispatch.${DEV_DOMAIN}:5173` of our own invention — is blocked, and would have required
   editing `compose/rtus/compose.yml` in this change.
3. **Windows works without a hosts-file edit.** `nip.io` wildcard-resolves to 127.0.0.1
   everywhere, which `*.localhost` does not on Windows.

*Alternatives considered:* (a) plain `localhost:5173` — no shared cookie parent with
`rtus-seh.${DEV_DOMAIN}`, so the subscription 401s with nothing in either log to explain it;
(b) a per-app `dispatch.${DEV_DOMAIN}:5173` hostname — solves the cookie, breaks CORS, and
buys nothing. Note `apps/dispatch-web/vite.config.ts` currently ships **no** `allowedHosts`
(the baseline removed it as moot on plain localhost); serving on a non-localhost host with
`vite dev --host` means restoring the scaffold's
`allowedHosts: ['127.0.0.1.nip.io', '.127.0.0.1.nip.io']`, or the dev server host-checks the
request.

**D11 — `last_contact` and `position_at` move independently, and the existing UPDATE
changes.** `dispatch-units-crud` D4 made every write bump `last_contact`; this change adds
`position_at`, the time the *fix* was taken. They are not the same event — an operator
editing a unit's radio channel is a contact, not a position report — and the roster spec
requires them to be independent. So the existing UPDATE keeps bumping `last_contact` and
must **not** touch `position_at`, which changes only when a write supplies a position.
*Alternative considered:* leaving `position_at` to follow `last_contact`. Rejected: it would
make every edit look like a fresh GPS fix, and the map's "fix age" would become a lie. This
closes what an earlier draft left as an open question — it is a change to shipped behaviour
and so belongs in the task list, not in an apply-time decision.

## Risks / Trade-offs

**R1 — The workshop's "one container, no auth" promise ends.** The stack goes from one
container to fifteen (fourteen platform services plus the dispatch PostgreSQL), several
pulled from `ghcr.io/mssfoobar`, and every attendee now needs a working login before they
see a unit. → The reproducibility gate in `tasks.md` proves a cold `down -v` / `up -d`
converges, `SETUP.md` and `README.md` are rewritten in the same change rather than left
stale, and the seeded accounts are documented where the old curl example lived. This is the
single largest cost of the change and it is worth re-confirming before apply.

**R2 — `WORKSHOP.md`'s bare `curl` stops working.** The 501 stubs are now behind a bearer,
so the documented `curl -i -X POST http://localhost:8081/v1/units/FU-101/assignment` answers
401 before it ever reaches the 501. → `WORKSHOP.md` gains a one-line token fetch (password
grant against the bundled `web` client, **with `scope=openid`**) that its curl example
reuses. The three exercises' acceptance criteria are otherwise untouched.

**R3 — The role→permission matrix is duplicated in Go, TypeScript and `roles.yaml`.** Three
copies drift. → Keep the matrix to two roles and one permission boundary (read vs write),
name the file each copy mirrors in a comment, and make the verification step decode a real
token rather than asserting the map in isolation. If it grows past a handful of roles, that
is the signal to move to AAS `/evaluate`.

**R4 — A projection can stay pending longer than one request.** The worker carries the
operator's token (D2a), so a delivery that outlives that token waits for the next
authenticated request from the same tenant to supply a fresh one. A tenant with no traffic
holds its backlog. → Acceptable here: the workshop has one tenant and an operator present by
definition. The unit row remains the source of truth (D7), the map's not-shown count is
computed from unit data rather than entity data so the page never under-reports the roster,
and a pending row is visible in the outbox table. If this ever needs to hold without an
operator, the answer is a scheduled job holding its own credential — a different design, not
a patch to this one.

**R5 — Cesium is a heavy dependency on a workshop laptop, and the skills disagree about the
alternative.** `@cesium/engine` plus `@cesium/widgets` add a large install and a WebGL
requirement. The two platform sources conflict on whether there is a lighter option:
`aoh-knowledge/services/gis.md` says "`MapLibreEngineProvider` ships as an empty stub —
picking MapLibre is not currently an option", while `aoh-gis-integration/references/components.md`
lists it as a usable "2D engine. Use instead of Cesium for a pure-2D map." → Proceed with
Cesium, which both sources agree works, in a 2D scene with Ion disabled so no external asset
quota is consumed. Resolve the conflict at apply time by trying `MapLibreEngineProvider`
against the installed SDK; if it works it is strictly better here, and the finding is worth
reporting upstream either way.

**R6 — A black map canvas.** Cesium fetches its workers and textures at runtime; without the
asset-copy step the canvas renders entirely black with only 404s in the console to explain
it. → The vite plugin runs on both `buildStart` and `configureServer` so dev and build are
covered, and the specs assert "no 404 under the Cesium base URL" as an observable outcome
rather than trusting the plugin.

**R7 — `@mssfoobar` image pulls.** The npm token checked into `.npmrc` covers packages, not
`ghcr.io` container images. → Confirm image pull access (or a preloaded image tarball, as
the repo already does for the devcontainer) before the workshop; this is a prerequisite of
apply, not a discovery for the day.

**R8 — The `realm-import.json` edit is skip-if-exists.** Keycloak's `start --import-realm`
imports only into an empty database, so adding the viewer user to an already-running stack
silently does nothing — sign-in then fails for an account the artifacts say exists, with no
error at import time to explain it. → The compose task pairs the edit with a project-wide
`down -v` and `up -d`, **not** `down -v iams-db`: Compose does not treat `-v` as a
per-service volume wipe when a service name is given, and podman-compose's `down` does not
accept service arguments at all, so the per-service form may silently do nothing — which is
the exact failure this risk is about. The reproducibility gate tears the whole stack anyway.

**R9 — The change is authored with tooling the workshop image does not carry.** Two tasks
need binaries that are absent from `.devcontainer/Dockerfile` (`node:24-bookworm-slim` plus
`ca-certificates curl git procps`): `aoh-compose`'s `bootstrap.py` needs **python3** — the
same gap that `apps/dispatch-svc/AGENTS.md` records as why that service was hand-written —
and there is no `golangci-lint` binary or `.golangci.yml` anywhere in this repo. →
Neither blocks an attendee: `compose/` is generated **once by the change author** and
committed, and attendees only consume it. The task list says so explicitly and uses the
repo's actual configured Go lint (`go vet`, via `pnpm lint`) rather than a linter nobody
here has. If the author also lacks python3, the fallback is to run `bootstrap.py` wherever
python3 exists and commit the output — it writes files, it does not need the stack.

## Migration Plan

1. Edit `realm-import.json` (the viewer seed user) **before** the first `up -d`, or after it
   with a project-wide `down -v` — the import is skip-if-exists (R8). Then bring the platform
   stack up (`aoh-compose` output), let `iams-init` then `project-aas-init` complete, and
   confirm that tokens for both seeded accounts carry the expected `active_tenant.roles`
   (`dispatch-dispatcher` for one, `dispatch-viewer` only for the other).
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
   Note this is separate from `/` itself: the scaffold ships no root `+page.svelte`, so
   dropping the baseline's `/` → `/units` redirect without adding a root route would leave
   `/` a 404 rather than a sign-in redirect. The task list adds the root redirect explicitly.
2. **Does the workshop need moving units?** With positions seeded and never updated, the map
   is live in mechanism but static in appearance, and the RTUS path is never visibly
   exercised. A tiny position-nudging loop (behind a flag, off by default) would make the
   live feed self-evident in a demo. Deliberately out of scope here — raise it as its own
   change if the deck needs movement.
3. **Is a second tenant worth standing up?** The tenant-scoping requirements are specified
   against a stack that runs exactly one tenant (`development`). The task list makes them
   provable by inserting a second tenant's row directly with `psql` and asserting the API
   never returns it — which tests the predicate without a second identity. What that cannot
   reach is the cross-tenant *GIS* read, so no such scenario is specified. Standing up a
   second AAS tenant and operator would close that gap and roughly double the seed surface;
   not worth it for a workshop, worth revisiting if this design is reused.

Questions this design closed rather than deferred: which realm users get which role (D2 —
the realm seeds only one interactive user, so the change adds a viewer to
`realm-import.json`) and whether `last_contact` still bumps on every write (D11 — yes, and
`position_at` moves independently, which means the existing UPDATE changes).
