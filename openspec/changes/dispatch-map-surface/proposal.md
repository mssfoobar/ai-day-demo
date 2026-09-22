## Why

The console has no map. For a C2 app that is a conspicuous gap — operators reason about
where things are — and `UBIQUITOUS_LANGUAGE.md` has recorded GIS as the natural successor
"once units carry positions" since the baseline.

This change does the smallest useful thing: it brings the **GIS module in as a surface** —
a real Cesium map, on real tiles, mounted from `@mssfoobar/gis-web-sdk` — and stops there.
No authentication, no `gis-service`, no RTUS, no entities, no new container. The workshop
gets a working map to build on, and two of the things deliberately left out become
exercises: integrating IAMS, and wiring field units onto this map.

That split is the point. Integrating a platform module is the skill the workshop teaches,
and it is a much better exercise when the surface already exists and the attendee's job is
to make it show *their* data rather than to get a WebGL canvas to paint at all.

## What Changes

- `apps/dispatch-web` gains a **Map** page at `/map`: `@mssfoobar/gis-web-sdk`'s
  `CesiumMapEngineProvider` + `Map`, a `MapBaseLayerProvider` with an OpenStreetMap XYZ
  source, and `MapLayerManager`. The console's header gains a link to it and the map links
  back.
- `<GisProvider>` mounts at the root layout, fed by the app's **existing** `ThemeProvider`
  dark-mode store, so the map repaints with the console's theme.
- The route opts out of SSR (`ssr = false`) — the Cesium engine touches browser globals at
  module init — and a vite plugin copies Cesium's runtime assets into `static/cesium/` so
  the canvas is not black.
- New dependencies: `@mssfoobar/gis-web-sdk`, `@mssfoobar/auth-sdk` (a peer),
  `@cesium/engine`, `@cesium/widgets`.

Deliberately **not** in this change, because each is a workshop exercise or belongs to one:

- **Authentication.** The GIS SDK gates auth on `IAM_URL`; unset means the auth hook is a
  passthrough, which is exactly the baseline's posture. Nothing here needs a token.
- **Field units on the map.** No position on the unit model, no `geo-entity` projection, no
  entity layer. `dispatch-svc` is not touched at all.
- **Live updates.** RTUS authorises the browser's SSE stream from an SDS session cookie, so
  a live feed is not reachable without IAMS. The map is static.
- Bookmarks, drawing, measurement and geofences — bookmarks need a BFF and `gis-service`.

The stack stays at **one container**. `pnpm start`, `pnpm stop`, `pnpm reset-db` and
`scripts/dev.mjs` are untouched, and so is every doc that describes them.

## Capabilities

### New Capabilities

- `dispatch-map`: the map surface — the route, its SSR posture, the Cesium asset contract,
  the theme binding, and the fact that it carries no application data.

### Modified Capabilities

- `dispatch-console`: gains a link to the map. Nothing else about the console changes.

## Existing AOH services considered

- `gis` (Geospatial Information System): **selected, SDK only.** The selection criteria say
  to use GIS when "a map display is a product requirement" — it is, and
  `@mssfoobar/gis-web-sdk` is how AOH renders one. What is *not* adopted is `gis-service`:
  the SDK makes no browser-side REST calls, so a map with base tiles needs no backend. The
  service arrives with the exercise that needs entity storage.
- `iams` (Keycloak + AAS): **ruled out, and supported as such.** The GIS integration skill
  documents `IAM_URL` as the auth gate — "unset → the hook is a passthrough" — so an
  unauthenticated map is a first-class mode rather than a workaround. Adopting IAMS would
  pull in Keycloak, AAS, SDS, Valkey and Traefik, break the baseline's one-container
  promise, and force the roster seed to be rewritten (the tenant id is assigned by AAS at
  stack-up, so a committed migration cannot know it). That is a change of its own; the
  reasoning is worked out in `dispatch-iams-and-unit-map`.
- `rtus` (Real-time Update Service): **ruled out.** Nothing publishes, and `rtus-seh`
  authorises its SSE stream from an SDS session cookie it cannot have without IAMS. Live
  updates arrive with the entity exercise.
- `dash` (Dashboard Service): **ruled out.** Checked because the catalogue defaults any
  dashboard surface to DASH. A full-bleed geospatial canvas is not a widget grid — the same
  exclusion `baseline-dispatch-console` recorded for the console.

## Impact

- **`apps/dispatch-web`**: a `/map` route and its `+page.ts`, `<GisProvider>` in the root
  layout, a link on the console page, `vite.config.ts` (Cesium asset plugin,
  `CESIUM_BASE_URL`, `ssr.noExternal`), `.gitignore` for the copied assets, and four new
  dependencies. `eslint.config.js` already ignores `static/cesium/`.
- **`apps/dispatch-svc`**: untouched.
- **Runtime dependencies**: none added. The existing PostgreSQL container is the whole
  stack, exactly as today.
- **Docs**: `apps/dispatch-web/README.md` and `AGENTS.md` gain the map route and the Cesium
  asset contract. `UBIQUITOUS_LANGUAGE.md` keeps its *field unit* ↔ `geo-entity` mapping
  conditional — units still carry no position — but corrects its guess (`field-unit` would
  be the `kind`, `track` the `entity_type`).
- **Workshop**: `WORKSHOP.md` and `USER_STORIES.md` gain an exercise for wiring field units
  onto this map. The three existing exercises are untouched.
