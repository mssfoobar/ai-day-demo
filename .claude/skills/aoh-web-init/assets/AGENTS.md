# AGENTS.md

Agent context for this SvelteKit app. App-specific architecture lives
here; monorepo-wide conventions live in the repo-root `AGENTS.md`.

- For project overview, layout, and how to run things locally, see `README.md`.
- For coding conventions AND what this scaffold provides as primitives
  (`@mssfoobar/ui` design system, Svelte 5 runes, sidebar navigation, gateway routes,
  OIDC flow gotchas, Pino logger arg order), the `aoh-conventions` skill
  auto-activates on AOH frontend code work; consult its `references/web.md`
  if not loaded.

## Architecture (load-bearing)

- **Design-system primitives** ship via `@mssfoobar/ui` — a published package
  on `npm.pkg.github.com`, NOT a workspace package. Import primitives by
  subpath: `import { Button } from "@mssfoobar/ui/button"`. The package's
  `styles/app.css` is `@import`-ed at the top of this app's `src/app.css`
  and owns every theme token (`--background`, `--primary`, `--bg-*`,
  `--text-*`, …); do **not** redeclare those tokens in `app.css`. Custom
  overrides go after the import as unlayered CSS.

  > 🛑 **Before writing any `+page.svelte` or `*.svelte` component, read
  > `aoh-conventions` → `references/web.md`.** It is a hard rule that
  > visual primitives (buttons, inputs, cards, dialogs, tables, …) come
  > from `@mssfoobar/ui`, not raw `<button>` / `<input>` / `<div class="rounded
  > border">`. Hand-rolling silently bypasses theme tokens and dark mode.
  > To see what's available in the version you have installed:
  > `cat node_modules/@mssfoobar/ui/package.json | jq .exports`. For a
  > worked import example see the `getting-started` template that ships
  > with the scaffold (composes `Card` + `Button` by subpath).
- **Sidebar navigation** is a hardcoded list in
  `src/lib/aoh/core/components/layout/nav.ts`. Add a `NavItem` row there (with an
  optional `roles` gate) when you add a feature page — the `Sidebar` and
  `Headerbar` read that list directly. No modlet auto-discovery.
- **Backends are reached only through the gateway** at
  `(private)/aoh/gateway/[...path]/+server.ts` — never call backends directly from
  the browser. Configure backend module routing in `gateway.config.ts`.
- **Auth flow** (OIDC PKCE + refresh) is scaffold-provided across the files under
  `(public)/aoh/api/auth/` and `(private)/+layout.server.ts`. Don't reimplement.
  Read user claims from `data` in private routes' `+page.server.ts`.
- **The `/` route is a login-stub, not a real page.** The root
  `+layout.server.ts` unconditionally redirects bare `/` to `LOGIN_PAGE`
  (defaults to `/aoh/api/auth/login`); after PKCE + callback, the user is
  sent to **`LOGIN_DESTINATION`**. Two rules follow:
  1. Put your real landing page at a named subpath
     (`(private)/board/+page.svelte`, `(private)/aoh/<module>/+page.svelte`, …)
     — NOT at `(private)/+page.svelte`. The root redirect would shadow it.
  2. Update `LOGIN_DESTINATION` (in `.env.development` AND
     `compose/<app>/compose.yml`) to point at that route. **Never set it to `/`**
     — that creates an infinite redirect loop (`/` → login → callback → `/` → …)
     which Chrome surfaces as `ERR_TOO_MANY_REDIRECTS` on the Keycloak host.
