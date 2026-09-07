# `@mssfoobar/gis-web-sdk` component & export catalog

Every public export, what it does, and the subpath to import it from.
The barrel (`@mssfoobar/gis-web-sdk`) re-exports the components; the
subpaths exist for granular imports / smaller dependency graphs. Both
forms work — importing everything from the barrel is simplest.

## Providers & engines (structural — pick the wrappers you need)

| Export | Subpath | Role |
|---|---|---|
| `GisProvider` | `/provider` | App-root theme provider: `dark_mode_store` (required) — its only prop. Mount once, high in the tree. |
| `CesiumMapEngineProvider` | `/engines/cesium` | 3D engine. Needs `PUBLIC_CESIUM_TOKEN` for terrain/imagery; falls back to OSM 2D without it. |
| `MapLibreEngineProvider` | `/engines/maplibre` | 2D engine. Use instead of Cesium for a pure-2D map. |
| `Map` (alias `GisMap`) | `/map` | The map itself. Props are the `MapOptions` type (import by name from the barrel) — see below. Wrap in exactly one engine provider. |

## Layers & data sources

| Export | Subpath | Role |
|---|---|---|
| `MapBaseLayerProvider` | `/base-layer-provider` | Declares a selectable base layer (`layer_name`). First declared = default. |
| `MapXyzSourceProvider` | `/xyz-source-provider` | XYZ tile source (`url` with `{z}/{x}/{y}`). Nest inside a base-layer provider. |
| `MapEntityLayerProvider` | `/entity-layer-provider` | Groups entities into a named, toggleable layer (`layer_name`). |
| `MapEntityProvider` | `/entity-provider` | Renders **all** entities of one `kind` in the map's entity store — a renderer + filter, **not** a data source (no `entities`/`data` prop). Data reaches the store via the RTUS feed or `map.upsert_entities(...)`. `geom_style` for polygons/lines; a typed `children` snippet for custom point markers. See `references/entities.md`. |
| `MapSingleEntityProvider` | `/single-entity-provider` | Renders ONE host-supplied entity passed directly via props (`id`, `entity_id`, `entity_type`, `geojson`) + a `children` snippet — declarative sugar over `map.upsert_entities`. Renders + tracks the camera like any entity. For fixed markers (a "you are here" pin), NOT a dynamic stream. Not part of a `kind` layer (won't show in `MapLayerManager`). |

## UI panels (drop inside `<GisMap>`)

| Export | Subpath | Role |
|---|---|---|
| `MapControlPanel` | `/control-panel` | Drawing / annotation toolbar (points, polylines, polygons). |
| `MapLayerManager` | `/layer-manager` | Per-layer visibility toggles (reads `map.state.entity_layers` + `base_layers`). Takes a trigger child. |
| `MapBookmarkManager` | `/bookmark-manager` | Camera bookmarks. `bookmarks` prop = initial list; create/delete via form actions — needs the BFF (`references/bff-bookmarks.md`). Takes a trigger child. |
| `MapEntitySearchBar` | `/entity-search-bar` | Search/locate entities on the map. |
| `MeasurementPanel` | `/measurement-panel` | Distance / area measurement tools. |
| `MapGeoStyleForm` | `/geo-style-form` | Editor for geometry styles. |
| `MapEntityStyleManager` | `/entity-style-manager` | Manage per-entity-kind styling. |
| `ImageGeoreferencingPanel` | `/image-georeferencing` | Georeference a raster image onto the map. |
| `LocationMapDisplay` | `/location-map-display` | Compact read-only map for displaying a single location. |
| `Flash` | `/flash` | Transient highlight/flash effect on an entity. |

## Non-component exports

| Export | Subpath | Role |
|---|---|---|
| `gisNav` | `/nav` | Navigation metadata: `{ code: "GIS", header: {name,url}, sidebar: [{name,url,icon}] }`. Feed to your sidebar/router. `url`s default to `/aoh/gis`. |
| types | `/types` | **Import types by name from the barrel** — `import type { MapOptions, CameraView, MapEntity, MapEventSubscriptions, EntityPosition, GeoEntity } from "@mssfoobar/gis-web-sdk"` (the barrel does `export type * from './types'`). A global `Gis.*` namespace also exists but only resolves when the SDK's `types-namespace.d.ts` is in your app's TS `include` — a plain consumer gets `Cannot find namespace 'Gis'`, so prefer the named imports. |
| utils | `/utils` | Geospatial helper utilities. |
| `styles/app.css` | `/styles/app.css` | The SDK's stylesheet (Tailwind `@source` registration). Chain into your CSS entry. |

## `<GisMap>` props (the `MapOptions` type)

| Prop | Type | Required | Notes |
|---|---|---|---|
| `initial_camera_view` | `CameraView` | yes | `{ position: [lon, lat] (EntityPosition), zoom: number (0–20), direction?: [pitch, yaw, roll] }`. |
| `batchUpdateInterval` | `number` (ms) | no | Entity-position batch flush interval. Default ~500ms. Lower for live feeds, raise for very large fleets. |
| `rtus_seh_url` | `URL` | no* | RTUS SEH SSE endpoint for the live feed. |
| `rtus_map_name` | `string` | no* | RTUS map name; must equal `gis-service`'s `RTUS_MAP_NAME`. |
| `user_id` | `string` | no* | From auth claims (aoh-web-init base: `data.user?.sub`). |
| `tenant_id` | `string` | no* | From auth claims (aoh-web-init base: `data.user?.active_tenant?.tenant_id`). |
| `event_subscriptions` | `MapEventSubscriptions` | no | Host interaction callbacks — `on_click`, `on_right_click`, `on_mouse_move`. **Requires gis-web-sdk ≥ 1.1.0.** Read once at init. See `references/entities.md` → "Reacting to clicks". |
| `children` | `Snippet` | yes | The nested providers/panels. |

\* The live feed subscribes **only when all four** RTUS fields are
present. Any missing → the map renders without a subscription (no error).
`on_click` / `on_right_click` payloads carry `{ position, cursor,
entity_ids, map_entity_ids }`; `on_mouse_move` carries `{ position,
cursor }`. (In gis-web-sdk < 1.1.0 these were typed but never fired and
could not be set — upgrade to use them.)

## `<GisProvider>` props (`GisProviderValue`)

`<GisProvider>` is the SDK's theme provider — `dark_mode_store` is its only
prop. (It used to also expose `gis_url` / `auth_adapter` / `feature_flags`,
which the SDK never read; they were removed. GIS REST goes through the host's
server-side BFF, so there's no client-side auth hook.)

| Prop | Type | Required | Notes |
|---|---|---|---|
| `dark_mode_store` | `Writable<boolean>` | yes | Host-owned theme store. The SDK only reacts to it. |
| `children` | `Snippet` | yes | App subtree. |

Helpers exported alongside it: `GetGisProviderContext()` (→ value or
`undefined`) and `GetDarkModeStore()` (→ provider's store or a fresh
`writable(false)` standalone). Components degrade gracefully with no
provider mounted — handy for Storybook/tests.

## Types & the `Gis.*` namespace

All SDK types are re-exported from the barrel (`@mssfoobar/gis-web-sdk`
does `export type * from './types'`), so import them **by name** from the
barrel — `import type { MapOptions, CameraView, MapEntity } from
"@mssfoobar/gis-web-sdk"`. The SDK also declares a global `Gis.*`
namespace, but that only resolves inside the SDK's own build (its
`types-namespace.d.ts` in the TS `include`); a plain consumer gets
`Cannot find namespace 'Gis'`, so **use the named barrel imports**. The
component docstrings shipped in the package are the authoritative prop
reference.
