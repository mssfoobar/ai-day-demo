## Context

`apps/dispatch-web` is an `aoh-web-init` scaffold with its auth layer removed
(`baseline-dispatch-console` D1) — no Keycloak, no SDS, no Traefik, no route groups, no
gateway. The whole stack is one PostgreSQL container and two natively-run processes, and
`README.md` and `SETUP.md` promise exactly that.

Three facts from the platform skills shape this design:

1. **The GIS SDK is config-agnostic.** It "reads no env vars, owns no token, maintains no
   global theme", and "makes **no GIS REST calls from the browser**". The only thing
   `<GisProvider>` takes is a dark-mode store (`aoh-gis-integration` →
   `references/auth-and-config.md`).
2. **Auth is gated on `IAM_URL`.** "Unset → the hook is a passthrough (open sandbox /
   tests); set → full OIDC." An unauthenticated map is therefore a supported mode, not a
   workaround.
3. **Cesium fetches its runtime assets lazily.** Without serving them the canvas is
   entirely black with only 404s to explain it — the single most common way this
   integration fails.

Together those mean a working map needs no AOH service at all. That is what makes this
change small enough to be worth doing on its own.

## Goals / Non-Goals

**Goals:**

- A real map, from the platform's own SDK, reachable from the console, that renders on a
  clean checkout with no new containers.
- Get the two things wrong-by-default right, because they are what an attendee would lose a
  day to: the Cesium asset copy, and `ssr = false`.
- Leave the surface in a state where the next integration is additive — an entity provider
  and a data source drop in beside what is here rather than replacing it.

**Non-Goals — each is a workshop exercise or belongs to one:**

- **Authentication.** Adopting IAMS is a change of its own; `dispatch-iams-and-unit-map`
  works out that reasoning in full.
- **Field units on the map.** No position on the unit model, no `geo-entity` projection, no
  entity layer. `dispatch-svc` is not touched.
- **Live updates.** Unreachable without IAMS — `rtus-seh` authorises its SSE stream from an
  SDS session cookie.
- Bookmarks, drawing, measurement, geofences. Bookmarks additionally need a BFF and
  `gis-service`.

## Runtime dependencies

| Dependency | Role | Owned by |
|---|---|---|
| `postgres` (dispatch) | The existing database behind `dispatch-svc`. **Unchanged** — this change does not touch the service or its schema | Existing — untouched by this change |
| `dispatch-web` | SvelteKit console; runs natively (`pnpm dev`), not composed | Existing — modified by this change |

**No dependency is added.** No `gis-service`, no RTUS, no IAMS, no SDS, no Traefik. That is
the headline property of this change: `pnpm start` keeps its "only PostgreSQL runs in a
container" promise, and the offline image bundle needs nothing new.

The base tiles come from the public OpenStreetMap tile server over the internet. That is a
third-party runtime dependency of a different kind, and it is the one thing here that will
not work on the workshop's offline network — see R4.

## API surface

This change adds and modifies **no HTTP endpoint**. `dispatch-svc` is untouched; the map
calls nothing.

| Route | Auth posture | Description |
|---|---|---|
| `/map` | Unauthenticated, like every route in this app | The map page. `ssr = false` |
| `/units` | Unauthenticated | The console — unchanged except for a link to `/map` |
| `/livez` · `/readyz` | Unauthenticated | Unchanged |

## UI / Design System

One surface, drawn in `openspec/changes/dispatch-map-surface/design/map-mock.html` with a
state switcher over its two states — **loaded** (tiles, layer panel, the "no entities yet"
note) and **tiles unavailable** (the offline case, R4) — plus a light/dark toggle, since the
theme is a dimension of both rather than a state of its own.

**Primitives.** The map itself is SDK-owned: `GisProvider` (root layout),
`CesiumMapEngineProvider`, `Map`, `MapBaseLayerProvider` + `MapXyzSourceProvider`, and
`MapLayerManager`. Everything around it — the page header, the link back to the console, the
"no entities yet" note — composes from `@mssfoobar/ui` (`Button`, `Card`, `Badge`). Nothing
is hand-rolled; `@mssfoobar/ui` ships all of it, and hand-rolling silently bypasses the theme
tokens and dark mode.

**Deliberately absent:** `MapEntityLayerProvider` and `MapEntityProvider`. Shipping an empty
entity provider now would only be a stub for the exercise to delete.

**Copy and tone.** Sentence case, operational register, per `aoh-design`. The empty-map note
reads as a statement of fact and a pointer, not an apology: "No entities are shown on this
map yet." — not "Oops, nothing here!"

## Decisions

**D1 — Tiles only; the map ships empty.** The surface lands with no data on it at all.
*Alternatives considered:* (a) drawing one or two hardcoded demo entities via
`map.upsert_entities`, so attendees see a marker before the exercise — rejected because it
is throwaway code the exercise immediately deletes, and because a marker that comes from
nowhere teaches the wrong mental model of where entity data lives; (b) going further and
wiring units properly — that is the exercise, and doing it here would leave the workshop
with one fewer. The cost is that the first thing an attendee sees is an empty map, which D2
addresses directly.

**D2 — The empty map states why it is empty.** A blank canvas is indistinguishable from a
broken one, and this map will be blank for every attendee until they do the exercise. So the
page says so and names the next step. This is the same reasoning the console already applies
to an unreachable service: an empty result and a failure must not look alike.

**D3 — No authentication, because the SDK supports its absence.**
`aoh-gis-integration`'s auth gate is `IAM_URL`: unset means the auth hook is a passthrough.
*Alternative considered:* adopting IAMS so the map matches a production posture. Rejected
here — it pulls in Keycloak, AAS, SDS, Valkey and Traefik, ends the baseline's
one-container promise, and forces the roster seed to be rewritten, because the tenant id is
assigned by AAS at stack-up and a committed migration cannot know it. All of that is real
work with real value; it is simply a different change, and it is already designed in
`dispatch-iams-and-unit-map`.

**D4 — Cesium, not MapLibre.** `CesiumMapEngineProvider` is the engine the SDK's own example
page uses, in `SCENE2D` with Ion disabled, so no Ion asset quota is consumed. *Alternative
considered:* `MapLibreEngineProvider`, which would be lighter. The two platform skills
disagree about whether it works at all — `aoh-knowledge/services/gis.md` calls it "an empty
stub … not currently an option", while `aoh-gis-integration/references/components.md` lists
it as a usable 2D engine — so this change takes the one both agree on and records the
conflict as R3.

**D5 — The map borrows the app's existing theme store.** `<GisProvider>` takes a
`Writable<boolean>`, and the app already has one in
`src/lib/aoh/core/provider/theme/ThemeProvider`. *Alternative considered:* a second store
local to the map. Rejected — two stores means two sources of truth for one visible property,
and the map would drift out of sync with the console on the first theme toggle.

**D6 — `/map`, not `/aoh/dispatch/map`.** `aoh-conventions` routes feature pages under
`(private)/aoh/{module}/`, but this app has no route groups: the baseline removed them with
the auth layer, and `apps/dispatch-web/AGENTS.md` records that as a deliberate deviation.
The console lives at `/units`, so the map lives at `/map` beside it. Moving both under
`(private)/aoh/dispatch/` is part of adopting IAMS, not part of adding a map.

**D7 — A link each way, not a navigation component.** The baseline has no `nav.ts`, no
`Sidebar` and no `Headerbar`, by decision. Two pages need two links, and introducing a
navigation component to hold them would be scope the baseline deliberately declined.

## Risks / Trade-offs

**R1 — A black map canvas.** Cesium fetches its workers, textures and shims at runtime from
`CESIUM_BASE_URL`; without them the canvas is entirely black, the layer toggle does nothing,
and the only evidence is 404s in the console. This is the documented number-one failure of
this integration. → The vite plugin runs on both `buildStart` and `configureServer` so dev
and build are both covered, and the spec asserts "no request under the Cesium base URL
answers 404" as an observable outcome rather than trusting the plugin.

**R2 — Cesium is heavy on a workshop laptop.** `@cesium/engine` plus `@cesium/widgets` are a
large install and need WebGL. → Accepted: it is the engine the SDK actually ships working,
the scene is 2D with Ion disabled, and the assets are copied locally rather than fetched from
a CDN. Attendees on a machine without WebGL will see the black canvas of R1 for a different
reason, which the page's own empty-state copy will not explain — worth a line in `SETUP.md`.

**R3 — The skills disagree about MapLibre.** `aoh-knowledge/services/gis.md` says
`MapLibreEngineProvider` "ships as an empty stub — picking MapLibre is not currently an
option"; `aoh-gis-integration/references/components.md` lists it as a working 2D engine. →
Proceed with Cesium, which both agree works. Trying MapLibre against the installed SDK is a
cheap experiment at apply time; if it works it is strictly lighter here, and the
contradiction is worth reporting upstream either way.

**R4 — The base tiles come from the public internet, and the workshop network has none.**
`SETUP.md` states there is no internet on the workshop network. OpenStreetMap tiles will not
load, so attendees will see the map's grid and controls over an empty background. → The map
still demonstrably works — the engine initialises, the layer panel functions, the empty-state
note renders — and this is a *tile* problem, not an integration problem. Decide before the
day whether to accept it, bundle an offline tile source, or point `MapXyzSourceProvider` at a
tile server inside the workshop network. This is the one thing in this change that the
offline constraint actually touches.

**R5 — `@mssfoobar/gis-web-sdk` must resolve from GitHub Packages.** The checked-in `.npmrc`
token covers the `@mssfoobar` scope, so this should work as the other five packages do. Note
`pnpm-workspace.yaml` sets `minimumReleaseAge: 10080`, so a version published in the last
seven days will not resolve. → Pin a version older than that, and confirm the install before
the day rather than on it.

## Migration Plan

There is nothing to migrate. No schema changes, no data, no containers, no config. The change
is additive to one app: on rollback, revert the commit and remove `static/cesium/`.

## Open Questions

1. **Where should the map's initial camera sit?** The SDK's example uses Singapore
   (`103.8189, 1.3521`, zoom 8), which matches the seeded roster's stations. Keeping that is
   the obvious default; it is called out only because it is the kind of value that gets
   copied without thought and then looks arbitrary to everyone afterwards.
2. **Does the workshop want an offline tile source?** See R4. It is a workshop-logistics
   decision rather than a repo one, and it is the only place the offline network affects this
   change.
