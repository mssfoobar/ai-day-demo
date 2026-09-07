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

The console now reads its roster from `dispatch-svc`, so it is no longer standalone.
From the **repo root**, one command starts the database, the service and this app:

```sh
pnpm install
pnpm start
```

Then open http://localhost:5173 — bare `/` redirects to `/units`.

Running `pnpm dev` in this directory alone still works, but with no service reachable the
console renders its **Units unavailable** state rather than a fleet.

**One container is required** — PostgreSQL, for the service. There is still no Keycloak,
no Traefik, no IAMS and no SDS.

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

| Path                                   | What                                                                    |
| -------------------------------------- | ----------------------------------------------------------------------- |
| `src/lib/aoh/dispatch/units.server.ts` | Service client: types, `listUnits()`, envelope unwrapping, wire mapping |
| `src/lib/aoh/dispatch/units.test.ts`   | Unit tests for the mapping layer                                        |
| `src/routes/units/+page.server.ts`     | Server `load` — the browser never calls the service                     |
| `src/routes/units/+page.svelte`        | The console page                                                        |
| `src/routes/+layout.server.ts`         | Redirects `/` to `/units`                                               |

Read the roster through `listUnits()`, never by importing the underlying array. That
accessor is the seam that now calls `dispatch-svc`. It is `.server.ts`, so importing it
from client code is a build error and the service URL cannot reach the browser bundle.

## Scripts

- `pnpm run dev` — development server with pino-pretty log formatting
- `pnpm run build` — production build (SvelteKit adapter-node)
- `pnpm run preview` — preview the production build
- `pnpm run check` — type-check with svelte-check
- `pnpm run lint` — prettier + eslint
- `pnpm run format` — prettier write
- `pnpm run test:unit` — Vitest
- `pnpm run test` — Vitest

There is **no end-to-end suite**. It was removed deliberately (see
`openspec/changes/dispatch-units-service`, design.md D6), so `lint`, `check-types`,
`build` and the unit tests are the automated checks on this app.

## Architecture

See `AGENTS.md` for conventions that still apply to this app.
