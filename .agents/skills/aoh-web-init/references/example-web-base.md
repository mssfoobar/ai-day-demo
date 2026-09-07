# Web-Base Generation Guide

This reference describes the architecture, patterns, and file contents for an AOH web-base project. The base project is scaffolded using `sv create` (the Svelte CLI), then customized. Files are categorized as:
- **MODIFY** — scaffolded by `sv create`, then modified for AOH requirements
- **GENERATE** — Claude writes fresh using current framework knowledge
- **FIXED** — copy verbatim from this reference
- **ASSET** — copy from the skill's `assets/` directory

## Table of Contents

1. [Project Structure](#project-structure)
2. [Fixed Config Files](#fixed-config-files) — copy verbatim
3. [Config Modifications](#config-modifications) — modify sv-scaffolded configs for AOH
4. [Generated Source Files](#generated-source-files) — Claude generates fresh
5. [Navigation](#navigation) — pattern reference
6. [UI Components](#ui-components)

---

## Project Structure

Generate this directory structure, then populate files as described below:

```
{repo-root}/                          # Turborepo root
├── turbo.json
├── apps/
│   └── {app-name}/                   # SvelteKit application (Turbo workspace)
│       ├── package.json              # ASSET — copy from assets/, replace `{APP_NAME}`
│       │                             # depends on `@mssfoobar/ui` (published, registry pinned)
│       ├── AGENTS.md                 # ASSET — copy from assets/ (app-level agent context)
│       ├── CLAUDE.md                 # ASSET — copy from assets/ (one-line `@AGENTS.md` import)
│       ├── svelte.config.js          # ASSET — copy from assets/ (adapter-node + $root alias)
│       ├── vite.config.ts            # ASSET — copy from assets/ (overwrites sv-generated)
│       ├── tsconfig.json             # ASSET — copy from assets/ (overwrites sv-generated)
│       ├── playwright.config.ts      # ASSET — copy from assets/ (overwrites sv-generated)
│       ├── eslint.config.js          # ASSET — copy from assets/ (AOH rules and ignores)
│       ├── Dockerfile                # ASSET — copy from assets/, replace placeholders
│       ├── gateway.config.ts         # ASSET — copy from assets/web-base/ (verbatim)
│       ├── aoh.config.ts             # FIXED
│       ├── .env.template             # ASSET — copy from assets/web-base/.env.template.tmpl, substitute {APP_NAME}
│       ├── .gitignore                # ASSET — copy from assets/web-base/ (overrides sv-generated)
│       ├── .prettierrc               # ASSET — copy from assets/web-base/ (adds prettier-plugin-tailwindcss)
│       ├── .prettierignore           # .prettierignore is NOT overridden — sv add prettier produces the right file
│       ├── aoh_colors.json           # ASSET — copy from assets/
│       ├── static/                   # ASSET — copy from assets/static/
│       │   ├── favicon.ico
│       │   └── images/logo.png
│       └── src/
│           ├── app.css               # ASSET — copy from assets/
│           │                         # @imports `@mssfoobar/ui/styles/app.css`; do NOT redeclare tokens
│           ├── app.html              # GENERATE
│           ├── app.d.ts              # ASSET — copy from assets/web-base/src/ (declares App.Error)
│           ├── hooks.server.ts       # ASSET — copy from assets/
│           ├── lib/
│           │   ├── index.ts          # GENERATE
│           │   ├── utils.ts          # GENERATE
│           │   ├── hooks/
│           │   │   └── is-mobile.svelte.ts  # GENERATE
│           │   └── aoh/
│           │       ├── core/
│           │       │   ├── constants.ts                          # GENERATE
│           │       │   ├── logger/Logger.ts                      # GENERATE
│           │       │   ├── browser_broadcaster/browser_broadcaster.ts  # GENERATE
│           │       │   ├── provider/
│           │       │   │   ├── auth/
│           │       │   │   │   ├── auth.ts                       # GENERATE
│           │       │   │   │   └── AuthProvider/index.svelte     # GENERATE
│           │       │   │   └── theme/
│           │       │   │       └── ThemeProvider/
│           │       │   │           ├── index.svelte              # GENERATE
│           │       │   │           └── theme.ts                  # GENERATE
│           │       │   ├── stores/
│           │       │   │   └── breadcrumb/breadcrumb.ts          # GENERATE
│           │       │   └── components/
│           │       │       └── layout/
│           │       │           ├── nav.ts                        # GENERATE
│           │       │           ├── Headerbar.svelte              # GENERATE
│           │       │           └── Sidebar.svelte                # GENERATE
│           │       └── iams/
│           │           ├── constants.ts                          # GENERATE
│           │           └── common.ts                             # GENERATE
│           └── routes/
│               ├── +layout.svelte           # GENERATE
│               ├── +layout.server.ts        # GENERATE
│               ├── +error.svelte            # ASSET — copy from assets/web-base/src/ (shared ErrorSurface)
│               ├── (private)/
│               │   ├── +layout.svelte       # GENERATE
│               │   ├── +layout.server.ts    # ASSET — copy from assets/private-layout.server.ts
│               │   ├── getting-started/+page.svelte  # GENERATE
│               │   └── aoh/
│               │       ├── gateway/[...path]/+server.ts      # ASSET — copy from assets/
│               │       └── iams/api/user/
│               │           ├── +server.ts                    # GENERATE
│               │           └── role/+server.ts               # GENERATE
│               └── (public)/
│                   ├── +layout.server.ts    # GENERATE
│                   ├── (health)/
│                   │   ├── livez/+server.ts   # GENERATE
│                   │   └── readyz/+server.ts  # GENERATE
│                   └── aoh/
│                       ├── debug/+page.svelte  # GENERATE
│                       └── api/auth/
│                           ├── login/+server.ts      # ASSET — copy from assets/auth-routes/
│                           ├── callback/+server.ts   # ASSET — copy from assets/auth-routes/
│                           ├── refresh/+server.ts    # ASSET — copy from assets/auth-routes/
│                           ├── logout/+server.ts     # ASSET — copy from assets/auth-routes/
│                           └── context/
│                               ├── +server.ts        # ASSET — copy from assets/auth-routes/
│                               └── [value]/+server.ts  # ASSET — copy from assets/auth-routes/
└── ...
```

---

## Fixed Config Files

Copy these files verbatim into the project.

### .npmrc
```
@mssfoobar:registry=https://npm.pkg.github.com
```

### .gitignore (app)
The bundled asset at `assets/web-base/.gitignore` is what gets copied to
`apps/{app-name}/.gitignore` in Step 2. It overrides what `sv create` /
`sv add` produce so the AOH-specific entries (notably the `!.env.template`
allowlist) survive. See the asset file for the full content.

### Design system: @mssfoobar/ui (published)

`@mssfoobar/ui` is a published package on `npm.pkg.github.com` — the
AOH design system. Apps consume it as a normal versioned dep (pinned in
`assets/package.json`) and import primitives by subpath:

```ts
import { Button } from "@mssfoobar/ui/button";
import { Card } from "@mssfoobar/ui/card";
```

The package owns the design tokens — semantic aliases (`--background`,
`--primary`, …) and the AOH Figma palette (`--bg-success-strong`,
`--text-error-strong`, …) ship inside `@mssfoobar/ui/styles/app.css`,
which the app's `src/app.css` `@import`s as the first line. Apps **must
not redeclare these tokens** — overriding them in the consuming app
silently diverges the design system. To customise, follow the layered
override pattern documented in the package's stylesheet header (declare
unlayered overrides *after* the import).

**Why a published package, not a workspace package?** Versioned releases
let multiple AOH monorepos pin a known-good design-system version without
each repo carrying its own scaffolded copy. Drift across apps is
controlled by the published version pin.

### gateway.config.ts
```typescript
export type ModuleConfig = {
	host: string;
	basePath?: string;
	getHeaders?: (context: { accessToken: string; requestHeaders?: Headers }) => Promise<Headers> | Headers;
};

export type GatewayConfig = {
	modules: Record<string, ModuleConfig>;
};

export const gatewayConfig: GatewayConfig = {
	modules: {
		// Add modules as needed, e.g.:
		// dash: {
		//   host: process.env.DASH_URL || 'http://localhost:8081',
		//   basePath: '/api',
		// },
	},
};

export function resolveModule(modulePath: string): { url: string; config: ModuleConfig } | null {
	const [moduleCode, ...restPath] = modulePath.split("/");
	const moduleConfig = gatewayConfig.modules[moduleCode];
	if (!moduleConfig) return null;
	const basePath = moduleConfig.basePath || "";
	const remainingPath = restPath.join("/");
	return {
		url: `${moduleConfig.host}${basePath}/${remainingPath}`,
		config: moduleConfig,
	};
}

export function resolveModuleUrl(modulePath: string): string | null {
	return resolveModule(modulePath)?.url ?? null;
}
```

### aoh.config.ts
```typescript
// Placeholder - used by the Web Base CLI to detect project root
// eslint-disable-next-line @typescript-eslint/no-empty-object-type
interface AohWebConfig {}
export const config: AohWebConfig = {};
```

### .env.template
```
## KEYCLOAK
IAM_URL=http://iams-keycloak.127.0.0.1.nip.io/realms/aoh/.well-known/openid-configuration
IAM_CLIENT_ID=web
SDS_URL=

## MISC
PUBLIC_DOMAIN=127.0.0.1.nip.io
PUBLIC_COOKIE_PREFIX=web
PUBLIC_COOKIE_SAMESITE=lax
ORIGIN=http://127.0.0.1.nip.io:5173
OIDC_ALLOW_INSECURE_REQUESTS=1

## SECURITY HEADERS
FRAME_ANCESTORS='self'
X_FRAME_OPTIONS=SAMEORIGIN

## LOGIN
LOGIN_DESTINATION=/getting-started
LOGIN_PAGE=

## BUILD
PUBLIC_STATIC_BUILD_VERSION=next
```

### .prettierrc
```json
{
	"useTabs": true,
	"tabWidth": 4,
	"singleQuote": false,
	"trailingComma": "es5",
	"printWidth": 120,
	"plugins": ["prettier-plugin-svelte"],
	"overrides": [
		{
			"files": "*.svelte",
			"options": {
				"parser": "svelte"
			}
		}
	]
}
```

### .prettierignore
Not part of the FIXED set — `sv add prettier` (run in Step 1) generates
this file with the right content for the AOH layout. Do not overwrite it.

---

## Config Modifications

The `sv create` scaffolding produces working config files for a vanilla SvelteKit project.
These modifications customize them for AOH. Read each generated file first, then apply the
changes described below — don't overwrite them from scratch.

### package.json

**Type**: ASSET — copy from `assets/package.json`, then replace `{APP_NAME}` with the
user's app name (the file's `"name"` field is `"@mssfoobar/{APP_NAME}"`). The asset
already contains:

- AOH-owned deps (`@mssfoobar/logger`, `@mssfoobar/sds-client`, `@mssfoobar/sse-client`,
  `@mssfoobar/cli`) at the versions AOH actually ships
- Framework core pinned exact (no carets) — `svelte`, `@sveltejs/kit`,
  `@sveltejs/adapter-node`, `vite`, `tailwindcss`, `typescript`
- AOH-specific scripts (`dev` piping through `pino-pretty`, the integration/unit test
  split, etc.) and the `lint-staged` config
- `cookie@0.x` pinned alongside `@types/cookie@0.x` — the asset `auth.ts` imports the
  legacy `CookieSerializeOptions` type which only exists in cookie@0.x

Copying it overwrites the sv-generated package.json wholesale; you don't need to merge
sv-tooling deps in by hand because they're already in the asset.

> **When to refresh `assets/package.json`**: bump it deliberately when you bump
> `SV_VERSION` in SKILL.md, or when AOH ships a new `@mssfoobar/*` major. Re-probe
> `npx sv@<new-version> create` to see what tooling versions sv now ships against,
> then update `assets/package.json` to match. Don't auto-bump on every scaffold —
> the whole point of ASSET is determinism.

### svelte.config.js

**Type**: ASSET — copy from `assets/svelte.config.js`. Uses `@sveltejs/adapter-node`
with `vitePreprocess` and `$root` alias. Overwrites the sv-generated config.

**Requires devDependencies**: `@sveltejs/adapter-node`, `@sveltejs/vite-plugin-svelte`

### eslint.config.js

**Type**: ASSET — copy from `assets/eslint.config.js`. Flat config with TypeScript,
Svelte, Prettier, AOH-specific rules (`no-console: "error"`, unused vars with `_` prefix
exception), and ignores. Overwrites the sv-generated config.

**Requires devDependencies**: `@eslint/js`, `typescript-eslint`, `eslint-plugin-svelte`, `eslint-config-prettier`, `globals`

### vite.config.ts

**Type**: ASSET — copy from `assets/vite.config.ts`. Includes the AOH `allowedHosts`,
build output dir, and test config. Overwrites the sv-generated config.

### tsconfig.json

**Type**: ASSET — copy from `assets/tsconfig.json`. Extends `./.svelte-kit/tsconfig.json`
with the AOH-required settings (strict mode, ES2022 target, bundler resolution, source
maps). Overwrites the sv-generated config.

### playwright.config.ts

**Type**: ASSET — copy from `assets/playwright.config.ts`. AOH e2e test config.
Overwrites the sv-generated config.

### Dockerfile

**Type**: ASSET — copy from `assets/Dockerfile`.

The Dockerfile follows the [pnpm Docker guide](https://pnpm.io/docker):
- **`pnpm fetch`** downloads all packages to the store from the lockfile (+ root `package.json` for the corepack pnpm pin) — good layer caching
- **`pnpm install --offline`** links from store to node_modules without network
- **`pnpm deploy --prod`** creates a self-contained production deployment with only production dependencies
- **`--mount=type=cache`** on the pnpm store for fast rebuilds
- **`--mount=type=secret`** for the GitHub packages auth token

Replace placeholders when copying:
- `{APP_NAME}` — the app name (used in `pnpm --filter` and `pnpm deploy --filter`)
- `{NODE_VERSION}` — Node.js major version (read from the project's `.node-version` or use current LTS)

(pnpm needs no placeholder — corepack activates the version pinned by `packageManager` in the root `package.json`, which the Dockerfile copies before `pnpm fetch`.)

---

## Generated Source Files

Claude generates all source files under `src/` using the latest Svelte/SvelteKit APIs. Below describes what each file must do — not the exact code, because syntax evolves across framework versions.

### src/app.html

Standard SvelteKit HTML shell. Includes `%sveltekit.head%` and `%sveltekit.body%`. Set `lang="en"`.

### src/app.d.ts

**Type**: ASSET — copy from `assets/web-base/src/app.d.ts`. It is an ASSET rather
than GENERATE because `App.Error` is half of the error contract: the hook returns
that shape and `+error.svelte` reads it, so a regenerated variant that drops a
field silently breaks the error page. It declares:

- `App.Error` with `errorCode?`, `timestamp?`, `userMessage?`, `isRetryable?` and
  `trace_id?` — the conformant AOH shape `handleError` returns. All optional,
  because an *expected* error (`error(status, …)`, including SvelteKit's own 404)
  bypasses `handleError` and arrives as `{ message }` alone. `errorMessage` and
  `details` are deliberately **not** declared: they are developer-facing and must
  never reach a user-facing surface.
- `App.Locals` with:
  - `authResult: AuthResult` (from the auth provider)
  - `clients?: { oidc_config?: Configuration }` (from openid-client)
  - `originalUrl?: string`
- `HTTPResponseBody<T>` generic type with fields: `data`, `message`, `sent_at`, `errors`, `page` (pagination with `number`, `size`, `total_records`, `count`)

### src/hooks.server.ts

**Type**: ASSET — copy from `assets/hooks.server.ts`. Handles OIDC discovery on startup,
env var validation, authentication via the auth provider, SDS client integration, and
security headers (Content-Security-Policy, X-Frame-Options).

It also exports **`handleError`** — the AOH error contract's server seam. An
unhandled exception becomes `{message, errorCode, timestamp, userMessage,
isRetryable}` plus `trace_id` *only* when a span is active, and is logged exactly
once (5xx → ERROR, 4xx → WARN) because this hook is the rendering layer. Two
invariants: the exception's own message never reaches the client (the returned
`message` is the resolved `userMessage`), and `trace_id` is the active span's or
absent — never a minted UUID, never the literal `unknown`. The empty
`ERROR_CATALOGUE` at the top of that block is where an app adds its
`errorCode` → copy entries.

**Requires dependencies**: `openid-client` (v6+ functional API), `dayjs`,
`@mssfoobar/errors`, `@opentelemetry/api`
**Requires internal modules**: `$lib/aoh/core/provider/auth/auth`, `$lib/aoh/core/logger/Logger`
**Env vars used**: `IAM_URL`, `IAM_CLIENT_ID`, `OIDC_ALLOW_INSECURE_REQUESTS`, `ORIGIN`, `LOGIN_DESTINATION`, `LOGIN_PAGE`, `SDS_URL`, `FRAME_ANCESTORS`, `X_FRAME_OPTIONS`, `NODE_ENV` (private); `PUBLIC_DOMAIN`, `PUBLIC_COOKIE_SAMESITE`, `PUBLIC_COOKIE_PREFIX` (public)

### src/lib/index.ts

Empty placeholder: just an empty file or a comment like `// Reexport your entry components here`.

### src/lib/utils.ts

Utility functions:
- `cn(...inputs)`: Combines `clsx()` and `twMerge()` for Tailwind class merging
- Type helpers: `WithoutChild<T>`, `WithoutChildren<T>`, `WithoutChildrenOrChild<T>`, `WithElementRef<T, U>`

### src/lib/hooks/is-mobile.svelte.ts

A Svelte 5 reactive hook that detects mobile viewport using `window.matchMedia`. Should use Svelte 5 runes (`$state`, `$effect`).

### src/lib/aoh/core/constants.ts

Constants for context cookie management:
- `CONTEXT_COOKIE_NAME = "context_value"`
- `CONTEXT_LABEL_COOKIE_NAME = "context_label"`
- `CONTEXT_COOKIE_MAX_AGE = 60 * 60 * 24 * 7` (1 week)
- `CONTEXT_SWITCH_ENDPOINT = "/aoh/api/auth/context"`

### src/lib/aoh/core/logger/Logger.ts

Logger wrapper using `@mssfoobar/logger`:
- Import `logger` from the bare `@mssfoobar/logger` specifier (requires `^1.0.5` — earlier registry releases had a broken `main`/missing `exports` map)
- Wrap in a Svelte readable store
- Export as `log`

### src/lib/aoh/core/browser_broadcaster/browser_broadcaster.ts

Cross-tab communication utility using `BroadcastChannel` API. Used for syncing auth state (logout, token refresh) across browser tabs.

### src/lib/aoh/core/provider/auth/auth.ts

The core authentication module. This is the most complex generated file. It handles:

1. **OIDC Authorization Code Flow with PKCE** — generates code verifier/challenge, builds auth URL, handles callback
2. **Token management** — access tokens, refresh tokens, ID tokens stored as HTTP-only cookies
3. **SDS (Session Data Store) integration** — optionally stores tokens in SDS via TCP instead of cookies (for large token payloads)
4. **Cookie management** — `COOKIES_TYPE_ENUM` for cookie names, `DEFAULT_COOKIE_OPTIONS` derived from env vars (`PUBLIC_COOKIE_PREFIX`, `PUBLIC_COOKIE_SAMESITE`)
5. **Key exports**:
   - `authenticate(event)` — main auth function, checks cookies → validates → refreshes if needed
   - `onAuthenticationSuccess(event, tokens)` — sets auth cookies after successful login
   - `onAuthenticationFailed(event)` — clears auth cookies
   - `validateAccessToken(config, accessToken)` — validates JWT via userinfo endpoint
   - `getSdsClient()` — creates SDS TCP client from `SDS_URL` env var
   - `isTenantAdmin(authResult)` — role check helper
   - `AuthResult` type — `{ authenticated: boolean, claims?: UserClaims, access_token?: string }`

Uses `openid-client` v6+ API (the newer functional API, not the class-based v5 API).

### src/lib/aoh/core/provider/auth/AuthProvider/index.svelte

Svelte 5 context provider component that:
- Receives `authResult` and `claims` as props
- Sets up a periodic token refresh (calls `/aoh/api/auth/refresh`)
- Provides auth context to child components via Svelte context API
- Listens for cross-tab auth events via BroadcastChannel

When using a `<script module>` block alongside a regular `<script>` block, import types
(`AuthResult`, `AuthClaims`) **only once** — either in `module` or `<script>`, not both.
Duplicating causes `Duplicate identifier` errors under svelte-check.

### src/lib/aoh/core/provider/theme/ThemeProvider/index.svelte

Svelte 5 component that:
- Accepts `mode` prop (`"light"` | `"dark"` | `"system"`)
- Toggles `.dark` class on the document root
- Persists preference to localStorage
- Provides theme context to children

### src/lib/aoh/core/provider/theme/ThemeProvider/theme.ts

Theme management logic:
- Functions to get/set theme preference
- System theme detection via `prefers-color-scheme` media query
- Theme toggle function

### src/lib/aoh/core/stores/breadcrumb/breadcrumb.ts

Svelte store for breadcrumb navigation state. Provides functions to set/get breadcrumb items with `name` and `url`.

### src/lib/aoh/core/components/layout/nav.ts

The single source of sidebar/header navigation. Exports a `NavItem` type (`name`, `url`, optional `group`, `icon`, `roles`), a hardcoded `navItems` array, and a `filterByRoles(items, roles)` helper. Add a row to `navItems` when you add a feature page — there is no modlet auto-discovery. The scaffold default ships a single `Getting started` entry.

### src/lib/aoh/core/components/layout/Sidebar.svelte

App sidebar built on `@mssfoobar/ui`'s `Sidebar`. Reads `navItems` from `nav.ts`, groups entries by their `group` label, role-filters them against the active tenant's roles, and marks the active route. Renders inside `Sidebar.Provider` (mounted in `(private)/+layout.svelte`). Props: `title`, `description`, `roles`, `userName`, `userEmail`, `defaultGroup`.

### src/lib/aoh/core/components/layout/Headerbar.svelte

Top header built on `@mssfoobar/ui`'s `Navbar`. Renders a route-derived breadcrumb (from `nav.ts`), the mobile sidebar trigger (`MobileTrigger`), the signed-in user (avatar + name), and the **logout** action (a full-page link to `/aoh/api/auth/logout` — the UI sidebar footer has no logout control). Add app-specific header controls (e.g. a notification bell) as children before the user block.

### src/lib/aoh/iams/constants.ts and common.ts

IAMS module utilities:
- `constants.ts`: IAMS-specific constants (API paths, role names)
- `common.ts`: Shared IAMS utility functions (e.g., building IAMS API URLs)

### Routes

#### Root routes:
- `+layout.svelte` — Imports `app.css`, Geist font, wraps children in `ThemeProvider` with `mode="dark"`. Import Geist as `@fontsource-variable/geist/index.css` (explicit CSS path — `@fontsource-variable/geist` bare has no type declaration so svelte-check fails).
- `+layout.server.ts` — Redirects `/` to login page
- `+error.svelte` — **ASSET**: copy from `assets/web-base/src/routes/+error.svelte`.
  The error page, built on `@mssfoobar/ui`'s shared `ErrorSurface` (subpath
  `@mssfoobar/ui/error-surface`, added in `ui@1.1.0` — see `NOTES.md`). It renders
  the resolved `userMessage`, a Retry control only when the failure is retryable,
  and `trace_id` as a copyable support reference only when one exists. It never
  renders `page.error.message`: for an expected error that string is
  framework/developer text, so the presentation is resolved from `page.status`
  through `@mssfoobar/errors` instead. Retry calls `invalidateAll()` — it must not
  navigate or reload.

#### Private routes (require authentication):
- `(private)/+layout.svelte` — Wraps content in `AuthProvider` + `Sidebar.Provider`; renders `Sidebar` + `Headerbar` + the page `main` + `Toaster`. Derives display name / active-tenant roles / tenant name from `data.user` and passes them to the sidebar/header.
- `(private)/+layout.server.ts` — **ASSET**: copy from `assets/private-layout.server.ts`. When user is unauthenticated, stores the intended path in the `aoh_redirect_after_auth` cookie and redirects to `LOGIN_API`. The callback route reads that cookie to return the user to their original destination — this cookie is the contract between the layout and the callback, so don't regenerate either half independently.
- `(private)/getting-started/+page.svelte` — Welcome/getting started page with a logout button that navigates to `/aoh/api/auth/logout`
- `(private)/aoh/gateway/[...path]/+server.ts` — **ASSET**: copy from `assets/gateway/[...path]/+server.ts`. API gateway proxy with caching, module resolution via `gateway.config.ts`, auth header forwarding. Supports GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD. **Requires**: `http-status-codes`, `@mssfoobar/logger`, `$root/gateway.config` (needs `$root` alias from svelte.config.js).
- `(private)/aoh/iams/api/user/+server.ts` — Returns current user info from auth claims
- `(private)/aoh/iams/api/user/role/+server.ts` — Returns current user's roles

#### Public routes (no auth required):
- `(public)/+layout.server.ts` — Passes through without auth check
- `(public)/(health)/livez/+server.ts` — Returns JSON with process uptime, version from `PUBLIC_STATIC_BUILD_VERSION`, memory usage
- `(public)/(health)/readyz/+server.ts` — Returns `200 OK` when app is ready
- `(public)/aoh/debug/+page.svelte` — Debug page showing environment variables (useful during development)

#### Auth routes — all **ASSET**, copy verbatim from `assets/auth-routes/`:

The auth routes form a single cohesive flow and were previously a source of generation bugs
(swallowed redirects in catch blocks, wrong SDS client method signatures, missing PKCE wiring).
Treating them as assets removes the drift. The flow:

1. Unauthenticated user hits a private route → `(private)/+layout.server.ts` stores the current
   path in the `aoh_redirect_after_auth` cookie (10-min lifetime) and 307-redirects to `LOGIN_API`.
2. `login/+server.ts` generates a PKCE verifier/challenge, stores the verifier (SDS temp session
   if `SDS_URL` is set, otherwise a cookie), then 307-redirects to Keycloak's authorization URL.
3. Keycloak redirects back to `ORIGIN/aoh/api/auth/callback`. `hooks.server.ts` runs
   `authenticate()` which exchanges the code for tokens and sets auth cookies/session via
   `onAuthenticationSuccess`.
4. `callback/+server.ts` reads the `aoh_redirect_after_auth` cookie, deletes it, and 307-redirects
   the user to their original destination (or `LOGIN_DESTINATION` if absent).

Files:
- `login/+server.ts` — Generate PKCE, store verifier, redirect to Keycloak. Uses SDS temp session when `SDS_URL` is set.
- `callback/+server.ts` — Read and clear `aoh_redirect_after_auth`, redirect to that path or `LOGIN_DESTINATION`.
- `refresh/+server.ts` — Refresh access token via SDS (`authSessionGetAccessToken`) or via `refreshTokenGrant` + cookie writes.
- `logout/+server.ts` — SDS: `authSessionDestroy`. OIDC cookie flow: call `refreshTokenGrant` to get an `id_token` hint, then `buildEndSessionUrl` for Keycloak's end_session_endpoint.
- `context/+server.ts` — GET: return `{ context }` from the `CONTEXT_COOKIE_NAME` cookie.
- `context/[value]/+server.ts` — GET: write `CONTEXT_COOKIE_NAME` cookie with `params.value`, 307-redirect to `ORIGIN`.

---

## Navigation

Sidebar/header navigation is a single hardcoded list in
`src/lib/aoh/core/components/layout/nav.ts` — there is no modlet auto-discovery.
Add a `NavItem` row when you add a feature page; `Sidebar.svelte` and
`Headerbar.svelte` read the list directly.

```typescript
import type { Component } from "svelte";

export type NavItem = {
	name: string;
	url: string;
	group?: string; // sidebar group label; entries sharing a group render together
	icon?: Component; // optional icon component
	roles?: string[]; // active-tenant roles allowed to see this entry; omit = everyone
};

export const navItems: NavItem[] = [
	{ name: "Getting started", url: "/getting-started" },
	// Add feature pages here, e.g.:
	// { name: "Dashboard", url: "/aoh/reports/dashboard", group: "Reports", roles: ["operations-team"] },
];

export function filterByRoles(items: NavItem[], roles: string[]): NavItem[] {
	return items.filter((item) => !item.roles || item.roles.some((r) => roles.includes(r)));
}
```

When the user requests specific modules during project setup, add their
`navItems` rows and create the corresponding route pages under
`src/routes/(private)/`.

---

## UI Components

UI primitives come from the published `@mssfoobar/ui` package — the AOH
design system. Imported by subpath:

```ts
import { Button } from "@mssfoobar/ui/button";
import { Card } from "@mssfoobar/ui/card";
```

`@mssfoobar/ui@0.0.1-alpha.2` currently ships:

> alert-dialog, avatar, badge, bottom-sheet, button, calendar, card,
> checkbox, combobox, command, date-picker, date-range-picker, drawer,
> dropdown-menu, input, label, menubar, pagination, popover, range-calendar,
> scroll-area, select, separator, sheet, sidebar, skeleton, slider, spinner,
> table, tabs, textarea, toast, toggle, toggle-group, tooltip
> (plus `@mssfoobar/ui/utils` for `cn()` and `@mssfoobar/ui/hooks/...`)

If a primitive isn't yet in the package, add it to `@mssfoobar/ui`
upstream rather than forking a copy app-side — the design system is the
single source of truth.
