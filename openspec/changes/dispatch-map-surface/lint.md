# Lint

Mechanical pre-apply lint over `proposal.md`, `design.md`, `specs/**/*.md` and `tasks.md`
for `dispatch-map-surface`. Each item is **PASS**, **FAIL**, or **N/A** with its reason.

> This change is deliberately small: one app, no new runtime dependency, no auth, no
> backend. Many of the template's checks exist to catch mistakes in territory this change
> never enters, so there are more N/As here than in a typical AOH change. §2 is the one
> that does not fit cleanly, and it is called out rather than smoothed over.

## 1. Authorization model

- [x] **N/A** — no authorization surface. The app has no authentication at all
      (`baseline-dispatch-console` D1) and this change adds none: design.md D3 records
      `IAM_URL` unset as the GIS SDK's supported passthrough mode.
- [x] **N/A** — same reason. No role, permission or claim is named anywhere in the change;
      `grep -ri 'active_tenant\|keycloak role\|realm role' ` over the artifacts returns only
      D3's explanation of what is *not* being adopted.
- [x] **N/A** — no JWT is read, so no scenario can assert a claim.

## 2. Session storage

- [ ] **N/A, and the template offers no N/A for this case — read the reason.** The check is
      "when `proposal.md` introduces or modifies any SvelteKit surface, `design.md`'s Runtime
      dependencies lists `sds-server` **and** `valkey`", and its only N/A is "no UI surface".
      This change *does* modify a SvelteKit surface and lists neither.

      It is a deliberate deviation, not an oversight. SDS is mandatory for an AOH web app
      **because such an app holds auth tokens server-side**; this one holds no token, has no
      session, and issues no cookie — the baseline removed the entire auth layer by decision,
      and `aoh-gis-integration` documents `IAM_URL` unset as a first-class mode rather than a
      degraded one. Adding `sds-server` and `valkey` here would compose two containers that
      nothing reads.

      The check becomes live the moment this app gains a session, which is what
      `dispatch-iams-and-unit-map` proposes — and there it passes, listing both. Flagging it
      here is the point: a reviewer should see that the rule was read and consciously not
      applied, rather than find it silently marked PASS.
- [x] **PASS** — no design text describes browser-side token storage, because no token
      exists.
- [x] **N/A** — no cookie is issued, so there is no prefix to match against `rtus-seh`'s
      seeded `cookienames`. RTUS is not part of this change.

## 2a. Real-time subscriptions

- [x] **N/A** — the change consumes no RTUS topic or map. `tasks.md` 1.7 omits
      `rtus_seh_url`, `rtus_map_name`, `user_id` and `tenant_id` from `<Map>` **specifically**
      so the SDK's live feed never attempts to subscribe (it subscribes only when all four
      are present), and `specs/dispatch-map/spec.md` asserts no request reaches RTUS.
- [x] **N/A** — no live topic, so no initial-fetch-plus-SSE race and no prepend site.

## 3. API contract

- [x] **N/A** — the change exposes no HTTP endpoint. `dispatch-svc` is untouched; the map
      calls nothing. `grep -rn '/api' specs design.md proposal.md tasks.md` returns **zero**
      hits, which here reflects that there is no API surface rather than that its paths are
      correct.
- [x] **N/A** — no gateway proxy, and none is needed: the browser makes no backend call.
- [x] **N/A** — no scenario asserts a response body, because no request is made.
- [x] **PASS** — `/livez` and `/readyz` remain mounted at the root and unauthenticated, as
      `design.md`'s route table records. This change does not touch them; they are listed so
      a reviewer can see they were considered rather than forgotten.

## 4. UI surfaces

- [x] **PASS** — `openspec/changes/dispatch-map-surface/design/map-mock.html` exists with a
      state switcher over both states the specs name — **loaded** and **tiles unavailable** —
      plus a light/dark toggle. Two states is the honest count: this surface has exactly two
      things that can vary, because it carries no data and depends on no backend.
- [x] **PASS** — `design.md`'s UI / Design System section enumerates both the
      `@mssfoobar/gis-web-sdk` components the map composes from (`GisProvider`,
      `CesiumMapEngineProvider`, `Map`, `MapBaseLayerProvider` + `MapXyzSourceProvider`,
      `MapLayerManager`) and the `@mssfoobar/ui` primitives its chrome uses (`Button`,
      `Card`, `Badge`), and names the mockup path.
- [x] **N/A** — no page route is gated by an application role, because there are no roles.
- [x] **PASS** — nothing is hand-rolled. `design.md`'s UI section states it, and `tasks.md`
      1.1 and 1.2 repeat the prohibition for the chrome and the map respectively.
- [x] **PASS** — `design.md` mandates `@mssfoobar/gis-web-sdk` and `tasks.md` 1.2 names the
      `aoh-gis-integration` skill in backticks and forbids hand-rolling the surface; 1.7
      composes the SDK tree directly, including the **required** `initial_camera_view`.
      (The UI-builder register in the tasks template lists `aoh-dashboard` only; a map on
      `gis-web-sdk` is the same category, and 1.2 says so.)

## 5. Cross-artifact consistency

- [x] **PASS** — both capabilities in `proposal.md` have a matching spec file, names
      identical: `dispatch-map` (new) and `dispatch-console` (modified).
- [x] **PASS** — 7 requirements and 18 scenarios; every requirement carries at least one
      `#### Scenario:` and `grep -rn '^### Scenario' specs/` returns nothing, so every
      scenario header is exactly four hashes. `openspec validate dispatch-map-surface
      --strict` passes.
- [x] **PASS** — the route table's three entries are all exercised: `/map` by
      `specs/dispatch-map/spec.md` throughout, `/units` by `specs/dispatch-console/spec.md`'s
      link scenarios, and `/livez` · `/readyz` are unchanged and asserted by neither, which
      the table itself states.
- [x] **PASS** — both rows of `design.md`'s Runtime dependencies table carry an owner
      (`Existing — untouched by this change`, `Existing — modified by this change`), and the
      section says in full sentences that **no** dependency is added.

## 6. Verification feasibility

- [x] **PASS** — every THEN is observable from outside: rendered tiles, a network log with
      no 404 under the Cesium base URL and no request to any backend, a toggled layer, a
      repainted palette, an exported `ssr = false`, and a `git status` that shows the copied
      assets ignored. The inspection-style scenarios ("the root layout is reviewed", "the
      map page source is reviewed") assert statically checkable facts and name the file.
- [x] **N/A** — no authentication scenario exists, so there is no token to obtain.

## 7. Tasks structure

- [x] **PASS** — both sections end in concrete commands: 1.12
      (`pnpm build && pnpm check-types && pnpm lint`, plus a `git status --porcelain` on the
      copied assets) and 2.1–2.3 (`pnpm start`, then named browser observations; `pnpm stop`
      to prove the map needs no backend; `pnpm build && pnpm preview` to prove the asset
      plugin covers the build hook as well as the dev one).
- [x] **N/A** — the change runs no compose command. It composes no infrastructure, so the
      dual-form rule has nothing to apply to. (`pnpm stop` in 2.2 is the repo's own script.)
- [x] **PASS** — no task runs `compose up` for an app; `dispatch-web` is exercised natively
      via `pnpm start` / `pnpm dev`, as it is today.
- [x] **PASS** — section order is Implement `dispatch-web` (1) → End-to-end verification (2).
      No Scaffold (the app exists), no Compose (no infrastructure), no Seed (no state), and
      only one app is touched.

## 8. Skill triggering

- [x] **N/A** — no Scaffold section; `apps/dispatch-web` already exists and is being
      extended.
- [x] **N/A** — no Compose section.
- [x] **PASS** — `tasks.md` 1.1, the first task of the only Implement section, references
      `aoh-conventions` in backticks and points at `references/web.md`, and additionally
      directs the implementer to `apps/dispatch-web/AGENTS.md` first, because several of that
      reference's rules do not apply to this app.
- [x] **N/A** — no Seed section and no IAMS authz artifact is touched.
- [x] **PASS** — `tasks.md` 1.1 references `aoh-design` in backticks and names the mockup
      path in full.
- [x] **PASS** — `tasks.md` 1.2 names `aoh-gis-integration` in backticks as the SDK skill for
      the map surface.

## 9. Native dev env documentation

- [x] **PASS** — `tasks.md` 1.11 is the last task before 1.12's verification and documents
      the env block, whose content is that **no variable changes**: `.env.development` is
      untouched, `IAM_URL` stays unset (which is what keeps the auth hook a passthrough), and
      `ORIGIN` is as today. Recording the absence explicitly is the point — the next reader
      would otherwise go looking for GIS configuration that does not exist.
- [x] **N/A** — `compose.override.yml` is unchanged; this change exposes no port and touches
      no compose file.

## 10. Reproducibility gate

- [x] **N/A** — the change composes no infrastructure and seeds nothing, so there is no
      volume to tear with `down -v` and no state to reconverge. `tasks.md` says so
      explicitly at the end of section 2 rather than leaving the omission to be noticed. The
      existing PostgreSQL container is untouched and `pnpm reset-db` means what it always
      meant.

## Resolution

Every item is **PASS** or **N/A** except **§2**, which is a **conscious deviation** rather
than a pass: an AOH web app is required to wire SDS, and this one does not, because it holds
no token and has no session. The reason is recorded above and in `design.md` D3; a reviewer
should agree with it explicitly rather than let it through as a formality.

The change is otherwise **apply-ready**. Three things are worth a human decision first, none
of them a lint failure:

1. **R4 — the base tiles come from the public internet, and `SETUP.md` says the workshop
   network has none.** The map will render its engine, controls and empty-state note over a
   blank background. That is a tile problem rather than an integration problem, but it is
   what attendees will see. Accept it, bundle an offline tile source, or point
   `MapXyzSourceProvider` at a tile server inside the network.
2. **R5 — `@mssfoobar/gis-web-sdk` must resolve from GitHub Packages**, and
   `pnpm-workspace.yaml`'s `minimumReleaseAge: 10080` means a version published within seven
   days will not. Confirm the install before the day.
3. **R3 — the platform skills contradict each other on `MapLibreEngineProvider`.** The design
   takes Cesium, which both agree works. If MapLibre is real it is strictly lighter here, and
   the contradiction is worth reporting upstream either way.
