## Context

The console is an `aoh-web-init` scaffold with its auth layer amputated
(`baseline-dispatch-console` D1) and a hand-written Go service behind it
(`dispatch-units-service` D1). Reads happen in a SvelteKit server `load`, writes in form
actions; the browser only ever talks to its own origin. `UBIQUITOUS_LANGUAGE.md` has mapped
our *field unit* onto a GIS `geo-entity` since the baseline and recorded that the mapping
becomes real *"once units carry positions. They do not today."* They still do not: this
change brings the map module in and leaves that mapping for the exercise to realise.

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
   browser's SSE stream by resolving the SDS session cookie. That dependency chain is why
   the map cannot be added before IAMS even though this change puts no data on it.

The consumers are workshop attendees on laptops, running the two apps natively against
composed-up infrastructure. Nothing here targets production.

## Goals / Non-Goals

**Goals:**

- Restore the console's authentication layer whole, on the platform's terms: OIDC/PKCE,
  SDS-held tokens, AAS-owned roles.
- Give `dispatch-svc` a real caller — tenant scoping and audit identity from the token
  instead of the `'workshop'` / `'system'` placeholders.
- Bring the GIS map **module** into the app — mounted, themed, auth-gated, Cesium assets
  served, live feed connected — and stop there. Putting dispatch data on it is deliberately
  left as a workshop exercise, which is the point: integrating a platform module against a
  running stack is the skill the workshop teaches, and it is a far better exercise when the
  surrounding infrastructure is already proven to work.
- Keep the change reversible in review: every new behaviour is additive to existing code
  paths, and the console's data-flow architecture (server-side reads, form-action writes)
  is untouched.

**Non-Goals — the first of these is the workshop exercise this change exists to set up:**

- **Field units on the map.** No position on the unit model, no `geo-entity` projection, no
  entity layer, no map↔console selection. `gis-service` and RTUS run, and the map is
  connected to them, but nothing writes and nothing is drawn.
- GIS bookmarks, drawing, measurement, geofences, or layer management beyond what ships.
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
| `rtus-pms` | Pub/sub control plane; holds the `gis` JSON map. Nothing publishes to it in this change | Platform — added by `aoh-compose` |
| `rtus-seh` | SSE delivery to the browser; authorises the subscription from the `web_auth_session_id` cookie via SDS | Platform — added by `aoh-compose` |
| `gis-db` | Geospatial PostgreSQL | Platform — added by `aoh-compose` |
| `gis-service` | Geo-entity REST API. **Runs but is not called by this change** — it is here so the workshop exercise that wires field units into it has a working backend to integrate against | Platform — added by `aoh-compose` |
| `otel-collector` | OTLP gateway. Neither app exports to it today — `dispatch-svc` has no OTEL wiring at all and `dispatch-web`'s `instrumentation.server.ts` is a no-op until `OTEL_EXPORTER_OTLP_ENDPOINT` is set. It arrives with the stack; wiring it is not in this change | Platform — added by `aoh-compose` |
| `postgres` (dispatch) | `dispatch-svc`'s own database — gains the seed-marker table | Existing — modified by this change |
| `dispatch-svc` | Go field-unit service; runs natively (`go run`), not composed | Existing — modified by this change |
| `dispatch-web` | SvelteKit console; runs natively (`pnpm dev`), not composed | Existing — modified by this change |

`rtus-pms` and `rtus-seh` must share `rtus.clustername` (`aoh_rtus` in the shipped
template) or SEH forms its own one-node Hazelcast cluster and every subscription silently
receives nothing.

## API surface

### Exposed by `dispatch-svc` (this change modifies the auth posture; it adds no route)

| Method | Path | Auth posture | Description | Fronting gateway route |
|---|---|---|---|---|
| GET | `/v1/units` | BearerAuth, role `dispatch-viewer` or `dispatch-dispatcher` | List the caller's tenant's units. **Not side-effect-free**: a dispatcher's first request against an unseeded tenant seeds it before answering (D7) | n/a — see D5 |
| GET | `/v1/units/{unit_code}` | BearerAuth, role `dispatch-viewer` or `dispatch-dispatcher` | One unit in the caller's tenant; 404 for another tenant's | n/a — see D5 |
| POST | `/v1/units` | BearerAuth, role `dispatch-dispatcher` | Create; `tenant_id` / `created_by` now from the token | n/a — see D5 |
| PUT | `/v1/units/{unit_code}` | BearerAuth, role `dispatch-dispatcher` | Replace; `occ_lock` semantics unchanged | n/a — see D5 |
| DELETE | `/v1/units/{unit_code}` | BearerAuth, role `dispatch-dispatcher` | Delete | n/a — see D5 |
| POST · DELETE | `/v1/units/{unit_code}/assignment` | BearerAuth, role `dispatch-dispatcher` | Workshop exercise 1 stub. Still answers 501 — but now only to an authorised caller | n/a — see D5 |
| GET | `/v1/units/{unit_code}/events` | BearerAuth, role `dispatch-viewer` or `dispatch-dispatcher` | Workshop exercise 2 stub. Still 501 | n/a — see D5 |
| PUT | `/v1/units/{unit_code}/crew` | BearerAuth, role `dispatch-dispatcher` | Workshop exercise 3 stub. Still 501 | n/a — see D5 |
| GET | `/livez` | Unauthenticated | Liveness — unchanged | n/a |
| GET | `/readyz` | Unauthenticated | Readiness — unchanged; still fails when the database is unreachable | n/a |

The stub rows cover four route/method pairs across three paths, and they are listed because
this change alters their posture even though it does not implement them: they are mounted today and answer 501 to anyone, and after this change an
unauthenticated caller gets 401 before ever reaching the 501. That is what breaks
`WORKSHOP.md`'s `curl` (R2). Unchanged from `dispatch-units-crud`:
`PATCH /v1/units/{unit_code}` and collection-level `PUT` / `DELETE` stay unmounted and
answer 405. Every 2xx body remains the AOH success
envelope (`data`, `message`, `sent_at`); errors remain the AOH error contract.

### Exposed by `dispatch-web`

| Route | Auth posture | Description |
|---|---|---|
| `/(public)/aoh/api/auth/login` · `/callback` · `/refresh` · `/logout` · `/context` · `/context/[value]` | PKCE (the flow itself) | All six OIDC endpoints the `aoh-web-init` scaffold ships, restored together. `context` / `context/[value]` are the scaffold's tenant-switching routes; this change adds no second tenant to switch to, but they return with the layer rather than being selectively omitted |
| `/(private)/aoh/dispatch/units` | Session required | The console, moved from `/units` |
| `/(private)/aoh/dispatch/map` | Session required | The map, `ssr = false` |
| `/livez` · `/readyz` | Unauthenticated | Unchanged |

### Consumed, not exposed

These are `gis-service`'s own endpoints. **This change calls none of them** — they are
recorded so the workshop exercise that projects field units into GIS starts from a named
surface rather than a search. Their paths are `gis-service`'s to define, not ours.

| Method | Path (on `gis-service`) | Called by | Purpose |
|---|---|---|---|
| PUT | `/geoentity` | *(the exercise)* | Upsert a geo-entity mirroring a unit |
| DELETE | `/geoentity/entity_id/{entity_id}` | *(the exercise)* | Remove a deleted unit's entity |
| GET | `/geoentity/entity_id/{entity_id}` | Verification only | Assert the projection landed |
| GET | `/geoentity` | Verification only | Assert a token is accepted by `gis-service` at all |

The browser's only cross-origin call is the SSE subscription to `rtus-seh`, carrying the
session cookie and no token. Nothing in this change calls `gis-service`; the "consumed"
table below is what the workshop exercise will fill in.

## UI / Design System

Two surfaces change. Clickable mockups, each with a state switcher covering every state the
specs name:

- `openspec/changes/dispatch-iams-and-unit-map/design/dispatch-map-mock.html` — the map:
  connected (base tiles, layer panel, live feed established and empty); live feed
  unavailable; and the viewer variant. It draws no field units, because this change puts
  none there.
- `openspec/changes/dispatch-iams-and-unit-map/design/dispatch-console-auth-mock.html` —
  the console's new states: signed in as a dispatcher (write controls present), signed in as
  a viewer (write controls absent), permission denied after a 403, **service unreachable**,
  **empty roster** seen by a viewer and by a dispatcher, and a unit that is unassigned and
  carries no capabilities.

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
`MapBaseLayerProvider` + `MapXyzSourceProvider` (OpenStreetMap tiles), and `MapLayerManager`. No tile client or marker renderer is written by hand.
`MapEntityLayerProvider` / `MapEntityProvider` are deliberately **absent** — they are what
the workshop exercise adds, along with the data behind them. `MapControlPanel` and
`MapBookmarkManager` are left out too; drawing and bookmarks are non-goals.

**Copy and tone.** Sentence case, operational register, no emoji, per `aoh-design`. "Live
updates are unavailable." — not "Oops! We couldn't connect." The permission-denied copy
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
there. Consequence: the Keycloak import must be re-run from empty volumes — carried as R7.

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

**D6 — Native dev runs on the platform's own dev origin, `http://${DEV_DOMAIN}:5173`.**
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

**D7 — The service seeds a tenant the first time a dispatcher asks it for anything.** The
baseline seeds the roster from `0002_seed.up.sql` on service start. That stops working here:
`iams-init` creates the tenant by `POST /admin/tenants` with only `{"name": "development"}`,
so **AAS assigns the id** — which is why `bootstrap.py` has to look it up by name — and a
hardcoded UUID in a checked-in migration is stale after the first `down -v`. Rows inserted by
SQL cannot carry the caller's identity into `created_by` either.

Everything the seed needs — the tenant id, the caller's identity, and the crew and assignment
shapes the write API withholds — is present on an authenticated request and nowhere else. So
that is where the seed happens: **after `BearerAuth`, on the first request from a caller holding
`dispatch-dispatcher`, the service seeds that caller's tenant**, in the same transaction that
records it has done so, and before the triggering request is served. No seed endpoint, no
second binary, no script, and nothing for an attendee to run.

If the seed cannot commit, the triggering request fails with a `DISPATCH_*` 5xx and no marker
row survives, so the next dispatcher request retries. The alternative — log it and serve an
empty roster — would show a dispatcher an empty console with no error, which is
indistinguishable from a tenant nobody has seeded yet and from one whose units were deleted
on purpose.

Two constraints make it safe rather than clever:

- **A marker, not an emptiness check.** The tempting trigger is "this tenant has no units."
  That is a trap: the console ships a working delete action, so a dispatcher who deletes the
  last unit would have the whole roster silently resurrected on their next click, mid-task,
  with nothing to explain it. A `tenant_seed` row written in the same transaction makes the
  rule *seed once per tenant, ever*. `INSERT … ON CONFLICT DO NOTHING` on that row is also
  what makes two simultaneous sign-ins safe: the loser sees the conflict and skips.
- **A dispatcher gate.** Seeding is a write, so it rides only a request from a caller who may
  write — a `dispatch-viewer`'s read must never perform one.

*Doing work on someone's request deserves scrutiny.* Seeding rides a request that did not ask
for it, which is the shape of a lot of bad behaviour. What makes it acceptable here is that it
is a **one-time bootstrap of an empty tenant**, gated on the same role the write would need
anyway, marked so it never repeats, and attributed to the dispatcher who triggered it — which
is true rather than convenient. A mechanism that ran repeatedly, or attributed one operator's
write to another, would not clear that bar.

*Alternatives considered:* (a) an external script writing the API plus raw SQL — needs a
Postgres driver the repo does not have or a `psql` the workshop image does not ship, and must
hand-roll `ON CONFLICT` against `unit_crew`'s unique constraint, the five-column all-or-
nothing assignment CHECK, and the audit columns; four ways to get the seed quietly wrong. (b) `POST /v1/units/seed`
— correct, but it puts a fixture-data route in a production API forever. (c) a `cmd/seed` CLI
— also correct, and the conservative choice, but it is one more entrypoint and one more thing
for an attendee to run. (d) accepting a roster with no crew and no assignment — breaks the
console's assigned-unit scenario, makes both mockups depict impossible data, and removes the
assigned unit workshop Exercise 1 starts from.

**The cost, stated plainly:** on a fresh stack a `dispatch-viewer` who signs in before any
dispatcher sees an **empty roster**, and the workshop's viewer account exists precisely to
demo the read-only role. Two things contain it: the trigger is any dispatcher request, not
specifically a login, so the roster exists from the moment a dispatcher touches the service;
and the console's empty state says so — "No units yet. A dispatcher signing in will populate
the roster." — rather than rendering a bare empty list. There is deliberately **no** fallback
emptiness check layered on top: one trigger, one marker.

Consequence: the repo's "migrations and seed data apply themselves when the service starts"
story changes; a migration must delete the pre-auth `'workshop'`-tenant rows that
`0002_seed.up.sql` inserts, or they linger invisibly; and `scripts/dev.mjs`, the root
`package.json` scripts, `README.md`, `SETUP.md`, both app READMEs, `WORKSHOP.md` and the
slides deck all describe a start and reset flow that no longer holds.

## Risks / Trade-offs

**R1 — The workshop's "one container, no auth" promise ends.** The stack goes from one
container to sixteen (fifteen platform services plus the dispatch PostgreSQL), several
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

**R4 — Cesium is a heavy dependency on a workshop laptop, and the skills disagree about the
alternative.** `@cesium/engine` plus `@cesium/widgets` add a large install and a WebGL
requirement. The two platform sources conflict on whether there is a lighter option:
`aoh-knowledge/services/gis.md` says "`MapLibreEngineProvider` ships as an empty stub —
picking MapLibre is not currently an option", while `aoh-gis-integration/references/components.md`
lists it as a usable "2D engine. Use instead of Cesium for a pure-2D map." → Proceed with
Cesium, which both sources agree works, in a 2D scene with Ion disabled so no external asset
quota is consumed. Resolve the conflict at apply time by trying `MapLibreEngineProvider`
against the installed SDK; if it works it is strictly better here, and the finding is worth
reporting upstream either way.

**R5 — A black map canvas.** Cesium fetches its workers and textures at runtime; without the
asset-copy step the canvas renders entirely black with only 404s in the console to explain
it. → The vite plugin runs on both `buildStart` and `configureServer` so dev and build are
covered, and the specs assert "no 404 under the Cesium base URL" as an observable outcome
rather than trusting the plugin.

**R6 — `@mssfoobar` image pulls, and the offline bundle.** The npm token checked into
`.npmrc` covers npm packages, not `ghcr.io` container images, so whoever applies this change
needs pull access for `iams-*`, `rtus-*` and `gis-service`. → Confirm that access before
starting, not at task 1.8.

There is a second, larger version of this that is **deliberately out of scope here**:
`SETUP.md` says the workshop network has no internet and ships a preloaded
`ai-day-workshop-images.tar.gz` carrying the workshop image and `postgres:16-alpine`. A
sixteen-service stack needs roughly ten more images in that bundle, and the bundle's size
grows accordingly. That is workshop-day logistics, owned by whoever runs the session, not by
this change — recorded here once so the decision is visible rather than discovered, and
deliberately left without a task.

**R7 — The `realm-import.json` edit is skip-if-exists.** Keycloak's `start --import-realm`
imports only into an empty database, so adding the viewer user to an already-running stack
silently does nothing — sign-in then fails for an account the artifacts say exists, with no
error at import time to explain it. → The compose task pairs the edit with a project-wide
`down -v` and `up -d`, **not** `down -v iams-db`: Compose does not treat `-v` as a
per-service volume wipe when a service name is given, and podman-compose's `down` does not
accept service arguments at all, so the per-service form may silently do nothing — which is
the exact failure this risk is about. The reproducibility gate tears the whole stack anyway.

**R8 — One authoring step needs tooling the workshop image does not carry.**
`aoh-compose`'s `bootstrap.py` needs **python3**, which is absent from
`.devcontainer/Dockerfile` (`node:24-bookworm-slim` plus `ca-certificates curl git procps`)
— the same gap `apps/dispatch-svc/AGENTS.md` records as why that service was hand-written.
→ It does not reach attendees: `compose/` is generated **once by the change author** and
committed, and attendees only consume it. If the author also lacks python3, run
`bootstrap.py` wherever python3 exists and commit the output — it writes files and does not
need the stack. Everything that runs against a live stack stays on Node, which the image
does have: the token fetch and claim decode in the seed verification, and both repo-root
scripts. The same audit removed `golangci-lint` from the task list — there is no
`.golangci.yml` or binary anywhere in this repo, and `go vet` is what `pnpm lint` actually
runs.

## Migration Plan

1. Edit `realm-import.json` (the viewer seed user) **before** the first `up -d`, or after it
   with a project-wide `down -v` — the import is skip-if-exists (R7). Then bring the platform
   stack up (`aoh-compose` output), let `iams-init` then `project-aas-init` complete, and
   confirm that tokens for both seeded accounts carry the expected `active_tenant.roles`
   (`dispatch-dispatcher` for one, `dispatch-viewer` only for the other).
2. Apply the `dispatch-svc` migrations: the seed-marker table (additive), then the removal of
   the pre-auth seeded rows. That removal is the one destructive step, and it is deliberate: those rows
   carry the placeholder tenant, so after this change no token can see them and they would
   linger invisibly.
3. Deploy `dispatch-svc` with bearer auth on. Anything still calling it unauthenticated —
   including a stale `pnpm dev` console — starts receiving 401, which is the intended signal.
4. Sign in as a dispatcher, or make any authenticated dispatcher request. That is what seeds
   the tenant (D7). There is no seed command to run.
5. Deploy `dispatch-web` with the restored auth layer and the map.

**Rollback:** revert both apps. The marker migration is additive, so a rolled-back binary
ignores it without error. The pre-auth row deletion is not reversible in
place — recover by tearing the stack (`compose down -v`), which discards the `tenant_seed`
marker along with the volume, so the roster reseeds itself on the next dispatcher request.
There is no seed command to run, here or anywhere. Note `pnpm reset-db` is **not** that command any
more: once the dispatch database joins the generated compose project it tears the whole
sixteen-service stack, so the root scripts have to be rewritten with this change. `gis-service` holds no data this change wrote, so
there is nothing to clean up there.

## Open Questions

1. **Should `/units` redirect to `/aoh/dispatch/units`?** A 301 would spare anyone with a
   bookmark or a slide deck pointing at the old path. Against: it keeps a route alive that
   the convention says should not exist, and the console's audience is one workshop cohort.
   Currently specified as *not* serving the console; flip it if the slides reference `/units`.
   Note this is separate from `/` itself: the scaffold ships no root `+page.svelte`, so
   dropping the baseline's `/` → `/units` redirect without adding a root route would leave
   `/` a 404 rather than a sign-in redirect. The task list adds the root redirect explicitly.
2. **How much should the exercise brief spell out?** Putting field units on the map is now
   a workshop exercise, and this change has already worked out most of its hard parts —
   `entity_id` = `unit_code`, `entity_type` `track`, `kind` as the grouping, the outbox
   pattern, and the fact that a client-credentials token carries no `active_tenant` claim so
   the projection must ride an operator's bearer. How much of that belongs in the brief and
   how much an attendee should rediscover is a teaching decision, not a technical one. The
   existing three exercises give the shape: a user story, acceptance criteria, pointers to
   the code to copy, and an explicit out-of-scope list.
3. **Is a second tenant worth standing up?** The tenant-scoping requirements are specified
   against a stack that runs exactly one tenant (`development`). The task list makes them
   provable by inserting a second tenant's row directly with `psql` and asserting the API
   never returns it — which tests the predicate without a second identity. Standing up a
   second AAS tenant and operator would be more faithful and roughly double the seed surface;
   not worth it for a workshop, worth revisiting if this design is reused.

A question this design closed rather than deferred: which realm users get which role (D2 —
the realm seeds only one interactive user, so the change adds a viewer to
`realm-import.json`).
