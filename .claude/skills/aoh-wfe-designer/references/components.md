# `@mssfoobar/wfe-web-sdk` — component, props & imperative-API catalog

Every user-facing export, what it does, and how the host drives it. The
authoritative prop reference is each component's own TypeScript types and
docstrings shipped in the installed `@mssfoobar/wfe-web-sdk` package; this is the
integration-level summary.

## Subpath exports

| Import | Surface |
|---|---|
| `@mssfoobar/wfe-web-sdk` (`.`) | The barrel — `WorkflowProvider`, `WorkflowDesigner`, `MonitorCanvas`, `MonitorSteps`, and the monitor helper fns below. |
| `@mssfoobar/wfe-web-sdk/provider` | `WorkflowProvider` only. |
| `@mssfoobar/wfe-web-sdk/designer` | The designer surface (`WorkflowDesigner`). |
| `@mssfoobar/wfe-web-sdk/monitor` | The monitor primitives (`MonitorCanvas`, `MonitorSteps`) + helpers. |
| `@mssfoobar/wfe-web-sdk/canvas` | The lower-level canvas shell (`WorkflowCanvas`) — most hosts use `WorkflowDesigner` instead. |
| `@mssfoobar/wfe-web-sdk/inspector` | The activity inspector panel (composed inside `WorkflowDesigner`). |
| `@mssfoobar/wfe-web-sdk/activity-sdk` | `Activity` type + helpers for authoring **custom activity property UIs** (see below). |
| `@mssfoobar/wfe-web-sdk/serialize` | DSL (de)serialization helpers (`workflow_json` ↔ designer graph). |
| `@mssfoobar/wfe-web-sdk/nav` | Nav metadata for sidebars/routers. |
| `@mssfoobar/wfe-web-sdk/styles/app.css` | The SDK stylesheet (Tailwind `@source` directives). |

## `<WorkflowProvider>` — the root provider

Wrap every WFE surface (designer and monitor) in one provider. It supplies the
service client, the Form-module clients, the theme store, and custom activity UIs
through Svelte context.

| Prop | Type | Required | Purpose |
|---|---|---|---|
| `client` | `WfeClient` | one of `client` **or** (`wfe_url` + `getToken`) | Pre-built service client (from `createWfeClient`). With the BFF pattern this is the only one you pass. |
| `wfe_url` | `string` | — | Alternative to `client`: the manager base URL the SDK calls **from the browser** (with `getToken`). The BFF pattern does **not** use this. |
| `getToken` | `() => string \| Promise<string>` | with `wfe_url` | Browser-side token hook. A no-op (`() => ""`) when the BFF attaches the token server-side. |
| `form_client` | `FormSubmitterClient` (`@mssfoobar/form-client/submitter`) | for Form steps | Renders/resumes human-task Form steps. |
| `amm_client` | `AmmClient` (`@mssfoobar/form-client/amm`) | for Form steps w/ attachments | Attachment client used by the Form renderer. |
| `dark_mode_store` | `Writable<boolean>` | recommended | Host-owned dark-mode store; the canvas/inspector/CodeMirror repaint on change. Use `createThemeMirrorStore()`. |
| `activities` | `Record<string, { component }>` | optional | Custom property-panel UIs keyed by `activity_type` (see below). |

Components fall back to standalone defaults when no provider is mounted (Storybook
use still works), but real apps always mount one provider above both pages.

## `<WorkflowDesigner>` — the designer

The canvas + inspector. **Ships no toolbar** — the host builds its own chrome and
drives the designer through its imperative API (grab it with `bind:this`).

Props:

| Prop | Type | Purpose |
|---|---|---|
| `bind:this` | `WorkflowDesignerApi` | Handle for the imperative API below. |
| `catalog` | `ServiceActivity[]` | Activity types the palette/inspector offer — `client.serviceActivity.list()`. |
| `eventCatalog` | `ServiceEvent[]` | Event types — `client.serviceEvent.list()`. |
| `templates` | `WorkflowTemplate[]` | Known templates (e.g. for CallActivity linking). |

Imperative API (`WorkflowDesignerApi`):

| Method | Effect |
|---|---|
| `loadTemplate(t: WorkflowTemplate)` | Render a template's graph onto the canvas. Call after the catalogues are set (await a `tick()` first). |
| `save({ name })` → `Promise<WorkflowTemplate>` | Validate, serialize, persist as a draft (`editable: true`). Rejects with a `.issues`-carrying error when invalid. |
| `publish({ name })` → `Promise<WorkflowTemplate>` | As `save`, but publishes (`editable: false`, immutable). |
| `validate()` → `{ message }[]` | Validate the current graph on demand; returns issues (empty = valid). |
| `exportJson({ name })` | Download the DSL as a file. |
| `importDsl(json)` | Load a DSL object onto the canvas. |
| `clear()` | Empty the canvas. |

`save`/`publish` reject with a `WorkflowValidationError` carrying `.issues` when
the graph is invalid — surface those in an issues bar, not a toast.

## Monitor primitives + helpers

The SDK ships read-only **view primitives**; the host builds the chrome (picker,
Start run, view toggle, breadcrumb) around them.

| Export | Kind | Purpose |
|---|---|---|
| `MonitorCanvas` | component | Status-colour-coded flow chart. Props: `template`, `catalog`, `events`, `onDrill(stepName, designerJson?)`. |
| `MonitorSteps` | component | Step-by-step checklist. Props: `template`, `events`, `onDrill`. |
| `workflowRunStatus(events)` | fn | Derive a `WorkflowRunStatus` (`running`/`completed`/`failed`/`canceled`/`terminated`) from history events. |
| `isTerminal(status)` | fn | True once the run can't change (disable Terminate). |
| `callActivityChild(workflow_json, stepName)` | fn | The inlined child DSL for a CallActivity step. |
| `callActivityRunIds(events)` | fn | `Map<stepName, childRunId>` from `CallActivityTaskStarted` events. |
| `callActivitySubgraphs(designer_json)` | fn | `Map<stepName, childDesignerJson>` for the saved child layout. |

History comes from `pollWorkflowHistory(client, workflowId, { intervalMs, signal })`
(an async generator in `@mssfoobar/wfe-client`) — re-point it whenever the watched
run changes; abort on unmount.

## The `wfe-client` surface (`@mssfoobar/wfe-client`)

`createWfeClient({ baseUrl, getToken })` returns a client with:

- `client.workflowTemplate.list({ size }) / get(id) / save(...) / publish(...) / delete(id)`
- `client.serviceActivity.list({ size })` · `client.serviceEvent.list({ size })` (read-only catalogues)
- `client.workflow.start({ template_id }) / terminate(id, { reason })`
- `client.validate.expression(...) / condition(...)`
- plus `pollWorkflowHistory(...)` and the `ServiceActivity` / `ServiceEvent` /
  `WorkflowTemplate` / `HistoryEvent` / `WorkflowRunStatus` types.

`baseUrl` is the prefix the client appends `/v1/<resource>` to. Point it at the
same-origin BFF (`/<mount>/api`) so the token stays server-side; `getToken` is then
a no-op. See `bff-and-auth.md`.

## Custom activity property UIs (`activities` prop)

Register a Svelte component per `activity_type` to replace the inspector's default
property form. The component takes an `activity: Activity` prop (from
`@mssfoobar/wfe-web-sdk/activity-sdk`) and reads/writes params through the **Activity
contract** — no local `$state` mirror, so values never go stale:

```svelte
<script lang="ts">
  import { Label } from "@mssfoobar/ui/label";
  import { Input } from "@mssfoobar/ui/input";
  import type { Activity } from "@mssfoobar/wfe-web-sdk/activity-sdk";
  let { activity }: { activity: Activity } = $props();
</script>

<Label>Title</Label>
<Input
  value={String(activity.getParameter("title") ?? "")}
  oninput={(e) => activity.setParameter("title", e.currentTarget.value)}
/>
```

Register it: `activities={{ InAppNotification: { component: InAppNotification } }}`
on `<WorkflowProvider>`. The key must equal the `activity_type` (and the manager's
`service_activity` catalogue must advertise that type for the node to be selectable).
