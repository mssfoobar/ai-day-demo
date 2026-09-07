# Entity rendering & the RTUS live feed

How the GIS map renders geospatial entities, styles them, draws custom
markers, and stays live via RTUS — the behaviour of the SDK's `Map` /
`MapEntityProvider` components and how you feed them data.

> **Types: import them by name from the barrel.** The snippets below
> write `Gis.MapEntity`, `Gis.MapEventSubscriptions`, `Gis.GeoEntity`,
> `Gis.CameraView`, etc. for brevity, but the global `Gis` namespace only
> resolves inside the SDK's own build. In a consumer app, import the type
> by name and drop the `Gis.` prefix:
> `import type { MapEntity, MapEventSubscriptions, GeoEntity, CameraView } from "@mssfoobar/gis-web-sdk";`
> then use `MapEntity<AircraftProperties>` in place of
> `Gis.MapEntity<AircraftProperties>`. Otherwise you get `Cannot find
> namespace 'Gis'`.

## The provider nesting

Entities are always two providers deep:

```svelte
<MapEntityLayerProvider layer_name="Aircraft">   <!-- toggleable named layer -->
  <MapEntityProvider kind="aircraft" … />        <!-- renders entities of one kind -->
</MapEntityLayerProvider>
```

- **`MapEntityLayerProvider`** groups entities under a `layer_name` that
  shows up in `<MapLayerManager>` with a visibility toggle.
- **`MapEntityProvider`** subscribes to entities of one **`kind`** (e.g.
  `"aircraft"`, `"boundary"`, `"airport"`, `"incident"`, `"responder"`).
  The `kind` is your app's entity classifier — the live feed and any
  REST-seeded entities are bucketed by it.

You can declare multiple layers, each with its own kind and styling.

## Styling polygons / lines: `geom_style`

For non-point geometries (polygons, polylines), pass a `geom_style`:

```svelte
<MapEntityProvider
  kind="boundary"
  geom_style={{
    fill: "#00FF00",
    "fill-opacity": 0.25,
    stroke: "#1E90FF",
    "stroke-opacity": 1,
  }}
/>
```

Keys mirror common GeoJSON/Mapbox paint props (`fill`, `fill-opacity`,
`stroke`, `stroke-opacity`, …). Without a style, geometries render with
the engine's defaults.

## Custom point markers: the typed `children` snippet

For points, render a fully custom marker by passing a `children` snippet
typed to `Gis.MapEntity`. The snippet receives each entity; branch on the
GeoJSON and read typed `properties`:

```svelte
<MapEntityLayerProvider layer_name="Aircraft">
  <MapEntityProvider kind="aircraft">
    {#snippet children(entity: Gis.MapEntity<AircraftProperties>)}
      {#if entity.geojson.type === "Feature"}
        <button
          type="button"
          class="pointer-events-auto rounded-md bg-orange-500 px-2 py-1 text-xs font-semibold text-white shadow"
          style:transform={`rotate(${entity.geojson.properties.track - 45}deg)`}
          aria-label={`Aircraft ${entity.geojson.properties.callsign ?? "unknown"}`}
        >
          {entity.geojson.properties.callsign ?? "—"}
        </button>
      {/if}
    {/snippet}
  </MapEntityProvider>
</MapEntityLayerProvider>
```

Where `AircraftProperties` is your app's typed GeoJSON properties shape,
e.g.:

```ts
interface AircraftProperties {
  callsign?: string;
  track: number; // heading in degrees
}
```

Without a snippet, entities render as default point markers. You only need
a snippet for the `kind`s you want custom styling on; every other `kind`
shows as a default point — that's expected, not a bug.

## The RTUS live feed

The map subscribes to **RTUS** (Real-Time Update Service) over SSE for
live entity create/update/remove. Wiring (on `<GisMap>`):

```svelte
<GisMap
  rtus_seh_url={rtusSehUrl}     {/* URL of the RTUS SEH endpoint */}
  rtus_map_name="gis"           {/* must equal gis-service's RTUS_MAP_NAME */}
  user_id={rtusUserId}          {/* from auth claims */}
  tenant_id={rtusTenantId}      {/* from auth claims */}
  batchUpdateInterval={500}
>
```

Critical behaviors:

- **All-or-nothing subscription.** The map subscribes only when
  `rtus_seh_url` AND `rtus_map_name` AND `user_id` AND `tenant_id` are
  all present. If any is missing it renders without a feed — no error,
  no log. First thing to check when "the map shows but nothing's live."
- **Per-tenant maps.** RTUS json-maps are keyed per tenant. `tenant_id`
  scopes which entities you see. A mismatch between the logged-in tenant
  and the tenant whose entities you expect = empty map.
- **Browser-side read path.** The authoritative live read is the browser
  SSE subscription via RTUS SEH (with the session cookie). The PMS side
  is write-only.
- **Keepalive.** The SSE heartbeat must exceed the RTUS keepalive window
  or you get false "Map Connection Lost" dialogs. The SDK's `Map` ships a
  35s heartbeat for this reason (≈2× the 15s RTUS keepalive).
- **`batchUpdateInterval`** controls how often position updates flush to
  the renderer — a throughput/latency knob, not a correctness one.

### Known sharp edge: orphaned markers

If the map misses RTUS *Removed* events (e.g. a disconnect), entities can
persist as orphaned markers. The robust fix is to **reconcile the entity
set against the `init` snapshot on every (re)connect** — drop anything
not in the snapshot. If you see stale markers that never disappear, this
is why.

## Reacting to clicks (marker → detail panel)

To open a detail side panel when an operator clicks an entity, pass an
`event_subscriptions` object to `<GisMap>` (**requires
`@mssfoobar/gis-web-sdk` ≥ 1.1.0**). The map fires `on_click` from its
default `pointer` interaction mode with the clicked entity IDs:

```svelte
<script lang="ts">
  let selected = $state<string | undefined>(undefined);

  const eventSubscriptions: Gis.MapEventSubscriptions = {
    on_click: ({ entity_ids, map_entity_ids, position, cursor }) => {
      // IMPORTANT: `map_entity_ids` holds your DOMAIN entity ids — the
      // `id` you published to RTUS, and the key into the map's entity
      // store. `entity_ids` holds the engine's internal per-render ids.
      // Use map_entity_ids[0] to resolve the clicked record. (This is the
      // SDK's own internal pattern: state.entities.get(map_entity_ids[0]).)
      selected = map_entity_ids?.[0];
    },
    // on_right_click and on_mouse_move carry the same shape
    // (on_mouse_move has no entity_ids/map_entity_ids).
  };
</script>

<GisMap {initial_camera_view} event_subscriptions={eventSubscriptions} …>
  …
</GisMap>

{#if selected}
  <!-- render @mssfoobar/ui Sheet / side panel for `selected` -->
{/if}
```

Notes:
- `on_click` fires for every pointer-mode click. Both id arrays are
  empty when clicking empty space (use that to close the panel).
- **`map_entity_ids` vs `entity_ids` (don't mix these up):**
  `map_entity_ids` are your **domain** entity IDs — the `id` your backend
  published to RTUS, and the key the map's entity store is keyed by. Use
  `map_entity_ids[0]` to look up the clicked record (the SDK itself does
  `state.entities.get(map_entity_ids[0])`). `entity_ids` are the render
  engine's **internal** per-feature IDs (e.g. Cesium's auto-generated
  object ids) — you rarely need them.
- Callbacks are read **once at init** — supply a stable object, don't
  swap it reactively after mount.
- Before 1.1.0 these callbacks were typed but never invoked and had no
  public setter — if click does nothing, check the installed SDK version.

### ⚠️ Do NOT mutate state inside the marker snippet

It's tempting to resolve the clicked record by building your own registry
of entities *inside* the `MapEntityProvider` `children` snippet — e.g.
`{@const _ = register(entity)}` that writes each entity into a `$state`
object or `SvelteMap`. **In Svelte 5 this throws and takes the whole
layer down** (markers never render — the symptom looks like "the trips
layer won't enable"):

```text
Svelte error: state_unsafe_mutation
Updating state inside `$derived(...)`, `$inspect(...)` or a template
expression is forbidden.
```

The marker snippet runs inside a reactive render context, so it must be
**pure** (render-only). Don't accumulate state in it.

**Resolve the clicked id by fetching it instead.** On `on_click`, call
the GIS service with the clicked id:

```ts
async function resolveClicked(id: string) {
  // Adjust this base path to however YOUR app reaches gis-service:
  //   - if you have the AOH gateway catch-all route, use
  //     `/aoh/gateway/gis/geoentity/...` (gateway injects the token);
  //   - otherwise proxy through your own `+server.ts` / load BFF.
  // There is NO gateway route by default — a hardcoded `/aoh/gateway/...`
  // in a gateway-less app silently 404s and the panel never opens.
  const base = "/aoh/gateway/gis/geoentity";
  // map_entity_ids[0] is usually the GIS UUID, but may be the business
  // entity_id depending on what the backend published — try both.
  let res = await fetch(`${base}/id/${id}`);
  if (!res.ok) res = await fetch(`${base}/entity_id/${id}`);
  return res.ok ? (await res.json()).data : undefined;
}

const eventSubscriptions: Gis.MapEventSubscriptions = {
  on_click: async ({ map_entity_ids }) => {
    const id = map_entity_ids?.[0];
    selected = id ? await resolveClicked(id) : undefined;
  },
};
```

Fetching on click is also the **right** pattern regardless of the
crash: the fresh record carries the current `occ_lock`, which you need
for any subsequent `PATCH /geoentity/id/{id}` (optimistic-concurrency
update). A snippet-built cache would hand you a stale lock.

> **Auth note:** if your app already has the AOH gateway route
> (`/aoh/gateway/<module>/[...path]`), these authenticated geo-entity
> reads/writes can go **straight through it** from client code — the
> gateway injects the server-side token. You only need a hand-written
> BFF (`references/bff-bookmarks.md`) for the bookmark **form actions**,
> not for plain geo-entity CRUD.

## Where do entities come from? (the one mental model)

**This is the single most important thing to understand.** There is one
entity store, `map.state.entities`. `<MapEntityProvider kind="…">` *renders*
what's in it (filtered by `geojson.properties.kind`) — it has **no** data
prop; you never hand it an array. Data gets **into** the store three ways,
and once stored they're indistinguishable:

1. **The RTUS live feed** — the platform's real-time channel.
2. **`map.upsert_entities(...)` / `map.remove_entities(...)`** — host code
   pushes its own data.
3. **`<MapSingleEntityProvider>`** — a declarative wrapper around (2) for a
   single entity.

So: pick how the *data* arrives (RTUS for shared real-time data; the host
API for your own data / dev / one-offs), then render it the same way with a
`MapEntityProvider kind="…"` (+ snippet) per kind.

### Option A — push your own entities (`map.upsert_entities`)

Use this for your app's domain records (incidents, trips, vehicles), for any
data you already have client-side, and for local dev with **no backend at
all**. Grab the map context from a child of `<Map>` and push plain
`GeoEntity` objects:

```svelte
<!-- A child of <Map>. A minimal "demo entity source" — push on a timer. -->
<script lang="ts">
  import { GetGisMapContext } from "@mssfoobar/gis-web-sdk/map";
  const map = GetGisMapContext();

  // Whenever your data changes, push it. upsert moves existing ids, adds new ones.
  function sync(records: MyRecord[]) {
    map.upsert_entities(records.map((r) => ({
      id: r.id,
      entity_id: r.id,
      entity_type: "track",                    // static | track | geofence | annotation
      geojson: {
        type: "Feature",
        geometry: { type: "Point", coordinates: [r.lon, r.lat] },
        properties: { kind: "incident", ...r }, // `kind` must match a MapEntityProvider
      },
    })));
  }
  // ...and map.remove_entities([id, …]) when records go away.
</script>
```

`upsert_entities` / `remove_entities` are **safe to call from anywhere** —
including inside a `$effect` or a subscription — because they apply their
writes on a microtask. **Never** write `map.state.entities` directly or set
`is_dirty` / `is_need_update` / `is_deleted` yourself: doing that from a
reaction self-references the reactive map and races the renderer, so the
marker silently never appears (this used to be a sharp edge; the API removes
it). You still declare a `<MapEntityProvider kind="incident">` (+ snippet) to
render them.

> **Advanced — host-side bridges.** `upsert_entities` takes a `GeoEntity` and
> builds the `MapEntity` for you. For the rare case that needs `MapEntity`-level
> control the upsert API doesn't expose (a custom `geojson` Feature, bespoke
> render flags) — e.g. bridging another live source (such as MSR replay) onto
> the GIS map — you *may* write `map.state.entities` directly, but **only
> deferred** (`queueMicrotask`), **never synchronously inside a reaction**. The
> rule that actually matters is "mutate the store *outside* the reaction," not
> "never touch the store" — that microtask detachment is exactly what the upsert
> API does for you.

### Option B — the RTUS live feed (shared, multi-client real-time)

For data that must be live across all clients (the platform's own tracks,
another team's service), a **backend publishes to RTUS** under a `kind` and
the map subscribes:

1. A backend writes entities to RTUS as GeoJSON features, each with a `kind`
   and a stable `id`. (`gis-service` itself does this via a
   transactional-outbox → RTUS publisher — the canonical pattern; a
   `PUT /geoentity` upsert enters that pipeline.)
2. The map subscribes to that RTUS map (`rtus_map_name`) for the user's
   tenant (the four `rtus_*` props on `<GisMap>`).
3. Declare a `MapEntityLayerProvider` + `MapEntityProvider kind="…"` per kind.

Internally the SDK's RTUS handler funnels through the **same**
`upsert_entities` / `remove_entities` path, so RTUS-fed and host-pushed
entities render identically. For development against a real `gis-service`,
`scripts/seed-geoentities.sh` upserts through `PUT /geoentity`.

### `MapSingleEntityProvider` — one entity, declaratively

`MapSingleEntityProvider` renders a **single** host-supplied entity straight
from props (`id`, `entity_id`, `entity_type`, `geojson`, plus a `children`
snippet) — it's sugar over `map.upsert_entities` for the one-off case. Use it
for a fixed "you are here" pin or a single highlighted location. It must be a
child of `<Map>`; it renders and tracks the camera like any other entity
(change its `geojson` prop and the marker moves). Give the entity a
`geojson.properties.kind` to enable click/hover hit-testing. Entities
rendered this way are **not** part of a `MapEntityLayerProvider` layer, so
they don't show in `MapLayerManager` — that's the intended trade-off for a
one-off. For a collection that grows/changes, use `upsert_entities` (Option
A) or RTUS (Option B) with a `MapEntityProvider`.

## Optimistic updates: act on the store first, reconcile second

**When the user mutates an entity (delete, move, edit an annotation), update
the store immediately — do *not* await the server round-trip.** This is the
single biggest perceived-latency win on the map, and the SDK is built for it.

Why it's safe: `map.state.entities` is a **local cache**, and
`upsert_entities` / `remove_entities` mutate it on the next microtask — no
network in the path. Both your call and the eventual server echo funnel
through one **idempotent** mutator (`Map/components/entity-changes.ts`):

- `remove_entities([id])` of an id that's already gone is a **no-op**.
- `upsert_entities([e])` is **last-writer-wins** and *revives* a just-removed
  id (it clears `is_deleted`).

So the server's later truth — whether echoed back over RTUS or returned from
your own write — simply re-applies over your optimistic change. Apply the
user's intent now; let reconciliation happen for free.

### The pattern (delete)

```ts
const map = GetGisMapContext();

async function deleteEntity(entity: Gis.GeoEntity) {
  map.remove_entities([entity.id]); // 1. optimistic: marker disappears now
  try {
    // 2. fire the backend write in the background (gateway or your BFF)
    const res = await fetch(`/aoh/gateway/gis/geoentity/id/${entity.id}`, { method: "DELETE" });
    if (!res.ok) throw new Error(`delete failed: ${res.status}`);
  } catch (err) {
    map.upsert_entities([entity]); // 3. roll back — put it back on failure
    throw err;
  }
}
```

Edit is the mirror image: `upsert_entities([edited])` first, then
`PATCH /geoentity/id/{id}` with the record's `occ_lock` (see "Reacting to
clicks" for why you fetch the fresh lock first); on a `409`/failure,
re-`upsert_entities` the server's authoritative copy to undo.

### The one caveat — RTUS-backed entities can briefly revive

For **host-owned data** (pushed via `upsert_entities`, no RTUS feed for that
`kind`) optimistic updates are clean: you own the store, nothing echoes back.

For **RTUS-backed entities**, the feed is still streaming. If an in-flight
`Updated` event for the id you just removed lands *before* your backend's
delete propagates to RTUS, the batch's `upsert` **revives** the marker
(`is_deleted` cleared) — a brief flicker until the authoritative `Removed`
arrives. Options: accept the flicker (usually < one `batchUpdateInterval`),
or keep a short-lived "pending-removed" guard set in the host and drop echoes
for those ids until your delete is confirmed. Don't try to "fix" this by
writing `is_deleted` yourself — that races the renderer (see Option A).

### Bookmarks: delete is optimistic, create isn't

`<MapBookmarkManager>` **delete** is optimistic out of the box (SDK ≥ the
release noted in its changeset): the entry leaves the panel on submit and is
restored if the BFF round-trip fails — the same act-first-reconcile shape,
implemented with a tiny `optimisticDeleteBookmark` helper (a good model for
your own list deletes). **Create** legitimately waits: the id is
server-assigned, so there's nothing to render until the round-trip returns.

Caveat — unlike the RTUS entity store, the bookmark store has **no echo
channel**: it's seeded once on mount and never reconciled, so a `success`
result is trusted outright (a backend that returns 2xx without actually
deleting will diverge until a reload). That's the deliberate trade-off for a
resource with no live feed; the "reconcile for free" guarantee above applies to
the RTUS-backed entity store, not to bookmarks.

## Camera & coordinates

- Positions are `[longitude, latitude]` (and optionally altitude) —
  **lon first**, GeoJSON order. A common bug is swapping to lat/lon.
- `Gis.CameraView` = `{ position: EntityPosition, zoom: number, direction?: [pitch, yaw, roll] }`.
- Zoom levels run 0–20 (OSM convention).
- The `Map` context exposes camera helpers (`get_camera_position`,
  `fly_to`, etc.) once mounted — see `Gis.MapOptions` / the `Map`
  component for the full callback surface.
