## 1. Scaffold `dispatch-web`

- [x] 1.1 Consult the `aoh-conventions` skill (web conventions) for AOH coding conventions before implementing.
- [x] 1.2 Confirm the `@mssfoobar` registry is reachable before scaffolding: `~/.npmrc` needs `//npm.pkg.github.com/:_authToken=<token>`, since `@mssfoobar/ui` is published to GitHub Packages. Without it `pnpm install` fails `401 Unauthorized`. This is a real prerequisite for every workshop attendee — record it in the app README (task 3.8).
- [x] 1.3 Scaffold `apps/dispatch-web` (package `@mssfoobar/dispatch-web`) using the `aoh-web-init` skill.
- [x] 1.4 Add `injectWorkspacePackages: true` and `onlyBuiltDependencies` to the root `pnpm-workspace.yaml` (YAML camelCase), leaving `minimumReleaseAge`, `trustPolicy`, and `blockExoticSubdeps` untouched. `sv add tailwindcss` appends its own `onlyBuiltDependencies` block — merge them into one key or the YAML is a duplicate-key error.
- [x] 1.5 Run `pnpm install` from the repo root. If a dependency is rejected by `minimumReleaseAge: 10080`, pin it to the newest version older than 7 days — do NOT weaken the workspace setting (design.md, Risks).
- [x] 1.6 Verify: `pnpm install && pnpm -F @mssfoobar/dispatch-web build` exits 0.

## 2. Remove the authentication layer

The scaffold performs OIDC discovery in a top-level `await` at server startup, so with no
identity provider every route returns HTTP 500. There is no inert state — auth must be
removed, not bypassed (design.md, Context + D1; specs/dispatch-console).

- [ ] 2.1 Consult the `aoh-conventions` skill (web conventions) before editing scaffold files, so what replaces the auth layer still follows AOH structure.
- [ ] 2.2 Strip `src/hooks.server.ts` to a no-auth server hook: remove the top-level `await discovery(...)`, the `authHandle` (SDS client, `authenticate()`, `locals.authResult`, `originalUrl`), and the OIDC env validation. Keep `createObservabilityHandle`, the security-header handling (`FRAME_ANCESTORS`, `X_FRAME_OPTIONS`), and the `handleError` export with its `App.Error` contract.
- [ ] 2.3 Delete the auth routes: `src/routes/(public)/aoh/api/auth/` in full (login, callback, refresh, logout, context, context/[value]).
- [ ] 2.4 Delete the `(private)` route group in full — its `+layout.svelte`, `+layout.server.ts`, `getting-started/`, the `aoh/gateway/[...path]/` proxy, and `aoh/iams/api/user/`. Move `(public)/(health)/livez` and `readyz` to plain `src/routes/livez/` and `readyz/` so they survive the group removal.
- [ ] 2.5 Delete the auth and IAMS library code: `src/lib/aoh/core/provider/auth/` (`auth.ts` and `AuthProvider/`) and `src/lib/aoh/iams/`. Delete `src/lib/aoh/core/components/layout/` (`Sidebar`, `Headerbar`, `nav.ts` and their `export.ts` files) per design.md D3, plus the breadcrumb store if nothing else references it.
- [ ] 2.6 Remove the `authResult`, `clients`, and `originalUrl` fields from `App.Locals` in `src/app.d.ts`, and drop the now-unused `AuthResult` / `openid-client` imports. Leave the `App.Error` block and `HTTPResponseBody<T>` intact.
- [ ] 2.7 Remove `gateway.config.ts` and the auth/OIDC/SDS entries from `.env.template` and `.env.development` (`IAM_*`, `OIDC_ALLOW_INSECURE_REQUESTS`, `SDS_URL`, `LOGIN_DESTINATION`, `LOGIN_PAGE`). Keep `PUBLIC_DOMAIN`, `PUBLIC_COOKIE_*` if still read, `ORIGIN`, the security headers, and the OTEL entries.
- [ ] 2.8 Prune now-unused dependencies from `apps/dispatch-web/package.json` (`openid-client`, the SDS client, and any auth-only package), then re-run `pnpm install` from the repo root.
- [ ] 2.9 Verify: `cd apps/dispatch-web && pnpm check` exits 0 and `grep -ri "oidc\|authResult\|keycloak\|SdsClient" src/` returns no match outside comments.

## 3. Implement the console

- [x] 3.1 Consult the `aoh-conventions` skill (web conventions) and the `aoh-design` skill before writing the page. Open `openspec/changes/baseline-dispatch-console/design/units-console-mock.html` in a browser as the visual reference.
- [x] 3.2 No AOH UI-builder skill applies. The surface is a master-detail list/detail page, not a dashboard, so `aoh-dashboard` and `@mssfoobar/dash-web-sdk` are deliberately NOT used — see proposal.md and design.md D4. Build from `@mssfoobar/ui` primitives by subpath.
- [x] 3.3 Create `src/lib/aoh/dispatch/roster.ts` (design.md D6): export the `UnitStatus` closed union, the `FieldUnit` type, the hardcoded 4–5 unit array, and the `listUnits()` accessor returning fresh objects. Every status appears at least once and ids are unique (specs/field-unit-roster).
- [ ] 3.4 Move the console page from `src/routes/(public)/units/+page.svelte` to `src/routes/units/+page.svelte`, since the route groups are gone (design.md D2). The page body is unchanged.
- [x] 3.5 Build the units pane from `Card`/`CardHeader`/`CardTitle`/`CardContent` (`@mssfoobar/ui/card`), `ScrollArea` (`@mssfoobar/ui/scroll-area`), one `Button variant="ghost"` per unit (`@mssfoobar/ui/button`) in a `<ul>`/`<li>`, `Separator` (`@mssfoobar/ui/separator`), and `Badge variant="soft"` with `color` per design.md (`@mssfoobar/ui/badge`). Icons from `@lucide/svelte/icons/<name>`. Detail pane from `@mssfoobar/ui/card` with `aria-live="polite"`, a description list of `Call sign` / `Unit ID` / `Status`, and the 32px `mouse-pointer-click` empty state. Copy strings verbatim from design.md §6. No command control.
- [x] 3.6 Wire selection: `let selectedId = $state<string | null>(null)`, nothing preselected (design.md D7, D8), `aria-current="true"` on the selected row. No network request, no navigation.
- [ ] 3.7 Repoint the root `+layout.server.ts` redirect for `/` at `/units` (design.md D2) and confirm the removed `LOGIN_PAGE` / `LOGIN_API` imports are gone.
- [ ] 3.8 Update `apps/dispatch-web/README.md` and `AGENTS.md` to describe an app with **no** authentication: record the `~/.npmrc` token prerequisite from 1.2, state that no container is required, and replace the scaffold's `(private)`/`(public)`, gateway, `nav.ts`, and `LOGIN_DESTINATION` guidance with what is actually true of this app.
- [x] 3.9 Unit tests (`vitest`): `listUnits()` returns 4–5 units, ids are unique, every status in the closed vocabulary appears at least once, order is stable across calls, and a caller cannot mutate the baseline (specs/field-unit-roster).
- [ ] 3.10 Verify: `cd apps/dispatch-web && pnpm build && pnpm check && pnpm lint && pnpm test:unit` all exit 0.

## 4. End-to-end verification

- [x] 4.1 Add an unauthenticated Playwright project with no `storageState` and no `dependencies`, and delete the scaffold's `setup` project and `tests/e2e/auth.setup.ts` — the Keycloak login they perform no longer exists.
- [x] 4.2 Write `tests/e2e/public/units-console.spec.ts` covering the acceptance criteria against a running dev server with no containers: the page loads without an auth redirect, `/` redirects to `/units`, 4–5 unit rows render with call sign / id / status, the empty-state prompt shows before selection, clicking a unit populates the detail pane, keyboard selection works, switching selection replaces the detail content, and selection issues no network request.
- [x] 4.3 Assert the light theme path explicitly (design.md Risks — the scaffold defaults `ThemeProvider` to `mode="dark"`, so light is never seen by default and can rot).
- [ ] 4.4 Add a spec asserting the no-auth posture from specs/dispatch-console: the server starts and serves HTTP 200 with no identity provider running, and no login/logout/callback/gateway route resolves.
- [ ] 4.5 Native dev smoke / E2E — no compose stack is involved, so this is the whole run story:
  ```bash
  cd apps/dispatch-web && pnpm dev &          # native, no containers
  cd apps/dispatch-web && pnpm exec playwright test --project=public-chromium
  ```
- [ ] 4.6 Clean-clone check (this change's actual acceptance criterion — an identical, controlled start state): from a fresh clone of the branch with no containers running, `pnpm install && pnpm -F @mssfoobar/dispatch-web dev` serves the console, and opening the dev server root redirects to `/units` and renders the units list and the empty detail pane.

<!-- No "Compose runtime dependencies" or "Seed / resource creation" section: design.md
     lists zero runtime dependencies and the change seeds no external state. The
     reproducibility gate is likewise dropped — it tears and rebuilds infra, and there is
     no infra. Task 4.6 is the equivalent guarantee for a no-infra change.

     Component tests were dropped deliberately: the AOH scaffold ships vitest with no DOM
     environment and no testing-library, so they would require new devDependencies under
     the workspace's 7-day minimumReleaseAge policy. Every behaviour they would cover is
     asserted in tasks 4.2-4.3 against the real rendered page. Facilitator-confirmed. -->
