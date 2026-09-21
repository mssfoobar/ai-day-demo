## 1. Compose runtime dependencies

> `bootstrap.py` needs **python3**, which the workshop devcontainer image does not carry
> (`node:24-bookworm-slim` plus `ca-certificates curl git procps`) — the same gap
> `apps/dispatch-svc/AGENTS.md` records as why that service was hand-written. This is a
> one-time authoring step: `compose/` is generated **once, by whoever applies this change**,
> and committed. Attendees only consume the committed output and never run the script. If
> the author's machine lacks python3, run it anywhere python3 exists and commit the result —
> it writes files and does not need the stack.

- [ ] 1.1 Generate the platform stack using the `aoh-compose` skill:
      `python3 .claude/skills/aoh-compose/scripts/bootstrap.py --repo-root . --services rtus,gis --custom-service name=dispatch-svc,type=go --custom-service name=dispatch-web,type=web`.
      `iams` (with `iams-aas`, `iams-web`, `iams-init`, `project-aas-init`), `sds` (with
      `valkey`), `traefik` and the `otel-collector` gateway come in automatically; `gis`
      pulls `rtus` transitively. Do not hand-write these files.
- [ ] 1.2 Fold the existing dispatch PostgreSQL into the generated layout: move the
      `postgres` service from `compose/compose.yml` into `compose/dispatch/compose.yml`
      (container `dispatch-postgres`, `name: aoh` so it joins the shared network) and add it
      to the regenerated top-level include list. Delete the now-false header comment that
      says there is "deliberately no Traefik, no Keycloak, no iams-aas, no sds-server and no
      valkey".
- [ ] 1.3 Confirm `rtus-pms` and `rtus-seh` both set the same `rtus.clustername`
      (`aoh_rtus` in the shipped template). A mismatch makes rtus-seh form its own one-node
      Hazelcast cluster and every SSE subscription silently receives nothing.
- [ ] 1.4 Leave `compose/rtus/compose.yml` **unedited**, and verify why that is safe:
      `rtus.session-id.cookienames` already carries `web_auth_session_id` (which
      `PUBLIC_COOKIE_PREFIX=web` produces), and rtus-seh's Traefik CORS middleware already
      lists `http://${DEV_DOMAIN}:5173` — the origin task 4.14 serves the console on. Serving
      it anywhere else (e.g. `dispatch.${DEV_DOMAIN}:5173`) would require adding that origin
      to `accesscontrolalloworiginlist` here, and the credentialed SSE request would be
      blocked until you did.
- [ ] 1.5 Add **one seed user** to `compose/iams/keycloak/realm-import.json`'s `users` array
      to serve as the viewer account, following the `aoh-knowledge` skill
      (`references/keycloak-realm-guide.md`) for the required shape. The shipped realm has
      only one interactive user, and `project-aas-init` cannot create one — it exits with
      "user not found in Keycloak". Add **no** OIDC client, **no** realm role and **no** claim
      mapper: application roles are AAS roles and belong in task 2.2, and nothing in this
      change uses a client-credentials grant (design.md D2a).
- [ ] 1.6 Record the native-dev port contract in `compose/compose.override.sample.yml`: the
      dispatch PostgreSQL stays on host **5432** (unchanged from today, and `SQL_PORT`'s
      default), so do **not** also publish `iams-db` there — the shipped sample allocates
      5432 to `iams-db`, and that collision is the one to avoid. `sds-server`'s TCP **5333**
      needs no entry at all: Traefik already publishes it. The real `compose.override.yml`
      stays gitignored; the contract does not.
- [ ] 1.7 Fill `compose/.env` from the template. Keep the shipped default
      `DEV_DOMAIN=127.0.0.1.nip.io` — nip.io wildcard-resolves to the loopback on every OS
      including Windows, and it is the domain the rtus-seh CORS list and every skill example
      already assume. Set `DEV_USER` / `DEV_PASSWORD`, plus the viewer account's credentials
      from 1.5. No client secret is needed.
- [ ] 1.8 Verify: run `podman compose -f compose/compose.yml up -d` (or `docker compose -f compose/compose.yml up -d`),
      then `podman compose -f compose/compose.yml ps` (or `docker compose -f compose/compose.yml ps`)
      shows `traefik`, `iams-keycloak`, `iams-aas`, `iams-web`, `sds-server`, `valkey`,
      `rtus-pms`, `rtus-seh`, `gis-service` and `dispatch-postgres` healthy, and `iams-init`
      and `project-aas-init` exited 0.

## 2. Seed / resource creation

- [ ] 2.1 Consult the `aoh-knowledge` skill (`references/services/iams.md` →
      "Project-level AAS bootstrap") before editing any seed artifact. Edit only
      `roles.yaml`: never `bootstrap.py`, never the Dockerfile, never
      `iams-aas-init.postman_collection.json`.
- [ ] 2.2 In `compose/iams/init/project-aas/roles.yaml`, under `tenant: development`, add the
      two application roles `dispatch-viewer` and `dispatch-dispatcher`, and under
      `assignments` give `${DEV_USER}` both roles and the viewer account from 1.5
      `dispatch-viewer` only. Use only the documented top-level keys — `roles`,
      `assignments`, `tenant_admins`, `groups`, `resources`, `scopes`, `resource_scopes`,
      `permissions`. There is no membership key: membership follows from `assignments`.
      Keep the file idempotent; `bootstrap.py` is GET-then-create per section.
- [ ] 2.3 Re-import the realm and re-run the bootstrap, because Keycloak's
      `start --import-realm` is skip-if-exists and task 1.5 edited the import file. Use a
      **project-wide** teardown — `podman compose -f compose/compose.yml down -v` (or
      `docker compose -f compose/compose.yml down -v`), then
      `podman compose -f compose/compose.yml up -d` (or `docker compose -f compose/compose.yml up -d`).
      Do **not** use `down -v iams-db`: Compose does not treat `-v` as a per-service volume
      wipe when a service is named, and podman-compose's `down` takes no service argument at
      all, so the per-service form can silently leave the old realm in place — the exact
      failure this step exists to prevent.
- [ ] 2.4 Verify (roles present, both accounts): for each of the two accounts, obtain a token
      and decode its payload —
      ```sh
      TOKEN=$(curl -fsS -X POST "http://iams-keycloak.${DEV_DOMAIN}/realms/aoh/protocol/openid-connect/token" \
        -d grant_type=password -d client_id=web -d scope=openid \
        --data-urlencode "username=${DEV_USER}" --data-urlencode "password=${DEV_PASSWORD}" \
        | python3 -c "import json,sys; print(json.load(sys.stdin)['access_token'])")
      python3 -c "import base64,json,sys; p=sys.argv[1].split('.')[1]; print(json.loads(base64.urlsafe_b64decode(p+'='*(-len(p)%4)))['active_tenant'])" "$TOKEN"
      ```
      The dispatcher's `active_tenant.roles` contains `dispatch-dispatcher`; the viewer's
      contains `dispatch-viewer` and not `dispatch-dispatcher`; both carry the `development`
      tenant's id. `scope=openid` is mandatory — AOH services validate via Keycloak's
      userinfo endpoint, which 403s on a token issued without it, surfacing as an opaque 401.
- [ ] 2.5 Verify (GIS accepts an operator token): `curl -fsS -o /dev/null -w '%{http_code}\n'
      -H "Authorization: Bearer $TOKEN" "http://gis.${DEV_DOMAIN}/geoentity"` prints `200`
      using the dispatcher token from 2.4. This is the token posture the outbox worker will
      use (design.md D2a); proving it here means the worker's first delivery is not the place
      it is discovered.

## 3. Implement `dispatch-svc`

- [ ] 3.1 Consult the `aoh-conventions` skill (`references/go.md` for layering, `aohhttp` and
      `BearerAuth`, `references/api.md` for the envelope and `/v{N}` paths,
      `references/database.md` for schema rules) and the `aoh-error-handling` skill for the
      new error codes, before writing code.
- [ ] 3.2 Add `migrations/0003_position_and_outbox.up.sql`: `position_lon`, `position_lat`,
      `position_at` on `{{SCHEMA}}.unit` with an all-or-nothing CHECK and range CHECKs
      (`[-180,180]` / `[-90,90]`); and a `{{SCHEMA}}.gis_outbox` table carrying the AOH
      mandatory columns plus `unit_code`, `intent` (`upsert` / `delete`), `payload jsonb`,
      `attempts`, `last_error`, `delivered_at`. Write against `{{SCHEMA}}`, never a
      hardcoded `dispatch.`.
- [ ] 3.3 Add `migrations/0004_seed_positions.up.sql`: move the seeded units to the
      `development` tenant id and give them coordinates. Keep the existing call sign ↔
      `unit_code` pairings (`FU-101` Alpha-1, `FU-102` Alpha-2, `FU-204` Bravo-1, `FU-205`
      Bravo-2, `FU-311` Charlie-1) and leave at least one unit un-positioned so the
      "not shown" count and the un-positioned detail state are exercised on first boot.
      Keep the `ON CONFLICT … DO UPDATE` idempotence — the reproducibility gate depends on it.
- [ ] 3.4 Add bearer authentication using `aoh-golib`'s shipped `aohhttp.BearerAuth`
      middleware — do not hand-roll offline JWKS validation. Mount it on `/v1/units` only, so
      `/livez` and `/readyz` stay unauthenticated. Put `sub`, `active_tenant.tenant_id` and
      `active_tenant.roles` on the request context. The middleware covers the four workshop
      stub routes too: they keep answering 501, but only to an authorised caller.
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
      returning 400 and an `aoherr.FieldDetail` per bad field. Keep `last_contact` bumping on
      every write as today, and do **not** let `position_at` follow it — `position_at` changes
      only when a write supplies a position (design.md D11).
- [ ] 3.8 Write the outbox row inside the same transaction as every create / replace /
      delete, deriving the intent from the unit's position **after** the write, not from the
      verb: has a position → `upsert`; no position, position cleared, or unit deleted →
      `delete` (design.md D6a). The handler must not call `gis-service`; a GIS outage must
      not fail a unit write.
- [ ] 3.9 Add the outbox worker. It delivers `PUT /geoentity` (upsert, keyed on `entity_id` =
      `unit_code`, `entity_type` `track`, `properties.kind` `field-unit`) or
      `DELETE /geoentity/entity_id/{unit_code}`, with backoff and at-least-once retry; a
      delete for an entity that is already gone counts as delivered. It authenticates with
      **the operator's access token**, captured by value from the request before the
      post-commit work detaches from the request context — a client-credentials token carries
      no `active_tenant` claim and `gis-service` resolves the tenant from it. Never persist a
      token: a row whose token has expired stays pending and is drained on a later
      authenticated request from the same tenant.
- [ ] 3.10 Extend `internal/config` using the scaffold's own key names:
      `IAMS_KEYCLOAK_HOST` (a URL **including the scheme**), `IAMS_KEYCLOAK_PORT`,
      `IAMS_KEYCLOAK_REALM`, `GIS_URL`, and the worker's poll interval. No client id or
      secret — there is no confidential client. Keep the existing Viper defaults pattern.
- [ ] 3.11 Go tests, following the consumer-declared-interface style the service already uses
      (`service.UnitReader`, `handler.UnitService`) with a fake repo and no mock framework —
      note these are the first tests in the repo, so establish the pattern rather than
      matching an existing file. Cover: the role projection, tenant scoping, position
      validation and clearing, the position-derived outbox intent, outbox-row-in-same-
      transaction, and worker idempotence against a stub `gis-service`. Do **not** add a
      frontend test framework — `apps/dispatch-web/AGENTS.md` records its absence as a
      decision to be raised, not reversed in passing.
- [ ] 3.12 Update `apps/dispatch-svc/AGENTS.md` (the "**No authentication**" bullet is now
      false — replace it with the bearer + tenant + role posture, and add the outbox worker to
      the load-bearing architecture list) and `apps/dispatch-svc/README.md` (its summary line
      says "no authentication").
- [ ] 3.13 Document the env block for a native run (`cd apps/dispatch-svc && go run ./cmd/server`):
      ```sh
      SQL_HOST=localhost SQL_PORT=5432 SQL_USER=dispatch SQL_PASSWORD=dispatch \
      SQL_DATABASE_NAME=dispatch SQL_SCHEMA_NAME=dispatch SQL_SSL_MODE=disable \
      IAMS_KEYCLOAK_HOST=http://iams-keycloak.${DEV_DOMAIN} IAMS_KEYCLOAK_PORT=80 \
      IAMS_KEYCLOAK_REALM=aoh \
      GIS_URL=http://gis.${DEV_DOMAIN} \
      HTTP_PORT=8081 HTTP_ALLOWED_ORIGINS=
      ```
      `IAMS_KEYCLOAK_HOST` carries the scheme (the scaffold's default is
      `http://iams-keycloak`); without it the composed Keycloak URL is unparseable.
      `HTTP_ALLOWED_ORIGINS` stays empty: the browser still never calls this service
      directly (design.md D5), so no CORS is required.
- [ ] 3.14 Verify: `cd apps/dispatch-svc && go build ./... && go vet ./... && go test ./... -count=1 -race`
      all exit 0. `go vet` is this repo's configured Go lint (`apps/dispatch-svc/package.json`,
      what `pnpm lint` runs); there is no `.golangci.yml` and no `golangci-lint` in the
      toolchain or the devcontainer image, so do not introduce one here.

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
      `src/lib/aoh/core/provider/auth/`, the SDS client, the `(public)/aoh/api/auth/*` routes
      — all six the scaffold ships (`login`, `callback`, `refresh`, `logout`, `context`,
      `context/[value]`), not just the four obvious ones — the `(private)` group and its
      layout, and `App.Locals` regaining `authResult` in `src/app.d.ts`. Tokens go to SDS;
      the browser gets only `web_auth_session_id`. Do **not** restore `gateway.config.ts` or
      the gateway proxy — design.md D5.
- [ ] 4.4 Keep `/livez` and `/readyz` **outside** the authenticated surface. They live at
      `src/routes/{livez,readyz}/+server.ts` today, outside any group; restoring the auth
      handle must not sweep them behind the sign-in redirect.
- [ ] 4.5 Move the console from `src/routes/units/` to
      `src/routes/(private)/aoh/dispatch/units/`, add `src/routes/(private)/aoh/dispatch/map/`,
      and add `nav.ts` with a `NavItem` for each, both gated on holding either application
      role. Point `LOGIN_DESTINATION` at the console's new path. Replace — do not merely
      delete — the baseline's `/` → `/units` redirect in `+layout.server.ts`: the scaffold
      ships no root `+page.svelte`, so dropping it without a replacement leaves `/` a 404
      rather than the sign-in flow the spec requires.
- [ ] 4.6 Add the role→permission projection mirroring `roles.yaml` (the TypeScript half of
      task 3.5, same two roles, same source-of-truth comment), read from
      `locals.authResult.claims.active_tenant.roles`. Hide add / edit / delete for a viewer
      and render the permission-denied card on a 403 from the service, exactly as the
      console-auth mockup's "Viewer" and "Permission denied" states show.
- [ ] 4.7 Add the signed-in operator to the restored `Navbar` — display name or username —
      with a sign-out control, as the mockup's header shows. This is what makes the
      "console identifies the signed-in operator" requirement real rather than implied by
      the layout.
- [ ] 4.8 Make `units.server.ts` send `Authorization: Bearer <token from SDS>` on every
      call, decode `position` at the wire boundary next to the existing status guard, and
      map a 401 to a re-authentication redirect and a 403 to the permission-denied state
      rather than the generic service-unavailable state.
- [ ] 4.9 Add the position section to `UnitDetail.svelte` — coordinates, fix time via the
      existing `sinceLabel`, a "No position reported." line when absent, and a **Show on
      map** control that carries the selected unit to the map route — per the mockup's
      "Unit with position" / "Unit without position" states.
- [ ] 4.10 Install the map dependencies as runtime `dependencies`:
      `@mssfoobar/gis-web-sdk@^2.0.0`, `@mssfoobar/auth-sdk`, `@cesium/engine@^25`,
      `@cesium/widgets@^15`. `@mssfoobar/sse-client` and `@mssfoobar/logger@^1.0.5` are
      already present at compatible versions. The SDK must be ≥ 1.1.0 for map interaction
      callbacks, which task 4.13's marker click depends on.
- [ ] 4.11 Add the Cesium asset-copy plugin to `vite.config.ts` (copy `Build/Workers`,
      `Source/Assets`, `Source/ThirdParty` from `@cesium/engine` and `Source` from
      `@cesium/widgets` into `static/cesium/`, on both `buildStart` and `configureServer`),
      define `CESIUM_BASE_URL`, and set `ssr.noExternal` to `[/^@mssfoobar\//, '@lucide/svelte',
      '@cesium/engine', '@cesium/widgets']`. In the same file restore the scaffold's
      `allowedHosts: ['127.0.0.1.nip.io', '.127.0.0.1.nip.io']` — the baseline removed it as
      moot on plain localhost, and `pnpm dev` runs `vite dev --host`, so serving on the
      `${DEV_DOMAIN}` origin without it is host-checked and refused. Add `static/cesium/` to
      `apps/dispatch-web/.gitignore`; `eslint.config.js` already ignores it, so leave that
      alone.
- [ ] 4.12 Mount `<GisProvider>` at the root `+layout.svelte`, fed by the app's existing
      `ThemeProvider` dark-mode store so the map repaints with the theme.
- [ ] 4.13 Build the map page: `+page.ts` exporting `ssr = false`; a `+page.server.ts` whose
      `load` returns the roster (for the counts in 4.14) and the user claims; and
      `+page.svelte` composing `CesiumMapEngineProvider` → `Map` (with the **required**
      `initial_camera_view`, plus `rtus_seh_url`, `rtus_map_name="gis"`, `user_id` from the
      `sub` claim, `tenant_id` from `active_tenant.tenant_id`, `batchUpdateInterval={500}`,
      and the `event_subscriptions` the marker-click handler needs) →
      `MapBaseLayerProvider` + `MapXyzSourceProvider` (OSM) → `MapEntityLayerProvider` +
      `MapEntityProvider kind="field-unit"` with a call-sign marker snippet →
      `MapLayerManager`. Take the user id from `data.user.sub` and the tenant from
      `data.user.active_tenant.tenant_id` — `data.user.id` / `data.user.tenantId` are
      undefined on this base and silently disable the live feed. Keep the marker snippet
      render-pure (mutating state inside it triggers `state_unsafe_mutation`). Do not fetch
      an entity list; the SDK's subscription is the only source of **entity** state
      (design.md D9) — the roster load above is unit data, which is a different thing.
- [ ] 4.14 Add the map's surrounding states from the mockup: the roster/not-shown counts
      (computed from the unit data loaded in 4.13, not from entities), the live / not-live
      badge, the "Live positions are unavailable." notice, the no-positioned-units empty
      state, and the selection card with **Open in console**. Wire map→console selection as a
      SvelteKit navigation, not a browser fetch; wire console→map by carrying the selected
      unit into the map route and flying the camera to it on mount, leaving the camera put
      when that unit has no position.
- [ ] 4.15 Update the docs that now assert the opposite of reality:
      `apps/dispatch-web/AGENTS.md` (the "🛑 This app has NO authentication" table goes; the
      map, Cesium assets and role gating join the load-bearing list),
      `apps/dispatch-web/README.md` ("This app has no authentication"), the root `README.md`
      (no-auth and one-container claims), `SETUP.md` (the full stack and both seeded
      accounts), `WORKSHOP.md` (its one `curl` example gains a `scope=openid` token fetch),
      `compose/compose.yml`'s header comment, and `UBIQUITOUS_LANGUAGE.md` (*position* and
      *fix time* join the term table; the *field unit* ↔ `geo-entity` mapping stops being
      conditional and is corrected — `field-unit` is a `kind`, `track` is the `entity_type`).
      Also sweep the inline comments that still assert no-auth in
      `apps/dispatch-web/{.env.template,.env.development,vite.config.ts,src/hooks.server.ts,src/app.d.ts}`
      and `apps/dispatch-svc/internal/repo/unit_repo.go`.
- [ ] 4.16 Document the env block for a native run (`cd apps/dispatch-web && pnpm dev`),
      added to `.env.development` and `.env.template`:
      ```sh
      ORIGIN=http://${DEV_DOMAIN}:5173
      IAM_URL=http://iams-keycloak.${DEV_DOMAIN}/realms/aoh
      IAM_CLIENT_ID=web
      OIDC_ALLOW_INSECURE_REQUESTS=1
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
      `OIDC_ALLOW_INSECURE_REQUESTS=1` is not optional over plain `http`: the scaffold's
      `discovery()` runs in a **top-level await**, so without it every route returns 500 at
      startup — the exact failure the baseline documented. `ORIGIN` must be the
      `${DEV_DOMAIN}` origin, not `localhost`, or the session cookie never reaches
      `rtus-seh` and its CORS list rejects the SSE request (design.md D10).
- [ ] 4.17 Verify: `cd apps/dispatch-web && pnpm build && pnpm check-types && pnpm lint` all
      exit 0, and
      `grep -rniE "no authentication|NO AUTH" apps README.md SETUP.md WORKSHOP.md compose/compose.yml`
      returns nothing.

## 5. End-to-end verification + reproducibility gate

- [ ] 5.1 Write `scripts/e2e-smoke.mjs` — Node, matching the repo's existing `scripts/*.mjs`
      convention and its Node 24 prerequisite. It SHALL, with no manual step: fetch a
      dispatcher token and a viewer token by password grant against the bundled `web` client
      **with `scope=openid`**; assert `GET /v1/units` is 401 with no token and 200 with the
      dispatcher token, and that the seeded roster comes back with positions; assert
      `POST /v1/units` is 403 with the viewer token and 201 with the dispatcher token; poll
      `gis-service` `GET /geoentity/entity_id/{unit_code}` until the new unit's entity appears
      and assert its `entity_type` is `track` and its `properties.kind` is `field-unit`; move
      the unit with a `PUT` and assert the entity's coordinates follow; **clear** the position
      with a `PUT` and assert the entity is removed; assert a unit row inserted directly under
      a different `tenant_id` never appears in `GET /v1/units`; delete the unit; and exit
      non-zero with a named assertion on any failure.
- [ ] 5.2 Walk the browser states the script cannot assert, against the running stack: sign
      in as the dispatcher and land on `/aoh/dispatch/units`; open `/aoh/dispatch/map` and
      confirm the roster renders with **no 404 under `/cesium/`** in the network log; move a
      unit with `PUT` in another terminal and watch the marker move without a reload; click a
      marker and confirm it opens in the console; select an un-positioned unit and confirm the
      camera stays put; leave the tab idle past the access-token lifetime and confirm the next
      action succeeds without a return to sign-in; sign out, sign in as the viewer, and
      confirm the add / edit / delete controls are absent and the map still renders.
- [ ] 5.3 Native dev E2E: `node scripts/e2e-smoke.mjs` exits 0.
- [ ] 5.4 **Reproducibility gate** — tear and rebuild infra, kill and restart the native dev
      processes for both apps using the env blocks from 3.13 and 4.16, and re-run the SAME
      E2E command from 5.3:
      ```bash
      pkill -f 'dispatch-svc/cmd/server' || true      # kill natives first: a stale process
      pkill -f 'vite dev' || true                     # holds 8081/5173 and hides the rebuild
      podman compose -f compose/compose.yml down -v   # or: docker compose -f compose/compose.yml down -v
      podman compose -f compose/compose.yml up -d     # or: docker compose -f compose/compose.yml up -d
      # wait for healthy, and for iams-init + project-aas-init to exit 0:
      podman compose -f compose/compose.yml ps        # or: docker compose -f compose/compose.yml ps
      cd apps/dispatch-svc && go run ./cmd/server &   # native, env block from 3.13
      cd apps/dispatch-web && pnpm dev &              # native, env block from 4.16
      node scripts/e2e-smoke.mjs                      # SAME command as 5.3
      ```
      The `-v` is what makes this meaningful: it discards the Keycloak, AAS, GIS, RTUS and
      dispatch volumes, so the realm import (including the viewer seed user), the AAS roles,
      the seeded units and their geo-entities must all reconverge from checked-in artifacts
      alone.
