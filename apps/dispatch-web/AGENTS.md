# AGENTS.md

Agent context for this SvelteKit app. App-specific architecture lives here;
monorepo-wide conventions live in the repo-root `AGENTS.md`.

- For project overview, layout, and how to run things locally, see `README.md`.
- For coding conventions and what this scaffold provides as primitives
  (`@mssfoobar/ui` design system, Svelte 5 runes, Pino logger arg order), the
  `aoh-conventions` skill auto-activates on AOH frontend code work; consult its
  `references/web.md` if not loaded — but read the exceptions below first.

## 🛑 This app has NO authentication

It was scaffolded from the AOH web-base and then had its entire auth layer **removed**
for the workshop baseline (openspec change `baseline-dispatch-console`, design.md D1).
Several `aoh-conventions/references/web.md` rules therefore do **not** apply here.
Do not "restore" these because a convention doc mentions them:

| `aoh-conventions` says                                   | Reality in this app                                               |
| -------------------------------------------------------- | ----------------------------------------------------------------- |
| Feature pages go under `src/routes/(private)/`           | There are no route groups. Pages go directly under `src/routes/`. |
| Reach backends through `(private)/aoh/gateway/[...path]` | There is no gateway and no `gateway.config.ts`.                   |
| Add a `NavItem` row to `nav.ts` for each page            | There is no `nav.ts`, no `Sidebar`, no `Headerbar`.               |
| Tokens live in SDS; `SDS_URL` is required                | No SDS, no tokens, no session.                                    |
| `LOGIN_DESTINATION` must point at a `(private)` route    | There is no login. `/` redirects to `/units`.                     |
| Read user claims from `data` in a private route          | There is no user.                                                 |

**Why removed rather than disabled:** the scaffold ran OIDC discovery in a top-level
`await` in `src/hooks.server.ts`. That executes at server startup, before routing, so
with no reachable identity provider every route returned HTTP 500 — including
unauthenticated ones. Auth here is all-or-nothing.

`tests/e2e/public/no-auth.spec.ts` asserts this posture. If you reintroduce auth, that
spec is the first thing that will fail, and it should be updated deliberately rather than
deleted.

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
  - `CardTitle` renders a `<div data-slot="card-title">`, **not** a heading element —
    `getByRole('heading')` will not find it in tests.
  - Lucide icons import per-icon: `import Radio from '@lucide/svelte/icons/radio'`.

- **Svelte 5 runes only.** `$state` / `$derived` / `$props` / `$effect`; never
  `export let` or `$:`.

- **There is no end-to-end suite.** Playwright was removed deliberately
  (`openspec/changes/dispatch-units-service`, design.md D6). `lint`, `check-types`,
  `build` and vitest are the automated checks. Don't reintroduce Playwright without
  raising it — its removal was a decision, not an oversight.

- **The roster comes from `dispatch-svc` over HTTP**, read in
  `src/routes/units/+page.server.ts` via `listUnits()` in
  `src/lib/aoh/dispatch/units.server.ts`. The `.server.ts` suffix is load-bearing:
  importing it from client code is a build error, which is what keeps `DISPATCH_SVC_URL`
  out of the browser bundle. The browser talks only to its own origin — there is no CORS
  config anywhere, and adding a client-side fetch to the service would need one.

- **Logic lives in pure modules, rendering in components.** `filters.ts` (search / filter /
  sort / counts) and `format.ts` (recency, colour maps) have no DOM and are unit-tested;
  `components/{StatusFilter,UnitRow,UnitDetail}.svelte` render them. Put a new behaviour
  in the pure module first and test it there — component tests are deliberately absent
  (no DOM test environment is configured), so untested logic inside a `.svelte` file is
  untested, full stop.

- **Types import from `types.ts`, not `units.server.ts`.** Client components must never
  import the server module — even `import type` from it is a smell, because the next
  person turns it into a value import and leaks `DISPATCH_SVC_URL` into the bundle.

- **Status summary tiles are the filter.** `StatusFilter` is a `ToggleGroup`; its counts
  are computed over the full roster on purpose. Don't "fix" them to reflect the filtered
  list.

- **An unassigned unit has no `assignment` key at all.** Branch on presence; the service
  omits it rather than sending an empty object.

- **The status vocabulary lives in the database** (a `CHECK` on `dispatch.unit.status`).
  The `UnitStatus` type is documentation, and `units.server.ts` guards unknown values at
  the wire boundary. Adding a status means a migration _and_ updating that guard.

- **Module code** lives under `src/lib/aoh/{module}/` (here: `dispatch`), per
  `aoh-conventions`. Pages live under `src/routes/`.
