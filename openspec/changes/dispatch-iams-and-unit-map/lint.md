# Lint

Mechanical pre-apply lint over `proposal.md`, `design.md`, `specs/**/*.md` and
`tasks.md` for `dispatch-iams-and-unit-map`. Every item below is **PASS**, **FAIL**, or
**N/A** with a one-line reason.

> **Second pass.** The first run of this lint marked every item PASS and was wrong to.
> Four parallel agent reviews found, among other things, that it certified a claim
> contradicted by `proposal.md` (§3), quoted a `grep` result that was not the result (§3),
> waved through an endpoint no scenario exercised (§5), and passed scenarios whose WHEN
> clause nothing in the change could reach (§6). Those artifacts have been fixed; the
> verdicts below are re-derived from the corrected files, and the checks that previously
> hid a defect say so.

## 1. Authorization model

- [x] **PASS** — `design.md`'s Runtime dependencies table lists `iams-keycloak` and
      `iams-aas` as separate rows with distinct roles ("OIDC provider; hosts the `aoh`
      realm…" / "Authorization control plane; owns the … tenant roles"), never bundled
      as "IAMS".
- [x] **PASS** — No application-level concern is attributed to Keycloak. Every occurrence of
      "realm role" across the artifacts is a **prohibition**
      (`specs/dispatch-access-control/spec.md`, `tasks.md` 1.5, `design.md` D2,
      `proposal.md`). The change now registers **no OIDC client at all**: an earlier draft
      added a confidential client for the projection worker, which `aoh-knowledge` →
      `integration-patterns.md` shows cannot work, because a client-credentials token carries
      no `active_tenant` claim and `gis-service` resolves the tenant from it. `design.md` D2a
      records the reversal and the posture that replaces it.
- [x] **PASS** — No scenario asserts the JWT carries `active_tenant.permissions`. The only
      spec mention is the negative assertion in `specs/dispatch-access-control/spec.md`
      ("it references no `active_tenant.permissions` claim"). Permission-level gating is an
      in-service role→permission projection over `active_tenant.roles` (`design.md` D3,
      `tasks.md` 3.7 / 4.6) — option (b) of the check.

## 2. Session storage

- [x] **PASS** — The change modifies a SvelteKit surface, and `design.md`'s Runtime
      dependencies lists both `sds-server` and `valkey`.
- [x] **PASS** — No artifact proposes browser-side token storage as the target posture.
      `design.md` D4 names the cookie-only fallback only to reject it, and
      `specs/dispatch-access-control/spec.md` requires "no cookie, `localStorage` entry, or
      server-rendered payload contains a JWT" — including across a token renewal.
- [x] **PASS** — Option (a): the platform default is kept. `tasks.md` 4.17 sets
      `PUBLIC_COOKIE_PREFIX=web`, and `tasks.md` 1.4 leaves `compose/rtus/compose.yml`
      unedited **and states the two reasons that is safe** — `rtus.session-id.cookienames`
      already carries `web_auth_session_id`, and rtus-seh's CORS middleware already lists
      `http://${DEV_DOMAIN}:5173`, the origin 4.17 serves on. An earlier draft served the
      console at `dispatch.${DEV_DOMAIN}:5173`, which is **not** in that CORS list and would
      have blocked every SSE connect; `design.md` D10 now records the origin choice and its
      three consequences.

## 2a. Real-time subscriptions

- [x] **PASS** — The map's live feed is owned by `@mssfoobar/gis-web-sdk`'s `Map`
      component, which subscribes through `@mssfoobar/sse-client`. No feature code wraps
      `EventSource`: `specs/dispatch-map/spec.md` states the SDK's subscription "is the only
      live-update path", and `tasks.md` 4.13 composes SDK components rather than a
      subscription of its own.
- [x] **PASS** — narrower than it first appears, so stated precisely: the map page **does**
      load a unit list (`tasks.md` 4.13's `+page.server.ts`, for the counts in 4.14), but
      that list is never merged with the live topic. Markers come only from the SDK's
      subscription; counts come only from the roster load. There is therefore no keyed list
      fed by both a fetch and an SSE prepend, and no `each_key_duplicate` site to dedup.
      `design.md` D9 decides this explicitly and `specs/dispatch-map/spec.md` asserts it
      observably ("the page issues no separate entity list request").

## 3. API contract

- [x] **PASS** — Every endpoint this change exposes uses `/v{N}/<resource>`: `/v1/units`,
      `/v1/units/{unit_code}`, `/v1/units/seed`, the four stub route/method pairs across three
      `/v1/units/{unit_code}/…` paths, plus root-mounted `/livez` and `/readyz`.
      `grep -rn '/api' specs design.md proposal.md tasks.md` returns **5 hits, not zero** —
      one each in `specs/dispatch-access-control/spec.md`, `design.md` and `proposal.md`, two
      in `tasks.md`, every one of them the SvelteKit `(public)/aoh/api/auth/*` route group or
      a reference to `aoh-conventions/references/api.md`. (Counted twice before and stated
      wrongly twice: first as zero, then as four with the `specs/` hit missed — which is why
      the count is now given per file.) No REST endpoint carries an
      `/api` prefix, so the verdict holds; the first pass of this lint claimed the grep
      returned nothing, which was false, and a fabricated mechanical result is the one thing
      a reader of a mechanical lint cannot be expected to re-check.
- [x] **N/A** — No fronting gateway proxy route exists, by decision. `design.md` D5 rejects
      restoring `gateway.config.ts` and `(private)/aoh/gateway/[...path]`: this change
      introduces no browser-side call to `dispatch-svc`, `aoh-conventions/web.md` forbids
      routing a server `load` through the proxy anyway, and the GIS SDK makes no browser-side
      REST calls (`aoh-gis-integration`). `tasks.md` 4.3 states the exclusion so it is not
      restored by reflex. **This N/A was previously indefensible**: `proposal.md` said in
      three places that the gateway *was* restored, including its module name. Those three
      passages now match D5.
- [x] **PASS** — vacuously, and stated precisely rather than dressed up: this change's
      scenarios assert **statuses and field presence**, not body envelopes. `grep -rn
      'errorCode\|sent_at\|details' specs/` returns nothing, and the single envelope token
      anywhere in `specs/` is one `data` (`specs/dispatch-units-api/spec.md`, "the response's
      `data` carries that position"). So no scenario asserts a forbidden shape because none
      asserts a shape at all; the envelope requirements it inherits from
      `dispatch-units-service` ("Responses use the AOH success envelope") are unchanged and
      not restated here. `design.md`'s API surface states the contract for both success and
      error.
- [x] **PASS** — `/livez` and `/readyz` appear in `design.md`'s API surface, mounted at root
      and marked Unauthenticated, and `specs/dispatch-access-control/spec.md`'s "Health probes
      stay outside the authenticated surface" covers **both** apps — the console's probes as
      well as the service's. `tasks.md` 3.6 and 4.4 each implement their half; an earlier
      draft specified both and implemented only one.

## 4. UI surfaces

- [x] **PASS** — Two mockups exist with state switchers covering every state their specs
      name. `design/dispatch-map-mock.html` — 7 states: live feed · positioned unit selected ·
      un-positioned unit selected · live feed unavailable · no positioned units · fully
      positioned roster · viewer read-only. `design/dispatch-console-auth-mock.html` —
      6 states: dispatcher · viewer · permission denied · service unreachable · unit with
      position · unit without position. Both carry a light/dark toggle. The un-positioned
      selection, fully-positioned-roster and service-unreachable states were added in this
      pass; the specs named them and the mocks did not have them.
- [x] **PASS** — `design.md`'s UI / Design System section enumerates the `@mssfoobar/ui`
      primitives the surfaces compose from and the `@mssfoobar/gis-web-sdk` components the
      map composes from. `specs/dispatch-map/spec.md` additionally names the mockup path.
- [x] **PASS** — Both gated page routes get option (a) **and** (b): `tasks.md` 4.5 adds a
      `nav.ts` `NavItem` per route gated on holding either application role, and `tasks.md`
      4.6 plus the console mock's "Permission denied" state specify the denied card.
- [x] **PASS** — No artifact proposes hand-rolling a shipped primitive. `design.md`'s UI
      section says so, `tasks.md` 4.1 repeats the prohibition, and both mocks now render the
      `Sidebar` + `Navbar` the restored layout brings back rather than implying a bespoke
      header.
- [x] **PASS** — `design.md` mandates `@mssfoobar/gis-web-sdk` and no task hand-rolls it:
      `tasks.md` 4.2 names the SDK's skill in backticks and forbids building the surface from
      raw primitives; 4.13 composes the SDK tree, including the **required**
      `initial_camera_view` prop and the `event_subscriptions` the marker click needs.
      (The UI-builder register in the tasks template lists `aoh-dashboard` only; a map on
      `gis-web-sdk` is the same category, and `tasks.md` 4.2 argues the analogy openly rather
      than leaving the omission to read as permission to hand-roll.)

## 5. Cross-artifact consistency

- [x] **PASS** — All six capabilities in `proposal.md` have a matching spec file, names
      identical: `dispatch-access-control`, `field-unit-geo-projection`, `dispatch-map`,
      `dispatch-console`, `dispatch-units-api`, `field-unit-roster`.
- [x] **PASS** — 37 requirement blocks (36 ADDED/MODIFIED plus one REMOVED) and 125
      scenarios; `grep -rn '^### Scenario' specs/` returns nothing, so every scenario header
      is exactly four hashes. The one scenario-less block is the REMOVED "The application has
      no authentication", which carries **Reason** and **Migration** instead — the shape
      `templates/spec.md` shows and every prior REMOVED block in this repo uses.
      `openspec validate dispatch-iams-and-unit-map --strict` passes.
- [x] **PASS** — Every endpoint in the API surface is exercised. Newly closed in this pass:
      `/(public)/aoh/api/auth/refresh`, which the first pass waved through as covered by the
      sign-in scenarios and was not — `specs/dispatch-access-control/spec.md` now carries "A
      session outlives the access token without the operator noticing"; and the three
      workshop stub routes, absent from the surface entirely although R2 depends on their
      posture changing — now listed, and covered by "The unimplemented stub routes are also
      protected". The rest: `GET /v1/units` → access-control 401/200 + units-api tenant
      scoping; `GET /v1/units/{unit_code}` → units-api 404; `POST`/`PUT`/`DELETE` →
      access-control viewer/dispatcher + units-api position + geo-projection delete;
      `/livez`+`/readyz` → access-control health probes; `POST /v1/units/seed` → units-api
      "Re-running the seed", "Seeding requires the dispatcher role" and "Seeded rows belong to
      the caller's tenant"; the page routes → dispatch-console and dispatch-map. Of the consumed `gis-service` endpoints, `PUT /geoentity` and
      `DELETE`/`GET /geoentity/entity_id/{id}` are exercised by geo-projection scenarios; the
      bare `GET /geoentity` collection is exercised by `tasks.md` 2.5 and by no scenario,
      which is acceptable because `design.md` marks that table "consumed, not exposed —
      **not** part of this change's API surface". Two further gaps closed
      in this pass: `DELETE /v1/units/{unit_code}/assignment` was in the surface and in no
      scenario (the stub-route scenario now names all four method/path pairs), and the
      surface listed four auth routes where `tasks.md` restores the scaffold's six — it now
      lists six, and "The scaffold's auth routes are restored as a set" covers `context` and
      `context/[value]`.
- [x] **PASS** — All 18 rows of `design.md`'s Runtime dependencies table carry an owner:
      fifteen `Platform — added by aoh-compose` (one noting `roles.yaml` is owned by this
      change) and three `Existing — modified by this change`. `iams-web` was added in this
      pass — it comes up unconditionally with the IAMS fragment and the table had omitted it.

## 6. Verification feasibility

- [x] **PASS** — Every THEN asserts something observable: an HTTP status, a JWT claim, a DB
      row or constraint, a `gis-service` response body, a browser cookie, a network-log
      absence, or a DOM state. The inspection-style scenarios ("the map page source is
      reviewed", "the outbox row is inspected") assert statically checkable facts about
      checked-in files and each names the artifact to inspect.

      The cross-tenant scenarios are reachable on a single-identity stack because
      `tasks.md` **5.3** creates the other tenant's row itself, with an explicit
      `compose exec … psql … INSERT` in both runtime forms, asserts all four scenarios
      against it, and tears it down so the script stays re-runnable. An earlier pass claimed
      this was fixed when only the spec wording had changed and no task inserted the row —
      the scenarios were still unreachable. The one that could not be salvaged, a
      cross-tenant *GIS* read, was removed rather than left unprovable; `design.md`'s Open
      Question 3 records what that leaves untested.
- [x] **PASS** — Token acquisition is named, never hand-minted: the password grant against
      the bundled `web` client for both operator accounts, **with `scope=openid`**.
      `tasks.md` 2.4 gives the exact `curl` and decode, and **5.2** makes the E2E script obtain
      both tokens the same way (5.1 is the seed step; the citation was stale after the
      section-5 renumbering). The scope is not decoration — AOH services validate through
      Keycloak's userinfo endpoint, which 403s on a token issued without it and surfaces as
      an opaque 401. The first pass omitted it everywhere.

## 7. Tasks structure

- [x] **PASS** — Every section ends with a concrete verification naming exact commands:
      1.8 (compose `up -d` / `ps`, naming the containers that must be healthy and the init
      containers that must exit 0), 2.4 + 2.5 (a runnable token fetch, claim decode, and a
      `curl` to `gis-service` asserting `200`), 3.16 (`go build && go vet && go test -race`),
      4.18 (`pnpm build && pnpm check-types && pnpm lint` plus a `grep` over the docs),
      5.6 (the reproducibility gate). Section 2's last step previously ended in prose with no
      command; it is now a `curl` with an asserted status.
- [x] **PASS** — All 9 compose invocations carry both forms (`podman compose` ×9,
      `docker compose` ×9). Both previously split pairs are fixed: 2.3's teardown, and 5.3's
      `exec … psql`, which had its podman form as a command and its docker form as prose —
      both are now written out in full so a line-oriented reader sees a runnable command
      either way.
- [x] **PASS** — No task runs `compose up` for `dispatch-svc` or `dispatch-web`; both are
      exercised natively (`go run ./cmd/server`, `pnpm dev`).
- [x] **PASS** — Section order is Compose (1) → Seed (2) → Implement `dispatch-svc` (3) →
      Implement `dispatch-web` (4) → E2E + gate (5). No Scaffold section, correctly: both
      apps exist and are being extended. Backend precedes frontend, and the producer
      (`dispatch-svc`, which fills GIS) precedes the consumer (`dispatch-web`).

## 8. Skill triggering

- [x] **N/A** — No Scaffold section; both apps already exist. (`aoh-web-init` is referenced
      in `tasks.md` 4.3 as the source of the auth layer's shape, not as a scaffolding step.)
- [x] **PASS** — `tasks.md` 1.1, the Compose section's first task, references `aoh-compose`
      in backticks with its `bootstrap.py` invocation.
- [x] **PASS** — `tasks.md` 3.1 and 4.1, the first task of each Implement section, reference
      `aoh-conventions` in backticks and name the specific references.
- [x] **PASS** — Both IAMS tasks name `aoh-knowledge` in backticks with the specific
      reference: 2.1 names `references/services/iams.md` for `roles.yaml`, and 1.5 names
      `references/keycloak-realm-guide.md` for the `realm-import.json` seed user.
- [x] **PASS** — `tasks.md` 4.1 references `aoh-design` in backticks and names both mockup
      paths in full.
- [x] **PASS** — `tasks.md` 4.2 names `aoh-gis-integration` in backticks as the SDK skill for
      the map surface.

## 9. Native dev env documentation

- [x] **PASS** — Each Implement section's last task before its verification is the env block:
      `tasks.md` 3.15 for the Go service and 4.17 for the SvelteKit app (4.16, the repo's
      start/reset tooling, sits between the docs task and the env block). The same task also
      documents the block the two repo-root `.mjs` scripts read — and names their env
      **source**, since a bare `node scripts/*.mjs` loads no app `.env`. Corrected in this
      pass, because the first draft would not have booted either process: `IAM_REALM` and
      `DISPATCH_SVC_CLIENT_ID`/`_SECRET` were invented (the scaffold's keys are
      `IAMS_KEYCLOAK_REALM` / `IAMS_KEYCLOAK_CLIENT_ID`, and the client pair is now
      unnecessary), `IAMS_KEYCLOAK_HOST` was missing its scheme, and
      `OIDC_ALLOW_INSECURE_REQUESTS=1` was absent — without which the scaffold's top-level
      `await discovery()` fails at startup over plain `http` and **every** route returns 500.
- [x] **PASS** — `tasks.md` 1.6 names the native-dev port contract and records it in
      `compose/compose.override.sample.yml`: the dispatch PostgreSQL keeps host **5432**, so
      `iams-db` must not also take it (the shipped sample allocates 5432 to `iams-db` — that
      collision is the point of the task), and `sds-server`'s **5333** needs no override
      entry because Traefik already publishes it.

## 10. Reproducibility gate

- [x] **PASS** — `tasks.md` 5.6 is the last task of section 5, inside the E2E section.
- [x] **PASS** — It runs a project-wide `down -v` then `up -d`, both in dual form. Note
      `tasks.md` 2.3 also now uses the project-wide form: the earlier `down -v iams-db` is
      not a per-service volume wipe in Compose and is not accepted at all by podman-compose,
      so it could have silently left the old realm in place — the exact failure that step
      exists to prevent (R8).
- [x] **PASS** — It **kills** and restarts both native processes, referring to the env blocks
      from 3.15 and 4.17. The Go server is killed by its listening port
      by `pkill` on **both** the `go run` parent and its compiled `exe/server` child: `go run`
      spawns the child under `$TMPDIR`, so killing only the parent leaves 8081 held. `pkill`
      is in `procps`, which the devcontainer installs and macOS has; an earlier draft used
      `lsof -ti … | xargs -r`, neither of which is available in the documented environment.
- [x] **PASS** — It re-runs `node scripts/e2e-smoke.mjs`, byte-identical to task 5.5's
      command, annotated "SAME command as 5.5". It also re-runs `scripts/seed-roster.mjs`
      first, because the roster is now part of what must reconverge (D12) rather than
      something a migration recreates.

## Resolution

All items are **PASS** or **N/A**. Two N/A, each with its reason recorded (no Scaffold
section; no gateway proxy route, by decision D5). `openspec validate
dispatch-iams-and-unit-map --strict` passes.

The change is **apply-ready**. Four things are a human call before `/opsx:apply`, not lint
failures:

1. **R1 in `design.md`** — this change ends the workshop's "one container, no
   authentication" property, which was a deliberate baseline decision
   (`baseline-dispatch-console` D1). Reversing it is the largest cost here and the one
   thing no amount of linting can decide.
2. **R7 / task 1.1** — image pull access to `ghcr.io/mssfoobar` for `iams-*`, `rtus-*` and
   `gis-service`. The checked-in `.npmrc` token covers npm packages, not container images.
   Confirm before starting, not at 1.8.
3. **R9** — `bootstrap.py` needs python3, which the workshop image does not carry. This is
   an authoring-time step whose output is committed, so it does not reach attendees, but
   whoever applies the change needs python3 somewhere. Everything that runs against a live
   stack is on Node, which the image has — an earlier draft used python3 in the seed
   verification, which would have failed in the documented environment.
4. **R5** — the two platform skills disagree on whether `MapLibreEngineProvider` is a usable
   2D engine or an empty stub. The design proceeds with Cesium, which both agree works;
   if MapLibre turns out to be real it is strictly lighter for this surface, and the
   contradiction is worth reporting upstream either way.

`design.md`'s Open Questions 1–3 are answerable during apply and do not gate it. The two
questions the first draft deferred — which realm users get which role, and whether
`last_contact` still bumps on every write — are now closed as D2 and D11, because both
turned out to change shipped artifacts rather than being apply-time details.
