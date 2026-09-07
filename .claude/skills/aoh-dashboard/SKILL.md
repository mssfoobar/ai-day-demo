---
name: aoh-dashboard
description: "Build dashboard pages (list/create/edit routes) and author widgets in an AOH SvelteKit app on @mssfoobar/dash-web-sdk, talking to the AOH dash backend — incl. the widgets a seeded dashboard renders (the seeding *mechanism* is owned by aoh-knowledge / aoh-compose). Use whenever building, customizing, or extending dashboard pages on dash-web-sdk — dashboard CRUD routes, DashboardCanvas and WidgetPicker, custom widgets via defineWidget, widget options (config schema, derived, actions), server load functions, providing the read/aggregation (or RTUS-fed live) data backend widgets need (build one if none exists), OR seeding/generating a dashboard ('make me a dashboard', 'set up a monitoring page', 'seed a dashboard'). Also trigger on adjacent concepts — 'widget config UI', 'DashboardClient', '42-column layout', or creating a dashboard programmatically. For scaffolding, dash backend bring-up, the seeding mechanism, or platform questions, defer to the sibling scaffolding / infra / knowledge skills."
allowed-tools: Read Write Edit Grep Glob Bash
---

# AOH Dashboards — Pages & Widgets

How to build dashboard management UI on top of `@mssfoobar/dash-web-sdk`, and how to author the widgets that live on those dashboards. Lives at the application-feature layer of an AOH project — assumes the scaffold and infra layers are already in place.

## Prerequisites — owned by other skills

This skill does not stand a project up from zero. The layers below it are owned by sibling skills in `.claude/skills/`:

| Need | Owning skill | What it gives you |
|---|---|---|
| A SvelteKit consumer app (OIDC + modlet + `@mssfoobar/ui` + Tailwind v4) inside `apps/<app-name>/` | **`aoh-web-init`** | The base frontend this skill writes pages into |
| A Go microservice (chi + sqlx + AOH middleware) if you're building a companion backend | **`aoh-go-init`** | Backing service for project-specific endpoints |
| The dash backend (`dash-service` + `dash-db`) running locally behind Traefik, plus Keycloak / Postgres | **`aoh-compose`** | `compose/dash/` brought up with `docker compose up`. Default route: `http://dash.${DEV_DOMAIN}` |
| Code conventions (Go, SvelteKit, DB, HTTP) | **`aoh-conventions`** | Don't restate them here — read its references |
| What every AOH service does, IAMS auth flow, AAS roles, dash REST endpoint specs | **`aoh-knowledge`** | `references/services/dash.md` + `dash-openapi.json` |

If the developer asks "how do I get started with dash from zero?", the chain is **aoh-compose → aoh-web-init → aoh-dashboards**. This skill picks up where the first two leave off.

## What this skill assumes

- The repo is a Turborepo monorepo with `apps/` and `packages/` (the layout `aoh-web-init` produces).
- A SvelteKit app exists at `apps/<app-name>/` with `@mssfoobar/dash-web-sdk` installed (`pnpm add @mssfoobar/dash-web-sdk` from inside that app).
- `DashboardClient` targets a single backend (`dashURL`) for dashboard, widget, category, **and tag** requests — tags are owned by the dash service itself, not a separate tag service, so there is no `tagURL`. The base URL is prepended to the request path, so whatever you set determines where requests land. It defaults to `""` (origin-relative — `/dashboard/...`, `/v1/tag/...`). Set it to wherever the dash service is reachable in your deployment.
- IAMS (Keycloak) issues the JWT. The SDK attaches it via the `getAccessToken` callback you pass to `DashboardClient` (`Authorization: Bearer <token>` on every request); omit it if cookies / transport auth already cover the requests. See `aoh-knowledge`'s `iams.md` for the token flow.

## When to read what

Don't load every reference up front — pull only the files relevant to the task.

| The developer is building / asking about... | Read |
|---|---|
| A new widget (any complexity) | `references/widgets.md` |
| A list page (CRUD over dashboards) | `references/list-page.md` |
| A create or edit page (`<DashboardCanvas>` + `<WidgetPicker>`) | `references/edit-create-pages.md` |
| Available API methods on `DashboardClient` | `references/sdk-client.md` |
| Widget sizing / the 42-column dashboard layout / widget meta | `references/widgets.md` (Sizing section) |
| Visual style, widget archetypes, badges, charts (LayerChart), renderer override hooks | `references/widget-style.md` |
| Seeding a dashboard — the widget `config` contract (the seed *mechanism* + auth live in `aoh-knowledge`'s `dash.md`) | `references/widgets.md` (Config field options) |

If you've never used this SDK before, skim **this file end to end first** — it gives the map. Then dive into one reference.

## The architecture in one paragraph

The SDK ships a `DashboardClient` (pure HTTP — no UI state), the `createDashboardEditor()` composable that bundles widget state + picker state + mode + save flows, a lean `<DashboardCanvas>` component that renders the editor's state (a 42-column dashboard layout by default, with the `<WidgetPicker>` integrated as a right-side overlay), and the `defineWidget()` API for authoring widgets. The consumer app builds an editor with `createDashboardEditor({ client, dashboardId?, mode, validate?, on*Success, on*Error })` and hands it to `<DashboardCanvas {editor} {factory} />` — no `bind:widgets` / `bind:dirty` / `selectedWidget` / `menuState` plumbing. Widgets are registered once at app start in a central `register.ts`, and the registered `widgetFactory` is passed to `<DashboardCanvas>`. There is no dynamic file discovery — every widget is explicitly imported and bound to its View component at the registration site.

## The data behind the widgets — always provide a backend

A dashboard is two halves: the DASH surface (config + layout, owned here) and the **data** the widgets render. **DASH stores config, never your data** — every count, metric, series, or list comes from a *domain* service. A dashboard is only real when a backend actually produces that data, so **always provide one**. The only question is whether you wire to a backend that exists or build a new one — both count as providing it:

- **A service already owns this data** — that service *is* the backend; wire the widgets to it (server `load` + a BFF route, or an SSE subscription). Don't stand up a second one.
- **Nothing exposes it yet** — then building the backend is part of *this* task. Don't stop at the frontend or leave widgets bound to a source that produces nothing (a polished dashboard rendering zeros):
  1. Prefer **extending the service that owns the domain**; if none does, scaffold a new one with **`aoh-go-init`**. The false choice to avoid is "AOH module vs. nothing" — your own domain service is usually the answer.
  2. Add the read / aggregation endpoint the widgets call (e.g. `GET /v1/<resource>/metrics`, standard envelope) per **`aoh-conventions`** (`api.md` for shape + `/v{N}` paths, `database.md` for the query), fronted by a SvelteKit BFF route so the browser never holds the token.
  3. For **live** widgets (real-time counts, status, positions), publish changes to **RTUS** from that service and subscribe with `@mssfoobar/sse-client` — DASH has no live channel of its own. See `aoh-knowledge`'s `rtus.md`.

A local stub (seeded values + a timer) is only ever a **temporary dev scaffold while you stand up the real backend** — never the finished state. Mark it clearly, name the endpoint / RTUS feed it stands in for, and replace it with the real data path before the dashboard is done.

Why: the most common way a dashboard task fails is shipping widgets bound to data that nothing produces. Providing the backend is part of building the dashboard, every time.

## Recommended module layout

Inside the consumer app (the one `aoh-web-init` scaffolded), organise dashboard code under a single module folder:

```
apps/<app-name>/
└── src/
    ├── lib/<your-module>/
    │   ├── widgets/
    │   │   ├── register.ts                ← One line per widget (widget + view binding)
    │   │   ├── Counter/
    │   │   │   ├── CounterWidget.svelte.ts ← defineWidget({...})
    │   │   │   └── View.svelte             ← Receives WidgetProps<typeof CounterWidget>
    │   │   └── Clock/
    │   │       ├── ClockWidget.svelte.ts
    │   │       ├── View.svelte
    │   │       └── WidgetConfig.svelte     ← Optional custom config UI
    │   ├── constants/
    │   │   └── widget.constants.ts         ← Side-effect imports register.ts → exports `factory`
    │   └── utils/
    │       └── sync-widget-types.ts        ← Pushes file-defined widgets to backend
    └── routes/(authed)/<your-prefix>/
        ├── +layout.svelte                  ← Readiness check, module nav
        ├── +page.svelte / .server.ts       ← Dashboard list
        ├── create/
        │   └── +page.svelte / .server.ts   ← Create new dashboard
        └── [id]/
            └── +page.svelte / .server.ts   ← View & edit one dashboard
```

`<your-module>` and `<your-prefix>` are project-specific (e.g. `lib/ops/` + `routes/(authed)/ops/`). The `(authed)` route group name matches whatever `aoh-web-init` produced for the post-login route — adjust to your scaffold.

## The three routes — what each one does

### `+page.svelte` (Dashboard list)

Paged list of dashboards with search, sort, multi-select delete, favourite toggle, and a Create button. Server load pulls the first page; client load handles search/pagination/sort.

Server-side responsibilities:
- `client.listDashboards({ sort, asc, page, size })` (initial page)
- Optionally check tenant-admin permissions for showing admin actions (see `aoh-knowledge`'s `iams.md` for the JWT role pattern)

Client-side responsibilities:
- DataTable wiring (columns, sort, pagination)
- Search debounce → `client.listDashboards({ ..., name })`
- Bulk delete (parallel `client.deleteDashboard` calls)
- Favourite toggle (`client.setFavourite`)
- Row click → `goto(\`/.../\${id}\`)`

**Full walkthrough:** `references/list-page.md`.

### `create/+page.svelte` (Create dashboard)

`createDashboardEditor({ client, mode: "create", validate, on*Success })` — no `dashboardId`, so `editor.saveDashboard(...)` auto-dispatches to `client.createDashboard`. User assembles widgets via `editor.setPickerState(true)`, names the dashboard, clicks Save.

Server-side responsibilities (parallel `Promise.all`):
- `client.listTags()` — for the tag picker
- `client.listWidgetTypes()` — for backend overrides
- `client.listCategories()` — for category names in the picker

Client-side responsibilities:
- `factory.applyOverrides(data.widget_types, data.categories)` on mount
- `syncWidgetTypes(factory, client, data.widget_types ?? [])` on mount (push any file-only widget types to backend)
- Build the `editor` once at the top of the page; reads/writes its state directly (no bindings)
- On save: `editor.saveDashboard({ name, description, favourite, tags })` — the editor's `validate` hook runs, then the API call, then `onDashboardSaveSuccess` fires with the new dashboard

**Full walkthrough:** `references/edit-create-pages.md`.

### `[id]/+page.svelte` (View + edit one dashboard)

`createDashboardEditor({ client, dashboardId, mode: "view", validate, on*Success, on*Error })` — passing `dashboardId` makes `editor.saveDashboard(...)` dispatch to `client.updateDashboard` with the internal `oldWidgetIds` delete-diff.

- View mode by default (dashboard non-editable, Edit/Delete/Favourite buttons visible)
- `editor.setMode("edit")` enables editing and reveals Add Widget / Assign Label / Save
- Saves go through `editor.saveDashboard(...)` → `client.updateDashboard(...)`
- Per-widget save: `editor.saveWidget(widget)` (wired automatically through the picker's "Confirm Changes" button)
- Optional: fullscreen mode, export-to-JSON

Server-side responsibilities (parallel `Promise.all`):
- `client.getDashboardById({ dashboard_id })`
- `client.listTags()`, `client.listWidgetTypes()`, `client.listCategories()` — same as create
- `client.getDashboardTags({ dashboard_id })` — resolve tag mappings against full tag list
- Redirect to an error page if the dashboard wasn't found

**Full walkthrough:** `references/edit-create-pages.md`.

### `+layout.svelte` (Module shell)

Tiny — fires a readiness check on mount via the SDK's `checkHealth()` (typically wrapped in a `+server.ts` route that holds the service URLs server-side) so users see a toast if the backend is unreachable. Optional but recommended.

## Writing a widget — the 30-second version

```ts
// CounterWidget.svelte.ts
import { defineWidget } from "@mssfoobar/dash-web-sdk/renderer";

export const CounterWidget = defineWidget({
  type: "counter",
  meta: { title: "Counter", icon: "hash", minWidth: 8, minHeight: 8, maxWidth: 14, maxHeight: 14 },
  config: {
    step:  { type: "number", default: 1, label: "Step", min: 1, max: 100 },
    count: { type: "number", default: 0, hidden: true },  // persisted, no form field
  },
  actions: {
    increment: (s) => { s.count += s.step; },
    reset:     (s) => { s.count = 0; },
  },
});
```

```svelte
<!-- View.svelte -->
<script lang="ts">
  import type { WidgetProps } from "@mssfoobar/dash-web-sdk/renderer";
  import { CounterWidget } from "./CounterWidget.svelte";
  let { model, context }: WidgetProps<typeof CounterWidget> = $props();
</script>

<div>
  <span>{model.count}</span>
  <button onclick={() => model.increment()}>+</button>
</div>
```

```ts
// register.ts
import { widgetFactory } from "@mssfoobar/dash-web-sdk/renderer";
import { CounterWidget } from "./Counter/CounterWidget.svelte";
import CounterView from "./Counter/View.svelte";

widgetFactory.register(CounterWidget, { view: CounterView });
```

That's the whole pattern. `model.count` and `model.step` are typed and reactive (inferred from `config`). Actions become methods (`model.increment()`). Persistence is automatic — `<DashboardCanvas>` mirrors `model.toConfig()` to `widget.config` on every change.

The widget definition and the View live in separate files because **they reference each other at the type level** — putting `view: View` inside `defineWidget()` would create a TypeScript cycle. Registration ties them together at one place (`register.ts`).

**For everything else** — derived/computed properties, custom config UIs, all field types, sizing, validation — read `references/widgets.md`.

## Seeding a dashboard

Seeding — provisioning a dashboard *before* a user opens the UI — is a platform concern. **Which mechanism to use, plus the auth / tenant / reproducibility context, lives in `aoh-knowledge` → `references/services/dash.md` ("Seeding a dashboard")**, not here. In short: for anything that must be reproducible (an openspec change, a default board every user sees on login) the dashboard is seeded by **`aoh-compose`'s `dash-init`** one-shot from a declarative `dashboard.yaml`; ad-hoc "make me a dashboard now" creation uses **`DashboardClient`** (the SDK this skill owns).

This skill owns only the **widget side** of a seed: the seed's `widget_type_id`s and `config` blobs must match the `defineWidget` types and config schemas you author here. So the work that belongs to this skill is:

1. Build the widgets the dashboard needs with `defineWidget` (see `references/widgets.md`) and register them in `widgets/register.ts`.
2. Plan widget positions across the 42-column grid (see `references/widgets.md` Sizing).
3. Produce a `config` blob per widget that matches its `defineWidget` schema exactly (mismatched types silently fall back to defaults) — that contract is in `references/widgets.md` (Config field options).

Then hand those to whichever mechanism `dash.md` selects (a `dashboard.yaml` entry for `dash-init`, or a `DashboardClient.createDashboard` payload for ad-hoc). **`references/widgets.md`** (Config field options) documents the widget-config contract; **`dash.md`** owns the mechanism + auth.

## Decision points (read this before coding)

### Do I need a custom config UI?

Most widgets only need fields the auto-generated form covers (`string`, `textarea`, `number`, `boolean`, `select`). **Default to the schema-driven form** — it's free, accessible, and matches the rest of the picker.

Reach for a `configComponent` only when:
- Fields need to be conditionally shown (e.g. show "currency code" only if `format === "currency"`)
- The user needs to pick from a dynamic list (e.g. fetched options)
- The config needs custom interactivity (drag-to-reorder, color picker, etc.)

Even when you provide a custom component, the `config` schema still defines the persisted shape — the custom UI just replaces the *rendering*. Persistence, hydration, and the `model.foo` accessors all work the same.

### Where does state belong — `config` or somewhere else?

Anything that needs to **survive a reload** goes in `config`. That includes runtime-mutable values like a counter (`count: { type: "number", default: 0, hidden: true }`). The `hidden` flag keeps it persisted but out of the form.

Ephemeral UI state (a popover's open/closed, a tooltip target) belongs in the View as ordinary `$state(...)` — it should not persist.

### Do I need a separate categories/tags table?

Tags are owned by the dash service itself (there is no separate tag service); you `listTags()` and `createTag()` through `DashboardClient`, which routes them to the dash backend. Categories are also local to the dash backend (`listCategories()`). Both are surfaced in the WidgetPicker via `factory.applyOverrides(widgetTypes, categories)`.

## File conventions

- Widget classes live in `.svelte.ts` files (Svelte 5 runes need preprocessing). Import via `./MyWidget.svelte` (drop the `.ts`).
- View components are plain `.svelte` files.
- Custom config components are plain `.svelte` files named `WidgetConfig.svelte` by convention.
- One `register.ts` per module. Side-effect import it from `widget.constants.ts` (or wherever you export the factory) so the factory is primed before any page renders.

For broader SvelteKit / Tailwind / file-naming rules, defer to **`aoh-conventions`** (specifically `references/web.md`) — don't restate them here.

## Common pitfalls

1. **Overriding `<DashboardCanvas config>` when you don't need to** — `<DashboardCanvas>` falls back to sensible defaults (42 columns, square cells, 0.5rem margin) and the backend assumes this layout. Only pass `config` when a page genuinely needs a different column count or margin.
2. **Reaching for `bind:instanceModels` (old pattern)** — The editor owns `instanceModels` now. To force re-hydration on cancel/import, set `editor.instanceModels = {}` and bump `canvasKey++` on the `{#key}` wrapper around `<DashboardCanvas>`.
3. **Importing the widget file inside its own View** — That works (and is required for typing) only because **the widget file no longer imports its View**. Keep that direction.
4. **Mutating `model` directly inside a View prop type expectation** — `model.count = 5` works (the schema setter handles it) but it's clearer to expose a named action: `actions: { setCount: (s, n) => s.count = n }`. Use direct assignment for one-off cases (e.g. input bindings); use actions for behavior you want to test or re-use.
5. **Server load returning a non-`ok` result without catching** — Always wrap `DashboardClient` calls in `.catch((err) => { log.error(...); return null; })` inside `Promise.all` so one failure doesn't crash the load function.
6. **Setting `ssr = false` everywhere** — The list page can keep SSR enabled. The create/edit pages need `export const ssr = false` because the dashboard renderer mounts to the DOM.

## What this skill won't do for you

- Pick column layouts. Plan widget positions yourself across the dashboard's 42 columns (see Sizing section of `references/widgets.md`).
- Design the visual language for your widget. Use your design system's semantic color tokens — never raw Tailwind colors. See **`aoh-design`** (and `references/widget-style.md` for widget-specific patterns) for the tokens that work with `@mssfoobar/ui`.
- Scaffold a SvelteKit app — that's **`aoh-web-init`**.
- Bring up the dash backend or Postgres — that's **`aoh-compose`** (it ships a `dash/` compose fragment).
- Write the Go for a data backend, or own a domain's schema — the *mechanics* are **`aoh-go-init`** + **`aoh-conventions`**. But note the boundary: every dashboard is backed by a real data service, and *ensuring that — building one whenever nothing already provides the data* — is this skill's job (see "The data behind the widgets — always provide a backend"). Those sibling skills supply the how; making sure the backend exists is owned here, not deferred away.
- Document AOH service catalogue or IAMS auth model — that's **`aoh-knowledge`**.
- Decide *how* a dashboard is seeded (`dash-init` vs SDK) or the seed's auth / tenant / reproducibility model — that's **`aoh-knowledge`** (`dash.md` → "Seeding a dashboard") + **`aoh-compose`** (the `dash-init` reconciler).
- Enforce code/style conventions — that's **`aoh-conventions`**.
