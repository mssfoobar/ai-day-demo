# SDK reference — `DashboardClient` and friends

The SDK ships from `@mssfoobar/dash-web-sdk` (HTTP client + types) and `@mssfoobar/dash-web-sdk/renderer` (UI components + widget API). This file lists every method/export you'll actually use in a management page.

## Construction

The client targets a **single backend** (`dashURL`). The dash service handles dashboard, widget, category, **and tag** requests — tags are owned by dash itself, not a separate tag service, so there is no `tagURL`. The base URL is prepended to the request path (`{dashURL}/dashboard/...`, `{dashURL}/v1/tag/...`). It defaults to `""`, which sends **origin-relative** requests (`/dashboard/...`, `/v1/tag/...`). Set it to wherever the dash service is reachable in your deployment.

```ts
import { DashboardClient } from "@mssfoobar/dash-web-sdk";

// In a +page.server.ts load function — pass SvelteKit's fetch.
// Origin-relative (default): works when your app proxies /dashboard and /v1/tag.
const client = new DashboardClient({ fetch });

// In a +page.svelte (browser) — origin-relative.
const client = new DashboardClient();

// Explicit service URL + auth callback.
const client = new DashboardClient({
  dashURL: "https://dash.example.com",
  getAccessToken: () => auth.getToken(),   // per request → Authorization: Bearer <token>
});
```

### `DashboardClientConfig`

| Option | Type | Default | Use when |
|---|---|---|---|
| `dashURL` | `string` | `""` (origin-relative) | Base URL of the dash service — used for **all** dashboard / widget / category / favourite **and tag** requests (and `checkHealth`). |
| `getAccessToken` | `() => string \| null \| undefined \| Promise<…>` | — | Returns the JWT to attach as `Authorization: Bearer` on every request. Called per request, so it can return a refreshed token. Omit if cookies / transport auth already authenticate the requests. |
| `timeout` | `number` | `30000` | Long requests |
| `fetch` | `typeof fetch` | global `fetch` | **Always in `+page.server.ts`** — pass SvelteKit's `fetch` so cookies & relative URLs work during SSR |

> Dashboard layout (column count, cell height, margin) is a UI concern, not a client concern. Pass overrides directly to the component: `<DashboardCanvas config={{ column: 24 }} />`. With no `config`, the dashboard is 42 columns wide with square auto-sized cells and 0.5rem margin between cells.

## The `Result` pattern

Every API method returns `Result<T>`. **Methods never throw on HTTP errors** — they return `{ ok: false, error: DashboardClientApiError }` so you can handle problems explicitly.

```ts
const result = await client.listDashboards();
if (!result.ok) {
  logger.error({ error: result.error });   // result.error has .status, .code, .message, .displayMessage
  toast.error("Could not load dashboards");
  return;
}
// result.data is the payload, result.page is pagination metadata (when applicable)
console.log(result.data);
```

`result.error.displayMessage` is a user-friendly string (mapped from HTTP status) — show it directly in toasts when you don't want to write the message yourself.

`Promise.reject` only happens for programming errors (bad config, etc.). For network/timeout failures, you get `{ ok: false, error: { status: 408, code: "request_timeout" } }`.

## Dashboard methods

| Method | Returns | Notes |
|---|---|---|
| `listDashboards({ sort?, asc?, page?, size?, name? })` | `Result<ListDashboardsResponse>` — `data: Dashboard[]`, `page: { count, number, size, total_records, sort }` | Paginated. `name` is a partial-match filter. |
| `getDashboardById({ dashboard_id })` | `Result<GetDashboardByIdResponse>` — includes `data.widgets` | The full dashboard with embedded widgets and tag refs. |
| `getDashboardByName({ name })` | `Result<GetDashboardByNameResponse>` | Used for uniqueness checks. |
| `createDashboard({ name, description, favourite?, tags?, widgets? })` | `Result<CreateDashboardResponse>` | Creates dashboard + widgets in one transaction. |
| `updateDashboard({ dashboard_id, name, description, favourite, occ_lock, tags?, oldWidgetIds?, widgets? })` | `Result<UpdateDashboardResponse>` | Pass `occ_lock` from your loaded copy. `oldWidgetIds` lets the server compute deletes. |
| `deleteDashboard({ dashboard_id })` | `Result<void>` | Cascades to widgets and tag mappings. |
| `setFavourite({ dashboard_id, favourite })` | `Result<void>` | Toggle without a full update. |
| `favouriteDashboard(...)` | alias of `setFavourite` | |

### `Dashboard` shape

```ts
type Dashboard = {
  id: string;
  occ_lock: number;             // optimistic concurrency token
  name: string;
  description: string;
  favourite: boolean;
  widgets?: Omit<Widget, "occ_lock" | "dashboard_id">[];
  tags?: Tag[];
};
```

## Widget methods

| Method | Returns | Notes |
|---|---|---|
| `getWidgetById(widgetId)` | `Result<{ data: Widget }>` | Used to refresh `occ_lock` before a per-widget save. |
| `addWidget({ dashboard_id, widget_type_id, row, column, width, height, config? })` | `Result<void>` | Add to an existing dashboard without a full update. |
| `editWidget({ id, occ_lock, dashboard_id, widget_type_id, row, column, width, height, config? })` | `Result<EditWidgetResponse>` | Per-widget save. Returns the new `occ_lock`. |
| `deleteWidget({ widget_id })` | `Result<void>` | |
| `listWidgetTypes()` | `Result<ListWidgetTypesResponse>` — `data: WidgetType[]` | Fetched on create/edit pages to apply overrides. |
| `createWidgetType({ id, name, icon, min_width, min_height, max_width?, max_height?, category_id?, limit?, enabled? })` | `Result<CreateWidgetTypeResponse>` | `syncWidgetTypes` calls this for each file-only widget. 409 means it already exists. |
| `updateWidgetType({ id, updates })` | `Result<UpdateWidgetTypeResponse>` | For admin tooling (rename, recategorize, etc.). |

### `Widget` shape

```ts
type Widget = {
  id: string;
  occ_lock: number;
  dashboard_id: string;
  widget_type_id: string;       // matches `MyWidget.type` on the file side
  row: number;
  column: number;
  width: number;
  height: number;
  config?: Record<string, unknown>;
};
```

### `WidgetType` shape

```ts
type WidgetType = {
  id: string;                   // matches MyWidget.type
  occ_lock?: number;
  name: string;
  icon: string;
  min_width: number;
  min_height: number;
  max_width?: number;
  max_height?: number;
  category_id?: string;
  limit?: number;
  enabled?: boolean;
};
```

## Tag methods

| Method | Returns | Notes |
|---|---|---|
| `listTags({ page?, size? })` | `Result<ListTagsResponse>` — paginated `data: Tag[]` | Tags live in a separate upstream service. |
| `createTag({ text, description? })` | `Result<CreateTagResponse>` | Used by `<TagSelector>` for on-the-fly creation. |
| `getDashboardTags({ dashboard_id })` | `Result<GetDashboardTagsResponse>` — `data: DashboardTag[]` | Returns mapping rows; resolve against `listTags()`. |
| `assignTagsToDashboard({ dashboard_id, tag_ids })` | `Result<void>` | Replace dashboard's tags. |
| `removeAllTags({ dashboard_id })` | `Result<void>` | |

```ts
type Tag = { id: string; text: string; description: string };
type DashboardTag = { id: string; dashboard_id: string; tag_id: string; occ_lock: number };
```

## Category methods

| Method | Returns |
|---|---|
| `listCategories({ page?, size? })` | `Result<ListCategoriesResponse>` |
| `createCategory({ name, description? })` | `Result<{ data: Category }>` |
| `deleteCategory({ id })` | `Result<void>` |

```ts
type Category = { id: string; occ_lock: number; name: string; description: string };
```

## Renderer exports

From `@mssfoobar/dash-web-sdk/renderer`:

### Components

| Export | Purpose |
|---|---|
| `DashboardCanvas` | Drag-and-drop layout — the main canvas. Consumes a `DashboardEditor` (`{editor}`) plus the `{factory}`. See `references/edit-create-pages.md` for usage. |
| `WidgetPicker` | Sidebar palette + config UI. Renders the auto-form unless the widget supplies a `configComponent`. Integrated into `<DashboardCanvas>` by default; pass `disablePicker` to render it yourself. |
| `ConfigPanel` | The schema-driven form renderer. You generally don't use this directly — `WidgetPicker` calls it. |
| `GridItem` | One cell wrapper. Not normally used directly. |
| `ErrorWidget` | Fallback when a widget fails to load. |

### Editor composable

| Export | Purpose |
|---|---|
| `createDashboardEditor(options)` | The keystone of the create/edit pages. Builds a `DashboardEditor` bundling widget state, instance models, dirty flag, picker/selection state, mode, and the per-widget + full-dashboard save flows. Hand the result to `<DashboardCanvas {editor} {factory} />`. Full walkthrough + the editor's property/method API in `references/edit-create-pages.md`. |

Its associated types — `DashboardEditor`, `DashboardEditorMode`, `DashboardWidget`, `DashboardWidgetInstance`, `DashboardEditorSaveParams`, `DashboardEditorSaveData`, `DashboardEditorValidation`, `CreateDashboardEditorOptions` — are all exported from `@mssfoobar/dash-web-sdk/renderer` alongside it.

### Widget authoring

| Export | Purpose |
|---|---|
| `defineWidget(spec)` | Build a widget class from a spec. See `references/widgets.md`. |
| `widgetFactory` | The shared singleton registry. Use `widgetFactory.register(MyWidget, { view })`. |
| `WidgetFactory` (class) | Build a private factory if you need one — uncommon. |

### Types

| Type | Use |
|---|---|
| `WidgetProps<typeof MyWidget>` | Type your View's `$props()` |
| `WidgetInstance<S, D, A>` | Manual instance type for advanced cases — usually `InstanceType<typeof MyWidget>` is enough |
| `WidgetClass<I>` | The static-side contract |
| `WidgetBaseInstance` | `{ hydrate, toConfig }` — minimum any widget exposes |
| `WidgetState<S>` | Inferred state type from a schema |
| `DefineWidgetSpec<S, D, A>` | The spec object's type |
| `DashboardWidget` | `{ id, widget_type_id, row, column, width, height, config?, occ_lock? }` — the shape `<DashboardCanvas>` works with |
| `DashboardProps`, `WidgetPickerProps`, `ConfigPanelProps`, `GridItemProps` | Props for each component |
| `ResolvedWidget` | `{ default: View, meta, WidgetClass, configSchema?, configComponent? }` — what `factory.get(type)` returns |

### Utilities

| Export | Purpose |
|---|---|
| `getIconComponent(name)` | Resolve a Lucide icon name (kebab-case) to a component. |
| `debounce(fn, ms)` | Simple debounce. |

## `factory` helpers

```ts
import { widgetFactory } from "@mssfoobar/dash-web-sdk/renderer";

// Register at app start
widgetFactory.register(MyWidget, { view: MyView, configComponent: MyConfig /* optional */ });

// In page mount — apply backend overrides
factory.applyOverrides(data.widget_types, data.categories);

// Lookup by type id
const resolved = await factory.get("counter");      // ResolvedWidget | undefined

// Lookup all (for the picker)
const all = await factory.getAll();                  // (ResolvedWidget & { path: string })[]

// Boolean check
factory.has("counter");

// All registered type ids
factory.types;                                       // string[]
```

`applyOverrides` is destructive — it replaces the previous overrides each call. That's fine for the typical pattern of "apply once on mount" because the server load fetches a fresh widget-types list.

## `syncWidgetTypes` (consumer utility)

Pushes any factory-registered widgets that aren't yet in the backend. The SDK doesn't ship this — copy it into your consumer at `src/lib/<your-module>/utils/sync-widget-types.ts`:

```ts
import type { DashboardClient, WidgetType } from "@mssfoobar/dash-web-sdk";
import type { WidgetFactory } from "@mssfoobar/dash-web-sdk/renderer";

export async function syncWidgetTypes(
  factory: WidgetFactory,
  client: DashboardClient,
  existingTypes: Partial<WidgetType>[],
): Promise<void> {
  const allWidgets = await factory.getAll();
  const existingIds = new Set(existingTypes.map((t) => t.id));

  for (const w of allWidgets) {
    if (existingIds.has(w.path)) continue;

    const result = await client.createWidgetType({
      id: w.path,
      name: w.meta.title,
      icon: w.meta.icon,
      min_width: w.meta.minWidth,
      min_height: w.meta.minHeight,
      max_width: w.meta.maxWidth,
      max_height: w.meta.maxHeight,
      limit: w.meta.limit,
      enabled: w.meta.enabled ?? true,
    });

    if (!result.ok) {
      // 409 Conflict = duplicate key — already exists, skip
      if (result.error?.status === 409) continue;
      throw result.error;
    }
  }
}
```

Call it on create/edit pages' `onMount` after `applyOverrides`. Errors are logged but don't block the page (the dashboard still loads even if sync fails).

## `checkHealth(token?)`

For module-level readiness checks (used in `+layout.svelte`):

```ts
const client = new DashboardClient({ dashURL: "http://..." });
const status = await client.checkHealth(accessToken);
// {
//   status: "ready" | "not_ready",
//   timestamp: "2025-01-15T...",
//   services: { dashboard: { status, response_time_ms } },
//   overall_response_time_ms: number,
// }
```

Hits `GET {dashURL}/health/live`. Tags are part of the dash service now, so there's no separate tag-service health check.

In practice you call this through a SvelteKit `+server.ts` API route (e.g. `/api/health`) that holds the URLs server-side, so the browser doesn't see internal hostnames.

## Things that will trip you up

- **Empty body responses** (204 No Content): `result.ok === true` but there's no `.data`. Methods that delete/assign return `Result<void>` for this reason.
- **`occ_lock` mismatch**: `updateDashboard` / `editWidget` return 409 if your `occ_lock` is stale. Refresh and retry.
- **`createWidgetType` 409**: not a bug — means the widget type already exists. `syncWidgetTypes` treats this as success.
- **Passing `client` between components**: fine, but each component should still get the SvelteKit `fetch` if they're called from a load function. Constructing a new `DashboardClient` per use is cheap.
- **`listTags` is paginated**: if your tenant has more than 10 tags, pass `{ size: 1000 }` or paginate. The auto-form's tag selector currently assumes a flat list.
