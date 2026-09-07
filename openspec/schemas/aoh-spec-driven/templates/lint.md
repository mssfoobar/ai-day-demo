# Lint

Mechanical pre-apply lint over every prior artifact —
`proposal.md`, `design.md`, `specs/**/*.md`, and `tasks.md`. Each check below
is **PASS** or **FAIL** with the offending line quoted, or **N/A** with the
one-line reason. Use this to catch wrong-by-default patterns before
`/opsx:apply` starts — fixing here costs minutes; fixing during apply costs
hours and lands broken assumptions in committed code.

If any item is **FAIL**, edit the offending artifact and re-run the lint.
Do NOT proceed to `/opsx:apply` while any FAIL remains. If a fix touches
upstream artifacts (proposal/design/specs), regenerate `tasks.md` before
re-linting — tasks may have inherited the broken assumption.

## 1. Authorization model

> Roles and permissions live in AAS, not Keycloak. The JWT carries roles only,
> not resolved permissions. (See `aoh-knowledge/services/iams.md`.)

- [ ] **PASS** — Runtime dependencies in `design.md` enumerates `iams-keycloak`
      AND `iams-aas` as separate entries (not bundled as "IAMS").
- [ ] **PASS** — No mention of "Keycloak role", "Keycloak permission", "realm
      role granting X", or per-app OIDC web client registration anywhere in
      `proposal.md` / `design.md` / `specs/**` for application-level concerns.
- [ ] **PASS** — No spec scenario asserts the JWT contains
      `active_tenant.permissions`. AAS surfaces `active_tenant.roles` only.
      Scenarios that need permission-level gating MUST either (a) assert
      `active_tenant.roles` membership, or (b) describe the in-service
      role→permission projection.

## 2. Session storage

> SDS is mandatory for every AOH web app. (See `aoh-knowledge/services/sds.md`.)

- [ ] **PASS** — When `proposal.md` introduces or modifies any SvelteKit
      surface, `design.md`'s Runtime dependencies lists `sds-server` AND
      `valkey`.
- [ ] **PASS** — No design.md text describes "tokens stored in cookies",
      "access token cookie", or other browser-side token storage as the
      target posture. Tokens belong server-side; the browser holds a session
      ID only.
- [ ] **PASS** — Every web app's `.env.template` and compose entry either
      (a) use the platform-default `PUBLIC_COOKIE_PREFIX=web` so the auth
      cookie matches the seeded `rtus.session-id.cookienames` on rtus-seh,
      OR (b) set a non-default prefix AND update
      `compose/rtus/compose.yml` to include the matching
      `<prefix>_auth_session_id` in `rtus.session-id.cookienames` in the
      same change. Mismatch causes silent 401s on every SSE connect.
      (See `aoh-knowledge/services/rtus.md` → "Cookie prefix convention".)
- [ ] **N/A** — Change has no UI surface (purely backend / infra).

## 2a. Real-time subscriptions (when the change consumes RTUS topics/maps)

> Use `@mssfoobar/sse-client`; dedup on prepend. (See
> `aoh-knowledge/services/rtus.md` → "SSE client SDK" and "Initial-fetch +
> SSE dedup".)

- [ ] **PASS** — Pages that subscribe to a live topic/map use
      `@mssfoobar/sse-client`'s `SSESubscribeClient`. No hand-rolled
      `EventSource` wrapper in feature code. (Test mocks may stub the
      SDK via dependency injection.)
- [ ] **PASS** — Every page that combines an initial `GET /list` with a
      live topic of the same domain has a deduplication step at the
      prepend site (`rows.filter((r) => r.id !== incoming.id)` before
      the spread, or equivalent). Without this filter, a fetch+SSE race
      throws Svelte's `each_key_duplicate` and breaks the table render.
- [ ] **N/A** — Change has no RTUS subscription on a UI surface.

## 3. API contract

> Every AOH service produces the canonical envelope. (See
> `aoh-conventions/api.md`.)

- [ ] **PASS** — Every endpoint in `design.md`'s API surface uses
      `/v{N}/<resource>` with no `/api` prefix.
- [ ] **PASS** — Every fronting gateway proxy route uses
      `/aoh/gateway/<module>/v{N}/...` (the proxy strips `/aoh/gateway/<module>/`
      and forwards to `${host}/v{N}/...`).
- [ ] **PASS** — No spec scenario asserts a non-envelope response shape.
      The AOH envelope is `{ data, message, sent_at, errors? }` for success and
      `{ data: null, message, sent_at, errors: [{ message: "<field>: <reason>" }] }`
      for failure. Shapes like `{ error: "..." }`,
      `{ errors: { field: "msg" } }`, or `{ status: "..." }` are forbidden.
- [ ] **PASS** — `/livez` and `/readyz` are listed on every new backend
      service's API surface, mounted at root, unauthenticated.

## 4. UI surfaces

> Visual primitives come from `@mssfoobar/ui`. Every page state has a clickable
> mockup. (See `aoh-design`, `aoh-conventions/web.md`.) The authoring-time
> counterpart to these checks lives in `aoh-design`'s "Design checklist before
> marking a UI proposal apply-ready" — keep both in sync when adding rules.

- [ ] **PASS** — `openspec/changes/<change>/design/<surface>-mock.html` exists
      with a state-switcher rendering every page state (empty, filled,
      validation errors, submitting, success, permission-denied as
      applicable).
- [ ] **PASS** — Every UI-touching spec scenario references the mockup file
      OR `design.md`'s UI / Design System section explicitly enumerates the
      `@mssfoobar/ui` primitives the surface composes from.
- [ ] **PASS** — Every page route gated by application roles has either
      (a) a `roles` entry on its `nav.ts` `NavItem` (the hardcoded sidebar
      nav), OR (b) an explicit permission-denied card described in design.md's
      UI section.
- [ ] **PASS** — No design.md text proposes hand-rolling a primitive that
      `@mssfoobar/ui` already ships (Button, Input, Card, Dialog, Select,
      Table, Sidebar, Toaster, etc.). Compositions on top of primitives are
      fine; redefinitions of primitives are not.
- [ ] **PASS** — If `design.md` mandates a platform UI SDK owned by a
      UI-builder skill (e.g. `@mssfoobar/dash-web-sdk`'s `<Grid>` / `defineWidget`
      via `aoh-dashboard`), no task hand-rolls that surface from raw
      `@mssfoobar/ui` primitives instead. A design that names the SDK while
      tasks hand-roll it is a FAIL — either build on the SDK, or amend
      `design.md` to drop it (and say so explicitly) before apply. Spec and
      implementation must agree.
- [ ] **N/A** — Change has no UI surface.

## 5. Cross-artifact consistency

> The artifacts only stay useful if they agree with each other.

- [ ] **PASS** — Every capability in `proposal.md`'s `## Capabilities` section
      has a corresponding `specs/<capability>/spec.md`. Names match exactly
      (kebab-case).
- [ ] **PASS** — Every requirement in `specs/**/*.md` has at least one
      `#### Scenario:` block. Scenario headers use exactly 4 hashtags
      (not 3, not bullets).
- [ ] **PASS** — Every endpoint in `design.md`'s API surface is exercised by
      at least one spec scenario (via WHEN/THEN against the path or behavior).
- [ ] **PASS** — Every runtime dependency in `design.md` has a clear owner
      column (`Platform — added by aoh-compose`, `This change`, `Existing — modified by this change`).
      A dependency without an owner is an undefined source of failure during apply.

## 6. Verification feasibility

> Every spec scenario must be testable. Tasks.md will need to express it as a
> command — flag specs that name behaviors no command can prove.

- [ ] **PASS** — Every scenario's THEN clause asserts something observable
      from outside the service: a HTTP status, a row in a DB, a JWT claim, a
      DOM state, a log line, or a returned envelope field. Scenarios asserting
      pure internal state ("the service caches the result") need a probe.
- [ ] **PASS** — Authentication scenarios name how the test obtains a token
      (bundled `web` client + password grant against Keycloak, NOT a
      hand-minted JWT).

## 7. Tasks structure

> `tasks.md` is the implementation contract. It must order work so dependencies
> are available before they're exercised, end every section with a runnable
> verification, and use the dual compose form. (See the `tasks` artifact
> instruction in the schema.)

- [ ] **PASS** — Every numbered section ends with a concrete verification
      step naming exact command(s) (e.g. `go build ./...`,
      `golangci-lint run ./...`, `pnpm build && pnpm check && pnpm test`,
      `pnpm test:e2e <spec>`). Vague verifications like "verify it works"
      are forbidden.
- [ ] **PASS** — Every compose command uses the dual form
      `podman compose <args>` (or `docker compose <args>`). One form alone
      misleads developers using the other runtime.
- [ ] **PASS** — No task does `compose up <app>` for any custom service
      under `apps/<name>/`. Custom apps run natively via `go run` /
      `pnpm dev` against composed-up infra — `compose up` of the app
      requires a pre-built image, which the workflow does NOT establish.
- [ ] **PASS** — Section order: Scaffold (if any) → Compose runtime
      dependencies (if any) → Seed / resource creation (if any) →
      Implement `<each app>` (backend first, then frontend; producers
      before consumers) → End-to-end verification + reproducibility gate.
      No section exercises infra that a later section composes.

## 8. Skill triggering

> Skills only fire when their name appears in backticks in a task. Tasks
> without backtick'd skill names produce hand-rolled work that drifts from
> AOH conventions.

- [ ] **PASS** — Every Scaffold task references its scaffolding skill in
      backticks: `aoh-go-init` for Go services, `aoh-web-init` for SvelteKit
      frontends.
- [ ] **PASS** — The Compose section's first task references `aoh-compose`
      in backticks.
- [ ] **PASS** — The first task of every Implement section references
      `aoh-conventions` in backticks.
- [ ] **PASS** — Tasks editing IAMS authz (`roles.yaml`,
      `realm-import.json`, AAS state) reference `aoh-knowledge` in
      backticks, naming the specific reference
      (`references/services/iams.md`).
- [ ] **PASS** — UI tasks reference `aoh-design` in backticks AND name the
      mockup file path (`openspec/changes/<change>/design/<surface>-mock.html`).
- [ ] **PASS** — If the surface is built by an AOH **UI-builder skill** (a
      platform SDK for a specific surface type — currently `aoh-dashboard` for a
      `@mssfoobar/dash-web-sdk` `<Grid>` of widgets; see the UI-builder list in
      tasks.md §5.2), the frontend Implement section names that skill in
      backticks. Without it the implementer hand-rolls the surface instead of
      building on its SDK.
- [ ] **N/A** — Change has no UI surface / no Scaffold step / no Seed step /
      no UI-builder surface, as applicable.

## 9. Native dev env documentation

> The implementing agent will start each app natively against compose infra.
> Without an env-var block in `tasks.md`, it reinvents env wiring at apply
> time and gets parts wrong (port mismatch with ORIGIN, missing SDS_URL,
> wrong schema name, etc.).

- [ ] **PASS** — Each Implement section's last task before verification
      documents the env-var block needed to start the app natively. For
      Go services: `SQL_*` (HOST/PORT/USER/PASSWORD/DATABASE_NAME/SCHEMA_NAME/SSL_MODE),
      `IAMS_KEYCLOAK_HOST` + `IAMS_KEYCLOAK_PORT`, `HTTP_PORT`,
      `HTTP_ALLOWED_ORIGINS`. For SvelteKit apps: the OIDC quartet
      (`IAM_URL`, `IAM_CLIENT_ID`, `PUBLIC_DOMAIN`, `PUBLIC_COOKIE_PREFIX`),
      `ORIGIN` matching the dev port, `SDS_URL=tcp://127.0.0.1:5333`, plus
      any per-app upstream URL like `<SERVICE>_URL`.
- [ ] **PASS** — The Compose section names which ports
      `compose/compose.override.yml` exposes on localhost (typically the
      service's primary DB on 5432 and `sds-server`'s 5333). The override
      file is gitignored, not committed, but the contract is documented.

## 10. Reproducibility gate

> The gate proves the seed artifacts converge on a fresh stack. A
> malformed gate doesn't catch the drift it was meant to catch.

- [ ] **PASS** — A reproducibility-gate task exists as the LAST task of
      the E2E section. NOT a separate "Reproducibility" section.
- [ ] **PASS** — The gate uses `down -v` + `up -d` (the `-v` is essential
      — without it, seeded volumes survive and the gate proves nothing).
- [ ] **PASS** — The gate kills + restarts native processes for every app
      the change touches, using the env block(s) documented in the
      Implement sections.
- [ ] **PASS** — The gate re-runs the SAME E2E command as the previous
      task (not an abbreviated, different, or per-step variation).
- [ ] **N/A** — Change is infra-only and has no E2E spec.

## Resolution

- All items **PASS** or **N/A** → the change is apply-ready. Proceed to
  `/opsx:apply`.
- Any **FAIL** → fix the cited artifact, then re-run this lint. If the fix
  is to an upstream artifact (proposal/design/specs), regenerate
  `tasks.md` before re-linting; otherwise the rerun will still see the
  inherited error in tasks.md.
