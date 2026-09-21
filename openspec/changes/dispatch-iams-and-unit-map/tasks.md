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
- [ ] 1.2 Fold the existing dispatch PostgreSQL into the generated layout. **Copy the
      `postgres` service block out of `compose/compose.yml` before running 1.1** —
      `bootstrap.py` regenerates that file from scratch as a `name:` + `include:` list and
      keeps only include entries, so both the service definition and the file's header
      comment are discarded by 1.1, not edited by you. Put the block in
      `compose/dispatch/compose.yml` (service name **`postgres`**, `container_name:
      dispatch-postgres`, `name: aoh` so it joins the shared network) and add
      `./dispatch/compose.yml` to the regenerated include list. Keep the service name
      `postgres`: `compose exec` and `compose up` take the service name, and 5.3 depends on
      it.
- [ ] 1.3 Confirm `rtus-pms` and `rtus-seh` both set the same `rtus.clustername`
      (`aoh_rtus` in the shipped template). A mismatch makes rtus-seh form its own one-node
      Hazelcast cluster and every SSE subscription silently receives nothing.
- [ ] 1.4 Leave `compose/rtus/compose.yml` **unedited**, and verify why that is safe:
      `rtus.session-id.cookienames` already carries `web_auth_session_id` (which
      `PUBLIC_COOKIE_PREFIX=web` produces), and rtus-seh's Traefik CORS middleware already
      lists `http://${DEV_DOMAIN}:5173` — the origin task 4.18 serves the console on. Serving
      it anywhere else (e.g. `dispatch.${DEV_DOMAIN}:5173`) would require adding that origin
      to `accesscontrolalloworiginlist` here, and the credentialed SSE request would be
      blocked until you did.
- [ ] 1.5 Add **one seed user** to `compose/iams/keycloak/realm-import.json`'s `users` array
      to serve as the viewer account, following the `aoh-knowledge` skill
      (`references/keycloak-realm-guide.md`) for the required shape. The shipped realm has
      only one interactive user, and `project-aas-init` cannot create one — it exits with
      "user not found in Keycloak". Add **no** OIDC client, **no** realm role and **no** claim
      mapper: application roles are AAS roles and belong in task 2.2, and nothing in this
      change uses a client-credentials grant (design.md D2a). Give the seed user
      `${DEV_PASSWORD}` as its credential: `iams-keycloak` forwards only `DEFAULT_REALM`,
      `DEV_DOMAIN`, `DEV_USER` and `DEV_PASSWORD` into the container, so a new
      `${VIEWER_PASSWORD}`-style placeholder would import **unresolved** and the viewer's
      password grant in 2.4 would fail. Reusing `DEV_PASSWORD` needs no compose edit; adding
      a variable would mean editing `compose/iams/compose.yml`'s `environment:` as well.
- [ ] 1.6 Record the native-dev port contract in `compose/compose.override.sample.yml`: the
      dispatch PostgreSQL stays on host **5432** (unchanged from today, and `SQL_PORT`'s
      default), so do **not** also publish `iams-db` there — the shipped sample allocates
      5432 to `iams-db`, and that collision is the one to avoid. `sds-server`'s TCP **5333**
      needs no entry at all: Traefik already publishes it. The real `compose.override.yml`
      stays gitignored; the contract does not.
- [ ] 1.7 Fill `compose/.env` from the template. Keep the shipped default
      `DEV_DOMAIN=127.0.0.1.nip.io` — nip.io wildcard-resolves to the loopback on every OS
      including Windows, and it is the domain the rtus-seh CORS list and every skill example
      already assume. Set `DEV_USER` / `DEV_PASSWORD`; the viewer account from 1.5 shares
      `DEV_PASSWORD`, so there is no second credential to fill in and no client secret.
- [ ] 1.8 Verify: run `podman compose -f compose/compose.yml up -d` (or `docker compose -f compose/compose.yml up -d`),
      then `podman compose -f compose/compose.yml ps` (or `docker compose -f compose/compose.yml ps`)
      shows `traefik`, `iams-keycloak`, `iams-aas`, `iams-web`, `sds-server`, `valkey`,
      `iams-db`, `rtus-db`, `rtus-pms`, `rtus-seh`, `gis-db`, `gis-service`, `otel-collector`
      and `postgres` (container `dispatch-postgres`) healthy, and `iams-init`
      and `project-aas-init` exited 0.

## 2. Seed / resource creation

- [ ] 2.1 Consult the `aoh-knowledge` skill (`references/services/iams.md` →
      "Project-level AAS bootstrap") before editing any seed artifact. Edit only
      `roles.yaml`: never `bootstrap.py`, never the Dockerfile, never
      `iams-aas-init.postman_collection.json`.
- [ ] 2.2 In `compose/iams/init/project-aas/roles.yaml`, under `tenant: development`, add the
      two application roles `dispatch-viewer` and `dispatch-dispatcher`, and under
      `assignments` give the dispatcher account both roles and the viewer account from 1.5
      `dispatch-viewer` only. Write **literal usernames** (`admin` is the shipped `DEV_USER`
      default): `bootstrap.py` reads this file with `yaml.safe_load` and performs no `${VAR}`
      substitution — unlike `realm-import.json` — so a literal `${DEV_USER}` is searched for
      verbatim in Keycloak and the bootstrap exits "user not found". Use only the documented top-level keys — `roles`,
      `assignments`, `tenant_admins`, `groups`, `resources`, `scopes`, `resource_scopes`,
      `permissions`. There is no membership key: membership follows from `assignments`.
      Keep the file idempotent; `bootstrap.py` is GET-then-create per section.
- [ ] 2.3 Re-run the stack so `project-aas-init` picks up 2.2's `roles.yaml` edit. Task 1.5
      edits `realm-import.json` *before* 1.8's first `up -d`, so the viewer user is already in
      the initial import — but if you reordered, or edited the realm after a stack was up,
      the import is skip-if-exists and only a volume teardown re-applies it (R8). Use a
      **project-wide** teardown — `podman compose -f compose/compose.yml down -v` (or `docker compose -f compose/compose.yml down -v`),
      then `podman compose -f compose/compose.yml up -d` (or `docker compose -f compose/compose.yml up -d`).
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
        | node -e "let d='';process.stdin.on('data',c=>d+=c).on('end',()=>console.log(JSON.parse(d).access_token))")
      node -e "console.log(JSON.parse(Buffer.from(process.argv[1].split('.')[1],'base64url')).active_tenant)" "$TOKEN"
      ```
      Node, not python3: the workshop image is `node:24-bookworm-slim` with no python3, and
      unlike `bootstrap.py` this step runs against a live stack rather than at authoring
      time.
      The dispatcher's `active_tenant.roles` contains `dispatch-dispatcher`; the viewer's
      contains `dispatch-viewer` and not `dispatch-dispatcher`; both carry the `development`
      tenant's id. Run `project-aas-init` a second time against the same stack and confirm
      the role set is unchanged — that is what "the role bootstrap is idempotent" asserts, and
      a single run cannot show it. `scope=openid` is mandatory — AOH services validate via Keycloak's
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
      `attempts`, `last_error`, `delivered_at`; and a `{{SCHEMA}}.tenant_seed` table with
      `tenant_id` as its primary key, which is the marker task 3.4 writes. Write against
      `{{SCHEMA}}`, never a hardcoded `dispatch.`.
- [ ] 3.3 Add `migrations/0004_drop_preauth_seed.up.sql`: delete the unit rows that
      `0002_seed.up.sql` inserted under the pre-auth placeholder tenant. Do **not** re-tenant
      them and do **not** add positions here — a committed migration cannot know the tenant
      id (AAS assigns it at stack-up; `bootstrap.py` has to look it up by name), and rows
      written in SQL bypass the outbox and would be permanently absent from the map
      (design.md D12). Seeding moves to task 3.4.
- [ ] 3.4 Implement seeding as service behaviour, not as a command (design.md D12). After
      `BearerAuth` and the role check, on a request from a caller holding
      `dispatch-dispatcher`, the service seeds that caller's tenant **if it has never been
      seeded**, in one transaction, and **before the triggering request is answered** — so the
      dispatcher's first `GET /v1/units` already returns the roster.
      - Gate on the `{{SCHEMA}}.tenant_seed` marker from 3.2, written in the same
        transaction with `INSERT … ON CONFLICT DO NOTHING`. That is both the "once per
        tenant, ever" rule and the concurrency guard: two simultaneous dispatchers race, one
        inserts, the other sees the conflict and skips.
      - Do **not** decide by counting units. Deleting units is a legitimate operator action
        and must not resurrect the roster — the console ships a working delete today.
      - Do **not** let a `dispatch-viewer` trigger it. Seeding is a write, and D2a forbids a
        reader performing writes elsewhere in this service; the rule has to hold in both
        places.
      - Write the baseline rows in full, reusing the `ON CONFLICT … DO UPDATE` shape
        `0002_seed.up.sql` already has (including `ON CONFLICT (unit_id, name)` for
        `unit_crew`): the call sign ↔ `unit_code` pairings (`FU-101` Alpha-1, `FU-102`
        Alpha-2, `FU-204` Bravo-1, `FU-205` Bravo-2, `FU-311` Charlie-1), each unit's status,
        type, station, sector, radio channel, shift and capabilities (`FU-311` keeps its
        deliberately empty list), the two assignments (`FU-102` → `INC-2841`, `FU-311` →
        `INC-2839`) **with all five assignment columns set**, since the all-or-nothing CHECK
        rejects a partial one, and the crew rows. Give every unit but `FU-204` a position, so
        the "not shown" count and the un-positioned detail state are both exercised.
      - Stamp `created_by` / `updated_by` from the caller's `sub` and `tenant_id` from their
        claim, and enqueue the ordinary outbox rows so the projection travels the same path,
        on the same token, as any other write.

      This lives in the service because everything it needs is only available on an
      authenticated request: the AAS-assigned tenant id, the caller's identity, the crew and
      assignment shapes the write API withholds, the outbox, the constraints, and the token
      that delivers the projection. There is no seed endpoint, no seed binary and no seed
      script — nothing for an attendee to run.
- [ ] 3.5 Add bearer authentication using `aoh-golib`'s shipped `aohhttp.BearerAuth`
      middleware — do not hand-roll offline JWKS validation. Mount it on `/v1/units` only, so
      `/livez` and `/readyz` stay unauthenticated. Put `sub`, `active_tenant.tenant_id` and
      `active_tenant.roles` on the request context. The middleware covers the four workshop
      stub routes too: they keep answering 501, but only to an authorised caller.
- [ ] 3.6 Add the role→permission projection in `internal/service`: a static map from role
      name to read/write permission that mirrors `roles.yaml`, with a comment naming that
      file as its source. Read `active_tenant.roles` only — **never**
      `active_tenant.permissions`; AAS does not emit it and code that expects it 403s every
      legitimate user. Gate writes on `dispatch-dispatcher`, reads on either role, and
      return 403 with a `DISPATCH_*` code otherwise.
- [ ] 3.7 Thread tenant and identity through service and repo: add
      `WHERE tenant_id = $n` to every query, set `created_by` / `updated_by` from `sub` and
      `tenant_id` from the claim, and ignore any of the three arriving in a request body.
- [ ] 3.8 Add `position` to `domain.Unit` (a `*Position` so an un-positioned unit omits the
      key, exactly as `Assignment` does) and to `domain.UnitInput`, with range validation
      returning 400 and an `aoherr.FieldDetail` per bad field. Keep `last_contact` bumping on
      every write as today, and do **not** let `position_at` follow it — `position_at` changes
      only when a write supplies a position (design.md D11).
- [ ] 3.9 Write the outbox row inside the same transaction as every create / replace /
      delete, deriving the intent from the unit's position **after** the write, not from the
      verb: has a position → `upsert`; no position, position cleared, or unit deleted →
      `delete` (design.md D6a). The handler must not call `gis-service`; a GIS outage must
      not fail a unit write.
- [ ] 3.10 Add the outbox worker. It delivers `PUT /geoentity` (upsert, keyed on `entity_id` =
      `unit_code`, `entity_type` `track`, `geojson.properties.kind` `field-unit`) or
      `DELETE /geoentity/entity_id/{unit_code}`, with backoff and at-least-once retry; a
      delete for an entity that is already gone counts as delivered. It authenticates with
      **the operator's access token**, captured by value from the request before the
      post-commit work detaches from the request context — a client-credentials token carries
      no `active_tenant` claim and `gis-service` resolves the tenant from it. Never persist a
      token, and bound retry by that token's **remaining lifetime**. A row still undelivered
      when the token expires stays pending, with its attempt count and last error recorded,
      and is left for the reconcile step (3.4) to flush. Do **not** drain it on a later
      request from another operator: that would attribute the entity to whoever happened to
      call next, and would make a `dispatch-viewer`'s read perform a GIS write (design.md
      D2a rejects this explicitly).
- [ ] 3.11 Extend `internal/config` using the scaffold's own key names:
      `IAMS_KEYCLOAK_HOST` (a URL **including the scheme**), `IAMS_KEYCLOAK_PORT`,
      `IAMS_KEYCLOAK_REALM`, `GIS_URL`, and the projection's retry budget — attempts and
      backoff, capped by the remaining lifetime of the token the delivery carries, which is
      the real bound (design.md D2a). There is **no** poll interval: the projection is triggered by the write that
      produced it and runs on the token that write carried, so a timer that woke with no
      credential could deliver nothing (design.md D2a). No client id or secret either — there
      is no confidential client. Keep the existing Viper defaults pattern.
- [ ] 3.12 Go tests, following the consumer-declared-interface style the service already uses
      (`service.UnitReader`, `handler.UnitService`) with a fake repo and no mock framework —
      note these are the first tests in the repo, so establish the pattern rather than
      matching an existing file. Cover: the role projection, tenant scoping, position
      validation and clearing, the position-derived outbox intent, outbox-row-in-same-
      transaction, and worker idempotence against a stub `gis-service`. Do **not** add a
      frontend test framework — `apps/dispatch-web/AGENTS.md` records its absence as a
      decision to be raised, not reversed in passing.
- [ ] 3.13 Update `apps/dispatch-svc/AGENTS.md` (the "**No authentication**" bullet is now
      false — replace it with the bearer + tenant + role posture, and add the outbox worker to
      the load-bearing architecture list) and `apps/dispatch-svc/README.md` (its summary line
      says "no authentication").
- [ ] 3.14 Document the env block for a native run (`cd apps/dispatch-svc && go run ./cmd/server`):
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
- [ ] 3.15 Verify: `cd apps/dispatch-svc && go build ./... && go vet ./... && go test ./... -count=1 -race`
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
      task 3.6, same two roles, same source-of-truth comment), read from
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
      `allowedHosts: ['127.0.0.1.nip.io', '.127.0.0.1.nip.io']` on **both** `server` and
      `preview`, as the scaffold does — the baseline removed it as moot on plain localhost,
      `pnpm dev` runs `vite dev --host`, and `pnpm preview` reads the same
      `.env.development`, so serving on the `${DEV_DOMAIN}` origin without it is host-checked
      and refused in both. Add `static/cesium/` to
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
      and `UBIQUITOUS_LANGUAGE.md`. Do **not** try to edit `compose/compose.yml`'s header
      comment — task 1.1 regenerates that file and the comment is already gone. In
      `UBIQUITOUS_LANGUAGE.md` there are **two** mappings to fix, not one: *position* and
      *fix time* join the term table, the *field unit* ↔ `geo-entity` mapping stops being
      conditional and is corrected (`field-unit` is the `kind`, `track` is the
      `entity_type`), and the existing claim that our *status* is "closest to a GIS `kind`"
      becomes false — `geojson.properties.kind` is now the fixed literal `field-unit`, so
      status maps to a GeoJSON property, not to `kind`.
      Also sweep the inline comments that still assert no-auth in
      `apps/dispatch-web/{.env.template,.env.development,vite.config.ts,src/hooks.server.ts,src/app.d.ts}`
      `apps/dispatch-svc/internal/repo/unit_repo.go`, `apps/dispatch-web/Dockerfile` (its
      comment explains which scaffold env vars were removed *because* there is no auth) and
      the console page's own header comment, which moves with the file in 4.5. Task 4.18's
      grep is the check that this sweep was complete — run it while editing, not after.
      Four more places assert a seeding story this change ends, and one of them instructs
      future agents to preserve the mechanism being removed — fix all four: `README.md`
      ("Migrations and seed data apply themselves when the service starts"),
      `apps/dispatch-svc/README.md` (the same claim, plus its layout line calling
      `migrations` the "schema + idempotent seed"), and `apps/dispatch-svc/AGENTS.md`
      ("**Migrations are embedded and run on start** … The seed is idempotent
      (`ON CONFLICT … DO UPDATE`); keep it that way"). The replacement is: migrations still
      apply on start; the roster is seeded by the service itself, once per tenant, on its
      first dispatcher request (design.md D12). In `SETUP.md`, the checkpoint itself moves — it currently
      says open `http://localhost:5173` and "if you see the dispatch console with a list of
      units, you are done"; after this change that is the wrong origin, sign-in comes first,
      and the roster is empty until the seed runs. In `WORKSHOP.md`, beyond the `curl`: its
      Exercise 2 hint tells attendees to add `0003_unit_event.up.sql`, a prefix task 3.2 now
      takes, and its "Done looks like" section ends with `pnpm reset-db && pnpm start`, which
      task 4.16 redefines. Four more files assert the old flow and are in no other task:
      `.devcontainer/post-create.sh` ("Postgres already runs as a sibling container" and the
      `localhost:5173` console URL), `apps/dispatch-web/README.md` (the `localhost:5173` open
      instruction, the `/` → `/units` redirect, and "**One container is required** … There is
      still no Keycloak, no Traefik, no IAMS and no SDS"), `apps/dispatch-svc/README.md` (the
      "only container in the workshop stack" line, the env table — which is no longer "all
      optional" and omits every variable task 3.12 adds — and the writable-field list task 3.9
      widens with `position`), and `USER_STORIES.md` + `WORKSHOP.md`'s "who did it (there is
      no login)", which is now false. The slides deck (`slides/slides.md`) also shows
      `pnpm start` and `http://localhost:5173`; update it or note explicitly that it is out of
      scope, rather than leaving it to be discovered on stage.
- [ ] 4.16 Render the unseeded roster as an explicit state, not a bare empty list: "No units
      yet. A dispatcher signing in will populate the roster." This is reachable whenever
      someone signs in before a dispatcher has — the viewer account on a fresh stack, most
      likely — and it is distinct from both the service-unreachable and permission-denied
      states the console-auth mockup already draws.
- [ ] 4.17 Update the repo's start and reset tooling, which this change breaks. None of it
      is optional — `pnpm start` is what `README.md` and `SETUP.md` tell attendees to run:
      - `scripts/dev.mjs` brings up **only** `postgres`. With the auth layer restored, the
        console's `hooks.server.ts` does OIDC discovery in a top-level `await`, so with no
        `iams-keycloak` running every route returns 500 — the all-or-nothing failure
        `design.md` opens with. It must bring up the whole stack; dropping the step instead
        leaves `pnpm start` with no infrastructure at all.
      - its readiness probe waits on `http://localhost:5173/units`; the console moves to
        `/aoh/dispatch/units` (4.5) and `/units` stops serving it, so the probe never passes.
        Probe an **unauthenticated** target (`/livez`): any `(private)` route redirects to
        Keycloak and `fetch` follows redirects, so it would report ready before it is.
      - it passes the service only `SQL_PORT`; the service now needs the 3.14 block, whose
        `IAMS_KEYCLOAK_HOST` default (`http://iams-keycloak`) resolves only inside the
        compose network. Without it `pnpm start` yields a service that cannot validate a
        token.
      - `composeRunner()` probes `podman` only, while every command in this change is
        dual-form; and `start:no-db` / `pnpm start --no-db` now means "skip the whole stack",
        which its README line does not say.
      - the compose **project name changes from `compose` to `aoh`** once `bootstrap.py`
        writes `name: aoh`. Every existing volume is renamed with it:
        `compose_dispatch-pgdata` → `aoh_dispatch-pgdata`, so an attendee's database is
        silently replaced by an empty one, and `compose.devcontainer.yml`'s three
        `node_modules` volumes are orphaned, forcing a full `pnpm install` — on a network
        `SETUP.md` says has no internet. Say so in `SETUP.md` and check
        `compose/compose.devcontainer.yml` still resolves against the renamed project.
      - its `WEB_URL` is `http://localhost:5173`; 4.18 requires the `${DEV_DOMAIN}` origin.
      - after 1.2 the dispatch database lives in the generated compose project, so
        `package.json`'s `stop` and `reset-db` (`compose down` / `down -v`) now tear the whole
        sixteen-service stack rather than one container. Rename or rescope them. There is no
        seed script to add: the service seeds itself on the first dispatcher request, so
        `pnpm start` gains no seeding step at all.
- [ ] 4.18 Document the env block for a native run (`cd apps/dispatch-web && pnpm dev`),
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
      `scripts/e2e-smoke.mjs` needs its own block, and its own **source**: it runs as a bare
      `node scripts/e2e-smoke.mjs` from the repo root, so nothing loads this app's
      `.env.development` for it. Point it at `compose/.env` (which task 1.7 already fills) or
      a root `.env`, and document which:
      ```sh
      IAM_URL=http://iams-keycloak.${DEV_DOMAIN}/realms/aoh
      IAM_CLIENT_ID=web
      DEV_USER=… DEV_PASSWORD=…            # dispatcher account
      VIEWER_USER=…                        # viewer account; shares DEV_PASSWORD (task 1.5)
      DISPATCH_SVC_URL=http://localhost:8081
      GIS_URL=http://gis.${DEV_DOMAIN}
      ```
      No `SQL_*`: nothing outside the service talks to PostgreSQL. The tenant-isolation step
      (5.3) is the one exception and it shells out to `compose exec`, not to a driver.
      `OIDC_ALLOW_INSECURE_REQUESTS=1` is not optional over plain `http`: the scaffold's
      `discovery()` runs in a **top-level await**, so without it every route returns 500 at
      startup — the exact failure the baseline documented. `ORIGIN` must be the
      `${DEV_DOMAIN}` origin, not `localhost`, or the session cookie never reaches
      `rtus-seh` and its CORS list rejects the SSE request (design.md D10).
- [ ] 4.19 Verify: `cd apps/dispatch-web && pnpm build && pnpm check-types && pnpm lint` all
      exit 0, and
      `grep -rniE "no authentication|NO AUTH" apps README.md SETUP.md WORKSHOP.md compose/compose.yml`
      returns nothing.

## 5. End-to-end verification + reproducibility gate

- [ ] 5.1 Confirm the seed triggers itself: against a freshly torn-down stack, the **first**
      `GET /v1/units` with a dispatcher token returns the five baseline units — at least one
      with crew, one with an assignment, one without a position — with no seed step having
      been run. Repeat the call and confirm the counts are unchanged; then delete a unit,
      call again, and confirm it stays deleted. Finally, confirm a viewer's first request
      against an unseeded tenant writes nothing and returns an empty roster. This is the
      whole seeding surface (design.md D12) and a prerequisite of everything below.
- [ ] 5.2 Write `scripts/e2e-smoke.mjs` — Node, matching the repo's existing `scripts/*.mjs`
      convention and its Node 24 prerequisite. It SHALL, with no manual step: fetch a
      dispatcher token and a viewer token by password grant against the bundled `web` client
      **with `scope=openid`**; assert `GET /v1/units` is 401 with no token and 200 with the
      dispatcher token, and that the seeded roster comes back with positions; assert
      `POST /v1/units` is 403 with the viewer token and 201 with the dispatcher token; poll
      `gis-service` `GET /geoentity/entity_id/{unit_code}` until the new unit's entity appears
      and assert its `entity_type` is `track` and its `geojson.properties.kind` is `field-unit`; move
      the unit with a `PUT` and assert the entity's coordinates follow; **clear** the position
      with a `PUT` and assert the entity is removed; delete the unit; assert that a replayed
      session-id cookie captured before sign-out is refused, and that a malformed token and an
      expired token each answer 401 (not only a missing one); and exit non-zero with a named
      assertion on any failure.
- [ ] 5.3 Add the tenant-isolation checks to the same script. It SHALL first insert a unit row
      under a **different** `tenant_id`, directly — **two** of them, per the note below:
      ```sh
      podman compose -f compose/compose.yml exec -T postgres psql -U dispatch -d dispatch -c "INSERT INTO dispatch.unit (unit_code, call_sign, status, unit_type, station, sector, radio_channel, shift, tenant_id) VALUES ('FU-101','Other-1','Idle','Ambulance','Elsewhere','Sector 9','TAC-9','Day','other-tenant')"
      docker compose -f compose/compose.yml exec -T postgres psql -U dispatch -d dispatch -c "INSERT INTO dispatch.unit (unit_code, call_sign, status, unit_type, station, sector, radio_channel, shift, tenant_id) VALUES ('FU-101','Other-1','Idle','Ambulance','Elsewhere','Sector 9','TAC-9','Day','other-tenant')"
      ```
      Insert **two** rows under `'other-tenant'`: one reusing a `unit_code` that already
      exists in the caller's tenant (for the list and read assertions) and one with a code the
      caller's tenant does not have (so "exists only under the other tenant" and "the same
      code may exist in two tenants" both have a subject). A single row cannot satisfy all
      four — with only a shared code, nothing exists solely in the other tenant and the create
      assertion would hit the caller's own row and 409. It SHALL then assert, with the dispatcher token: that row is absent
      from `GET /v1/units`; `GET /v1/units/{unit_code}` returns the caller's own unit and
      never the other tenant's; `PUT` and `DELETE` against a `unit_code` that exists **only**
      under the other tenant answer 404 and leave that row byte-identical; and creating a unit
      with the other tenant's `unit_code` in the caller's tenant succeeds. This is what makes
      `specs/dispatch-units-api`'s four cross-tenant scenarios provable on a stack that stands
      up one identity — the predicate under test is the `WHERE tenant_id = $n` clause, not
      Keycloak. Tear the injected row down at the end so the script stays re-runnable.
- [ ] 5.4 Walk the browser states the script cannot assert, against the running stack: sign
      in as the dispatcher and land on `/aoh/dispatch/units`; open `/aoh/dispatch/map` and
      confirm the roster renders with **no 404 under `/cesium/`** in the network log; move a
      unit with `PUT` in another terminal and watch the marker move without a reload; click a
      marker and confirm it opens in the console; select an un-positioned unit and confirm the
      camera stays put; leave the tab idle past the access-token lifetime and confirm the next
      action succeeds without a return to sign-in; create a positioned unit and delete it in
      another terminal and confirm the marker appears and disappears without a reload; give
      `FU-204` a position and confirm the not-shown count disappears, then clear every
      position and confirm the empty state; stop `rtus-seh`
      (`podman compose -f compose/compose.yml stop rtus-seh`, or `docker compose -f compose/compose.yml stop rtus-seh`)
      and confirm the map still renders its base layer with the "live positions are
      unavailable" notice; stop `gis-service` the same way, write a unit, and confirm the
      write succeeds and its outbox row is left pending with an attempt count; restart both;
      sign out, sign in as the viewer, and confirm the add / edit / delete controls are absent
      and the map still renders.
- [ ] 5.5 Native dev E2E: `node scripts/e2e-smoke.mjs` exits 0.
- [ ] 5.6 **Reproducibility gate** — tear and rebuild infra, kill and restart the native dev
      processes for both apps using the env blocks from 3.14 and 4.18, and re-run the SAME
      E2E command from 5.5:
      ```bash
      # Kill natives first: a stale process holds 8081/5173 and hides the rebuild.
      # `go run` spawns a compiled child under $TMPDIR, so kill BOTH the parent and
      # the child — a pattern matching only 'cmd/server' leaves the child holding 8081.
      # pkill is in procps (present in the devcontainer and on macOS); lsof is not.
      pkill -f 'go run ./cmd/server' || true          # dispatch-svc parent
      pkill -f 'exe/server' || true                   # dispatch-svc compiled child
      pkill -f 'vite dev' || true                     # dispatch-web
      podman compose -f compose/compose.yml down -v   # or: docker compose -f compose/compose.yml down -v
      podman compose -f compose/compose.yml up -d     # or: docker compose -f compose/compose.yml up -d
      # wait for healthy, and for iams-init + project-aas-init to exit 0:
      podman compose -f compose/compose.yml ps        # or: docker compose -f compose/compose.yml ps
      cd apps/dispatch-svc && go run ./cmd/server &   # native, env block from 3.14
      cd apps/dispatch-web && pnpm dev &              # native, env block from 4.18
      until curl -fsS http://localhost:8081/readyz >/dev/null; do sleep 1; done
      node scripts/e2e-smoke.mjs                      # SAME command as 5.5
                                                      # (its first dispatcher call seeds)
      ```
      The `-v` is what makes this meaningful: it discards the Keycloak, AAS, GIS, RTUS and
      dispatch volumes, so the realm import (including the viewer seed user), the AAS roles,
      the seeded units and their geo-entities must all reconverge from checked-in artifacts
      alone — the roster included, which needs no step here precisely because the service
      seeds itself on the E2E script's first dispatcher request.
