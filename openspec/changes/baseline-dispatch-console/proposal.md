## Why

Workshop attendees currently have nothing to start from — `apps/` and `packages/` are
both empty, so the first hour of any session is spent scaffolding instead of learning.
A checked-in baseline console gives every attendee an identical, controlled starting
state that renders with `pnpm dev` and no containers at all.

## What Changes

- Scaffold a SvelteKit app at `apps/dispatch-web` (package `@mssfoobar/dispatch-web`)
  from the `aoh-web-init` web-base, then **remove its authentication layer entirely**.
  The workshop is explicitly unauthenticated: no Keycloak, no Traefik, no IAMS, no SDS.
- **BREAKING vs. the scaffold**: delete the OIDC discovery in `hooks.server.ts`, the
  `(public)/aoh/api/auth/*` routes, the `(private)` route group and its layout, the
  `AuthProvider`, the SDS client, the IAMS module, and the bearer-token gateway proxy.
  These are not dormant — they are gone. The app keeps `@mssfoobar/ui`, the theme
  provider, the `app.css` token layer, the OpenTelemetry bootstrap, and the AOH folder
  layout.
- Add a single dispatch console page at `/units`: a left **Units** list and a right
  **unit detail** pane, selection-driven, composed from `@mssfoobar/ui` primitives.
- Seed 4–5 hardcoded field units in a typed in-app module behind a `listUnits()`
  accessor, each with a call sign, an ID, and a status of `Available`, `En route`, or
  `Idle`.

Explicit non-goals for this change: no command or dispatch action on the detail pane,
no backend service, no database, no map, no real-time updates, no persistence, and no
authentication of any kind.

The console is a user-facing surface, so it conforms to the AOH design system and
consumes `@mssfoobar/ui` primitives rather than hand-rolled markup.

### Relationship to the follow-up change

This change is the **frontend half** of the workshop baseline. A separate change,
`dispatch-units-service`, adds a Go + PostgreSQL backend exposing `/v1/units` and
repoints `listUnits()` at it. That split is deliberate: the accessor seam introduced
here is what lets the backend land without touching the console page.

## Capabilities

### New Capabilities

- `dispatch-console`: the baseline operator console surface — the units list, the
  selection behavior, and the detail pane, including the initial empty state, and its
  unauthenticated, container-free access posture.
- `field-unit-roster`: the shape and provenance of the baseline unit data — call sign,
  identifier, status vocabulary, and the read accessor the console consumes.

### Modified Capabilities

None. `openspec/specs/` is empty; this is the repo's first change.

## Existing AOH services considered

- `iams` (Keycloak + AAS): **ruled out — deliberately removed.** Every AOH web app
  normally authenticates through IAMS, and the `aoh-web-init` scaffold wires it in by
  default. The workshop is specified as having no authentication, so the scaffold's auth
  layer is stripped rather than deferred. This was reached the hard way: the scaffold's
  `hooks.server.ts` performs OIDC discovery in a **top-level await**, so it runs at
  server boot before any routing and returns HTTP 500 on *every* route — a `(public)`
  route group does not avoid it. Deferring auth is therefore not possible; it is either
  present and running, or removed.
- `sds` (Session Data Store): **ruled out.** SDS is mandatory only because AOH web apps
  hold auth tokens server-side. With no authentication there is no session and no token
  to store.
- `dash` (Dashboard Service): **ruled out.** The catalogue defaults any *dashboard*
  surface to DASH via `@mssfoobar/dash-web-sdk`, so it was checked first. It does not
  apply: the requested surface is a master–detail list/detail screen (Main List +
  Details archetypes), not a widget grid, which is the catalogue's stated exclusion.
- `gis` (Geospatial Information System): **ruled out for this change.** Field units map
  cleanly onto GIS `geo-entity` rows and GIS is the natural successor once units carry
  positions, but this baseline renders no map and stores no coordinates — the catalogue
  says to skip GIS when coordinates are never visualised.
- `rtus` (Real-time Update Service): **ruled out.** The roster is hardcoded and static;
  there is no event to push and no second session to keep in sync.

## Impact

- **New code**: `apps/dispatch-web/` — the whole app, currently the only workspace member.
- **Removed code**: the scaffold's entire auth surface (see What Changes). This is the
  main review risk in the change and the reason the `aoh-web-init` conventions around
  `(private)` / `(public)` route groups no longer apply to this app.
- **Repo**: `apps/*` is already globbed by `pnpm-workspace.yaml` and `turbo.json`;
  `injectWorkspacePackages` and `onlyBuiltDependencies` are added to the workspace file
  as the scaffold requires, and a package-scoped build override is added to `turbo.json`.
- **Dependencies**: adds `@mssfoobar/ui`, `@lucide/svelte`, SvelteKit, Svelte 5, Tailwind
  CSS v4, and the observability packages `aoh-web-init` pins. Note the repo's
  `minimumReleaseAge: 10080` and `trustPolicy: no-downgrade` — versions published in the
  last 7 days will not resolve. `@mssfoobar/ui` comes from GitHub Packages, so a
  `~/.npmrc` token is a per-developer prerequisite.
- **APIs**: none. No endpoint is added and no backend is called.
- **Runtime dependencies**: none. `pnpm dev` is the whole run story.
- **Docs**: a `UBIQUITOUS_LANGUAGE.md` at the repo root records the project's own terms
  (*field unit*, *call sign*, *status*) and maps them to platform vocabulary, as the
  root `AGENTS.md` requires.
