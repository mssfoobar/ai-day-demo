# dispatch-web

Baseline AOH dispatch console for workshops.

A single console page — a left **Units** list and a right **unit detail** pane — built
from `@mssfoobar/ui` primitives. It is the identical, controlled state every workshop
attendee starts from.

## Prerequisites

One thing is **not** optional, and it is the only thing that can stop you before the app
runs: `@mssfoobar/ui` is published to GitHub Packages, so `~/.npmrc` needs a token

```
//npm.pkg.github.com/:_authToken=<your GitHub token with read:packages>
```

Without it `pnpm install` fails with `401 Unauthorized`. The `@mssfoobar` registry itself
is already mapped in this app's `.npmrc`; only the token is per-developer.

## Development

```sh
pnpm install     # from the repo root
pnpm run dev
```

Then open http://localhost:5173 — bare `/` redirects to `/units`.

**No containers are required.** No Keycloak, no Traefik, no IAMS, no `sds-server`, no
database. If something tells you to run `docker compose` or `podman compose` to see this
page, that instruction is for a different app.

`.env.development` is **checked in on purpose** — `pnpm dev` reads it through `env-cmd`
and fails outright without it, and the workshop needs every attendee on byte-identical
config. It holds no secret.

## This app has no authentication

That is a deliberate property of the workshop, not an oversight, and it is worth stating
plainly because it makes this app diverge from the standard AOH web-base:

| Standard AOH web app                                | This app                                       |
| --------------------------------------------------- | ---------------------------------------------- |
| OIDC login via Keycloak (IAMS)                      | None — no login, no logout, no session         |
| `(private)` / `(public)` route groups               | None — routes sit directly under `src/routes/` |
| `/aoh/gateway/<module>/…` proxy with a bearer token | None — no `gateway.config.ts`                  |
| Tokens in `sds-server`, session id in a cookie      | No tokens, no session cookie                   |
| Sidebar + headerbar from `nav.ts`, role-filtered    | None — the console renders bare                |

The auth layer was **removed**, not disabled. That distinction matters: the scaffold
performed OIDC discovery in a top-level `await` in `src/hooks.server.ts`, which runs at
server startup before any routing. With no identity provider reachable, _every_ route
returned HTTP 500 — an unauthenticated route did not escape it. There is no "auth present
but inert" state, so it is either running against a real Keycloak or gone.

Reinstating auth means restoring the whole set together — `hooks.server.ts`'s discovery
and auth handle, `src/lib/aoh/core/provider/auth/`, the `(public)/aoh/api/auth/*` routes,
the `(private)` group and layout, the gateway proxy, and the `App.Locals` fields in
`src/app.d.ts` — and running `iams-keycloak`, `iams-aas`, `sds-server` and `valkey`.
Re-scaffolding those files with `aoh-web-init` is the sane way to do it.

## Where the console code lives

| Path                                     | What                                                   |
| ---------------------------------------- | ------------------------------------------------------ |
| `src/lib/aoh/dispatch/roster.ts`         | `FieldUnit`, the `UnitStatus` union, and `listUnits()` |
| `src/lib/aoh/dispatch/roster.test.ts`    | Unit tests for the roster                              |
| `src/routes/units/+page.svelte`          | The console page                                       |
| `src/routes/+layout.server.ts`           | Redirects `/` to `/units`                              |
| `tests/e2e/public/units-console.spec.ts` | Console acceptance coverage                            |
| `tests/e2e/public/no-auth.spec.ts`       | Asserts the no-auth posture stays true                 |

Read the roster through `listUnits()`, never by importing the underlying array. That
accessor is the seam the follow-up change repoints at a real backend service without
touching the page.

## Scripts

- `pnpm run dev` — development server with pino-pretty log formatting
- `pnpm run build` — production build (SvelteKit adapter-node)
- `pnpm run preview` — preview the production build
- `pnpm run check` — type-check with svelte-check
- `pnpm run lint` — prettier + eslint
- `pnpm run format` — prettier write
- `pnpm run test:unit` — Vitest
- `pnpm run test` — Playwright + Vitest

Playwright has a single `public-chromium` project; it starts the dev server itself and
needs nothing else running:

```sh
pnpm exec playwright test
```

## Architecture

See `AGENTS.md` for conventions that still apply to this app.
