# Lint

Mechanical pre-apply lint over `proposal.md`, `design.md`, `specs/**/*.md` and
`tasks.md` for `dispatch-iams-and-unit-map`. Every item below is **PASS**, **FAIL**, or
**N/A** with a one-line reason.

## 1. Authorization model

- [x] **PASS** — `design.md`'s Runtime dependencies table lists `iams-keycloak` and
      `iams-aas` as separate rows with distinct roles ("OIDC provider; hosts the `aoh`
      realm…" / "Authorization control plane; owns the … tenant roles"), never bundled
      as "IAMS".
- [x] **PASS** — No application-level concern is attributed to Keycloak. Every occurrence
      of "realm role" / "Keycloak realm roles" across the artifacts is a **prohibition**:
      `specs/dispatch-access-control/spec.md` ("They SHALL NOT be Keycloak realm roles",
      "SHALL NOT add a realm role"), `tasks.md:25` ("Add **no** realm role"),
      `design.md` D2, `proposal.md` ("not as Keycloak realm roles"). The one OIDC client
      this change registers is **confidential**, for a server-to-server service account —
      not a per-app *web* client; `design.md` D2 cites
      `aoh-knowledge/services/iams.md`'s rule that a confidential backend is exactly the
      case that warrants one, and the console reuses the bundled `web` PKCE client.
- [x] **PASS** — No scenario asserts the JWT carries `active_tenant.permissions`. The only
      spec mention is the negative assertion in
      `specs/dispatch-access-control/spec.md` ("it references no
      `active_tenant.permissions` claim"). Permission-level gating is specified as an
      in-service role→permission projection over `active_tenant.roles`
      (`design.md` D3, `tasks.md` 3.5 / 4.5), which is option (b) of the check.

## 2. Session storage

- [x] **PASS** — The change modifies a SvelteKit surface, and `design.md`'s Runtime
      dependencies lists both `sds-server` and `valkey`.
- [x] **PASS** — No artifact proposes browser-side token storage as the target posture.
      `design.md` D4 names the cookie-only fallback only to reject it, and
      `specs/dispatch-access-control/spec.md` requires "no cookie, `localStorage` entry, or
      server-rendered payload contains a JWT".
- [x] **PASS** — Option (a): the platform default is kept.
      `tasks.md` 4.14 sets `PUBLIC_COOKIE_PREFIX=web` in `.env.development` /
      `.env.template`, `tasks.md` 1.4 confirms `rtus-seh`'s seeded
      `rtus.session-id.cookienames` already carries `web_auth_session_id` and explicitly
      leaves `compose/rtus/compose.yml` untouched, and
      `specs/dispatch-access-control/spec.md` asserts the match.

## 2a. Real-time subscriptions

- [x] **PASS** — The map's live feed is owned by `@mssfoobar/gis-web-sdk`'s `Map`
      component, which subscribes through `@mssfoobar/sse-client`. No feature code wraps
      `EventSource`: `specs/dispatch-map/spec.md` states "Feature code SHALL NOT hand-roll
      an `EventSource`; the SDK's `@mssfoobar/sse-client`-based subscription is the only
      live-update path", and `tasks.md` 4.11 composes the SDK components rather than a
      subscription of its own.
- [x] **PASS** — Holds by construction: the map page does **not** combine an initial
      `GET /list` with the live feed, so no fetch/SSE race and no prepend site exists.
      `design.md` D9 makes this an explicit decision ("The map has one source of entity
      state"), `specs/dispatch-map/spec.md` asserts it observably ("entity state comes only
      from the SDK's subscription, and the page issues no separate entity list request"),
      and `tasks.md` 4.11 forbids fetching an entity list in `load`. The console's
      server-side roster read is unit data from `dispatch-svc`, not entity data from the
      `gis` map, so the two never merge into one list.

## 3. API contract

- [x] **PASS** — Every endpoint this change exposes uses `/v{N}/<resource>`:
      `/v1/units`, `/v1/units/{unit_code}`, plus root-mounted `/livez` and `/readyz`.
      `grep -rn '/api' specs design.md proposal.md tasks.md` returns nothing. The
      `/geoentity` paths in `design.md`'s "Consumed, not exposed" table are
      `gis-service`'s own published surface, called server-to-server; they are not
      endpoints this change defines, and the table says so.
- [x] **N/A** — No fronting gateway proxy route exists. `design.md` D5 decides against
      restoring `gateway.config.ts` and `(private)/aoh/gateway/[...path]`: this change
      introduces no browser-side call to `dispatch-svc` (reads stay in a server `load`,
      writes in form actions), and `aoh-conventions/web.md` forbids routing a server
      `load` through the proxy anyway. `tasks.md` 4.3 states the exclusion explicitly so
      the implementer does not restore it by reflex.
- [x] **PASS** — No scenario asserts a forbidden shape. Success assertions use the AOH
      envelope (`data` / `message` / `sent_at`), inherited unchanged from
      `dispatch-units-service`'s "Responses use the AOH success envelope". Failure
      assertions use the AOH error contract's `errorCode` / `details`, which
      `aoh-conventions/api.md` designates for new and migrated paths and which this repo
      already uses (`DISPATCH_UNIT_STALE`, `DISPATCH_NOT_IMPLEMENTED`). No
      `{ error: "..." }`, `{ errors: { field: "msg" } }` or `{ status: "..." }` appears.
- [x] **PASS** — `/livez` and `/readyz` appear in `design.md`'s API surface, mounted at
      root and marked Unauthenticated. This change adds no new backend service, but the
      probes are listed anyway because restoring auth is exactly what could sweep them
      behind it — `specs/dispatch-access-control/spec.md`'s "Health probes stay outside
      the authenticated surface" asserts both apps' probes stay open.

## 4. UI surfaces

- [x] **PASS** — Two mockups exist, each with a state switcher covering every state its
      specs name:
      `design/dispatch-map-mock.html` (live feed · unit selected · live feed unavailable ·
      no positioned units · viewer read-only, plus a light/dark toggle) and
      `design/dispatch-console-auth-mock.html` (dispatcher · viewer · permission denied ·
      unit with position · unit without position).
- [x] **PASS** — `design.md`'s UI / Design System section enumerates the `@mssfoobar/ui`
      primitives the surfaces compose from (`Button`, `Card`/`CardHeader`/`CardContent`/
      `CardTitle`, `Badge`, `Separator`, `ScrollArea`, `Sheet`, `AlertDialog`, `Select`,
      `Input`, `Toaster`, `Sidebar`, `Navbar`) and the `@mssfoobar/gis-web-sdk`
      components the map composes from. `specs/dispatch-map/spec.md` additionally names
      the mockup path.
- [x] **PASS** — Both gated page routes get option (a) **and** (b):
      `tasks.md` 4.4 adds a `nav.ts` `NavItem` per route gated on holding either
      application role, and `tasks.md` 4.5 plus
      `design/dispatch-console-auth-mock.html`'s "Permission denied" state specify the
      denied card. `specs/dispatch-map/spec.md` and
      `specs/dispatch-access-control/spec.md` assert both behaviours.
- [x] **PASS** — No artifact proposes hand-rolling a shipped primitive. `design.md`'s UI
      section states "Nothing here is hand-rolled — `@mssfoobar/ui` ships every one of
      these", and `tasks.md` 4.1 repeats the prohibition on raw `<button>` / `<input>` /
      hand-rolled `div` equivalents.
- [x] **PASS** — `design.md` mandates `@mssfoobar/gis-web-sdk` for the map and no task
      hand-rolls it: `tasks.md` 4.2 names the SDK's skill in backticks and forbids
      building the surface from raw `@mssfoobar/ui` primitives, and 4.11 composes the SDK
      component tree directly. (Note: the UI-builder register in the tasks template lists
      `aoh-dashboard` only; a map on `gis-web-sdk` is the same category, and `tasks.md`
      4.2 says so rather than leaving the omission to be read as permission to hand-roll.)

## 5. Cross-artifact consistency

- [x] **PASS** — All six capabilities in `proposal.md` have a matching spec file, names
      identical: new — `dispatch-access-control`, `field-unit-geo-projection`,
      `dispatch-map`; modified — `dispatch-console`, `dispatch-units-api`,
      `field-unit-roster`. `ls specs/` returns exactly those six directories.
- [x] **PASS** — Every ADDED and MODIFIED requirement carries at least one
      `#### Scenario:` with exactly four hashes (35 requirement blocks — 34 ADDED/MODIFIED
      plus one REMOVED — and 111 scenarios;
      `grep -rn '^### Scenario' specs/` returns nothing). The single requirement without
      scenarios is the REMOVED block `specs/dispatch-console/spec.md` → "The application
      has no authentication", which carries **Reason** and **Migration** instead, as the
      delta format requires. `openspec validate dispatch-iams-and-unit-map --strict`
      passes.
- [x] **PASS** — Every endpoint in the API surface is exercised:
      `GET /v1/units` → access-control "A request with no token is rejected" / "A valid
      token is accepted", units-api "A list returns only the caller's tenant";
      `GET /v1/units/{unit_code}` → units-api "Another tenant's unit is not found";
      `POST` / `PUT` / `DELETE /v1/units[/{unit_code}]` → access-control "A viewer cannot
      write" / "A dispatcher can write", units-api "Setting a position" / "Clearing a
      position", geo-projection "The entity goes with the unit";
      `/livez` + `/readyz` → access-control "Health probes stay outside the authenticated
      surface"; the auth routes → access-control's sign-in and sign-out scenarios; the two
      page routes → dispatch-console "The console answers on its module route" and
      dispatch-map "The map renders the fleet" / "Unauthenticated access". The consumed
      `gis-service` endpoints are exercised by geo-projection's projection scenarios.
- [x] **PASS** — All 17 rows of `design.md`'s Runtime dependencies table carry an owner:
      fourteen `Platform — added by aoh-compose` (one of them noting `roles.yaml` is owned
      by this change) and three `Existing — modified by this change` (the dispatch
      PostgreSQL, `dispatch-svc`, `dispatch-web`).

## 6. Verification feasibility

- [x] **PASS** — Every THEN asserts something observable from outside: an HTTP status
      (401 / 403 / 404 / 200), a JWT claim (`active_tenant.roles`,
      `active_tenant.tenant_id`), a DB row or constraint (outbox rows, the all-or-nothing
      position CHECK), a `gis-service` response body (`entity_type`, `properties.kind`,
      coordinates), a browser cookie, a network-log absence (no 404 under the Cesium base
      URL), or a DOM state (markers, the denied card, the un-positioned count). The
      handful of inspection-style scenarios ("the map page source is reviewed", "the
      authorization path in either app is reviewed", "the outbox row is inspected") assert
      statically checkable facts about checked-in files, not hidden runtime state, and
      each names the artifact to inspect.
- [x] **PASS** — Token acquisition is named, never hand-minted.
      `specs/dispatch-access-control/spec.md` says "obtained … via the password grant
      against the bundled `web` client on the `aoh` realm" for both operator tokens, and
      the client-credentials grant for the service account. `tasks.md` 2.5 gives the exact
      `curl`, and 5.1 makes the E2E script obtain both operator tokens the same way.

## 7. Tasks structure

- [x] **PASS** — Every section ends with a concrete verification naming exact commands:
      1.8 (`podman compose … up -d` / `ps`, dual form, naming the containers that must be
      healthy and the init containers that must exit 0), 2.5 + 2.6 (`curl` token fetch and
      decode; `GET /geoentity` with the service-account token), 3.14
      (`go build ./... && golangci-lint run ./... && go test ./... -count=1 -race`), 4.15
      (`pnpm build && pnpm check-types && pnpm lint`, plus a `grep` that the stale
      no-authentication claims are gone), 5.4 (the reproducibility gate).
- [x] **PASS** — Every compose invocation is in the dual `podman compose <args>` (or
      `docker compose <args>`) form: `tasks.md` 1.8, 2.4 and 5.4.
- [x] **PASS** — No task runs `compose up` for `dispatch-svc` or `dispatch-web`. Both are
      exercised natively (`go run ./cmd/server`, `pnpm dev`) against composed-up infra, in
      3.13 / 4.14 and in the gate.
- [x] **PASS** — Section order is Compose (1) → Seed (2) → Implement `dispatch-svc` (3) →
      Implement `dispatch-web` (4) → E2E + gate (5). No Scaffold section, correctly: both
      apps already exist and are being extended. Backend precedes frontend, and the
      producer (`dispatch-svc`, which fills GIS) precedes the consumer (`dispatch-web`,
      which renders it). No section exercises infra a later section composes — the AAS
      roles in section 2 depend only on the stack section 1 brings up.

## 8. Skill triggering

- [x] **N/A** — No Scaffold section; both apps already exist. (`aoh-web-init` is
      referenced in `tasks.md` 4.3 as the source of the auth layer's shape, not as a
      scaffolding step.)
- [x] **PASS** — `tasks.md` 1.1, the Compose section's first task, references
      `aoh-compose` in backticks and gives its `bootstrap.py` invocation.
- [x] **PASS** — `tasks.md` 3.1 and 4.1, the first task of each Implement section,
      reference `aoh-conventions` in backticks and name the specific references
      (`references/go.md`, `references/api.md`, `references/database.md`,
      `references/web.md`).
- [x] **PASS** — Both IAMS-authz tasks name `aoh-knowledge` in backticks with the specific
      reference: `tasks.md` 2.1 names `references/services/iams.md` for the `roles.yaml`
      edit, and 1.5 names `references/keycloak-realm-guide.md` for the `realm-import.json`
      edit.
- [x] **PASS** — `tasks.md` 4.1 references `aoh-design` in backticks and names both mockup
      paths in full.
- [x] **PASS** — `tasks.md` 4.2 names `aoh-gis-integration` in backticks as the SDK skill
      for the map surface.

## 9. Native dev env documentation

- [x] **PASS** — Each Implement section's last task before its verification is the env
      block. `tasks.md` 3.13 covers the Go service (`SQL_HOST` / `SQL_PORT` / `SQL_USER` /
      `SQL_PASSWORD` / `SQL_DATABASE_NAME` / `SQL_SCHEMA_NAME` / `SQL_SSL_MODE`,
      `IAMS_KEYCLOAK_HOST` + `IAMS_KEYCLOAK_PORT`, `HTTP_PORT`, `HTTP_ALLOWED_ORIGINS`,
      plus `IAM_REALM`, `GIS_URL` and the client-credentials pair the outbox worker needs).
      `tasks.md` 4.14 covers the SvelteKit app (the OIDC quartet `IAM_URL`,
      `IAM_CLIENT_ID`, `PUBLIC_DOMAIN`, `PUBLIC_COOKIE_PREFIX`; `ORIGIN` matching the dev
      port; `SDS_URL=tcp://127.0.0.1:5333`; and the upstream URLs `DISPATCH_SVC_URL` and
      `PUBLIC_RTUS_SEH_URL`).
- [x] **PASS** — `tasks.md` 1.6 names the ports the override exposes on localhost — **5432**
      for the dispatch PostgreSQL and **5333** for `sds-server`'s TCP entrypoint — and
      records them in `compose/compose.override.sample.yml`, noting the real
      `compose.override.yml` stays gitignored while the contract does not.

## 10. Reproducibility gate

- [x] **PASS** — `tasks.md` 5.4 is the last task of section 5, inside the E2E section, not
      a separate trailing section.
- [x] **PASS** — The gate runs `podman compose -f compose/compose.yml down -v` followed by
      `up -d` (both in dual form), and the task text states why the `-v` is what makes it
      meaningful.
- [x] **PASS** — It restarts both native processes — `cd apps/dispatch-svc && go run
      ./cmd/server &` and `cd apps/dispatch-web && pnpm dev &` — referring to the env
      blocks from 3.13 and 4.14.
- [x] **PASS** — It re-runs `node scripts/e2e-smoke.mjs`, byte-identical to task 5.3's
      command, annotated "SAME command as 5.3".

## Resolution

All items are **PASS** or **N/A**. Two N/A, each with its reason recorded above (no
Scaffold section; no gateway proxy route by decision D5). `openspec validate
dispatch-iams-and-unit-map --strict` passes.

The change is **apply-ready**. Two things are worth a human decision before
`/opsx:apply` rather than during it — they are not lint failures, but they change what
apply is worth:

1. **R1 in `design.md`** — this change ends the workshop's "one container, no
   authentication" property. That was a deliberate baseline decision
   (`baseline-dispatch-console` D1), and reversing it is the single largest cost here.
2. **R8 / task 1.1** — image pull access to `ghcr.io/mssfoobar` for `iams-*`, `rtus-*` and
   `gis-service` is a prerequisite of the very first task. The checked-in `.npmrc` token
   covers npm packages, not container images. Confirm before starting, not at 1.8.

`design.md`'s Open Questions 1–4 are answerable during apply and do not gate it.
