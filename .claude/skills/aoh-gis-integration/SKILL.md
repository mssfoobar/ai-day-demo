---
name: aoh-gis-integration
description: >
  Integrate the AOH GIS module (`@mssfoobar/gis-web-sdk`) into a SvelteKit
  consumer app — mount the Cesium/MapLibre map, render live entities,
  wire bookmarks through a server-side BFF, connect the RTUS live feed,
  and supply auth + theme via `<GisProvider>`. Use this skill whenever a
  user wants to add a map / GIS to their AOH app, render geospatial
  entities, show aircraft/vehicle/incident markers, add bookmarks or a
  drawing/measurement panel, consume `@mssfoobar/gis-web-sdk`, or wire the
  GIS service (`gis-service`) into a SvelteKit frontend. Trigger phrases:
  "add a map", "integrate GIS", "use gis-web-sdk", "render entities on a
  map", "GIS bookmarks", "live map feed", "mount GisProvider", "show
  aircraft on the map", "consume the GIS module".
license: Proprietary
metadata:
  owner: AOH Platform Team
  status: v1
---

# AOH GIS Integration

Wire the AOH **GIS module** into a SvelteKit consumer app. The module
ships as the npm package **`@mssfoobar/gis-web-sdk`** (published to GitHub
Packages under the `@mssfoobar` scope); the backend is the Go
**`gis-service`** (REST + an RTUS live feed). This skill is the
integration playbook — it tells you the exact components, props, route
setup, and server-side BFF wiring to get a working, themed, live map.

> **Package naming — read this first.** The published SDK is
> **`@mssfoobar/gis-web-sdk`**. It is *not* `@mssfoobar/gis-sdk` and not
> `@mssfoobar/gis-app` — those names do not exist on the registry.
> Likewise the backend is the Go service **`gis-service`**, not the
> retired Java `gis-app`. If you see those old names anywhere, they're
> stale.

## What you're wiring (mental model)

```
<GisProvider>                         ← app-root: theme store (dark_mode_store)
  <CesiumMapEngineProvider>           ← 3D engine (or MapLibre for 2D)
    <GisMap initial_camera_view rtus_* batchUpdateInterval>   ← the map itself
      <MapEntityLayerProvider> <MapEntityProvider kind=…/>    ← live entities
      <MapBaseLayerProvider> <MapXyzSourceProvider url=…/>    ← base tiles
      <MapBookmarkManager bookmarks={data.bookmarks}>…        ← bookmarks (needs BFF)
      <MapLayerManager>…  <MapControlPanel/>                  ← UI panels
    </GisMap>
  </CesiumMapEngineProvider>
</GisProvider>
```

Two halves to every integration:

1. **Client (`.svelte`)** — the component tree above. Renders the map.
2. **Server (`+page.server.ts` BFF)** — proxies bookmark CRUD (and any
   other GIS REST calls) to `gis-service` with a **server-side** bearer
   token. The browser never sees the token or the gateway URL directly.

## Prerequisites & install

The SDK is a Svelte 5 package. Peer requirements:

- **SvelteKit + Svelte 5** (runes). The map components are `.svelte`
  files consumed via the `svelte` export condition.
- **Tailwind v4** is assumed by the SDK's styles (`@source` registration).
  The components don't *require* Tailwind to function, but the shipped
  look does.
- Peer packages: `@mssfoobar/auth-sdk`, `@mssfoobar/ui`,
  `@mssfoobar/logger`, `@mssfoobar/sse-client`.

Authenticate the scope to GitHub Packages (project-level `.npmrc`):

```ini
@mssfoobar:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${NODE_AUTH_TOKEN}
```
…with `NODE_AUTH_TOKEN` exported (a PAT/`gh auth token` with
`read:packages`). Then:

```bash
pnpm add @mssfoobar/gis-web-sdk@^2.0.0 @mssfoobar/auth-sdk @mssfoobar/ui \
         @mssfoobar/logger@^1.0.5 @mssfoobar/sse-client
```

> Use **gis-web-sdk ≥ 1.1.0** if you need map interaction callbacks
> (`on_click` for click-to-detail) — earlier versions can't wire them.

> **Pin `@mssfoobar/logger` ≥ 1.0.5.** Every earlier registry release is
> broken for the bare specifier the SDK imports: all of `1.0.0`–`1.0.4`
> declare a `main` of `dist/index.js` that doesn't exist in the tarball
> and ship no `exports` map. The actual file location differs by
> version — `1.0.0` ships `dist/Logger.js`; `1.0.1`–`1.0.4` ship
> `Logger.js` at the package root. The SDK's components import bare
> `@mssfoobar/logger`, so any of those versions break `vite build` with a
> resolve error. `1.0.5` has a correct `exports` map and also keeps
> the legacy `@mssfoobar/logger/dist/Logger` deep-import path working.
> If you're stuck on an older version and can't bump, the fallback is a
> vite `resolve.alias` (see the failure-modes table).

> **You also need two Cesium packages as *direct* deps** — see Step 2.
> The SDK only declares them transitively, and pnpm won't let the
> consumer's vite config resolve them otherwise. Add them now (as runtime
> `dependencies`, so a devDep-pruning build can't drop them):
>
> ```bash
> pnpm add @cesium/engine@^25 @cesium/widgets@^15
> ```

## Integration steps

### Step 1. Opt the map route out of SSR

Cesium and MapLibre touch browser globals (`window`, WebGL) at
module-init. The map route **must not** server-render. Add a `+page.ts`
next to your map `+page.svelte`:

```ts
// src/routes/<your-map-route>/+page.ts
export const ssr = false;
```

### Step 2. Serve CesiumJS static assets (or the map renders black)

**Do not skip this — it is the #1 cause of a dead map.** CesiumJS fetches
its Web Workers, textures, widget CSS, and third-party shims **at
runtime** from a global `CESIUM_BASE_URL`. If you mount
`<CesiumMapEngineProvider>` without serving those assets, the map canvas
is **entirely black**, base-layer toggles do nothing (the engine never
initializes), and the console shows 404s like:

```
GET …/node_modules/.vite/deps/Assets/Images/ion-credit.png            404
GET …/node_modules/.vite/deps/Assets/IAU2006_XYS/IAU2006_XYS_18.json  404
GET …/node_modules/.vite/deps/Assets/approximateTerrainHeights.json   404
```

Four things are required. Build your `vite.config.ts` from the block
below — it is complete and copy-pasteable. The essentials:

1. **Direct deps (the pnpm gotcha).** `@cesium/engine` and
   `@cesium/widgets` are only **transitive** deps of `@mssfoobar/gis-web-sdk`
   (it declares `@cesium/engine ^25`, `@cesium/widgets ^15`). Under pnpm's
   strict `node_modules` layout they are **not** resolvable from the
   consumer app, so `require.resolve('@cesium/engine/package.json')` throws
   `MODULE_NOT_FOUND` and the asset-copy plugin can't find them. Add them
   as **direct** deps first, at the SDK's versions (runtime
   `dependencies`, **not** devDeps, so a prune-before-build step can't drop
   them):
   ```bash
   pnpm add @cesium/engine@^25 @cesium/widgets@^15
   ```

2. **Copy the asset dirs into `static/cesium/`** via an inline vite plugin
   (runs on `buildStart` + `configureServer`, `fs.cpSync` recursive +
   force + dereference). The plugin copies four leaf directories:

   | From (resolved via `require.resolve`) | To (under `static/cesium/`) |
   |---|---|
   | `@cesium/engine` → `Build/Workers` | `Workers` |
   | `@cesium/engine` → `Source/Assets` | `Assets` |
   | `@cesium/engine` → `Source/ThirdParty` | `ThirdParty` |
   | `@cesium/widgets` → `Source` | `Widgets` |

3. **Define `CESIUM_BASE_URL` + bundle the engine for SSR.** Paste this
   into your `vite.config.ts` (merge the `plugins` / `ssr.noExternal`
   entries into your existing config — a SvelteKit + Tailwind v4 app
   already has `sveltekit()` + `tailwindcss()`):

   ```ts
   import { createRequire } from "node:module";
   import fs from "node:fs";
   import path from "node:path";

   import tailwindcss from "@tailwindcss/vite";
   import { sveltekit } from "@sveltejs/kit/vite";
   import { defineConfig, type Plugin } from "vite";

   // CesiumJS ships static runtime assets (Web Workers, textures, the widget
   // CSS/JS, third-party shims) that the runtime fetches lazily from
   // `CESIUM_BASE_URL`. Without serving them, every Worker / texture request
   // 404s and the canvas stays black. We copy four directories into the
   // SvelteKit `static/` folder so they're served at
   // `/cesium/{Workers,Assets,ThirdParty,Widgets}/...`. Resolve the @cesium
   // packages via `require.resolve` to handle pnpm's `.pnpm/...` symlink
   // layout robustly.
   const require = createRequire(import.meta.url);
   const cesiumEngineDir = path.dirname(require.resolve("@cesium/engine/package.json"));
   const cesiumWidgetsDir = path.dirname(require.resolve("@cesium/widgets/package.json"));

   const CESIUM_BASE_URL = "/cesium";

   interface CesiumAssetCopySpec {
     from: string;
     to: string;
   }

   const CESIUM_COPY_SPECS: CesiumAssetCopySpec[] = [
     { from: path.join(cesiumEngineDir, "Build/Workers"), to: "Workers" },
     { from: path.join(cesiumEngineDir, "Source/Assets"), to: "Assets" },
     { from: path.join(cesiumEngineDir, "Source/ThirdParty"), to: "ThirdParty" },
     { from: path.join(cesiumWidgetsDir, "Source"), to: "Widgets" },
   ];

   /**
    * Sync Cesium's static assets into `static/cesium/` so SvelteKit's
    * `static/` handler serves them at runtime. Runs on every dev start and
    * every build start — idempotent, overwrite-safe via `force: true`.
    */
   function cesiumAssetCopyPlugin(staticRoot: string): Plugin {
     const cesiumStaticRoot = path.join(staticRoot, "cesium");

     function syncOnce() {
       fs.mkdirSync(cesiumStaticRoot, { recursive: true });
       for (const { from, to } of CESIUM_COPY_SPECS) {
         const dest = path.join(cesiumStaticRoot, to);
         fs.cpSync(from, dest, { recursive: true, force: true, dereference: true });
       }
     }

     return {
       name: "cesium-asset-copy",
       buildStart() {
         syncOnce();
       },
       configureServer() {
         syncOnce();
       },
     };
   }

   const staticRoot = path.resolve(import.meta.dirname, "static");

   export default defineConfig({
     plugins: [cesiumAssetCopyPlugin(staticRoot), sveltekit(), tailwindcss()],
     define: {
       CESIUM_BASE_URL: JSON.stringify(CESIUM_BASE_URL),
     },
     ssr: {
       // Bundle the SDK's compiled `.svelte` (they must traverse the Svelte
       // plugin) + the Cesium packages (so a pnpm-strict runtime image that
       // copies only the root node_modules doesn't have to resolve the
       // `.pnpm/...` virtual store). `devalue` is bundled for the same reason
       // — the GIS BFF uses it to decode superForm's `dataType: 'json'`
       // payload (Step 6).
       noExternal: [
         /^@mssfoobar\//,
         "@lucide/svelte",
         "@cesium/engine",
         "@cesium/widgets",
         "devalue",
       ],
     },
   });
   ```

4. **Gitignore + lint-ignore the generated assets** — add `static/cesium/`
   to your app's `.gitignore` AND to your linter's `ignores` (e.g.
   `eslint.config.js`). They're vendor files copied from `node_modules`;
   without the ignore, the linter reports thousands of errors in the copied
   workers.

### Step 3. Mount `<GisProvider>` (usually at the layout root)

`<GisProvider>` is the GIS SDK's **theme provider**. It owns exactly one
thing — the host app's dark-mode store — and the map's palette repaints when
it changes:

| Prop | Type | Required | Purpose |
|---|---|---|---|
| `dark_mode_store` | `Writable<boolean>` | **yes** | Host-owned theme store; the map palette repaints on change. |

The host owns the theme store; the SDK only *reacts* to it (no global
singleton, no SDK-internal `localStorage`). Mount it at your root
`+layout.svelte` so every GIS component across routes shares one provider:

```svelte
<script lang="ts">
  import { writable } from "svelte/store";
  import { GisProvider } from "@mssfoobar/gis-web-sdk";

  const darkModeStore = writable(false);  // wire to your real theme toggle
</script>

<GisProvider dark_mode_store={darkModeStore}>
  {@render children()}
</GisProvider>
```

> **No `gis_url` / `auth_adapter` props.** The SDK makes no GIS REST calls
> from the browser — all GIS data flows through the host's server-side BFF
> (bookmarks; Step 6) and the RTUS live feed. There is therefore no
> client-side auth hook to supply. (Earlier SDK versions carried inert
> `gis_url` / `auth_adapter` / `feature_flags` props the SDK never read;
> they were removed.)

Components fall back to a standalone light-mode store when no provider is
mounted, so isolated/Storybook use still works.

> Mounting `<GisProvider>` *inside* the page gives a self-contained
> widget — fine to start — but **lift it to the layout root** once more
> than one route uses GIS so entity providers / search / bookmarks can
> be shared.

### Step 4. Resolve config from env (host concern, not the SDK's)

The SDK does **not** read env vars — the host injects them. The env vars in
play (see the gis manifest for the canonical list):

| Var | Side | Purpose |
|---|---|---|
| `GIS_URL` | server | BFF → `gis-service` base URL. Unset → BFF stub mode. |
| `PUBLIC_RTUS_SEH_URL` | browser | RTUS SEH SSE endpoint for the live feed. Unset → no live feed. |
| `PUBLIC_CESIUM_TOKEN` | browser | Cesium ion token for 3D terrain/imagery. Unset → OSM-only 2D. |

All authenticated GIS REST (bookmarks, geo-entity reads) goes through the
host's **server-side** BFF with a server-held token (Step 6) — the browser
never holds a GIS base URL or token, so there's no public GIS API URL to
resolve here.

### Step 5. Build the map page

The component tree, with the props that matter — including a typed custom
entity snippet:

```svelte
<script lang="ts" module>
  import {
    Map as GisMap,
    MapBaseLayerProvider,
    MapXyzSourceProvider,
    MapEntityLayerProvider,
    MapEntityProvider,
    MapBookmarkManager,
    MapLayerManager,
    MapControlPanel,
    CesiumMapEngineProvider,
    type CameraView,
    type MapEntity,
  } from "@mssfoobar/gis-web-sdk";
  import { Button } from "@mssfoobar/ui/button";
  import { env as publicEnv } from "$env/dynamic/public";
  import type { PageData } from "./$types";

  const INITIAL_CAMERA_VIEW: CameraView = {
    position: [103.8189, 1.3521], // [lon, lat] — Singapore
    zoom: 8,
  };
  const RTUS_MAP_NAME = "gis"; // must match gis-service RTUS_MAP_NAME

  function resolveRtusSehUrl(): URL | undefined {
    const v = publicEnv.PUBLIC_RTUS_SEH_URL;
    if (!v) return undefined;
    try { return new URL(v); } catch { return undefined; }
  }
</script>

<script lang="ts">
  let { data }: { data: PageData } = $props();

  // Live feed only subscribes when ALL of seh_url + map_name + user_id +
  // tenant_id are present; otherwise the map renders without a feed.
  //
  // On the aoh-web-init base, `data.user` is `authResult.claims` (the
  // decoded JWT), so the user id is the `sub` claim and the tenant is
  // `active_tenant.tenant_id` — NOT `data.user.id` / `data.user.tenantId`
  // (those are undefined on this base and silently disable the feed). The
  // (private) layout load returns `{ user: authResult.claims }`; thread it
  // through your `+page.server.ts` load as `user`. Server-side — the token
  // never reaches the browser, only the claims.
  const rtusSehUrl = resolveRtusSehUrl();
  const rtusUserId = $derived(data.user?.sub ?? undefined);
  const rtusTenantId = $derived(data.user?.active_tenant?.tenant_id ?? undefined);
</script>

<div class="relative h-full min-h-0 w-full overflow-hidden">
  <CesiumMapEngineProvider>
    <GisMap
      initial_camera_view={INITIAL_CAMERA_VIEW}
      batchUpdateInterval={500}
      rtus_seh_url={rtusSehUrl}
      rtus_map_name={RTUS_MAP_NAME}
      user_id={rtusUserId}
      tenant_id={rtusTenantId}
    >
      <!-- Entities: MapEntityLayerProvider > MapEntityProvider, keyed by `kind`. -->
      <MapEntityLayerProvider layer_name="Boundary">
        <MapEntityProvider
          kind="boundary"
          geom_style={{ fill: "#00FF00", "fill-opacity": 0.25, stroke: "#1E90FF", "stroke-opacity": 1 }}
        />
      </MapEntityLayerProvider>

      <!-- Custom marker via a typed snippet (see references/entities.md). -->
      <MapEntityLayerProvider layer_name="Aircraft">
        <MapEntityProvider kind="aircraft">
          {#snippet children(entity: MapEntity)}
            {#if entity.geojson.type === "Feature"}
              <button type="button" class="rounded bg-orange-500 px-2 py-1 text-xs text-white">
                {entity.geojson.properties.callsign ?? "—"}
              </button>
            {/if}
          {/snippet}
        </MapEntityProvider>
      </MapEntityLayerProvider>

      <!-- Base layers: first declared is the default. -->
      <MapBaseLayerProvider layer_name="Open Street Maps">
        <MapXyzSourceProvider url={`https://tile.openstreetmap.org/{z}/{x}/{y}.png`} />
      </MapBaseLayerProvider>

      <!-- Bookmarks: `bookmarks` is loaded by the BFF (Step 6). -->
      <MapBookmarkManager bookmarks={data.bookmarks}>
        <Button variant="secondary" size="sm" class="pointer-events-auto shadow-md">Bookmarks</Button>
      </MapBookmarkManager>

      <MapLayerManager>
        <Button variant="secondary" size="sm" class="pointer-events-auto shadow-md">Layers</Button>
      </MapLayerManager>

      <MapControlPanel />
    </GisMap>
  </CesiumMapEngineProvider>
</div>
```

**Import types by name from the barrel.** All SDK types are re-exported
from `@mssfoobar/gis-web-sdk` — prefer named type-imports:

```ts
import type { MapEntity, MapEventSubscriptions, CameraView, MapOptions } from "@mssfoobar/gis-web-sdk";
```

> **Aside — the `Gis.*` global.** The type module also declares a global
> `Gis` namespace (`Gis.MapEntity`, `Gis.CameraView`, …). It only resolves
> if the SDK's `types-namespace.d.ts` is in your app's TS `include`; a
> plain consumer that just imports the package gets `Cannot find namespace
> 'Gis'`. **Use the named barrel imports above** — they always resolve.

**Key `<GisMap>` props** (the `MapOptions` type):

| Prop | Type | Notes |
|---|---|---|
| `initial_camera_view` | `CameraView` | `{ position: [lon, lat], zoom, direction? }`. |
| `batchUpdateInterval` | `number` (ms) | How often batched entity-position updates flush. 500 is a sane default; lower for live feeds, raise for huge fleets. |
| `rtus_seh_url` | `URL \| undefined` | RTUS SEH SSE endpoint. |
| `rtus_map_name` | `string` | Must match `gis-service`'s `RTUS_MAP_NAME` (e.g. `"gis"`). |
| `user_id` / `tenant_id` | `string` | From auth claims — on the aoh-web-init base, `data.user?.sub` and `data.user?.active_tenant?.tenant_id`. Required (with the two above) for the live feed to subscribe. |
| `event_subscriptions` | `MapEventSubscriptions` | Host click/hover callbacks (`on_click`, …). **gis-web-sdk ≥ 1.1.0.** |

> **3D vs 2D:** use `CesiumMapEngineProvider` (3D, needs `PUBLIC_CESIUM_TOKEN`
> for terrain/imagery; falls back to OSM 2D without it) or
> `MapLibreEngineProvider` (`@mssfoobar/gis-web-sdk/engines/maplibre`) for
> pure 2D. Pick exactly one and wrap `<GisMap>` in it.

> **Where do map entities come from? (the one mental model)** Entities live
> in one store, `map.state.entities`, and **render** through
> `<MapEntityProvider kind="…">` (a renderer + filter — it has no data prop;
> it draws every entity whose `geojson.properties.kind` matches). Data gets
> **into** the store three ways, all equivalent once stored:
> 1. **The RTUS live feed** — the platform's real-time channel. A backend
>    publishes entities to RTUS under a `kind`; the map subscribes (wire the
>    four `rtus_*` props). This is how shared, multi-client live data flows.
> 2. **`map.upsert_entities(entities)` / `map.remove_entities(ids)`** — the
>    host pushes its *own* data straight onto the map. Grab the map via
>    `bind:map` or a child component's `GetGisMapContext()`. Safe to call
>    from anywhere (auto-deferred; no internal flags). This is how you render
>    your app's records (incidents, trips) **without** routing through RTUS —
>    including with no backend at all (dev/demo).
> 3. **`<MapSingleEntityProvider>`** — a declarative wrapper around (2) for a
>    single host-supplied entity (a "you are here" pin). It renders + tracks
>    the camera like any other entity.
>
> Full recipes (typed snippets, click-to-detail, the host-push + RTUS
> patterns) are in `references/entities.md`.

### Step 6. Wire the BFF for bookmarks (`+page.server.ts`)

`<MapBookmarkManager>` posts to **form actions** `?/create_bookmark` and
`?/delete_bookmark` on the same route, and reads an initial `bookmarks`
list from `load`. The BFF proxies these to `gis-service` with a
server-side token. This is the single most error-prone part — there are
three non-obvious gotchas. **Read `references/bff-bookmarks.md` for the
full, copy-pasteable implementation.** The gotchas:

1. **Auth token handling must fail *closed*.** When auth is enabled
   (`IAM_URL` set) read the user's token from `locals.authResult`
   (SDS-backed, server-side only). If there's no valid token, send **no**
   `Authorization` and let the upstream 401 — never substitute a dev/test
   identity against a real service (that's a fail-open identity swap).
   Only when auth is disabled may you use a deterministic dev bearer.
2. **The form payload is devalue-encoded, not JSON.** `<MapBookmarkManager>`
   drives its forms with superforms' `superForm({ dataType: 'json' })`,
   which serializes with **devalue** under `__superform_json` field(s).
   Parse with `devalue.parse(formData.getAll('__superform_json').join(''))`,
   not `JSON.parse`. A naive parse silently yields an empty payload and
   every create fails validation.
3. **Actions must return a `SuperValidated`-shaped `form`.** The client's
   `superForm` enhance throws "No form data returned from ActionResult"
   unless the action result carries a `form` object with `id` (""),
   `valid`, `errors`, `data`. Without it the component's `onResult` (which
   adds/removes the bookmark on the map) never runs. `create` also returns
   the created `bookmark`; `delete`'s `onResult` reads
   `result.data.form.data.id`.

Bookmark REST contract on `gis-service`: `GET /bookmark` (→ `{ data: [...] }`),
`POST /bookmark` (→ `{ data: {...} }`), `DELETE /bookmark/id/{id}` (→ 204).
Validate `lon ∈ [-180,180]`, `lat ∈ [-90,90]` before the round-trip.

### Step 7. Navigation (optional)

The SDK exports nav metadata: `import { gisNav } from "@mssfoobar/gis-web-sdk/nav"`.
The two shapes are **not** identical:

- `gisNav.header` is `{ name, url }` — **no `icon`**.
- `gisNav.sidebar` is `{ name, url, icon }[]`, where `icon` is a Svelte
  `Component` (a `@lucide/svelte` icon).

On the `aoh-web-init` base, the sidebar reads `navItems: NavItem[]` from
`src/lib/aoh/core/components/layout/nav.ts`, where
`NavItem = { name; url; group?; icon?: Component; roles? }`. `icon` is
**optional** there, so both shapes assign cleanly — feed `gisNav.sidebar`
straight in (its `icon` satisfies the optional `Component`), and if you
also surface the header entry, drop it in without an icon:

```ts
// src/lib/aoh/core/components/layout/nav.ts
import { gisNav } from "@mssfoobar/gis-web-sdk/nav";

export const navItems: NavItem[] = [
  ...gisNav.sidebar,               // { name, url, icon } — icon is a Component
  // header entry has no icon; NavItem.icon is optional, so this type-checks:
  // { name: gisNav.header.name, url: gisNav.header.url },
];
```

`url`s default to `/aoh/gis` — override if you mounted GIS elsewhere.

### Step 8. Styles

The SDK's Tailwind `@source` directives must be in your CSS pipeline.
Chain the SDK + ui stylesheets through a **single** CSS entry (so the
`@source` directives aggregate in one Tailwind pass) rather than two JS
imports — a CSS-level `@import`, not a JS import:

```css
/* src/app.css */
@import "@mssfoobar/ui/styles/app.css";           /* owns @import 'tailwindcss' + @theme */
@import "@mssfoobar/gis-web-sdk/styles/app.css";
```

The first `@import` must own `@import 'tailwindcss'` + the `@theme` block
(here `@mssfoobar/ui`); the GIS line is a thin `@source` bridge that puts
the SDK's compiled output under Tailwind's content scan. Two separate JS
imports run in separate Tailwind passes and the SDK's `@source` globs are
lost.

## Verify it works

1. `pnpm dev` → visit the route. Map renders (OSM tiles even without a
   Cesium token).
2. **Bookmarks**: create one via the panel → it appears; reload → it
   persists (BFF stub or real service). If create silently no-ops, you
   hit gotcha #2 (devalue) or #3 (SuperValidated shape).
3. **Entities (no backend)**: drop a child of `<GisMap>` that calls
   `map.upsert_entities(...)` on a timer (see the copy-pasteable snippet in
   `references/entities.md` → "push your own entities") and declare a
   matching `MapEntityProvider kind="…"` — markers should render and move
   with no RTUS/gis-service at all. This is the fastest way to confirm your
   render wiring.
4. **Live feed**: with `PUBLIC_RTUS_SEH_URL` + auth claims set, entities
   pushed to RTUS appear/move. If nothing shows, confirm all four
   (`rtus_seh_url`, `rtus_map_name`, `user_id`, `tenant_id`) are present —
   the map silently skips subscription if any is missing.
5. No SSR error on load → Step 1 (`ssr = false`) is in place.
6. **Map actually draws** (tiles + camera, not a black rectangle). A
   black canvas with `Assets/*`/worker 404s in the console means Step 2
   (Cesium assets) isn't done.

## Common failure modes

| Symptom | Cause | Fix |
|---|---|---|
| `window is not defined` / WebGL error on load | Route is server-rendering Cesium | Add `+page.ts` with `export const ssr = false` (Step 1). |
| **Map canvas is black**; console 404s for `Assets/*.json`, `Images/ion-credit.png`, worker chunks under `.vite/deps/Assets/…`; base-layer toggles do nothing | Cesium runtime assets not served / `CESIUM_BASE_URL` unset | Add the `cesiumAssetCopyPlugin` + `define CESIUM_BASE_URL='/cesium'` + `ssr.noExternal` (Step 2). Also add `@cesium/engine`+`@cesium/widgets` as **direct** deps — they're only transitive via the SDK and unresolvable under pnpm. |
| `vite build` fails resolving `@mssfoobar/logger` (`dist/index.js` not found or bare specifier unresolvable) | Every published `@mssfoobar/logger` ≤ 1.0.4 is broken: stale `main`, no `exports` map (1.0.1–1.0.4 ship files at the package root); the SDK's components import the bare specifier | Bump to `@mssfoobar/logger@^1.0.5`. If stuck on ≤ 1.0.4, alias the bare specifier in `vite.config.ts` (it's ESM, so use `createRequire`): `const require = createRequire(import.meta.url);` then `resolve.alias: [{ find: /^@mssfoobar\/logger$/, replacement: require.resolve('@mssfoobar/logger/Logger.js') }]` (1.0.1–1.0.4 root layout; use `…/dist/Logger.js` for 1.0.0). |
| `npm error 404 @mssfoobar/gis-sdk` | Wrong package name | It's `@mssfoobar/gis-web-sdk`. |
| Create bookmark silently does nothing | BFF parsed payload as JSON, or returned no `form` | devalue-parse `__superform_json`; return a `SuperValidated`-shaped `form` (Step 6, gotchas 2 & 3). |
| Map renders but no live entities from RTUS | A required RTUS prop missing | Provide all of `rtus_seh_url`, `rtus_map_name`, `user_id`, `tenant_id` (the feed silently skips subscription if any is absent). Or, to render without a feed, push your own via `map.upsert_entities(...)`. |
| My REST data (incidents/trips) won't render via `MapEntityProvider` | `MapEntityProvider` has no data prop — it only *renders* what's in the store | Put your records in the store: call `map.upsert_entities(entities)` (host-owned data, no backend needed), or have a backend publish them to RTUS under a `kind`. Declare a `MapEntityProvider kind="…"` matching their `properties.kind`. See `references/entities.md`. |
| Markers never appear / freeze when I push my own entities | Writing `map.state.entities` **synchronously inside a `$effect`** (self-references the reactive map and races the provider) | Use `map.upsert_entities(...)` / `map.remove_entities(...)` — they auto-defer and hide the internal flags. Never hand-write `is_dirty`/`is_need_update`/`is_deleted` or `map.state.entities.set` from a reaction. |
| `on_click` / marker click does nothing | SDK < 1.1.0, or callbacks swapped after mount | Upgrade to `@mssfoobar/gis-web-sdk` ≥ 1.1.0; pass a stable `event_subscriptions` to `<GisMap>` (read once at init). |
| `state_unsafe_mutation` crash; an entity layer won't render (markers never appear, e.g. "can't enable trips") | A marker `children` snippet mutates `$state` during render (e.g. registering each entity into a `SvelteMap`/registry) | Keep the snippet **pure** (render-only). Resolve a clicked id by fetching `GET /geoentity/id/{id}` (fallback `/geoentity/entity_id/{id}`), not from a snippet-built registry. See `references/entities.md` → "Reacting to clicks". |
| Entities appear but never disappear | Missed RTUS Removed events / orphaned keys | Reconcile against the `init` snapshot on (re)connect (see `references/entities.md` → "orphaned markers"). |
| User actions (delete/edit an entity or annotation) feel laggy — the UI waits for the server | Mutating only **after** the `await` returns (the slow path an agent writes by default) | Update the store **first**: call `map.remove_entities([id])` / `map.upsert_entities([e])`, fire the backend write after, roll back on failure. The mutator is idempotent so the RTUS echo reconciles for free. See `references/entities.md` → "Optimistic updates". |
| Map theme doesn't follow app dark mode | No `dark_mode_store` wired through `<GisProvider>` | Pass a host-owned `Writable<boolean>` (Step 3). |
| 401 from `gis-service` on bookmarks | BFF sent no/dev token against real auth | Forward `locals.authResult` token; fail closed, don't swap identity (Step 6, gotcha 1). |
| Only the **last-declared** entity layer shows on fresh load; the others' toggles start off even though every provider set `is_visible: true` | `MapControlPanel` ≤ 1.2.0 force-hid all entity layers except the last registered in its mount pass | Upgrade `@mssfoobar/gis-web-sdk` to a version whose changelog notes the entity-layer-visibility fix (visibility is left as each provider declared). On affected versions, re-assert each layer's `is_visible` **after first paint** (e.g. `setTimeout`) — the panel's stomp runs after an async dynamic import inside its `onMount`, so a host `onMount` alone can still be overridden. |
| Layer toggles don't survive navigation | `MapLayerManager` visibility is in-memory; persistence is host-owned | Persist the toggle state yourself (store + sessionStorage); the SDK keeps no session state. |
| Map canvas grows past viewport, controls drop below fold | Unbounded map container height | Bound the shell: `h-svh min-h-0 overflow-hidden` on the layout, `min-h-0` on the map wrapper. |

## References

- `references/components.md` — the full component + props catalog (every
  SDK export, what it does, when to use it).
- `references/bff-bookmarks.md` — the complete, copy-pasteable
  `+page.server.ts` BFF (auth, devalue parsing, SuperValidated shape,
  stub fallback).
- `references/entities.md` — entity rendering deep-dive: `kind`,
  `geom_style`, typed custom-marker snippets, **the one entity model** (RTUS
  feed vs `map.upsert_entities` vs `MapSingleEntityProvider`; how to plot
  your own domain data, backend-free), **optimistic updates** (mutate the
  store first, reconcile via the idempotent mutator), and **click-to-detail**
  wiring via `event_subscriptions`.
- `references/auth-and-config.md` — the server-side BFF token flow + env
  resolution + the SDS server-side token pattern.
