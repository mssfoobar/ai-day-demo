# GIS — Geospatial Information System

## What It Does

GIS provides geospatial data management for the AOH platform — storing, querying, and
rendering entities on maps. It handles map overlays, entity tracking, and real-time
position updates.

Use GIS when your application needs maps, location data, or spatial queries.

## Architecture

```
┌──────────────┐   REST    ┌─────────────┐  RTUS   ┌──────────┐
│ Your Backend │─────────►│ gis-service │────────►│ RTUS-PMS │
│              │          │ (Go API)    │ publish │ (map:gis)│
└──────────────┘          └──────┬──────┘         └──────────┘
                                 │                      │
                            ┌────▼─────┐          ┌─────▼─────┐
                            │  GIS-DB  │          │ RTUS-SEH  │
                            │(Postgres)│          │  (SSE)    │
                            └──────────┘          └─────┬─────┘
                                                        │ SSE
┌──────────────┐   mounts   ┌────────────────┐         │
│  Host App    │──────────►│ @mssfoobar/     │◄────────┘
│  (SvelteKit) │           │ gis-web-sdk     │
└──────────────┘           └────────────────┘
```

## Components

| Component | Image | Purpose |
|-----------|-------|---------|
| **gis-db** | `postgres:17.0` | Geospatial data storage |
| **gis-service** | `ghcr.io/mssfoobar/ops-hub/gis-service` (v3.0.0) | REST API for entity CRUD (Go microservice) |
| **@mssfoobar/gis-web-sdk** | *(npm package — not an image)* | Embeddable Svelte map SDK a host app mounts |

## Key Concepts

### Entities

GIS stores geospatial entities — any object with a location. Each entity wraps a standard
GeoJSON Feature inside a `geojson` field, plus top-level metadata fields (`entity_id`,
`entity_type`, `occ_lock`).

### Entity types

Four reserved entity types classify behaviour:
- **static** — entities that don't move
- **track** — entities with real-time position updates
- **geofence** — geofence areas that can trigger events
- **annotation** — graphical overlays (arrows, markers, etc.)

Currently these are reserved names and don't change processing, but are planned for
type-specific behaviour in the future.

### Kind

`Kind` is an optional property inside a GeoJSON Feature's `properties` field. It groups
entities by business category (e.g., "aircraft", "vehicle", "patrol"). Kind is used for:
- Grouping entities in entity layers
- Applying geometry styles per kind
- Filtering/querying via `GET /geoentity/kind/{kind}`

### Real-time integration

GIS publishes entity changes to RTUS map named `"gis"` via a **transactional outbox**:
each entity write records an event row in an outbox table inside the same database
transaction, and a background worker polls the outbox and publishes those events to RTUS
with retry. This means:
- When an entity's position changes, connected browsers get instant updates
- A host app mounting `@mssfoobar/gis-web-sdk` subscribes to the `gis` RTUS map for live map rendering
- Your frontend can also subscribe to see entity movements in real-time

### Batch operations

GIS supports batch operations for up to 1000 entities per request (the `GIS_BATCH_LIMIT`
config) — useful for bulk imports or scenarios where many entities update simultaneously.

### Optimistic concurrency

GIS uses optimistic concurrency control via the `occ_lock` field. PATCH operations require
`occ_lock` in the request body. The `GIS_PUTS_IGNORE_OCC_LOCK` config can disable this for
high-frequency position updates (PUT/upsert operations).

### Database

PostgreSQL. The DB schema is carried over verbatim from the legacy service ("schema parity,
redesign later"). The service runs golang-migrate on boot to apply migrations, but does NOT
issue `CREATE SCHEMA` — it sets `search_path` against a schema that is created out-of-band
(the legacy "preliquibase" step). Real-time change propagation is done by the transactional
outbox worker (see Real-time integration), not by logical-replication / CDC.

### Frontend architecture (`@mssfoobar/gis-web-sdk`)

The GIS frontend ships as an embeddable Svelte SDK (`@mssfoobar/gis-web-sdk`) that a host app
mounts — there is no standalone deployed frontend service (see the `aoh-gis-integration`
skill). It uses CesiumJS as its map rendering engine and is built as composable Svelte
components with a swappable engine interface:

Key components:
- **Map** — core component, takes `initial_camera_view`, `rtus_seh_url`, `rtus_map_name`,
  `user_id`, `tenant_id`, `batchUpdateInterval`
- **CesiumMapEngineProvider** — wraps Map to provide the CesiumJS renderer
- **MapBaseLayerProvider** + **MapXyzSourceProvider** — define map tile layers (streets, terrain)
- **MapEntityLayerProvider** + **MapEntityProvider** — render entities by kind with custom
  Svelte snippets for point entities and geom_style for shapes
- **MapLayerManager** — prebuilt UI for toggling layer visibility
- **MapBookmarkManager** — prebuilt UI for managing camera bookmarks
- **MapControlPanel** — toolbar with zoom, drawing tools, colour pickers

Map engine exposes drawing modes (`set_mode_view`, `set_mode_pointer`, `set_mode_draw_polyline`,
`set_mode_draw_rectangle`, `set_mode_draw_ellipse`), annotation functions (`annotate_point`,
`annotate_polyline`, `annotate_polygon`), and camera control (`fly_to`, `set_camera_behaviour`).

Types are namespaced under `Gis` (e.g., `Gis.GisMap`, `Gis.GeoEntity`, `Gis.CameraView`,
`Gis.GeomStyle`) in `src/lib/aoh/gis/types.d.ts`.

### Frontend defaults (`@mssfoobar/gis-web-sdk`)

These are the out-of-the-box values shipped by `@mssfoobar/gis-web-sdk` and its example
page (`src/routes/(private)/aoh/gis/+page.svelte`). They are sensible-default starting
points, not values that are deeply wired in — the example page is the seam where the
consumer app customises them.

| Concern | Default | Notes |
|---|---|---|
| Map rendering engine | **CesiumJS** (`@cesium/engine` + `@cesium/widgets`) | The only working engine. `MapLibreEngineProvider` ships as an empty stub — picking MapLibre is not currently an option. |
| Cesium scene mode | **`SCENE2D`** | Set in `CesiumMapEngineProvider`. Rotation and tilt are disabled; only wheel-zoom and drag-pan are wired in 2D. The component does include logic to switch between 2D / 2.5D / 3D, but defaults to 2D on init. |
| Cesium Ion / world imagery | **Disabled** | The Viewer is constructed with `baseLayer: false` and `baseLayerPicker: false`. Cesium Ion's default imagery, terrain, and OSM Buildings are NOT loaded. `Cesium.Ion.defaultAccessToken` is settable via a prop but is empty by default — so no Ion asset quota is consumed unless the consumer opts in. |
| Base map tile servers | **None baked in** — wired by the consumer page via `MapBaseLayerProvider` + `MapXyzSourceProvider`. The shipped example wires two public XYZ slippy-map sources: **OpenStreetMap** (`https://tile.openstreetmap.org/{z}/{x}/{y}.png`) and **Carto Light** (`https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}.png`). | Both are public community tile servers. Fine for dev/demos, but production deployments with non-trivial traffic should swap to a self-hosted tileserver or a commercial provider — OSM's and Carto's free tiles have usage policies that prohibit heavy production use. |
| XYZ source rendering | `Cesium.UrlTemplateImageryProvider` | `MapXyzSourceProvider` passes the URL template through unchanged; standard `{z}/{x}/{y}` and `{s}` (subdomain) placeholders work. |
| Default camera | **Singapore** — lon `103.8189`, lat `1.3521`, zoom `8` | Hardcoded in the example page as `SINGAPORE_LONG_LAT` / `DEFAULT_ZOOM_LEVEL`. Pass a different `INITIAL_CAMERA_VIEW` to `<Map>` to override. |
| Live update batching | `batchUpdateInterval=500` (ms) | Example-page value. RTUS entity changes are batched for this interval before being applied to the map to avoid thrashing on high-frequency tracks. |
| Sidebar / navbar entry | `/aoh/gis`, code `GIS`, label `Map`, Lucide `map` icon | Wired in the app's route config. |

**Frontend env vars** (from `.env.template`):

| Var | Purpose |
|---|---|
| `GIS_URL` | gis-service REST API base — server-side fetches. |
| `PUBLIC_RTUS_SEH_URL` | RTUS SEH base for the browser's SSE subscription. Required — the page throws if missing. |
| `PUBLIC_GIS_RTUS_MAP_NAME` | RTUS map to subscribe to. Conventionally `gis` (matches the JSON Map gis-service publishes to). |
| `IAM_URL` / `IAM_CLIENT_ID=gis` | Standard AOH OIDC — Keycloak realm + this app's client. |
| `SDS_URL` | Optional. If set, auth tokens are stored server-side via SDS instead of cookies. |

**Swapping the base map.** To use a different tile server (e.g. a self-hosted MBTiles
server, MapTiler, Stadia Maps), replace the `<MapXyzSourceProvider url="…" />` lines
inside `<MapBaseLayerProvider>` on the page. Multiple `MapBaseLayerProvider` blocks
become user-selectable layers in `MapLayerManager` — order in source = order in the
menu. The engine only understands XYZ raster sources today (`source.type === "xyz"`);
vector tiles, WMS, and WMTS are not wired.

## API Endpoints (gis-service)

All list endpoints support pagination (`page`, `size`, `sort` query params). Pagination is
0-indexed (page starts at 0).

### Entity CRUD

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/geoentity` | List entities (paginated) |
| `POST` | `/geoentity` | Create entity |
| `PUT` | `/geoentity` | Upsert entity (create or update by ID) |
| `POST` | `/geoentity/batch` | Batch create |
| `PUT` | `/geoentity/batch` | Batch upsert |
| `GET` | `/geoentity/id/{id}` | Get entity by internal UUID |
| `GET` | `/geoentity/entity_id/{entityId}` | Get entity by business entity ID |
| `PATCH` | `/geoentity/id/{id}` | Partial update by UUID (requires `occ_lock`) |
| `PATCH` | `/geoentity/entity_id/{entityId}` | Partial update by entity ID (requires `occ_lock`) |
| `DELETE` | `/geoentity/id/{id}` | Delete by UUID |
| `DELETE` | `/geoentity/entity_id/{entityId}` | Delete by entity ID |
| `DELETE` | `/geoentity/id/batch` | Batch delete by UUIDs |
| `DELETE` | `/geoentity/entity_id/batch` | Batch delete by entity IDs |

### Query

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/geoentity/kind/{kind}` | Get all entities of a specific kind |
| `GET` | `/geoentity/property?property={key}&value={value}` | Get single entity by property key-value |

### Bookmarks

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/bookmark` | List bookmarks (paginated) |
| `POST` | `/bookmark` | Create bookmark (name, lon, lat, alt, zoom, pitch, yaw, roll) |
| `GET` | `/bookmark/id/{id}` | Get bookmark |
| `PATCH` | `/bookmark/id/{id}` | Update bookmark (requires `occ_lock`) |
| `DELETE` | `/bookmark/id/{id}` | Delete bookmark |

### Sync

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `POST` | `/sync/full` | Full data sync |
| `POST` | `/sync/partial` | Partial sync |
| `GET` | `/sync/diff` | Get diff since last sync |

## Data Model

```
GeoEntity:
  id: UUID
  occ_lock: integer (optimistic concurrency)
  geojson:
    type: "Feature"
    geometry:
      type: "Point" | "Polygon" | "LineString" | ...
      coordinates: [longitude, latitude]
    properties:
      name: string
      kind: string (optional — business category for grouping)
      ... (custom properties)
  entity_id: string (optional — custom business ID, e.g. "vehicle-001")
  entity_type: string (static | track | geofence | annotation)
  created_at, updated_at: timestamp

Bookmark:
  id: UUID
  name: string
  lon: number (-180 to 180)
  lat: number (-90 to 90)
  alt: number (altitude)
  zoom: number (0-20)
  pitch: number (camera pitch)
  yaw: number (camera yaw)
  roll: number (camera roll)
  occ_lock: integer
```

## Who Uses GIS?

| Service | How it uses GIS |
|---------|----------------|
| **Your service** | Store any location-aware data |

## Access

| URL | What |
|-----|------|
| `http://gis.${DEV_DOMAIN}` | gis-service REST API |
| `http://gis.${DEV_DOMAIN}/swagger-ui/index.html` | Swagger UI |

## Dependencies

- **iams** (Keycloak + AAS for auth)
- **rtus** (real-time entity updates via map `"gis"`)
