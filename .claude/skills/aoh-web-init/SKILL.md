---
name: aoh-web-init
description: >
  Scaffold a new AOH web-base SvelteKit project inside a Turborepo monorepo — uses sv create
  to scaffold the base project, then customizes configs and generates AOH source files for
  OIDC authentication with Keycloak, a default sidebar/header layout, AOH UI components from
  `@mssfoobar/ui`, and Tailwind CSS v4. Use this skill whenever the user wants to create a new web-base project,
  set up web-base, initialize an AOH web app, bootstrap a web frontend with Keycloak auth, or
  scaffold a SvelteKit frontend. Also trigger when the user says "create web-base",
  "new web-base project", "set up web-base", "init web-base", or "scaffold web-base".
compatibility: Requires Turborepo (turbo CLI), Node.js, pnpm, and sv CLI (npx sv)
allowed-tools: Read Write Edit Bash(turbo:*) Bash(pnpm:*) Bash(npx:*) Bash(cp:*) Bash(echo:*) Bash(rm:*) Bash(mv:*) Glob Grep
---

# Web-Base Init

Scaffold a complete AOH web-base SvelteKit project inside a Turborepo monorepo. Uses `sv create`
to scaffold the base project with up-to-date configs, then customizes for AOH. Infrastructure
setup (Docker Compose, Keycloak, etc.) is managed separately.

The project is a SvelteKit app with OIDC authentication (Keycloak), a default sidebar/header
layout driven by a hardcoded `nav.ts`, AOH UI components from `@mssfoobar/ui`, and Tailwind CSS v4.

## References

- `references/example-web-base.md` — complete generation guide with project structure,
  file categories (GENERATE, FIXED, ASSET), and what each file must do.

Files are categorized as:

- **MODIFY** — Scaffolded by `sv create`, then modified for AOH requirements. Read the
  generated file first, then apply the changes described in the reference — don't
  overwrite from scratch.
- **GENERATE** — Claude writes fresh using current framework knowledge (Svelte, SvelteKit,
  Vite, TypeScript, openid-client). The reference describes _what_ each file must do, not
  the exact code — because framework syntax evolves across versions.
- **FIXED** — Copy verbatim from the reference file.
- **ASSET** — Copy from `assets/`. Binary or large files: `app.css`, `aoh_colors.json`,
  `static/favicon.ico`, `static/images/logo.png`, `AGENTS.md`, `CLAUDE.md`.

## Gather inputs

Ask the user for the following. If already provided in their prompt, use those values.

1. **App name** (required) — directory name and package.json name. Must be lowercase,
   hyphen-separated (e.g. `my-web-app`).
2. **Description** (optional) — one-line project description.

## Steps

### Step 1. Scaffold with sv create

Use the Svelte CLI to scaffold the base SvelteKit project. This ensures config files
match the latest Svelte/SvelteKit/Vite versions instead of generating them from scratch.

`sv` is pinned to a specific version below (currently `0.15.1`). It's still pre-1.0,
so its flag names move between minor releases — leaving the version unpinned means a
silent breakage the next time someone runs the skill against a newer `sv`. Bump
`SV_VERSION` deliberately when you want to ride a new release, and re-test the flags.

The flags below pin every interactive choice so the CLI never prompts — leaving any
option implicit makes `sv` fall back to a TTY prompt the agent can't answer:

- `--no-dir-check` / `--no-download-check` / `--no-git-check` — skip the "directory
  not empty?", "download package?", and "git working tree dirty?" prompts.
- `addon=opt:val` syntax for `tailwindcss` and `vitest` — these add-ons declare
  options (`plugins`, `usages`), and `sv add` prompts whenever any option is left
  unspecified, even if you want the default. Pass them explicitly to silence it.
- `npx --yes` — auto-confirm npx's "package not found, install?" prompt on cold caches.

```bash
SV_VERSION=0.15.1
npx --yes sv@$SV_VERSION create apps/{app-name} \
    --template minimal --types ts \
    --no-install --no-add-ons \
    --no-dir-check --no-download-check
```

Then add the required tooling add-ons (same pinned `sv`):

```bash
cd apps/{app-name}
npx --yes sv@$SV_VERSION add prettier --no-install --no-git-check --no-download-check
npx --yes sv@$SV_VERSION add eslint --no-install --no-git-check --no-download-check
npx --yes sv@$SV_VERSION add 'tailwindcss=plugins:none' --no-install --no-git-check --no-download-check
npx --yes sv@$SV_VERSION add 'vitest=usages:unit,component' --no-install --no-git-check --no-download-check
npx --yes sv@$SV_VERSION add playwright --no-install --no-git-check --no-download-check
```

> Pinning `sv` pins the scaffolding *generator*, not every dep inside the generated
> `package.json` — those entries can still use carets and resolve forward on
> `pnpm install`. If you need fully reproducible deps, lean on a checked-in
> `pnpm-lock.yaml` with `--frozen-lockfile` rather than expecting `sv` to do it.

> If `sv create` fails because `apps/{app-name}` already exists (e.g. from a prior
> turbo workspace step), remove the directory first and rerun. The turbo workspace
> registration is handled by pnpm-workspace.yaml's `apps/*` glob.

### Step 2. Copy fixed and asset files

Read `references/example-web-base.md` and:

1. **Copy all FIXED files** from the "Fixed Config Files" section — these are verbatim
   copies that overwrite or add to the scaffolded project
2. **Copy ASSET files** (these overwrite sv-generated equivalents where applicable):
   - `assets/package.json` -> `apps/{app-name}/package.json` (replace the placeholder package name `aoh-app-name` with the user's `{app-name}`, so `"name"` becomes `"@mssfoobar/{app-name}"`. The placeholder uses a schema-valid form on purpose — `{APP_NAME}` would fail npm's name regex and trip JSON-schema linters. The file already contains the AOH `@mssfoobar/*` deps, framework pins, scripts, and the `packageManager` pin)
   - `assets/AGENTS.md` -> `apps/{app-name}/AGENTS.md` (verbatim — this is the app-level agent-context source of truth, loaded cross-tool whenever someone works inside the new app, so it carries the AOH architecture notes future feature work needs: design-system import discipline (`@mssfoobar/ui` primitives, theme-token rules), sidebar navigation (`nav.ts`), gateway.config registration, logger import path, the `no-console` lint rule, runes vs reactive syntax)
   - `assets/CLAUDE.md` -> `apps/{app-name}/CLAUDE.md` (verbatim — a one-line `@AGENTS.md` import so Claude Code picks up the AGENTS.md content; never edit this file directly, edit `AGENTS.md`)
   - `assets/app.css` -> `apps/{app-name}/src/app.css`
   - `assets/aoh_colors.json` -> `apps/{app-name}/aoh_colors.json`
   - `assets/svelte.config.js` -> `apps/{app-name}/svelte.config.js`
   - `assets/eslint.config.js` -> `apps/{app-name}/eslint.config.js`
   - `assets/vite.config.ts` -> `apps/{app-name}/vite.config.ts`
   - `assets/tsconfig.json` -> `apps/{app-name}/tsconfig.json`
   - `assets/playwright.config.ts` -> `apps/{app-name}/playwright.config.ts` (defines a `setup` project that performs the Keycloak login and a `chromium` project that reuses its `storageState`, so every `(private)`-page E2E runs authenticated without per-spec login wiring)
   - `assets/tests/e2e/auth.setup.ts` -> `apps/{app-name}/tests/e2e/auth.setup.ts` (the `setup` project above; OIDC PKCE login via the bundled `web` client, saves `playwright/.auth/user.json`. Feature specs go under `tests/e2e/*.spec.ts` and inherit the authenticated state)
   - `assets/hooks.server.ts` -> `apps/{app-name}/src/hooks.server.ts` (verbatim — wraps the auth handle with `@mssfoobar/observability`'s `createObservabilityHandle` via `sequence(...)`, so the server span is named by route. It also exports **`handleError`**, the AOH error contract's server seam: an unhandled exception becomes `{errorCode, timestamp, userMessage, isRetryable}` plus `trace_id` *only* when a span is active, logged exactly once (5xx → ERROR, 4xx → WARN). The exception's own message never reaches the client, and `trace_id` is never a minted UUID or the literal `unknown`. Pair it with the `App.Error` declaration in `src/app.d.ts` and `src/routes/+error.svelte` — the three are one unit, so don't copy one without the others)
   - `assets/instrumentation.server.ts` -> `apps/{app-name}/src/instrumentation.server.ts` (substitute `{APP_NAME}` → `{app-name}`, giving `service.name = web-{app-name}`). The OpenTelemetry bootstrap (born instrumented): SvelteKit runs it before any other server module. Gated on `experimental.instrumentation.server: true` in `svelte.config.js` (already set in the asset) + `@sveltejs/adapter-node` >= 5.3.0 (the asset `package.json` pins a compatible version) — SvelteKit throws at build if the file exists without the flag/adapter support. No-op until `OTEL_EXPORTER_OTLP_ENDPOINT` is set.
   - `assets/auth.ts` -> `apps/{app-name}/src/lib/aoh/core/provider/auth/auth.ts`
   - `assets/gateway/[...path]/+server.ts` -> `apps/{app-name}/src/routes/(private)/aoh/gateway/[...path]/+server.ts`
   - `assets/private-layout.server.ts` -> `apps/{app-name}/src/routes/(private)/+layout.server.ts`
   - `assets/auth-routes/login/+server.ts` -> `apps/{app-name}/src/routes/(public)/aoh/api/auth/login/+server.ts`
   - `assets/auth-routes/callback/+server.ts` -> `apps/{app-name}/src/routes/(public)/aoh/api/auth/callback/+server.ts`
   - `assets/auth-routes/refresh/+server.ts` -> `apps/{app-name}/src/routes/(public)/aoh/api/auth/refresh/+server.ts`
   - `assets/auth-routes/logout/+server.ts` -> `apps/{app-name}/src/routes/(public)/aoh/api/auth/logout/+server.ts`
   - `assets/auth-routes/context/+server.ts` -> `apps/{app-name}/src/routes/(public)/aoh/api/auth/context/+server.ts`
   - `assets/auth-routes/context/[value]/+server.ts` -> `apps/{app-name}/src/routes/(public)/aoh/api/auth/context/[value]/+server.ts`
   - `assets/static/favicon.ico` -> `apps/{app-name}/static/favicon.ico`
   - `assets/static/images/logo.png` -> `apps/{app-name}/static/images/logo.png`
   - `assets/Dockerfile` -> `apps/{app-name}/Dockerfile` (replace `{APP_NAME}`, `{NODE_VERSION}` → `22`; pnpm needs no substitution — corepack resolves it from the root `package.json`'s `packageManager` pin at build time)
   - `assets/web-base/gateway.config.ts` -> `apps/{app-name}/gateway.config.ts` (verbatim — Svelte's `$root` alias resolves to the app root; the gateway `+server.ts` imports `$root/gateway.config`)
   - `assets/web-base/.env.template.tmpl` -> `apps/{app-name}/.env.template` (substitute `{APP_NAME}` → `{app-name}`; the resulting `.env.template` is what Step 5 copies to `.env.development`)
   - `assets/web-base/README.md.tmpl` -> `apps/{app-name}/README.md` (substitute `{APP_NAME}` → `{app-name}` and `{APP_DESCRIPTION}` → `{description}`)
   - `assets/web-base/.gitignore` -> `apps/{app-name}/.gitignore` (overwrites the sv-generated one. The AOH version adds `!.env.template` / `!.env.**.template` allowlists so the env template above isn't silently ignored, plus `.aoh`, `playwright-report`, `allure-*`, and `coverage`. `.prettierignore` is not overridden — `sv add prettier` produces the right file already)
   - `assets/web-base/.prettierrc` -> `apps/{app-name}/.prettierrc` (overwrites the one `sv add prettier` writes, which sets `tailwindStylesheet: ./src/routes/layout.css` — but Step 2 deletes that file as an sv straggler, leaving prettier-plugin-tailwindcss to error `ENOENT` on every file. The AOH version points at `./src/app.css`.)

The auth routes and `(private)/+layout.server.ts` work as a single cohesive flow: the private
layout stores the intended path in `aoh_redirect_after_auth` when an unauthenticated user hits
a private route, the login route generates PKCE and redirects to Keycloak, and the callback
reads that cookie to send the user to their original destination. Copy them verbatim — these
were previously GENERATE but the generated versions kept diverging from the working flow
(e.g. swallowing redirects in catch blocks, using wrong SDS client method signatures).

3. **Remove sv-create stragglers** — `sv add` for `playwright`, `vitest`, and `tailwindcss`
   leaves demo/example files at paths the AOH layout uses for real routes/code. They'd
   shadow our output and svelte-check would error on them (vitest's example imports
   `vitest-browser-svelte`, which we don't depend on). Delete them once the assets
   are in place:

   ```bash
   cd apps/{app-name}
   rm -f src/routes/+page.svelte src/routes/layout.css
   rm -rf src/routes/demo src/lib/vitest-examples
   ```

   The sv-generated `src/routes/+layout.svelte` is intentionally overwritten by the
   reference implementation's version (it sets up `ThemeProvider` and imports the
   AOH `app.css`).

### Step 3. Generate source files

`package.json`, `vite.config.ts`, `tsconfig.json`, `playwright.config.ts`,
`svelte.config.js`, and `eslint.config.js` were all already overwritten as ASSETs in
Step 2 — the AOH versions win over what `sv create` produced. No config merge needed
at this stage.

Using the reference guide's "Generated Source Files" section:

1. **Copy the reference src tree** — Source files (AuthProvider,
   ThemeProvider, IAMS routes, layout components, etc.) are bundled at
   `assets/web-base/src/`. Bulk-copy them into the app — they're already written
   against current Svelte 5 runes (`$props`, `$state`, `$effect`, `$derived`),
   SvelteKit, and `openid-client` v6+ functional API. Re-generating them from
   scratch is what introduces the "wrong SDS method signature / swallowed redirect"
   bugs that pushed us to make them assets in the first place.

   ```bash
   # cp -R copies the directory contents (note the trailing /. on the source).
   cp -R .claude/skills/aoh-web-init/assets/web-base/src/. apps/{app-name}/src/
   ```

   After the bulk copy, re-overwrite the canonical ASSETs from Step 2 — `auth.ts`,
   `hooks.server.ts`, the gateway `+server.ts`, `(private)/+layout.server.ts`, and
   the `auth-routes/` files — because the flat `assets/` versions are kept in sync
   while `assets/web-base/src/` is a frozen reference snapshot. (If the two ever
   diverge, the flat assets win.)

2. **Rename the `getting-started` template** — `assets/web-base/src/routes/(private)/getting-started/+page.svelte.tmpl`
   has `.tmpl` so it doesn't collide with `sv create` if invoked accidentally.
   Substitute placeholders and rename to `+page.svelte`:

   ```bash
   sed -e 's/{APP_NAME}/{app-name}/g' \
       -e 's/{APP_DESCRIPTION}/{description}/g' \
       apps/{app-name}/src/routes/'(private)'/getting-started/+page.svelte.tmpl \
       > apps/{app-name}/src/routes/'(private)'/getting-started/+page.svelte
   rm apps/{app-name}/src/routes/'(private)'/getting-started/+page.svelte.tmpl
   ```

If the user explicitly asks for a custom feature ("add a `dashboard` page", "wire
in a `vehicles` page"), generate that fresh on top of the copied reference tree (add
its `nav.ts` row) — don't try to extend the bundled snapshot in-place.

### Step 4. Update turbo.json

Add a package-scoped task override for the SvelteKit app in `turbo.json`. The key
is the **package name** from the app's `package.json` (scoped, e.g.
`@mssfoobar/{app-name}`) — not the directory name; with a bare directory name the
override silently never applies:

```json
"@mssfoobar/{app-name}#build": {
  "inputs": ["$TURBO_DEFAULT$", ".env*"],
  "outputs": ["build/**", ".svelte-kit/**"]
}
```

This ensures Turbo caches SvelteKit builds correctly with the right outputs instead of
using the default Next.js `.next/**` config.

### Step 5. Install dependencies

Two workspace-root settings must match what the scaffold expects:

- **`package.json` → `packageManager`** must equal `pnpm@10.30.3` (or whatever `assets/package.json` pins). pnpm refuses to mix versions across a workspace; fresh `create-turbo` repos commonly ship `pnpm@9.x`. **Scaffolding into an existing monorepo?** Don't change the root's `packageManager` — that pin governs the whole workspace, and bumping it can break the other packages or a `corepack`-less checkout. Instead align the *app's* pin down to the repo's existing `packageManager`.
- **`pnpm-workspace.yaml` → `injectWorkspacePackages: true`** (camelCase, NOT kebab-case — pnpm-workspace.yaml uses YAML-style keys; `.npmrc` uses kebab-case). Without it, the Dockerfile's `pnpm deploy` errors `ERR_PNPM_DEPLOY_NONINJECTED_WORKSPACE`. The setting is workspace-level — app-level `.npmrc` doesn't satisfy it.
- **`pnpm-workspace.yaml` → `onlyBuiltDependencies: [esbuild]`** — pnpm ≥10 blocks dependency build scripts by default; without this, install warns `Ignored build scripts: esbuild` and vite later fails at dev/build time. Add any other packages the install warning names (e.g. `core-js-bundle`).

Edit both files directly (don't go through `npm config` / `pnpm config` — those rewrite the file). Both edits are idempotent; re-running is safe.

Then install (env file first so `pnpm dev` works immediately after):

```bash
cd apps/{app-name}
cp .env.template .env.development
# Run pnpm install at repo root so the workspace picks up the new app.
cd ../.. && pnpm install
```

Install failure modes:

- **`401 Unauthorized` on `@mssfoobar/ui`** — `npm.pkg.github.com` auth token is
  missing/expired in `~/.npmrc` (`//npm.pkg.github.com/:_authToken=...`). Fix the
  token; don't fall back to manual file copies.
- **`Your pnpm version is incompatible … Expected: 10.x.x, Got: 9.x.x`** — your
  global pnpm doesn't match the `packageManager` field. Recommended fix:
  `corepack enable && corepack use pnpm@10.30.3` (auto-activates per-repo). When
  bumping `assets/package.json`'s `packageManager`, also update the root
  `package.json` (the Dockerfile reads the pin from there via corepack).
- **`ERR_PNPM_NO_MATCHING_VERSION`** — a pinned dep version is unpublished. Run
  `pnpm view <pkg> versions --json`, pick the closest available release, and
  update **both** the user's app `package.json` AND `assets/package.json` so the
  next scaffold doesn't hit the same wall.

### Step 6. Verify the build

End by formatting, then running all checks so the generated project is
proven-working — not just generated. `pnpm format` first: the app is assembled
from three sources (`sv create`, `sv add`, and the AOH assets), which don't share
one formatter pass, so the raw tree is not prettier-clean. Formatting normalizes
it to the AOH `.prettierrc`, after which `pnpm lint` (prettier `--check` + eslint)
passes:

```bash
cd apps/{app-name}
pnpm format
pnpm exec svelte-kit sync
pnpm exec svelte-check --tsconfig ./tsconfig.json --threshold error
pnpm build
pnpm lint
```

If any of these fail, fix the root cause before reporting success.

The assets under `assets/` have been shaped to avoid specific failure modes
(legacy cookie type, unpublished Geist patch versions, missing
`@mssfoobar/logger` exports map, design-system token shadowing, etc.). When
modifying any asset file, read `NOTES.md` in this skill directory for the
rationale of the current pins and import paths — re-introducing one of those
failures isn't caught by the build above, only by users running the scaffolded
app.

## After scaffolding: how to author pages

The scaffold lays down the auth flow, gateway, layout, and one worked-example
page (`(private)/getting-started/+page.svelte`). For authoring new pages or
components, the relevant skills are:

- `aoh-design` — design-system contract, page archetypes, `@mssfoobar/ui`
  primitive inventory and composition discipline (the authority for "what to
  use and why").
- `aoh-conventions` (`references/web.md`) — code-level mechanics: import
  subpaths, theme-token usage, API quirks (`Badge` variant/color split,
  `Toaster` not auto-mounted, etc.), Svelte 5 runes, gateway calls.

The `getting-started/+page.svelte` template the scaffold just copied is a
worked import-pattern example — mirror it. Don't guess `@mssfoobar/ui`
subpaths; read the package's `exports` field
(`cat apps/{app-name}/node_modules/@mssfoobar/ui/package.json | jq .exports`).

### Moving the landing page → update `LOGIN_DESTINATION`

The scaffold's root `+layout.server.ts` unconditionally redirects bare `/` to
`LOGIN_PAGE` (defaults to the bundled `/aoh/api/auth/login`), which runs PKCE,
calls back, and then sends the user to **`LOGIN_DESTINATION`**. Both env vars
are wired through `.env.development` (local `pnpm dev`) and
`compose/<app>/compose.yml` (containerised).

When you delete the scaffolded `getting-started` page (or move your real
landing route to e.g. `/board`), you **must** update `LOGIN_DESTINATION` in
**both** places to match — otherwise post-callback lands on a 404 route.

Two hard rules to call out, because they're the failure mode that bit during
this skill's first real use:

- `LOGIN_DESTINATION` must point to a real authenticated route under
  `(private)/` (e.g. `/board`, `/aoh/<module>/...`). It is consumed by the
  callback handler and by the root layout's `/`-redirect, so a path that 404s
  surfaces immediately after login.
- **`LOGIN_DESTINATION` must NOT be `/`.** `/` is the login-stub itself
  (root `+layout.server.ts` redirects it to the login flow). Setting
  `LOGIN_DESTINATION=/` creates an infinite redirect loop:
  `GET /` → login → Keycloak (SSO active) → callback → `GET /` → login → ...
  Chrome surfaces this as `ERR_TOO_MANY_REDIRECTS` on the Keycloak host.

Same caveat applies if you put a real `+page.svelte` at `(private)/+page.svelte`
expecting it to render at `/` — it can't, because the root layout intercepts
`/` first. Put your landing page at a named subpath
(`(private)/board/+page.svelte`, `(private)/aoh/tasking/+page.svelte`, etc.)
and point `LOGIN_DESTINATION` there.
