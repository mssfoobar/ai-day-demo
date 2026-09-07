---
name: aoh-wfe-designer
description: >
  Integrate the AOH WFE designer + monitor SDK (`@mssfoobar/wfe-web-sdk`) into a
  SvelteKit consumer app — mount the BPMN-style workflow designer and the
  read-only run monitor, drive them through their imperative API, wire the
  `@mssfoobar/wfe-client` through a server-side BFF, and supply theme + Form
  clients via `<WorkflowProvider>`. Use this skill whenever a user wants to add
  a workflow designer or monitor to their AOH app, embed `@mssfoobar/wfe-web-sdk`,
  build a host toolbar that saves/publishes/validates workflows, render a live
  workflow-execution view, or wire the WFE module (workflow-manager) into a
  SvelteKit frontend. Trigger phrases: "integrate the workflow designer",
  "embed wfe-web-sdk", "add a workflow designer", "add a workflow monitor",
  "WorkflowProvider", "WorkflowDesigner", "mount the WFE designer", "consume the
  WFE SDK", "show a running workflow".
license: Proprietary
metadata:
  owner: AOH Platform Team
  status: v1
allowed-tools: Read Write Edit Bash(pnpm:*) Glob Grep
---

# AOH WFE Designer + Monitor Integration

Wire the AOH **WFE module's** embeddable UI into a SvelteKit consumer app. The
UI ships as the npm package **`@mssfoobar/wfe-web-sdk`** (GitHub Packages, `@mssfoobar`
scope): a BPMN-style **designer** (canvas + inspector) and a read-only run
**monitor**. The backend is the Go **workflow-manager** (REST `/v1`) plus the
**workflow-engine** (Temporal). This skill is the integration playbook — the exact
components, props, route setup, and server-side BFF wiring to get a working,
themed designer and live monitor.

Integration is bespoke to your app's layout, auth, and CSS pipeline — so this
skill is a playbook, not a generator. The components, props, provider, routes,
and BFF wiring below (plus the copy-pasteable snippets in `references/`) stand
on their own; adapt them into your app. Example routes use your own generic
paths — a designer at `src/routes/<your-route>/`, its BFF under
`src/routes/<your-route>/api/`, and helpers under `src/lib/wfe/`.

> **Package naming — read this first.** The published SDK is **`@mssfoobar/wfe-web-sdk`**.
> Its wire layer is **`@mssfoobar/wfe-client`** (the HTTP client + types) and
> **`@mssfoobar/wfe-types`**. The form packages — **`@mssfoobar/form-client`** (the
> injected `form_client` / `amm_client`) + **`@mssfoobar/form-web-sdk`** (the renderer)
> — are **peers**, needed only for human-task Form steps. There is no
> `@mssfoobar/wfe-web` / `wfe-designer`
> package and no standalone designer *service* — the designer is this SDK, mounted
> in a host app.

## What you're wiring (mental model)

```
<WorkflowProvider client form_client amm_client dark_mode_store activities>
  <WorkflowDesigner bind:this catalog eventCatalog templates/>   ← designer page
  — or —
  <MonitorCanvas .../>  <MonitorSteps .../>                      ← monitor page
</WorkflowProvider>

client (createWfeClient, baseUrl=/<mount>/api)
   │  same-origin — no token in the browser
   ▼
src/routes/<mount>/api/[...path]/+server.ts   (BFF: attaches the user's token)
   ▼
workflow-manager  ${WFE_URL}/v1/*
```

Two halves to every page:

1. **Client (`.svelte`)** — `<WorkflowProvider>` + the designer or monitor
   components, plus the **host-built toolbar** (the SDK ships none).
2. **Server (`+server.ts` BFF)** — a catch-all proxy that forwards `/v1/*` to the
   workflow-manager with a **server-side** bearer token. The browser never sees
   the token or the manager URL.

## Prerequisites & install

The SDK is a Svelte 5 package. Peer requirements:

- **SvelteKit + Svelte 5** (runes). Components are `.svelte` consumed via the
  `svelte` export condition.
- **Tailwind v4** — the SDK's styles register via `@source` (Step 10).
- Peer packages: `@mssfoobar/wfe-client`, `@mssfoobar/wfe-types`,
  `@mssfoobar/form-client` + `@mssfoobar/form-web-sdk` (Form steps), `@mssfoobar/ui`,
  `@mssfoobar/logger`, `@lucide/svelte`, `svelte ^5`.

Authenticate the scope to GitHub Packages (project-level `.npmrc`):

```ini
@mssfoobar:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${NODE_AUTH_TOKEN}
```

…with `NODE_AUTH_TOKEN` exported (a PAT / `gh auth token` with `read:packages`).
Then install:

```bash
pnpm add @mssfoobar/wfe-web-sdk @mssfoobar/wfe-client @mssfoobar/wfe-types \
         @mssfoobar/form-client @mssfoobar/form-web-sdk @mssfoobar/ui @mssfoobar/logger @lucide/svelte
```

> **No extra deps for the canvas.** The SDK **bundles** its commercial JointJS+
> (Rappid) canvas into `dist/` — consumers install nothing for it, and there is
> no token needed at install. The `@codemirror/*` + `@lezer/*` packages the expr
> editor uses are regular `dependencies` of the SDK, pulled in transitively.

> **SSR caveat (no action needed, just know it).** Rappid runs jQuery/Backbone at
> import and would crash SSR — but the SDK's canvas is an **SSR-safe shell** that
> dynamically imports the real canvas on mount, and the CodeMirror editors create
> their `EditorView` in `onMount` only. So the designer/monitor routes **server-render
> fine with no `export const ssr = false`** needed. Don't statically re-export the
> SDK's canvas internals from a server-reachable module.

## Integration steps

### Step 1. Install the SDK + the wfe-client

Per "Prerequisites & install" above. `@mssfoobar/wfe-client` is what talks to the
manager; `@mssfoobar/wfe-web-sdk` provides the UI.

### Step 2. Add the server-side BFF proxy

A catch-all route forwards the SDK's REST calls to the manager with a server-held
token. Create `src/routes/<mount>/api/[...path]/+server.ts` — it maps
`/<mount>/api/v1/<resource>` → `${WFE_URL}/v1/<resource>`, attaches the user's
token, and **fails closed** (401) when `IAM_URL` is set but there's no session.
Full walkthrough: `references/bff-and-auth.md`.

### Step 3. Build the client against the BFF

```ts
import { createWfeClient } from "@mssfoobar/wfe-client";
// baseUrl is the prefix the client appends `/v1/<resource>` to. Point it at the
// same-origin BFF so the token stays server-side — getToken is a no-op.
const client = createWfeClient({ baseUrl: "/<mount>/api", getToken: () => "" });
```

(The SDK *can* call the manager directly with `<WorkflowProvider wfe_url getToken>`,
but that puts the token in the browser — use the BFF. See `references/bff-and-auth.md`.)

### Step 4. Mount `<WorkflowProvider>`

Wrap both the designer and the monitor in one provider (per page is fine; lift to a
shared layout once both routes exist).

| Prop | Type | Required | Purpose |
|---|---|---|---|
| `client` | `WfeClient` | **yes** (BFF mode) | The `createWfeClient` instance from Step 3. |
| `form_client` | `FormSubmitterClient` (`@mssfoobar/form-client/submitter`) | Form steps | Renders/resumes human-task Form steps. |
| `amm_client` | `AmmClient` (`@mssfoobar/form-client/amm`) | Form steps w/ attachments | Attachment client for the Form renderer. |
| `dark_mode_store` | `Writable<boolean>` | recommended | Host-owned theme store; canvas/inspector/editors repaint on change (Step 7). |
| `activities` | `Record<string, { component }>` | optional | Custom property-panel UIs per `activity_type` (`references/components.md`). |

### Step 5. Build the designer page

`<WorkflowDesigner>` is the canvas + inspector and **ships no toolbar** — the host
builds chrome and drives it via its imperative API (grabbed with `bind:this`):
`loadTemplate` / `save` / `publish` / `validate` / `exportJson` / `importDsl` /
`clear`. Fetch the catalogues + templates from the manager on mount and pass them
in:

```svelte
let designer = $state<WorkflowDesignerApi>();
// onMount: catalog = (await client.serviceActivity.list({size:200})).data;
//          eventCatalog = (await client.serviceEvent.list({size:200})).data;
//          templates = (await client.workflowTemplate.list({size:100})).data;
//          await tick(); designer.loadTemplate(templates[0]);
<WorkflowDesigner bind:this={designer} {catalog} {eventCatalog} {templates} />
```

`save`/`publish` reject with an error carrying `.issues` when the graph is invalid
— render those in an issues bar, not a toast. **`await tick()` before
`loadTemplate`** so the catalogues reach the designer before it resolves the graph.
Build the toolbar (Save / Publish / Validate / Export / Import / Clear buttons)
around the imperative API and wire the catalogue fetches in `onMount` as sketched
above. Full API: `references/components.md`.

### Step 6. Build the monitor page

The SDK ships read-only **view primitives**; the host builds the chrome (workflow
picker, Start run, flow/steps toggle, breadcrumb):

- `<MonitorCanvas template catalog events onDrill/>` — status-coloured flow chart.
- `<MonitorSteps template events onDrill/>` — step checklist.
- `pollWorkflowHistory(client, workflowId, {intervalMs, signal})` — async generator
  of history events; re-point it when the watched run changes, abort on unmount.
- `workflowRunStatus(events)` / `isTerminal(status)` — run status + terminal check.
- CallActivity drill-down: `callActivityChild` / `callActivityRunIds` /
  `callActivitySubgraphs` resolve a step's child DSL, run id, and saved layout.

Full API: `references/components.md`.

### Step 7. Wire the theme-mirror store

The SDK has no access to your shell's `class="dark"` toggle, so mirror it into a
`Writable<boolean>` and pass it as `dark_mode_store`. Drop this helper in once
(e.g. `src/lib/wfe/theme-mirror.ts`) — it wires a `MutationObserver` on `<html>`
on mount and tears it down on destroy:

```ts
import { onMount } from "svelte";
import { writable, type Writable } from "svelte/store";

/** Mirror the app's `<html class="dark">` toggle into a store the WFE SDK
 *  subscribes to via `<WorkflowProvider dark_mode_store={…}>`. Call once
 *  during component init. */
export function createThemeMirrorStore(): Writable<boolean> {
	const dark = writable(false);
	onMount(() => {
		const el = document.documentElement;
		const sync = () => dark.set(el.classList.contains("dark"));
		sync();
		const observer = new MutationObserver(sync);
		observer.observe(el, { attributes: true, attributeFilter: ["class"] });
		return () => observer.disconnect();
	});
	return dark;
}
```

```ts
const darkMode = createThemeMirrorStore();   // -> <WorkflowProvider dark_mode_store={darkMode}>
```

### Step 8. Form steps (delegated to the Form module)

WFE stores no form schema — human-task **Form steps are delegated to the AOH Form
module (SurveyJS)**. Inject `form_client` (`FormSubmitterClient`) and `amm_client`
(`AmmClient`) into `<WorkflowProvider>` and the SDK renders the Form-step picker (in
the designer) and renders + resumes the form (in the monitor) itself — no host
wiring beyond providing the clients. They need a reachable Form-module backend
(through your gateway BFF). Omit both props if your workflows use no Form steps.
(See the `aoh-knowledge` WFE entry for the Form delegation. To construct the
`form_client` / `amm_client` or embed forms standalone — outside a workflow — see
the **`aoh-form-integration`** skill.)

### Step 9. Navigation (optional)

The SDK exports nav metadata: `import { ... } from "@mssfoobar/wfe-web-sdk/nav"`. Feed
it to your sidebar/router; override the default `url`s if you mounted WFE elsewhere
than `/aoh/wfe`.

### Step 10. Styles

The SDK's Tailwind `@source` directives must be in your CSS pipeline. Chain the SDK
stylesheet through your single CSS entry (so `@source` directives aggregate in one
Tailwind pass) rather than a separate JS import:

```css
/* src/app.css */
@import "@mssfoobar/wfe-web-sdk/styles/app.css";
```

Keep this in your **single** Tailwind CSS entry (a CSS-level `@import`, not a JS
import) so every package's `@source` directives aggregate into one Tailwind pass;
splitting them across multiple entries drops the SDK's classes from the content
scan.

## Verify it works

1. `pnpm dev` → visit `/<mount>`. The designer canvas renders (empty is fine).
2. **Catalogues load**: the activity palette/inspector offer types → the BFF +
   `WFE_URL` are reaching the manager, and its `service_activity` catalogue is
   seeded (a worker registered its activities — see `aoh-wfe-worker`).
3. **Save/publish**: draw a graph → Save → it appears in the template picker;
   reload → it persists. Invalid graphs surface in the issues bar.
4. **Monitor**: `/<mount>/monitor` → pick a template → Start run → the flow chart
   colours as the run progresses. CallActivity nodes drill in.
5. **Theme**: toggle the shell's dark mode → the canvas/inspector repaint (Step 7).
6. **No SSR crash** on load (the SDK guards browser globals — Step "SSR caveat").

## Common failure modes

| Symptom | Cause | Fix |
|---|---|---|
| `npm error 404 @mssfoobar/wfe-designer` / `@mssfoobar/wfe-web` | Wrong package name | It's `@mssfoobar/wfe-web-sdk` (+ `@mssfoobar/wfe-client`, `@mssfoobar/wfe-types`). |
| Designer canvas empty; palette has no activities | The manager's `service_activity` catalogue is empty, or the BFF isn't reaching the manager | Seed the catalogue (run a worker + its seed — see `aoh-wfe-worker`); confirm `WFE_URL` + the BFF (Step 2). |
| Designer loads a template but the graph is blank / mis-resolved | `loadTemplate` called before `catalog`/`eventCatalog` reached the component | `await tick()` after setting the catalogues, before `loadTemplate` (Step 5). |
| 401 from the manager on every call | BFF didn't forward a token (auth on, no session) — fail-closed working as designed, or `locals.authResult` not populated | Sign in; confirm `hooks.server.ts` (auth-sdk) sets `locals.authResult`. With auth off, ensure `IAM_URL` is unset so the dev bearer is used. |
| `503 WFE_URL is not configured` | The BFF env var is unset | Set `WFE_URL` (manager base, **no** `/v1`) on the server. |
| Save silently does nothing / no toast | Validation failed; `.issues` not surfaced | Read the rejected error's `.issues` into an issues bar (Step 5). |
| Map/canvas theme doesn't follow app dark mode | No `dark_mode_store` wired | Pass `createThemeMirrorStore()` to `<WorkflowProvider>` (Step 7). |
| Form-step picker is empty / forms won't render | `form_client` (+ `amm_client`) not injected, or no Form-module backend | Provide both on `<WorkflowProvider>` and ensure the Form module is reachable (Step 8). |
| Styles look unstyled / Tailwind classes missing | SDK `@source` directives not in the CSS pass | `@import "@mssfoobar/wfe-web-sdk/styles/app.css"` from your single CSS entry (Step 10). |
| Calls go to `${WFE_URL}/v1/v1/...` (double `/v1`) | `baseUrl` already includes `/v1`, or `WFE_URL` does | `baseUrl` = `/<mount>/api` (client adds `/v1`); `WFE_URL` = manager base, no `/v1`. |

## What this skill won't do for you

- Scaffold a SvelteKit app, set up OIDC auth / `hooks.server.ts` / SDS — that's
  **`aoh-web-init`**. This skill assumes a working app with `locals.authResult`.
- Bring up the workflow-manager / engine / Temporal / Postgres — that's
  **`aoh-compose`** (the `wfe` service).
- Build or register activities (so the designer has anything to offer) — that's
  **`aoh-wfe-worker`** (the activity worker + the `service_activity` seed).
- Explain the WFE domain model, the DSL, or the REST contract — that's the
  **`aoh-knowledge`** WFE entry.

## References

- `references/components.md` — the full SDK export + props + imperative-API catalog
  (`WorkflowProvider`, `WorkflowDesigner` API, monitor primitives + helpers, the
  `wfe-client` surface, custom activity UIs).
- `references/bff-and-auth.md` — the same-origin BFF proxy, the fail-closed
  server-side token flow, the `WFE_URL`/`IAM_URL` gate, and the `/v1` path mapping.
- Domain / API — the WFE domain model, DSL, and REST contract: the
  **`aoh-knowledge`** skill's WFE entry.
