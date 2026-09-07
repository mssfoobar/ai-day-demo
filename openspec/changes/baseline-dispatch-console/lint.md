# Lint

Mechanical pre-apply lint over `proposal.md`, `design.md`, `specs/**/*.md`, and
`tasks.md` for `baseline-dispatch-console`.

**Result: apply-ready.** Every item is PASS or N/A.

Re-run after the change was re-scoped mid-apply: authentication is now **removed**, not
deferred, and the backend service moved to a separate change (`dispatch-units-service`).
The first pass of this lint marked §1/§2 N/A with the reason "no auth surface" while the
app still contained a dormant auth layer — that reasoning was **wrong**, because the
scaffold performs OIDC discovery in a top-level `await` at server startup, independent of
routing. The N/As below rest on the auth layer being gone from the codebase, which is
verified by `tests/e2e/public/no-auth.spec.ts` rather than asserted here.

## 1. Authorization model

- [x] **N/A** — `iams-keycloak` / `iams-aas` in Runtime dependencies: no auth surface.
      The app has no authentication (proposal.md, design.md D1) and zero runtime
      dependencies.
- [x] **PASS** — No "Keycloak role", "Keycloak permission", "realm role granting X", or
      per-app OIDC client registration for application concerns. Keycloak is named only
      as the thing that was removed and what restoring it would require.
- [x] **PASS** — No spec scenario asserts `active_tenant.permissions`. No scenario asserts
      any JWT claim; there is no token in this change.

## 2. Session storage

- [x] **N/A** — `sds-server` + `valkey`: no auth surface. SDS exists to hold auth tokens
      server-side; with no authentication there is no session and no token. The SDS client
      dependency was removed from `package.json`, and
      `tests/e2e/public/no-auth.spec.ts` asserts no session cookie is set.
- [x] **PASS** — No browser-side token storage is described anywhere; there are no tokens.
- [x] **N/A** — Cookie-prefix / rtus `cookienames` contract: the auth cookie no longer
      exists and no RTUS is composed.

## 2a. Real-time subscriptions

- [x] **N/A** — No RTUS subscription on any UI surface; the roster is static
      (proposal.md).

## 3. API contract

- [x] **N/A** — `/v{N}/<resource>` paths: the change adds no API endpoint. The single row
      in design.md's API surface is `GET /units`, labelled a SvelteKit page route. No
      `/api` prefix appears anywhere (verified by search).
- [x] **N/A** — Gateway proxy routes: the gateway was deleted with the auth layer.
- [x] **PASS** — No spec scenario asserts a non-envelope response shape; no scenario
      asserts a response body at all.
- [x] **N/A** — `/livez` + `/readyz` on new backend services: this change introduces no
      backend service. The scaffold's health routes are retained and are covered by
      `no-auth.spec.ts`.

## 4. UI surfaces

- [x] **PASS** — `design/units-console-mock.html` exists and is clickable, rendering both
      states this page has — **empty** (initial) and **filled** (unit selected) — plus a
      working light/dark toggle. Validation-error, submitting, success and
      permission-denied states do not exist on this surface: no form, no action, no role
      gate (specs/dispatch-console, "No command actions in the baseline").
- [x] **PASS** — design.md's UI / Design System §2 enumerates every primitive by subpath,
      verified against the installed `@mssfoobar/ui@1.1.0` `exports` field.
- [x] **N/A** — Role-gated page routes: no route is role-gated; there are no roles.
- [x] **PASS** — No hand-rolling of a shipped primitive. design.md D4 composes the row from
      `Button variant="ghost"`; §3 frames the empty state and detail rows as compositions.
      `tasks.md` 3.5 repeats the subpath list so the contract is checkable at review.
- [x] **PASS** — Design and tasks agree on the UI-builder question: DASH is ruled out with
      a reason in proposal.md and restated in `tasks.md` 3.2. No task hand-rolls a surface
      the design assigns to an SDK.

## 5. Cross-artifact consistency

- [x] **PASS** — Both capabilities in proposal.md have matching spec files with exactly
      matching kebab-case names: `specs/dispatch-console/`, `specs/field-unit-roster/`.
- [x] **PASS** — Every requirement has at least one scenario and every scenario header uses
      exactly 4 hashtags. Verified by search: dispatch-console 6 requirements / 17
      scenarios, field-unit-roster 4 requirements / 9 scenarios, zero 3-hashtag headers,
      zero requirements without a following scenario.
- [x] **PASS** — The one endpoint in design.md's API surface (`GET /units`) is exercised by
      spec scenarios and by `tests/e2e/public/units-console.spec.ts`.
- [x] **PASS** — The Runtime dependencies table's owner column reads "n/a — this change
      introduces no runtime dependency", so the empty set is explicit.

## 6. Verification feasibility

- [x] **PASS** — Every THEN clause asserts something observable: DOM state, an HTTP status,
      absence of a network request, absence of a cookie, or a build result. Each is realised
      by a concrete assertion in `tests/e2e/public/` or `roster.test.ts`.
- [x] **N/A** — Token acquisition in authentication scenarios: there are no authentication
      scenarios, only scenarios asserting authentication's absence.

## 7. Tasks structure

- [x] **PASS** — Every numbered section ends with a concrete verification: 1.6
      `pnpm install && pnpm -F @mssfoobar/dispatch-web build`; 2.9 `pnpm check` plus a
      residual-reference grep; 3.10 `pnpm build && pnpm check && pnpm lint && pnpm test:unit`;
      4.5/4.6 the Playwright run and the clean-clone check. No "verify it works".
- [x] **N/A** — Dual `podman compose` / `docker compose` form: `tasks.md` contains no
      compose command, because the change composes nothing.
- [x] **PASS** — No task runs `compose up` against a custom app; `apps/dispatch-web` is
      exercised natively via `pnpm dev`.
- [x] **PASS** — Section order is Scaffold (1) → Remove the authentication layer (2) →
      Implement the console (3) → End-to-end verification (4). Compose and Seed are dropped
      as the schema's section menu allows, and sections are numbered contiguously. Section 2
      is a removal step specific to this change, ordered before the console work it unblocks.
- [x] **PASS** — No section exercises infra a later section composes; none is composed.

## 8. Skill triggering

- [x] **PASS** — `tasks.md` 1.3 names `aoh-web-init` in backticks for the SvelteKit scaffold.
- [x] **N/A** — `aoh-compose` in a Compose section: there is no Compose section.
- [x] **PASS** — `tasks.md` 3.1 (Implement) and 2.1 (Remove) both name `aoh-conventions` in
      backticks, as does 1.1.
- [x] **N/A** — `aoh-knowledge` on IAMS authz tasks: no task edits `roles.yaml`,
      `realm-import.json`, or AAS state.
- [x] **PASS** — `tasks.md` 3.1 names `aoh-design` in backticks AND the mockup path
      `openspec/changes/baseline-dispatch-console/design/units-console-mock.html`.
- [x] **N/A** — UI-builder skill: the surface matches none; `aoh-dashboard` is ruled out
      with a reason in proposal.md and `tasks.md` 3.2.

## 9. Native dev env documentation

- [x] **PASS** — `tasks.md` 2.7 defines the env surface that remains after the auth removal
      (`ORIGIN`, the security headers, the OTEL entries) and names every variable removed.
      The checked-in `.env.development` and `.env.template` carry the same set with
      per-variable comments. The OIDC quartet and `SDS_URL` the schema normally requires are
      deliberately absent: the code that read them is gone.
- [x] **N/A** — `compose.override.yml` port contract: there is no Compose section and no
      override file.

## 10. Reproducibility gate

- [x] **N/A** — All four gate checks. The schema's `tasks` instruction makes the gate
      "conditional on infra being composed — drop it for changes with no infra delta", and
      this change has no infra: `down -v` + `up -d` would tear and rebuild nothing.
      `tasks.md` 4.6 is the equivalent guarantee — a clean clone with no containers must
      reach the rendered console — and it was executed, not merely written: a fresh clone
      installed in 11.8s and all 14 Playwright specs passed.

## Resolution

All items **PASS** or **N/A**. The change is apply-ready and, in fact, applied — every
task in `tasks.md` is complete and verified.

Two things a reviewer should consciously accept:

- **The auth layer was deleted from a scaffold the `aoh-web-init` skill says to copy
  verbatim.** This is the change's main review risk. It is enumerated in design.md D1, in
  the app README, and in the app `AGENTS.md`, and it is guarded by
  `tests/e2e/public/no-auth.spec.ts`. The reason it is not optional is recorded in
  design.md's Context: OIDC discovery ran in a top-level `await`, so an unauthenticated
  route group did not escape it.
- **The `~/.npmrc` GitHub Packages token** remains a per-developer prerequisite
  (`tasks.md` 1.2). No lint check covers it, and it is the only thing that can still stop
  an attendee at `pnpm install`.
