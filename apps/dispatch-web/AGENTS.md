# AGENTS.md

Agent context for this SvelteKit app. App-specific architecture lives here;
monorepo-wide conventions live in the repo-root `AGENTS.md`.

- For project overview, layout, and how to run things locally, see `README.md`.
- For coding conventions and what this scaffold provides as primitives
  (`@mssfoobar/ui` design system, Svelte 5 runes, Pino logger arg order), the
  `aoh-conventions` skill auto-activates on AOH frontend code work; consult its
  `references/web.md` if not loaded — but read the exceptions below first.

## Authentication is back, and it is all-or-nothing

The baseline stripped this app's auth layer; `dispatch-iams-and-unit-map` restored it
whole. `aoh-conventions/references/web.md` now applies as written — `(private)` route
groups, SDS-held tokens, `nav.ts`, claims from `data`. The one rule that still does **not**
apply is the gateway (see below).

**Why "all-or-nothing" is the load-bearing phrase:** `src/hooks.server.ts` runs OIDC
discovery in a **top-level `await`**. That executes at server startup, before routing, so
with no reachable `iams-keycloak` _every_ route returns 500 — `/livez` included. There is
no "auth present but inert" state. Two consequences worth knowing before you debug
anything here:

- `pnpm dev` needs the compose stack up. A 500 on every route means Keycloak, not your code.
- `OIDC_ALLOW_INSECURE_REQUESTS=1` is required over plain http. Without it that same
  discovery throws at startup.

- **The route guard is `(private)/+layout.server.ts`, not the hooks handle.** The handle
  only _resolves_ the session onto `locals`. Moving the redirect into it would sweep
  `/livez`, `/readyz` and the `(public)/aoh/api/auth/*` routes behind the sign-in flow —
  Kubernetes probes them without credentials, and the auth routes have to be reachable to
  a visitor who is, by definition, not yet signed in.
- **No gateway proxy, deliberately.** `gateway.config.ts` and
  `(private)/aoh/gateway/[...path]` are NOT restored. Nothing here calls `dispatch-svc`
  from the browser, and `aoh-conventions` forbids routing a server `load` through the
  proxy anyway (the proxy reads a bearer a server sub-request does not carry, so it 401s
  before forwarding). Do not restore it by reflex because a convention doc mentions it;
  restore it when the first browser-side call appears.
- **Tokens never reach the browser.** They live in SDS; the browser holds only
  `web_auth_session_id`. Anything returned from a `load` is serialised into the page
  payload, so a token must never be returned from one — `bearerFrom(locals)` is
  server-only, and `(private)/+layout.server.ts` returns claims, not the token.
- **Roles are AAS tenant roles, and there is no `active_tenant.permissions` claim.**
  `src/lib/aoh/dispatch/permissions.ts` projects `active_tenant.roles` onto read/write. It
  mirrors `compose/iams/init/project-aas/roles.yaml`, and its Go twin is
  `apps/dispatch-svc/internal/service/permissions.go` — all three move together. Code that
  gates on a resolved permission list hides every control from every legitimate user.
- **Hiding a control is not a permission check.** The service enforces the same rule
  independently; the console's `canWrite` decides what to _render_, and a 403 coming back
  anyway renders the permission-denied state.

## The map

- **It is SDK-owned.** `/aoh/dispatch/map` composes `@mssfoobar/gis-web-sdk`. Never
  hand-roll a map renderer, tile client, entity layer or marker from `@mssfoobar/ui`
  primitives.
- **Import components from their SUBPATHS, as default exports.** Each subpath resolves to
  one `.svelte` file. Going through the barrel would also drag the Cesium engine into
  every module that touches the SDK — including the root layout, which SSRs, and Cesium
  touches browser globals at module init.
- **`+page.ts` exports `ssr = false`** for the same reason. Do not remove it.
- **Cesium's runtime assets are copied into `static/cesium/`** by a `vite.config.ts`
  plugin, on `buildStart` **and** `configureServer` — a plugin hooked only to the build
  leaves `pnpm dev` with a black canvas and 404s under `/cesium/`. The directory is
  gitignored and lint-ignored; it is vendor output.
- **`svelte.config.js` uses a custom TypeScript preprocessor, not
  `vitePreprocess({ script: true })`.** Read the comment there before changing it: the SDK
  ships `.svelte` containing TypeScript, Svelte's native stripping cannot handle its
  optional parameters, and vitePreprocess's script half then deletes imports a block does
  not itself use — which silently breaks `GisProvider` at SSR time.
- **Entity state has ONE source: the SDK's RTUS subscription.** The map page loads the unit
  roster too, but only for the counts, and never merges the two. Adding a second entity
  source reintroduces a fetch/SSE race that needs dedup at the prepend site.
- **The marker snippet must stay render-pure.** Accumulating state inside it throws
  `state_unsafe_mutation` and takes the whole layer down — the symptom looks like "the
  layer won't enable". An event handler is fine; it runs on click, not during render.
- **The console is served on `${DEV_DOMAIN}`, not localhost.** The session cookie must
  reach `rtus-seh.${DEV_DOMAIN}`, and that origin is the one rtus-seh's CORS list already
  permits. `vite.config.ts`'s `allowedHosts` is the other half of that choice.

## Architecture (load-bearing)

- **Design-system primitives** ship via `@mssfoobar/ui` — a published package on
  `npm.pkg.github.com`, NOT a workspace package. Import primitives by subpath:
  `import { Button } from "@mssfoobar/ui/button"`. The package's `styles/app.css` is
  `@import`-ed at the top of this app's `src/app.css` and owns every theme token
  (`--background`, `--primary`, `--bg-*`, `--text-*`, …); do **not** redeclare those
  tokens. Custom overrides go after the import as unlayered CSS.

  > **Before writing any `.svelte` file, read `aoh-conventions` → `references/web.md`.**
  > It is a hard rule that visual primitives (buttons, inputs, cards, dialogs, tables, …)
  > come from `@mssfoobar/ui`, not raw `<button>` / `<input>` / `<div class="rounded
border">`. Hand-rolling silently bypasses theme tokens and dark mode.
  > To see what the installed version ships:
  > `cat node_modules/@mssfoobar/ui/package.json | jq .exports`.

- **Tailwind only scans this app because `src/app.css` says so.** The
  `@import 'tailwindcss'` lives inside `@mssfoobar/ui/styles/app.css`, so Tailwind v4
  roots its automatic content detection at the _package_ (its own `@source "../"`), not
  here. `src/app.css` therefore carries an explicit `@source './'`. **Do not remove it**,
  and if you add source outside `src/`, add a `@source` for it. Without it a utility this
  app uses but the package doesn't is emitted into the DOM with no CSS rule behind it —
  the class is present, the token is defined, and nothing renders. That is how the
  selected-row highlight (`bg-accent`) first shipped invisible.

- **API quirks worth knowing** (they fail silently, not loudly):
  - `Badge` splits shape from palette: `<Badge variant="soft" color="success">`.
    Writing `variant="success"` renders a default solid badge with no error.
  - `CardTitle` renders a `<div data-slot="card-title">`, **not** a heading element.
  - Lucide icons import per-icon: `import Radio from '@lucide/svelte/icons/radio'`.

- **Svelte 5 runes only.** `$state` / `$derived` / `$props` / `$effect`; never
  `export let` or `$:`.

- **There is no automated test suite** — no Playwright, no vitest — by decision. `lint`,
  `check-types` and `build` are the checks. Don't reintroduce a test framework without
  raising it; the removal was a decision, not an oversight.

- **The roster comes from `dispatch-svc` over HTTP**, read in
  `src/routes/(private)/aoh/dispatch/units/+page.server.ts` via `listUnits()` in
  `src/lib/aoh/dispatch/units.server.ts`. The `.server.ts` suffix is load-bearing:
  importing it from client code is a build error, which is what keeps `DISPATCH_SVC_URL`
  and the bearer out of the browser bundle. The browser talks only to its own origin —
  there is no CORS config anywhere, and adding a client-side fetch to the service would
  need one.
- **An empty roster is a state, not an empty list — and its wording must stay
  seed-state-independent.** A tenant is empty either because no dispatcher has triggered
  the seed yet or because a dispatcher deleted every unit. Nothing in the API tells the two
  apart, and for the second "a dispatcher will populate it" is false. Vary the role-gated
  action, never the statement.
- **A write is a REPLACE, so the edit form round-trips the unit's position through hidden
  inputs** (`positionLon`/`positionLat`/`positionAt`). Positions are not authored here —
  but omitting them would clear the fix and delete the unit's marker from the map. `at`
  rides along too, so an edit that reported no new location leaves the fix time where it
  was instead of looking like a fresh GPS report.

- **Logic lives in pure modules, rendering in components.** `filters.ts` (search / filter /
  sort / counts), `format.ts` (recency, colour maps) and `forms.ts` (form parsing) have no
  DOM; `components/{StatusFilter,UnitRow,UnitDetail,UnitForm}.svelte` render them. Put a
  new behaviour in the pure module first.

- **Types import from `types.ts`, not `units.server.ts`.** Client components must never
  import the server module — even `import type` from it is a smell, because the next
  person turns it into a value import and leaks `DISPATCH_SVC_URL` into the bundle.

- **Status summary tiles are the filter.** `StatusFilter` is a `ToggleGroup`; its counts
  are computed over the full roster on purpose. Don't "fix" them to reflect the filtered
  list.

- **Writes go through form actions, never a client fetch.** `+page.server.ts` has
  `create` / `update` / `delete`; the forms use `use:enhance`. That keeps the browser
  same-origin (no CORS, no service URL in the bundle) and lets validation come back as
  `fail(400, { errors })` rendered beside the field. `parseUnitForm` in `forms.ts` is the
  console's first-pass validation; the service validates again.
- **Always echo `occLock`.** Edit and delete forms carry a hidden `occLock` from the unit
  the user is looking at. The service answers 409 `DISPATCH_UNIT_STALE` if someone else
  wrote first, which the action turns into a form-level message. Don't "fix" a 409 by
  re-reading and retrying silently — the user needs to see that their view was stale.
- **Field-error UI is a sibling `<p class="text-destructive text-xs">`.** `@mssfoobar/ui`
  ships no `FormMessage`; this is the documented composition (`aoh-conventions/web.md`).
- **`<Toaster />` is mounted in `+layout.svelte`.** Without it every `toast.*()` call is a
  silent no-op and a write looks like nothing happened.

- **An unassigned unit has no `assignment` key at all.** Branch on presence; the service
  omits it rather than sending an empty object.

- **The status vocabulary lives in the database** (a `CHECK` on `dispatch.unit.status`).
  The `UnitStatus` type is documentation, and `units.server.ts` guards unknown values at
  the wire boundary. Adding a status means a migration _and_ updating that guard.

- **Module code** lives under `src/lib/aoh/{module}/` (here: `dispatch`), per
  `aoh-conventions`. Pages live under `src/routes/`.
