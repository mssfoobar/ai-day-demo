# dispatch-web

AOH dispatch console for workshops.

Two pages behind an IAMS sign-in: a **console** (`/aoh/dispatch/units`) with a left Units
list and a right unit-detail pane, and a **map** (`/aoh/dispatch/map`) rendering the same
units live on a geospatial canvas. The chrome is `@mssfoobar/ui`; the map is
`@mssfoobar/gis-web-sdk`. It is the identical, controlled state every workshop attendee
starts from.

## Prerequisites

One thing is **not** optional, and it is the only thing that can stop you before the app
runs: `@mssfoobar/ui` is published to GitHub Packages, so `~/.npmrc` needs a token

```
//npm.pkg.github.com/:_authToken=<your GitHub token with read:packages>
```

Without it `pnpm install` fails with `401 Unauthorized`. The `@mssfoobar` registry itself
is already mapped in this app's `.npmrc`; only the token is per-developer.

## Development

The console reads its roster from `dispatch-svc` and authenticates against IAMS, so it is
not standalone. From the **repo root**, one command starts the stack, the service and this
app:

```sh
pnpm install
pnpm start
```

Then open <http://127.0.0.1.nip.io:5173> and sign in; bare `/` leads to the sign-in flow
and then to `/aoh/dispatch/units`.

**Not `localhost`.** The session cookie is issued on `127.0.0.1.nip.io` so it reaches
`rtus-seh.127.0.0.1.nip.io`, which is what authorises the map's SSE subscription — and that
origin is the one rtus-seh's CORS allow-list already contains. Served on localhost the map
silently never connects, with nothing in either log to explain it. `nip.io`
wildcard-resolves to the loopback on every OS, including Windows, so there is no hosts file
to edit. `vite.config.ts` allowlists the host; `pnpm dev` runs `vite dev --host`.

**The whole platform stack is required**, not just PostgreSQL: `iams-keycloak`, `iams-aas`,
`sds-server`, `valkey`, `rtus-pms`, `rtus-seh` and `gis-service`. Running `pnpm dev` here
with no Keycloak reachable does **not** degrade gracefully — OIDC discovery runs in a
top-level `await`, so every route returns 500, including the health probes.

`.env.development` is **checked in on purpose** — `pnpm dev` reads it through `env-cmd`
and fails outright without it, and the workshop needs every attendee on byte-identical
config. It holds no secret; the password in it is the stack's shipped dev default.

## Authentication

Operators sign in through `iams-keycloak` with OIDC Authorization Code + PKCE, against the
`aoh` realm's bundled public `web` client. **Tokens live server-side in SDS**; the browser
holds only an opaque `web_auth_session_id` cookie. Nothing returned from a `load` carries a
token — everything a load returns is serialised into the page payload.

| Concern       | Here                                                                                 |
| ------------- | ------------------------------------------------------------------------------------ |
| Sign-in       | `(public)/aoh/api/auth/*` — all six scaffold routes                                  |
| Session       | `sds-server`, keyed by `web_auth_session_id`                                         |
| Route guard   | `(private)/+layout.server.ts` redirects; the hooks handle only resolves              |
| Roles         | AAS tenant roles in `active_tenant.roles` (`dispatch-viewer`, `dispatch-dispatcher`) |
| Permissions   | projected in-process — `src/lib/aoh/dispatch/permissions.ts`                         |
| Backend calls | server-side only, with `Authorization: Bearer` from SDS                              |

Two deliberate omissions:

- **No gateway proxy.** `gateway.config.ts` and `(private)/aoh/gateway/[...path]` are not
  restored. Nothing here calls `dispatch-svc` from the browser — reads are in a server
  `load`, writes in form actions — and `aoh-conventions` is explicit that a server `load`
  must not route through the proxy anyway. When a browser-side call first appears, the
  gateway arrives with it.
- **No `active_tenant.permissions`.** AAS does not emit one. `permissions.ts` projects
  role names onto read/write, mirroring `compose/iams/init/project-aas/roles.yaml`, whose
  Go twin is `apps/dispatch-svc/internal/service/permissions.go`.

Hiding a write control is presentation, not authorization: the service enforces the same
role rule independently, and a viewer who submits a write anyway gets a 403 rendered as
the permission-denied state.

## The map

`/aoh/dispatch/map` composes `@mssfoobar/gis-web-sdk` — `GisProvider` (mounted once at the
root layout, fed the app's dark-mode store), `CesiumMapEngineProvider`, `Map`,
`MapBaseLayerProvider` + `MapXyzSourceProvider` (OpenStreetMap tiles),
`MapEntityLayerProvider` + `MapEntityProvider kind="field-unit"`, and `MapLayerManager`.
No renderer, tile client or marker layer is hand-rolled.

Three things about it are load-bearing:

- **`+page.ts` exports `ssr = false`.** The Cesium engine touches browser globals at module
  init, so server-rendering the route throws before anything renders.
- **Cesium's runtime assets are copied into `static/cesium/`** by a plugin in
  `vite.config.ts`, on both `buildStart` and `configureServer`. The engine fetches its
  workers and textures lazily from `CESIUM_BASE_URL`; without the copy the canvas renders
  entirely black with only 404s to explain it. The copied directory is gitignored and
  lint-ignored — it is vendor output, not source.
- **Entity state has exactly one source**: the SDK's RTUS subscription. The page loads the
  unit roster for its counts, but never merges it with the live topic, so there is no
  fetch/SSE race to reconcile. Deriving "not shown" from entities rather than units would
  also make a projection that is still draining look like a smaller roster.

## Where the console code lives

| Path                                       | What                                                                         |
| ------------------------------------------ | ---------------------------------------------------------------------------- |
| `src/lib/aoh/dispatch/types.ts`            | `FieldUnit` / `Crew` / `Assignment` / `Position` — client-safe, no `$env`    |
| `src/lib/aoh/dispatch/units.server.ts`     | Service client: `listUnits()`, the bearer, envelope unwrapping, wire mapping |
| `src/lib/aoh/dispatch/permissions.ts`      | Role → permission projection, mirroring `roles.yaml`                         |
| `src/lib/aoh/dispatch/filters.ts`          | Search, status filter, sort, fleet counts — pure                             |
| `src/lib/aoh/dispatch/format.ts`           | Recency labels and status/priority colour maps — pure                        |
| `src/lib/aoh/dispatch/components/`         | `StatusFilter`, `UnitRow`, `UnitDetail` — composed from `@mssfoobar/ui`      |
| `src/lib/aoh/core/provider/auth/`          | The OIDC + SDS auth layer, from the `aoh-web-init` scaffold                  |
| `src/lib/aoh/core/components/layout/`      | `nav.ts`, `Sidebar`, `Headerbar` — the restored chrome                       |
| `src/routes/(private)/aoh/dispatch/units/` | The console page and its form actions                                        |
| `src/routes/(private)/aoh/dispatch/map/`   | The map page (`ssr = false`)                                                 |
| `src/routes/(public)/aoh/api/auth/`        | The six OIDC endpoints                                                       |
| `src/routes/+layout.server.ts`             | Redirects `/` to the console, which leads to sign-in                         |

Keyboard: `/` focuses search, `Esc` clears it, `↑ ↓ Home End` move through the visible
list, `Enter`/`Space` select. Clicking a status tile filters the list; counts stay
fleet-wide.

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

There is **no automated test suite in this app** — no Playwright, no vitest — by decision.
`lint`, `check-types` and `build` are its automated checks. End-to-end coverage lives at
the repo root as `pnpm e2e` (`scripts/e2e-smoke.mjs`), which drives the real stack.

## Architecture

See `AGENTS.md` for conventions that still apply to this app.
