## 1. Compose runtime dependencies

- [ ] 1.1 Generate the platform stack using the `aoh-compose` skill:
      `python3 .claude/skills/aoh-compose/scripts/bootstrap.py --repo-root . --services rtus,gis --custom-service name=dispatch-svc,type=go --custom-service name=dispatch-web,type=web`.
      `iams` (with `iams-aas`, `iams-init`, `project-aas-init`), `sds` (with `valkey`),
      `traefik` and the `otel` gateway come in automatically; `gis` pulls `rtus`
      transitively. Do not hand-write these files.
- [ ] 1.2 Fold the existing dispatch PostgreSQL into the generated layout: move the
      `postgres` service from `compose/compose.yml` into `compose/dispatch/compose.yml`
      (container `dispatch-postgres`, `name: aoh` so it joins the shared network) and add it
      to the regenerated top-level include list. Delete the now-false header comment that
      says there is "deliberately no Traefik, no Keycloak, no iams-aas, no sds-server and no
      valkey".
- [ ] 1.3 Confirm `rtus-pms` and `rtus-seh` both set the same `rtus.clustername`
      (`aoh_rtus` in the shipped template). A mismatch makes rtus-seh form its own one-node
      Hazelcast cluster and every SSE subscription silently receives nothing.
- [ ] 1.4 Confirm `rtus-seh`'s `rtus.session-id.cookienames` includes `web_auth_session_id`
      and leave it alone — `dispatch-web` keeps the platform-default
      `PUBLIC_COOKIE_PREFIX=web`, so no `compose/rtus/compose.yml` edit is needed
      (design.md D4).
- [ ] 1.5 Add the confidential `dispatch-svc` client to
      `compose/iams/keycloak/realm-import.json`: `publicClient: false`,
      `serviceAccountsEnabled: true`, the `openid` scope, and a secret sourced from
      `compose/.env`. Consult the `aoh-knowledge` skill
      (`references/keycloak-realm-guide.md`) first. Add **no** realm role, **no** claim
      mapper and **no** second public client — application roles are AAS roles and belong in
      task 2.2 (design.md D2).
- [ ] 1.6 Document, in `compose/compose.override.sample.yml`, the localhost ports a native
      dev run needs exposed: **5432** for the dispatch PostgreSQL and **5333** for
      `sds-server`'s TCP entrypoint. The real `compose.override.yml` stays gitignored;
      the contract does not.
- [ ] 1.7 Fill `compose/.env` from the template: `DEV_DOMAIN`, `DEV_USER`, `DEV_PASSWORD`
      and the `dispatch-svc` client secret. Pick a `DEV_DOMAIN` under `.localhost` (e.g.
      `aoh.localhost`) so the console can be served from a sibling hostname and the session
      cookie reaches `rtus-seh` (design.md D10).
- [ ] 1.8 Verify: run `podman compose -f compose/compose.yml up -d` (or `docker compose -f compose/compose.yml up -d`),
      then `podman compose -f compose/compose.yml ps` (or `docker compose -f compose/compose.yml ps`)
      shows `traefik`, `iams-keycloak`, `iams-aas`, `sds-server`, `valkey`, `rtus-pms`,
      `rtus-seh`, `gis-service` and `dispatch-postgres` healthy, and `iams-init` and
      `project-aas-init` exited 0.

## 2. Seed / resource creation

- [ ] 2.1 Consult the `aoh-knowledge` skill (`references/services/iams.md` →
      "Project-level AAS bootstrap") before editing any seed artifact. Edit only
      `roles.yaml`: never `bootstrap.py`, never the Dockerfile, never
      `iams-aas-init.postman_collection.json`.
- [ ] 2.2 In `compose/iams/init/project-aas/roles.yaml`, under `tenant: development`, add the
      two application roles `dispatch-viewer` and `dispatch-dispatcher`, and under
      `assignments` give the seeded dispatcher user both and the seeded viewer user
      `dispatch-viewer` only. Keep the file idempotent — `bootstrap.py` is GET-then-create
      per section.
- [ ] 2.3 In the same file, add the `dispatch-svc` client's service-account user
      (`service-account-dispatch-svc`) as a member of the `development` tenant so
      `gis-service` accepts the outbox worker's writes.
- [ ] 2.4 Re-import the realm and re-run the bootstrap, because Keycloak's
      `start --import-realm` is skip-if-exists and task 1.5 edited the import file:
      `podman compose -f compose/compose.yml down -v iams-db` (or `docker compose -f compose/compose.yml down -v iams-db`),
      then `podman compose -f compose/compose.yml up -d` (or `docker compose -f compose/compose.yml up -d`).
      A plain restart or `--force-recreate iams-keycloak` will NOT re-import.
- [ ] 2.5 Verify (roles present): obtain a token for the seeded dispatcher with the password
      grant against the bundled `web` client —
      `curl -s -d 'grant_type=password' -d 'client_id=web' -d "username=${DEV_USER}" -d "password=${DEV_PASSWORD}" "http://iams-keycloak.${DEV_DOMAIN}/realms/aoh/protocol/openid-connect/token"`
      — and decode its payload; `active_tenant.roles` contains `dispatch-dispatcher` and
      `active_tenant.tenant_id` is the `development` tenant's id. Repeat for the viewer user
      and confirm it contains `dispatch-viewer` and not `dispatch-dispatcher`.
- [ ] 2.6 Verify (service account works): obtain a client-credentials token for
      `dispatch-svc` and call `GET /geoentity` on `gis-service` with it; the response status
      is 200, not 401 or 403.

## 3. Implement `dispatch-svc`

- [ ] 3.1 Consult the `aoh-conventions` skill (`references/go.md` for layering and
      `aohhttp`, `references/api.md` for the envelope and `/v{N}` paths,
      `references/database.md` for schema rules) and the `aoh-error-handling` skill for the
      new error codes, before writing code.
- [ ] 3.2 Add `migrations/0003_position_and_outbox.up.sql`: `position_lon`, `position_lat`,
      `position_at` on `{{SCHEMA}}.unit` with an all-or-nothing CHECK and range CHECKs
      (`[-180,180]` / `[-90,90]`); and a `{{SCHEMA}}.gis_outbox` table carrying the AOH
      mandatory columns plus `unit_code`, `intent` (`upsert` / `delete`), `payload jsonb`,
      `attempts`, `last_error`, `delivered_at`. Write against `{{SCHEMA}}`, never a
      hardcoded `dispatch.`.
- [ ] 3.3 Add `migrations/0004_seed_positions.up.sql`: move the seeded units to the
      `development` tenant id and give them coordinates. Keep the
      `ON CONFLICT … DO UPDATE` idempotence — the reproducibility gate depends on it.
- [ ] 3.4 Add bearer authentication: validate the token offline against the `aoh` realm's
      JWKS, reject with 401 on missing / malformed / wrongly-signed / expired, and leave
      `/livez` and `/readyz` unmounted from the middleware. Put `sub`,
      `active_tenant.tenant_id` and `active_tenant.roles` on the request context.
- [ ] 3.5 Add the role→permission projection in `internal/service`: a static map from role
      name to read/write permission that mirrors `roles.yaml`, with a comment naming that
      file as its source. Read `active_tenant.roles` only — **never**
      `active_tenant.permissions`; AAS does not emit it and code that expects it 403s every
      legitimate user. Gate writes on `dispatch-dispatcher`, reads on either role, and
      return 403 with a `DISPATCH_*` code otherwise.
- [ ] 3.6 Thread tenant and identity through service and repo: add
      `WHERE tenant_id = $n` to every query, set `created_by` / `updated_by` from `sub` and
      `tenant_id` from the claim, and ignore any of the three arriving in a request body.
- [ ] 3.7 Add `position` to `domain.Unit` (a `*Position` so an un-positioned unit omits the
      key, exactly as `Assignment` does) and to `domain.UnitInput`, with range validation
      returning 400 and an `aoherr.FieldDetail` per bad field.
- [ ] 3.8 Write the outbox row inside the same transaction as every create / replace /
      delete. The handler must not call `gis-service`; a GIS outage must not fail a unit
      write.
- [ ] 3.9 Add the outbox worker: poll undelivered rows, `PUT /geoentity` (upsert, keyed on
      `entity_id` = `unit_code`, `entity_type` `track`, `properties.kind` `field-unit`) or
      `DELETE /geoentity/entity_id/{unit_code}`, with backoff and at-least-once retry. A
      delete for an entity that is already gone counts as delivered. Authenticate with the
      client-credentials grant for the `dispatch-svc` client, caching the token until it
      nears expiry.
- [ ] 3.10 Extend `internal/config`: `IAMS_KEYCLOAK_HOST`, `IAMS_KEYCLOAK_PORT`,
      `IAM_REALM`, `GIS_URL`, `DISPATCH_SVC_CLIENT_ID`, `DISPATCH_SVC_CLIENT_SECRET`, and
      the worker's poll interval. Keep the existing Viper defaults pattern; fail fast on a
      missing secret.
- [ ] 3.11 Tests, following the existing consumer-declared-interface style (a fake repo, no
      mock framework): the role projection, tenant scoping, position validation and
      clearing, outbox-row-in-same-transaction, and worker idempotence against a stub
      `gis-service`.
- [ ] 3.12 Update `apps/dispatch-svc/AGENTS.md`: the "**No authentication**" bullet is now
      false and is replaced by the bearer + tenant + role posture, and the outbox worker
      joins the load-bearing architecture list.
- [ ] 3.13 Document the env block for a native run (`cd apps/dispatch-svc && go run ./cmd/server`):
      ```sh
      SQL_HOST=localhost SQL_PORT=5432 SQL_USER=dispatch SQL_PASSWORD=dispatch \
      SQL_DATABASE_NAME=dispatch SQL_SCHEMA_NAME=dispatch SQL_SSL_MODE=disable \
      IAMS_KEYCLOAK_HOST=iams-keycloak.${DEV_DOMAIN} IAMS_KEYCLOAK_PORT=80 IAM_REALM=aoh \
      GIS_URL=http://gis.${DEV_DOMAIN} \
      DISPATCH_SVC_CLIENT_ID=dispatch-svc DISPATCH_SVC_CLIENT_SECRET=<from compose/.env> \
      HTTP_PORT=8081 HTTP_ALLOWED_ORIGINS=
      ```
      `HTTP_ALLOWED_ORIGINS` stays empty: the browser still never calls this service
      directly (design.md D5), so no CORS is required.
- [ ] 3.14 Verify: `cd apps/dispatch-svc && go build ./... && golangci-lint run ./... && go test ./... -count=1 -race` all exit 0.

## 4. Implement `dispatch-web`

- [ ] 4.1 Consult the `aoh-conventions` skill (`references/web.md`) and the `aoh-design`
      skill before writing any `.svelte` file. Open both mockups in a browser as visual
      reference: `openspec/changes/dispatch-iams-and-unit-map/design/dispatch-console-auth-mock.html`
      and `openspec/changes/dispatch-iams-and-unit-map/design/dispatch-map-mock.html`.
      Visual primitives come from `@mssfoobar/ui` subpaths — no raw `<button>` /
      `<input>` / hand-rolled `div` equivalents.
- [ ] 4.2 The map surface is built by a platform SDK, so ALSO consult the
      `aoh-gis-integration` skill and compose the map from `@mssfoobar/gis-web-sdk`. Do NOT
      hand-roll a map renderer, tile client, entity layer or marker from raw
      `@mssfoobar/ui` primitives. (The UI-builder list in the tasks template registers
      `aoh-dashboard` only; GIS is the same category for a map surface.)
- [ ] 4.3 Restore the auth layer as one piece, per the `aoh-web-init` scaffold: OIDC
      discovery and the auth handle in `src/hooks.server.ts`,
      `src/lib/aoh/core/provider/auth/`, the SDS client, the
      `(public)/aoh/api/auth/{login,callback,logout,refresh}` routes, the `(private)` group
      and its layout, and `App.Locals` regaining `authResult` in `src/app.d.ts`. Tokens go
      to SDS; the browser gets only `web_auth_session_id`. Do not restore
      `gateway.config.ts` or the gateway proxy — design.md D5.
- [ ] 4.4 Move the console from `src/routes/units/` to
      `src/routes/(private)/aoh/dispatch/units/`, add `src/routes/(private)/aoh/dispatch/map/`,
      and add `nav.ts` with a `NavItem` for each, both gated on holding either application
      role. Point `LOGIN_DESTINATION` at the console's new path and drop the
      `/` → `/units` redirect in `+layout.server.ts`.
- [ ] 4.5 Add the role→permission projection mirroring `roles.yaml` (the TypeScript half of
      task 3.5, same two roles, same source-of-truth comment), read from
      `locals.authResult.claims.active_tenant.roles`. Hide add / edit / delete for a viewer
      and render the permission-denied card on a 403 from the service, exactly as the
      console-auth mockup's "Viewer" and "Permission denied" states show.
- [ ] 4.6 Make `units.server.ts` send `Authorization: Bearer <token from SDS>` on every
      call, decode `position` at the wire boundary next to the existing status guard, and
      map a 401 to a re-authentication redirect and a 403 to the permission-denied state
      rather than the generic service-unavailable state.
- [ ] 4.7 Add the position section to `UnitDetail.svelte` — coordinates, fix time via the
      existing `sinceLabel`, a "No position reported." line when absent, and a **Show on
      map** control — per the mockup's "Unit with position" / "Unit without position"
      states.
- [ ] 4.8 Install the map dependencies as runtime `dependencies`:
      `@mssfoobar/gis-web-sdk@^2.0.0`, `@mssfoobar/auth-sdk`, `@cesium/engine@^25`,
      `@cesium/widgets@^15`. `@mssfoobar/sse-client` and `@mssfoobar/logger@^1.0.5` are
      already present at compatible versions.
- [ ] 4.9 Add the Cesium asset-copy plugin to `vite.config.ts` (copy `Build/Workers`,
      `Source/Assets`, `Source/ThirdParty` from `@cesium/engine` and `Source` from
      `@cesium/widgets` into `static/cesium/`, on both `buildStart` and `configureServer`),
      define `CESIUM_BASE_URL`, and add the `@mssfoobar/*` + Cesium packages to
      `ssr.noExternal`. Add `static/cesium/` to `.gitignore` **and** to
      `eslint.config.js`'s `ignores` — without the lint ignore the copied workers produce
      thousands of errors.
- [ ] 4.10 Mount `<GisProvider>` at the root `+layout.svelte`, fed by the app's existing
      `ThemeProvider` dark-mode store so the map repaints with the theme.
- [ ] 4.11 Build the map page: `+page.ts` exporting `ssr = false`, and `+page.svelte`
      composing `CesiumMapEngineProvider` → `Map` (with `rtus_seh_url`, `rtus_map_name="gis"`,
      `user_id` from the `sub` claim, `tenant_id` from `active_tenant.tenant_id`,
      `batchUpdateInterval={500}`) → `MapBaseLayerProvider` + `MapXyzSourceProvider` (OSM) →
      `MapEntityLayerProvider` + `MapEntityProvider kind="field-unit"` with a call-sign
      marker snippet → `MapLayerManager`. Take the user id from `data.user.sub` and the
      tenant from `data.user.active_tenant.tenant_id` — `data.user.id` / `data.user.tenantId`
      are undefined on this base and silently disable the live feed. Do not fetch an entity
      list in `load`; the SDK's subscription is the only source of entity state
      (design.md D9).
- [ ] 4.12 Add the map's surrounding states from the mockup: the fleet/un-positioned counts
      (computed from unit data, not entity data), the live / not-live badge, the
      "Live positions are unavailable." notice, the no-positioned-units empty state, and the
      selection card with **Open in console**. Wire map→console selection as a SvelteKit
      navigation, not a browser fetch.
- [ ] 4.13 Update the docs that now assert the opposite of reality:
      `apps/dispatch-web/AGENTS.md` (the "🛑 This app has NO authentication" table goes; the
      map, Cesium assets and role gating join the load-bearing list), `README.md` (no-auth
      and one-container claims), `SETUP.md` (the full stack, the seeded accounts, and the
      Windows `*.localhost` hosts entry), `WORKSHOP.md` (the `curl` examples gain a token
      fetch), and `UBIQUITOUS_LANGUAGE.md` (*position* and *fix time* join the term table;
      the *field unit* ↔ `geo-entity` mapping stops being conditional and is corrected —
      `field-unit` is a `kind`, `track` is the `entity_type`).
- [ ] 4.14 Document the env block for a native run (`cd apps/dispatch-web && pnpm dev`),
      added to `.env.development` and `.env.template`:
      ```sh
      ORIGIN=http://dispatch.${DEV_DOMAIN}:5173
      IAM_URL=http://iams-keycloak.${DEV_DOMAIN}/realms/aoh
      IAM_CLIENT_ID=web
      PUBLIC_DOMAIN=${DEV_DOMAIN}
      PUBLIC_COOKIE_PREFIX=web
      SDS_URL=tcp://127.0.0.1:5333
      DISPATCH_SVC_URL=http://localhost:8081
      PUBLIC_RTUS_SEH_URL=http://rtus-seh.${DEV_DOMAIN}
      LOGIN_DESTINATION=/aoh/dispatch/units
      FRAME_ANCESTORS="'self'"
      X_FRAME_OPTIONS=SAMEORIGIN
      PUBLIC_STATIC_BUILD_VERSION=dev
      ```
      `ORIGIN` must be the `${DEV_DOMAIN}` hostname, not `localhost`, or the session cookie
      will not travel to `rtus-seh` and every SSE connect 401s silently (design.md D10).
- [ ] 4.15 Verify: `cd apps/dispatch-web && pnpm build && pnpm check-types && pnpm lint` all
      exit 0, and `grep -ri "no authentication" apps/dispatch-web/AGENTS.md README.md` returns
      nothing.

## 5. End-to-end verification + reproducibility gate

- [ ] 5.1 Write `scripts/e2e-smoke.mjs` — Node, matching the repo's existing `scripts/*.mjs`
      convention and its Node 24 prerequisite. It SHALL, with no manual step: fetch a
      dispatcher token and a viewer token by password grant against the bundled `web`
      client; assert `GET /v1/units` is 401 with no token, 200 with the dispatcher token, and
      that the seeded roster comes back with positions; assert `POST /v1/units` is 403 with
      the viewer token and 201 with the dispatcher token; poll `gis-service`
      `GET /geoentity/entity_id/{unit_code}` until the new unit's entity appears and assert
      its `entity_type` is `track` and its `properties.kind` is `field-unit`; move the unit
      with a `PUT` and assert the entity's coordinates follow; delete the unit and assert the
      entity goes; and exit non-zero with a named assertion on any failure.
- [ ] 5.2 Walk the browser states the script cannot assert, against the running stack: sign
      in as the dispatcher and land on `/aoh/dispatch/units`; open `/aoh/dispatch/map` and
      confirm the fleet renders with **no 404 under `/cesium/`** in the network log; move a
      unit with `PUT` in another terminal and watch the marker move without a reload; click a
      marker and confirm it opens in the console; sign out, sign in as the viewer, and
      confirm the add / edit / delete controls are absent and the map still renders.
- [ ] 5.3 Native dev E2E: `node scripts/e2e-smoke.mjs` exits 0.
- [ ] 5.4 **Reproducibility gate** — tear and rebuild infra, restart the native dev processes
      for both apps using the env blocks from 3.13 and 4.14, and re-run the SAME E2E command
      from 5.3:
      ```bash
      podman compose -f compose/compose.yml down -v      # or: docker compose -f compose/compose.yml down -v
      podman compose -f compose/compose.yml up -d        # or: docker compose -f compose/compose.yml up -d
      # wait for healthy, and for iams-init + project-aas-init to exit 0:
      podman compose -f compose/compose.yml ps           # or: docker compose -f compose/compose.yml ps
      cd apps/dispatch-svc && go run ./cmd/server &      # native, env block from 3.13
      cd apps/dispatch-web && pnpm dev &                 # native, env block from 4.14
      node scripts/e2e-smoke.mjs                         # SAME command as 5.3
      ```
      The `-v` is what makes this meaningful: it discards the Keycloak, AAS, GIS, RTUS and
      dispatch volumes, so the realm import, the AAS roles, the seeded units and their
      geo-entities must all reconverge from checked-in artifacts alone.
