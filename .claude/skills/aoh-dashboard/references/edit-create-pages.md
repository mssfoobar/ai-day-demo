# Create & Edit pages — `createDashboardEditor` + `<DashboardCanvas>`

Both pages now follow the same shape: build a `DashboardEditor` with `createDashboardEditor({ client, dashboardId?, mode, validate?, on*Success, on*Error })`, hand it to `<DashboardCanvas>`, then call `editor.setMode(...)` / `editor.setPickerState(...)` / `editor.saveDashboard(...)` from your toolbar.

| | Create | Edit |
|---|---|---|
| `createDashboardEditor` mode | `"create"` (no `dashboardId`) | `"view"` initially, toggled to `"edit"` |
| `saveDashboard` dispatches to | `client.createDashboard` | `client.updateDashboard` (with internal `oldWidgetIds` diff) |
| Delete | n/a | `client.deleteDashboard(...)` with confirmation |
| Cancel | "Leave?" dialog → `goto(HOME_URL)` | `editor.resetWidgets(...)` + restore from local `backupDashboard` |
| Fullscreen / Export | n/a | Yes (view mode only) |

The editor owns the widget array, instance models, dirty flag, picker open state, selected widget id, drag state, and the `previousWidgetIds` snapshot used for the update-dashboard delete-diff. The consumer page reads/writes editor state directly (no `bind:widgets` / `bind:dirty` / `selectedWidget` plumbing).

## Both pages disable SSR

```ts
// +page.server.ts (and same for [id]/+page.server.ts)
export const ssr = false;
```

The `<DashboardCanvas>` component mounts a drag-and-drop layout engine to the DOM, which doesn't work server-side. Set `ssr = false` and rely on client-side hydration.

## Server load — what to fetch

Both pages need the same auxiliary data (tags, widget types, categories). The edit page additionally needs the dashboard and its tag mappings.

### Create page

```ts
// create/+page.server.ts
import type { PageServerLoad } from "./$types";
import { DashboardClient } from "@mssfoobar/dash-web-sdk";

export const ssr = false;

export const load: PageServerLoad = async ({ fetch }) => {
  const client = new DashboardClient({ fetch });

  const [tagsResult, widgetTypesResult, categoriesResult] = await Promise.all([
    client.listTags().catch(() => null),
    client.listWidgetTypes().catch(() => null),
    client.listCategories().catch(() => null),
  ]);

  return {
    tags: tagsResult?.ok ? tagsResult.data : [],
    categories: categoriesResult?.ok ? categoriesResult.data : [],
    widget_types: widgetTypesResult?.ok ? widgetTypesResult.data : [],
  };
};
```

### Edit page

```ts
// [id]/+page.server.ts
import { redirect } from "@sveltejs/kit";
import type { PageServerLoad } from "./$types";
import { DashboardClient } from "@mssfoobar/dash-web-sdk";

export const ssr = false;

export const load: PageServerLoad = async ({ fetch, params, url }) => {
  const client = new DashboardClient({ fetch });
  const dashboardId = params.id;
  const edit_mode = url.searchParams.get("edit") ?? "false";

  const [dashboardResult, tagsResult, widgetTypesResult, categoriesResult, dashboardTagsResult] =
    await Promise.all([
      client.getDashboardById({ dashboard_id: dashboardId }).catch(() => null),
      client.listTags().catch(() => null),
      client.listWidgetTypes().catch(() => null),
      client.listCategories().catch(() => null),
      client.getDashboardTags({ dashboard_id: dashboardId }).catch(() => null),
    ]);

  const dashboard = dashboardResult?.ok ? dashboardResult.data : undefined;
  if (!dashboard?.id) redirect(303, "/error");

  const allTags = (tagsResult?.ok ? tagsResult.data : null) ?? [];
  const tagMappings = (dashboardTagsResult?.ok ? dashboardTagsResult.data : null) ?? [];
  const assignedTagIds = new Set(tagMappings.map((m) => m.tag_id));
  dashboard.tags = allTags.filter((t) => assignedTagIds.has(t.id));

  return {
    dashboard,
    tags: allTags,
    widget_types: widgetTypesResult?.ok ? widgetTypesResult.data : [],
    categories: categoriesResult?.ok ? categoriesResult.data : [],
    edit_mode,
  };
};
```

`getDashboardTags` returns mapping rows (`{ dashboard_id, tag_id, ... }`); resolve them against `listTags()` to get the full `Tag[]` for the UI.

## Page setup — both pages

```svelte
<script lang="ts">
  import { onMount, untrack } from "svelte";
  import { DashboardClient, type Tag } from "@mssfoobar/dash-web-sdk";
  import { DashboardCanvas, createDashboardEditor } from "@mssfoobar/dash-web-sdk/renderer";
  import { factory } from "$lib/<your-module>/constants/widget.constants";
  import { syncWidgetTypes } from "$lib/<your-module>/utils/sync-widget-types";

  let { data } = $props();
  const client = new DashboardClient();

  // Snapshot the load `data` for one-time state seeding. `untrack` keeps
  // Svelte 5 from flagging state_referenced_locally on the initialisers.
  const initialData = untrack(() => data);

  // Apply backend overrides BEFORE the editor / <DashboardCanvas> mount
  if (initialData.widget_types?.length) {
    factory.applyOverrides(initialData.widget_types, initialData.categories);
  }
</script>
```

Then construct the editor — this is where create and edit diverge.

## Create page — the editor

```ts
const editor = createDashboardEditor({
  client,
  // No dashboardId → mode defaults to "create".
  mode: "create",
  // Pre-save validation. Runs before any network call; falsy → abort with
  // the error string surfaced via onDashboardSaveError.
  validate: async () => {
    const nameOk = await isDashboardNameAvailable(newDashboard.name ?? "");
    return { valid: nameOk };
  },
  onDashboardSaveSuccess: (data) => {
    toast.success("Dashboard has been created successfully.");
    goto(`${HOME_URL}/${data.id}`);
  },
  onDashboardSaveError: () => toast.error("There was an error creating the dashboard."),
});
```

The page state collapses to dashboard metadata + tag wiring:

```ts
let newDashboard: Partial<Dashboard> = $state({ name: "", description: "", favourite: false });
let tags: Tag[] = $state(initialData.tags ?? []);
let selectedTags: Tag[] = $state([]);
let isDashboardNameAlreadyTaken: boolean = $state(false);
```

Save handler — purely the trigger; validation + dispatch + side-effects live in the editor's options:

```ts
async function onSave() {
  await editor.saveDashboard({
    name: newDashboard.name ?? "",
    description: newDashboard.description ?? "",
    favourite: newDashboard.favourite,
    tags: selectedTags,
  });
}
```

## Edit page — the editor

```ts
let dashboard: DashboardWithTags = $state({ ...initialData.dashboard, tags: initialData.dashboard.tags ?? [] });
let backupDashboard: DashboardWithTags = $state({ ...initialData.dashboard, tags: initialData.dashboard.tags ?? [] });
let previousWidgets: DashboardWidget[] = $state([]);

const editor = createDashboardEditor({
  client,
  dashboardId: initialData.dashboard.id,
  mode: "view",
  validate: async () => {
    const v = await validateDashboard(dashboard.name, dashboard.description);
    return v.valid ? { valid: true } : { valid: false };
  },
  onWidgetSaveSuccess: () => toast.success("Widget configuration saved."),
  onWidgetSaveError: () => toast.error("Failed to save widget configuration."),
  onDashboardSaveSuccess: (data) => {
    dashboard.occ_lock = data.occ_lock ?? dashboard.occ_lock;
    Object.assign(backupDashboard, { ...dashboard, tags: [...selectedTags] });
    previousWidgets = JSON.parse(JSON.stringify(editor.widgets));
    editor.setMode("view");
    toast.success("Dashboard has been updated successfully.");
    goto(`/aoh/dash/${dashboard.id}`, { invalidateAll: true });
  },
  onDashboardSaveError: () => toast.error("There was an error saving the dashboard."),
});
```

`previousWidgets` is kept on the page for the **cancel** flow (which needs the full widget objects, not just ids). The editor tracks `previousWidgetIds` separately for the `oldWidgetIds` delete-diff at save time.

Save handler:
```ts
async function onSave() {
  await editor.saveDashboard({
    name: dashboard.name,
    description: dashboard.description,
    favourite: dashboard.favourite,
    occ_lock: dashboard.occ_lock ?? 0,
    tags: selectedTags,
  });
}
```

Cancel handler:
```ts
function onCancel() {
  editor.resetWidgets(JSON.parse(JSON.stringify(previousWidgets)));
  Object.assign(dashboard, { ...backupDashboard, tags: [...backupDashboard.tags] });
  selectedTags = [...backupDashboard.tags];
  editor.setMode("view");
  goto(`/aoh/dash/${dashboard.id}`, { replaceState: true });
}
```

`editor.resetWidgets(restored)` replaces the widget array, clears `instanceModels` (forces re-hydration from the restored config), and closes the picker — in one call.

## onMount — populate widgets, sync widget types

For both pages:

```ts
onMount(() => {
  // ... breadcrumb setup ...

  // Sync file-defined widget types up to the backend so admins can override
  // their meta later (safe to fire and forget — errors are logged).
  syncWidgetTypes(factory, client, data.widget_types ?? []);
});
```

For the **edit page** additionally:

```ts
onMount(() => {
  // ... breadcrumb setup ...

  if (data.edit_mode === "true") editor.setMode("edit");

  // Map API widgets to DashboardWidget shape and seed the editor
  editor.widgets = (dashboard.widgets ?? [])
    .filter((w) => w.widget_type_id !== "")
    .map((w) => ({
      id: w.id,
      widget_type_id: w.widget_type_id,
      row: w.row,
      column: w.column,
      width: w.width,
      height: w.height,
      config: w.config,
      occ_lock: (w as Record<string, unknown>).occ_lock as number | undefined,
    }));

  previousWidgets = JSON.parse(JSON.stringify(editor.widgets));
  Object.assign(backupDashboard, { ...dashboard, tags: [...dashboard.tags] });
  selectedTags = [...dashboard.tags];

  syncWidgetTypes(factory, client, data.widget_types ?? []);
});
```

`<DashboardCanvas>` takes its first snapshot for the delete-diff after it loads widgets, so the consumer doesn't need to seed `previousWidgetIds` explicitly.

## The toolbar

Both pages drive the editor from toolbar buttons:

```svelte
<!-- Add widget (opens picker in palette mode) -->
<Button onclick={() => editor.setPickerState(true)}>Add Widget</Button>

<!-- Save (edit page) -->
<Button onclick={onSave} disabled={!isDashboardDirty(dashboard)}>Save</Button>

<!-- Enter edit mode (edit page, view mode) -->
{#if editor.mode !== "edit"}
  <Button onclick={() => editor.setMode("edit")}>Edit</Button>
{/if}

<!-- Cancel (edit page, edit mode) -->
{#if editor.mode === "edit"}
  <Button onclick={onCancel}>Cancel</Button>
{/if}
```

`isDashboardDirty(dashboard)` checks `editor.dirty` (widget movement / add / delete tracked by `<DashboardCanvas>` → editor) OR diffs `dashboard.name` / `description` / `tags` against `backupDashboard` (those live outside the editor).

## Dashboard markup

That's the whole thing:

```svelte
<div class="w-full grow flex h-0 rounded relative mt-2">
  <div class="w-full h-full overflow-y-auto overflow-x-hidden bg-background">
    {#key canvasKey}
      <DashboardCanvas
        {editor}
        {factory}
        onlimitreached={(title, limit) =>
          toast.error(`"${title}" is limited to ${limit} instance${limit === 1 ? "" : "s"} per dashboard.`)}
      />
    {/key}
  </div>
</div>
```

The integrated WidgetPicker slides in as a right-side overlay when `editor.pickerOpen` is true and `editor.mode !== "view"`. If you need it elsewhere (bottom drawer, modal, separate column), pass `disablePicker={true}` to `<DashboardCanvas>` and render `<WidgetPicker>` yourself bound to the editor's state.

The `{#key canvasKey}` wrapper is used for cancel/import flows — bumping `canvasKey++` after a `editor.resetWidgets(...)` or `editor.widgets = [...imported]` re-mounts `<DashboardCanvas>` so it re-reads its internal layout state from scratch.

## Editor API reference

| Property | Type | Purpose |
|---|---|---|
| `widgets` | `DashboardWidget[]` (read/write) | The widget array. Mutate to add/remove externally; `<DashboardCanvas>` mutates as the user drags or clicks Add Widget. |
| `instanceModels` | `Record<string, WidgetInstance>` (read/write) | Per-widget model instances. Reset to `{}` to force re-hydration from `widgets[].config`. |
| `dirty` | `boolean` (read/write) | True after any local change. Cleared on successful save. |
| `mode` | `"create" \| "edit" \| "view"` (read-only) | Current mode. |
| `pickerOpen` | `boolean` (read-only) | Whether the integrated picker panel is showing. |
| `selectedWidgetId` | `string \| undefined` (read-only) | Id of the widget the picker is currently configuring. |
| `isDragging` | `boolean` (read-only) | True while a widget drag is in progress. |
| `dashboardId` | `string \| undefined` (read-only) | Mirrors the option passed to `createDashboardEditor`. |
| `client` | `DashboardClient \| undefined` (read-only) | Mirrors the option. `undefined` for a client-free editor (see *Running without a backend*). |

| Method | Effect |
|---|---|
| `setMode(value)` | Set edit / view / create mode. `"view"` also closes the picker. |
| `setPickerState(value)` | Open / close the integrated picker. Opening clears the current selection (palette mode). |
| `selectWidget(id?)` | Programmatically select a widget (also opens the picker in config mode if `id` is set). |
| `saveWidget(widget)` | Per-widget save. Calls `client.editWidget(...)` with a fresh `occ_lock` re-fetched first. Fires `onWidgetSaveSuccess` / `onWidgetSaveError`. |
| `saveDashboard(params)` | Full-dashboard save. Runs `validate` hook, dispatches to `client.createDashboard` (create mode) or `client.updateDashboard` (edit mode) with internal `oldWidgetIds` diff. Fires `onDashboardSaveSuccess` / `onDashboardSaveError`. |
| `snapshotWidgetIds()` | Re-snapshot the current `widgets[].id` list as the baseline for the next save's delete-diff. `<DashboardCanvas>` calls this once after widgets first load and after each successful save; consumer rarely needs it. |
| `resetWidgets(restored)` | Replace widgets, clear instanceModels, clear dirty, close picker. The cancel-flow helper. |

## Escape hatches

If the default save flow doesn't fit (custom endpoint, audit logging, batched saves), pass overrides to `createDashboardEditor`:

```ts
const editor = createDashboardEditor({
  client,
  dashboardId,
  saveWidget: async (widget) => {
    // Your own HTTP call. Return `Result<EditWidgetResponse>` shape.
    return await myCustomClient.patch(`/widgets/${widget.id}`, widget);
  },
  saveDashboard: async (params, widgets, oldWidgetIds) => {
    return await myCustomClient.put(`/dashboards/${dashboardId}`, {
      ...params,
      widgets,
      oldWidgetIds,
    });
  },
});
```

The internal `validate` hook still runs before your `saveDashboard` override. The `on*Success` / `on*Error` callbacks fire on the override's Result, same as the default flow.

## Running without a backend (client-free editor)

`client` is **optional** on `createDashboardEditor`. Omit it to run the editor fully client-side — render and edit the grid with no dash backend at all — and read the layout out through `<DashboardCanvas bind:value>` (a `DashboardWidget[]` snapshot that re-emits whenever the layout or any widget's config changes). Persist it yourself (localStorage, your own API, a file). This is the pattern a no-backend demo or a self-hosted-persistence page uses.

```svelte
<script lang="ts">
  import { DashboardCanvas, createDashboardEditor, type DashboardWidget } from "@mssfoobar/dash-web-sdk/renderer";
  import { factory } from "$lib/<your-module>/constants/widget.constants";

  // No `client` → nothing hits the network.
  const editor = createDashboardEditor({ mode: "view" });

  // Live, config-synced snapshot of the grid.
  let grid = $state<DashboardWidget[]>([]);

  $effect(() => {
    localStorage.setItem("my-dashboard", JSON.stringify(grid));
  });
</script>

<DashboardCanvas {editor} {factory} bind:value={grid} />
```

Caveats when there's no `client`:

- **The built-in save flow can't run.** `editor.saveDashboard(...)` / `editor.saveWidget(...)` need either a `client` or the `saveDashboard` / `saveWidget` overrides from *Escape hatches* above — without both, `saveDashboard` returns a configuration-error `Result` rather than dereferencing an undefined client. Drive persistence off `bind:value` instead, or supply the overrides.
- **No `applyOverrides` source.** With no server load you have no backend `widget_types` / `categories` to apply — the picker shows whatever the `factory` was registered with, using each widget's file-defined `meta`.
- **Seed the initial grid** by assigning `editor.widgets = [...]` (and bumping the `{#key canvasKey}` wrapper, per the cancel/import note above) before first paint.

## Import / export

The editor doesn't know about export formats — that stays in the page. Build a `DashboardImportSchema` JSON from `editor.widgets` + dashboard metadata:

```ts
function onExport() {
  const exportData = {
    version: 1,
    dashboard: { name: dashboard.name, description: dashboard.description, favourite: dashboard.favourite },
    tags: selectedTags.map((t) => t.text),
    widgets: editor.widgets.map((w) => ({
      widget_type_id: w.widget_type_id,
      row: w.row, column: w.column,
      width: w.width, height: w.height,
      config: w.config,
    })),
  };
  // ... blob / download ...
}
```

For import, mutate the editor:

```ts
function applyImport(schema) {
  dashboard.name = schema.dashboard.name;
  dashboard.description = schema.dashboard.description;
  selectedTags = /* resolve from tags */;
  editor.widgets = schema.widgets.map((w) => ({ id: crypto.randomUUID(), ...w }));
  editor.instanceModels = {};
  canvasKey++;
  editor.dirty = true;
}
```

Setting `editor.widgets = [...]` + `editor.instanceModels = {}` + bumping `canvasKey` is the canonical "fresh re-hydration" pattern. `<DashboardCanvas>`'s `{#key canvasKey}` wrapper re-mounts and rebuilds models from the new `widget.config` blobs.

## Common pitfalls

- **Forgetting `untrack` on load-data initialisers.** Svelte 5 warns about `state_referenced_locally`. Snapshot once with `const initialData = untrack(() => data);` and read from `initialData.x` for state seeding.
- **Mutating editor state inside `$derived`.** Editor's getter properties are reactive — read them in templates and `$derived` freely, but don't write through `editor.widgets[i].x = ...`; mutate by `editor.widgets = editor.widgets.map(...)` so the reactivity proxy notices.
- **Skipping `canvasKey++` after `editor.widgets = [...]`.** `<DashboardCanvas>` keeps an in-memory layout cache; the key wrapper forces a clean re-mount so the visual state can't drift from the data array.
- **Showing `onlimitreached` toast text that doesn't match the resolved title.** `<DashboardCanvas>` passes the title from `mod.meta.title` (after `factory.applyOverrides`), so just relay it.
- **Calling `editor.saveDashboard()` without setting a `validate` hook on the create page.** The default flow will create even with invalid input. Validation is the consumer's job; the hook is just where to run it.
- **Importing `WidgetPicker` separately if you want a different layout.** It's still exported. Set `<DashboardCanvas disablePicker={true}>` and render `<WidgetPicker>` wherever you like, bound to the editor.
