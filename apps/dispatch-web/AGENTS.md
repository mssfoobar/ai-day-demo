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

- **API quirks worth knowing** (they fail silently, not loudly):
  - `Badge` splits shape from palette: `<Badge variant="soft" color="success">`.
    Writing `variant="success"` renders a default solid badge with no error.
  - `CardTitle` renders a `<div data-slot="card-title">`, **not** a heading element —
    `getByRole('heading')` will not find it in tests.
  - Lucide icons import per-icon: `import Radio from '@lucide/svelte/icons/radio'`.

- **Svelte 5 runes only.** `$state` / `$derived` / `$props` / `$effect`; never
  `export let` or `$:`.

- **The console page is server-rendered.** In E2E tests, wait for hydration before
  clicking — the rows exist and are "actionable" before Svelte attaches handlers, and a
  click in that window does nothing. `tests/e2e/public/units-console.spec.ts` has a
  `gotoUnits()` helper that does this; use it rather than a bare `page.goto`.

- **Roster access goes through `listUnits()`** in `src/lib/aoh/dispatch/roster.ts`, never
  by importing the underlying array. That accessor is the seam a follow-up change
  repoints at a backend service without touching the page.

- **Module code** lives under `src/lib/aoh/{module}/` (here: `dispatch`), per
  `aoh-conventions`. Pages live under `src/routes/`.
