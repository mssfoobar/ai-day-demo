# Lint

Mechanical pre-apply lint over `proposal.md`, `design.md`, `specs/**/*.md`, and
`tasks.md` for `baseline-dispatch-console`.

**Result: apply-ready.** Every item is PASS or N/A. Two items failed on the first
pass and were fixed in this pass — see *Fixes applied* at the end; both fixes are
recorded there rather than silently absorbed.

The dominant N/A reason across §1, §2, §3, and §10 is *no auth surface* / *no runtime
dependency* — both sanctioned N/A reasons in the schema. They trace to a single
deliberate, facilitator-confirmed decision (design.md **D1**: scaffold the full AOH
web-base but place the console in the scaffold's existing `(public)` group so the
baseline runs with zero containers). That decision is the change's whole point, so the
N/As are load-bearing and should be read, not skimmed.

## 1. Authorization model

- [x] **N/A** — Runtime dependencies enumerating `iams-keycloak` / `iams-aas`: no auth
      surface. The change introduces zero runtime dependencies (design.md, Runtime
      dependencies) and the console route is unauthenticated by design (D1).
- [x] **PASS** — No "Keycloak role", "Keycloak permission", "realm role granting X", or
      per-app OIDC client registration for application concerns anywhere in
      `proposal.md` / `design.md` / `specs/**`. Keycloak is named only as the thing the
      baseline deliberately does *not* require (`design.md:203` D2, "an attendee opening
      `localhost` lands on a Keycloak redirect that cannot resolve") — a statement about
      a redirect target, not an authz attribution.
- [x] **PASS** — No spec scenario asserts `active_tenant.permissions`. No scenario
      asserts any JWT claim; there is no token in this change.

## 2. Session storage

- [x] **N/A** — `sds-server` + `valkey` in Runtime dependencies: no auth surface. The
      check's premise is session storage for an authenticated web app; this change ships
      no authenticated session, so there is no session to store. `SDS_URL` is checked in
      empty and unexercised (`tasks.md` 2.9). **This is the deviation a reviewer should
      confirm**: the moment the auth exercise moves the route into `(private)`, that
      change MUST add `sds-server` and `valkey` to its runtime dependencies. Recorded in
      design.md's Migration Plan step 1.
- [x] **PASS** — No text describes browser-side token storage as the target posture. The
      only cookie mentioned is the scaffold's PKCE-verifier fallback, named as unused
      (`tasks.md` 2.9).
- [x] **PASS** — Option (a): `PUBLIC_COOKIE_PREFIX=web`, the platform default, is carried
      through unchanged into the checked-in `.env.development` (`tasks.md` 2.9, with an
      inline "do not change" note citing the rtus cookie-name contract). No RTUS is
      composed, so no `cookienames` update is owed.

## 2a. Real-time subscriptions

- [x] **N/A** — No RTUS subscription on any UI surface. `rtus` was surveyed and ruled out
      in `proposal.md` ("the roster is hardcoded and static; there is no event to push").

## 3. API contract

- [x] **N/A** — `/v{N}/<resource>` paths: the change adds no API endpoint. The single row
      in design.md's API surface is `GET /units`, explicitly labelled "A SvelteKit page
      route, not an API endpoint". No `/api` prefix appears anywhere in the change
      (verified by search).
- [x] **N/A** — Gateway proxy routes: the scaffolded `(private)/aoh/gateway/[...path]`
      route is checked in but not exercised by this change (design.md, API surface).
- [x] **PASS** — No spec scenario asserts a non-envelope response shape. No scenario
      asserts a response body at all; the roster is a build-time import, not a fetch.
- [x] **N/A** — `/livez` + `/readyz` on new backend services: this change introduces no
      backend service. The scaffold's own health routes are created by `aoh-web-init` and
      left untouched (design.md, API surface).

## 4. UI surfaces

- [x] **PASS** — `design/units-console-mock.html` exists and is clickable. It renders
      every state this page has: **empty** (no selection, the initial state) and
      **filled** (a unit selected), reachable by clicking a row, plus a working
      light/dark toggle covering both themes. The states the check also lists — validation
      errors, submitting, success, permission-denied — do not exist on this surface: it
      has no form, no action, and no role gate (specs/dispatch-console, "No command
      actions in the baseline").
- [x] **PASS** — `design.md`'s UI / Design System §2 enumerates every primitive by
      subpath (`@mssfoobar/ui/card`, `/scroll-area`, `/button`, `/separator`, `/badge`)
      in a table, and §3 names the primitives each composite is built from.
- [x] **N/A** — Role-gated page routes: the console route is not gated by application
      roles; it is deliberately unauthenticated (D1).
- [x] **PASS** — No hand-rolling of a shipped primitive. design.md **D3** is explicit
      that the selectable row is a *composition* over `Button variant="ghost"` rather
      than a bespoke row component, and §3 frames the empty state and detail rows as
      compositions of `Card` + `Badge` + a Lucide glyph. `tasks.md` 2.5 repeats the
      subpath list so the contract is checkable at review.
- [x] **PASS** — Design and tasks agree on the UI-builder question. `design.md` does NOT
      mandate `@mssfoobar/dash-web-sdk`; `proposal.md` rules DASH out with a reason, and
      `tasks.md` 2.2 states the same exclusion and cites both. No task hand-rolls a
      surface that the design assigns to an SDK.

## 5. Cross-artifact consistency

- [x] **PASS** — Both capabilities in `proposal.md` have matching spec files with exactly
      matching kebab-case names: `specs/dispatch-console/spec.md`,
      `specs/field-unit-roster/spec.md`.
- [x] **PASS** — Every requirement has at least one scenario, and every scenario header
      uses exactly 4 hashtags. Verified by search: dispatch-console 6 requirements / 14
      scenarios, field-unit-roster 4 requirements / 9 scenarios, zero 3-hashtag
      `### Scenario:` headers, zero requirements without a following scenario.
- [x] **PASS** — The one endpoint in design.md's API surface (`GET /units`) is exercised
      by spec scenarios: "Console renders from a clean checkout" and "No container is
      required" (specs/dispatch-console). *Fixed this pass — see Fix 1.*
- [x] **PASS** — The Runtime dependencies table's owner column reads "n/a — this change
      introduces no runtime dependency", so the empty set is explicit rather than an
      undefined source of failure. *Fixed this pass — see Fix 1.*

## 6. Verification feasibility

- [x] **PASS** — Every THEN clause asserts something observable from outside: DOM state
      (row counts, badge labels, `aria-current`, focus ring, empty-state copy), absence
      of a network request (interceptable in Playwright), or a build result
      (`pnpm check-types` fails on an out-of-vocabulary status). The one scenario phrased
      about two people — "Roster is identical across attendees and runs" — has an
      observable THEN (same units, ids, statuses, and order), provable by reload
      comparison, which `tasks.md` 3.5 realises as a clean-clone check.
- [x] **N/A** — Token acquisition in authentication scenarios: no auth surface, so there
      are no authentication scenarios.

## 7. Tasks structure

- [x] **PASS** — Every numbered section ends with a concrete, runnable verification:
      1.6 `pnpm install && pnpm -F @mssfoobar/dispatch-web build`; 2.13
      `pnpm build && pnpm check && pnpm lint && pnpm test:unit`; 3.4 `pnpm dev` +
      `pnpm exec playwright test tests/e2e/units-console.spec.ts`, with 3.5 naming the
      clean-clone commands. No "verify it works".
- [x] **N/A** — Dual `podman compose` / `docker compose` form: `tasks.md` contains no
      compose command, because the change composes nothing.
- [x] **PASS** — No task runs `compose up` against a custom app. `apps/dispatch-web` is
      exercised natively via `pnpm dev` (3.4, 3.5), which is the convention the check
      protects.
- [x] **PASS** — Section order is Scaffold (1) → Implement `dispatch-web` (2) → End-to-end
      verification (3). The Compose and Seed sections are dropped as the schema's section
      menu allows, and the remaining sections are renumbered contiguously. No section
      exercises infra a later section composes, since none is composed.

## 8. Skill triggering

- [x] **PASS** — `tasks.md` 1.3 names `aoh-web-init` in backticks for the SvelteKit
      scaffold.
- [x] **N/A** — `aoh-compose` in a Compose section: there is no Compose section.
- [x] **PASS** — `tasks.md` 2.1, the first task of the only Implement section, names
      `aoh-conventions` in backticks. 1.1 names it as well.
- [x] **N/A** — `aoh-knowledge` on IAMS authz tasks: no task edits `roles.yaml`,
      `realm-import.json`, or AAS state.
- [x] **PASS** — `tasks.md` 2.1 names `aoh-design` in backticks AND the mockup path
      `openspec/changes/baseline-dispatch-console/design/units-console-mock.html`.
- [x] **N/A** — UI-builder skill: the surface is a master-detail list/detail page, matching
      no UI-builder skill. The one candidate (`aoh-dashboard`) is ruled out with a reason
      in `proposal.md` and `tasks.md` 2.2.

## 9. Native dev env documentation

- [x] **PASS** — `tasks.md` 2.9, the last task before the section's verification,
      documents the full env-var block needed to start the app natively: the OIDC quartet
      (`IAM_URL`, `IAM_CLIENT_ID`, `PUBLIC_DOMAIN`, `PUBLIC_COOKIE_PREFIX`), `ORIGIN`
      matched to the vite dev port, and `SDS_URL` — present and deliberately empty, with
      the reason stated. No `<SERVICE>_URL` is owed, because the console calls no backend.
      *Fixed this pass — see Fix 2.*
- [x] **N/A** — `compose.override.yml` port contract: there is no Compose section and no
      override file, because nothing is composed.

## 10. Reproducibility gate

- [x] **N/A** — All four gate checks. The schema's `tasks` instruction makes the gate
      "conditional on infra being composed — drop it for changes with no infra delta",
      and this change has no infra delta: `down -v` + `up -d` would tear and rebuild
      nothing, proving nothing. `tasks.md` 3.5 is the equivalent guarantee for a
      no-infra change — a clean clone with no containers must reach the rendered console —
      and `tasks.md` carries a trailing comment recording why the gate was dropped.

## Fixes applied this pass

Two items FAILed on the first pass and were fixed before this lint was finalised:

1. **§5.3 + §5.4 (design.md)** — the API surface table listed `GET /livez` and
   `GET /readyz`, neither exercised by any spec scenario, and the Runtime dependencies
   table used a bare `—` in the owner column. Fixed by removing the two health rows (they
   belong to the `aoh-web-init` scaffold, not to this change's surface, and a sentence
   now says so) and by making the owner cell state the empty set explicitly.
2. **§9.1 (tasks.md)** — task 2.9 pointed at the scaffold's `.env.template` instead of
   documenting the env block in `tasks.md` itself, which is exactly the omission the check
   exists to catch: `pnpm dev` runs through `env-cmd` and fails outright on a missing
   `.env.development`. Fixed by inlining the full block, with `SDS_URL` empty and
   `LOGIN_DESTINATION` realigned to `/units` to match design.md D2.

Both fixes are confined to the artifacts cited. Fix 1 touches `design.md` upstream of
`tasks.md`, so per the schema the inheritance was re-checked: it removed two rows the
tasks never referenced and reworded a table cell, changing no assumption `tasks.md`
builds on, and `tasks.md` was edited and re-linted in this same pass. Sections 7, 8, and
9 were re-verified after the edit.

## Resolution

All items **PASS** or **N/A**. The change is apply-ready — proceed to `/opsx:apply`.

Before applying, a reviewer should consciously accept two things this lint surfaced
rather than blocked:

- The **deferred-auth posture** behind the §1/§2 N/As (design.md D1), and the
  obligation it creates: the auth exercise must add `sds-server` + `valkey`.
- The **`~/.npmrc` GitHub Packages token** prerequisite (`tasks.md` 1.2). No lint check
  covers it, and it is the one thing that can still stop an attendee at
  `pnpm install` despite the zero-container design.
