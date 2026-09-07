# Widgets — `defineWidget()` reference

Authoring a widget means filling in a single spec object and binding it to a Svelte view component at registration time. The schema is the single source of truth — it defines persisted state, form controls, defaults, and types.

## Table of contents

- [The four files](#the-four-files) — one widget = a class file + View + optional WidgetConfig + register line
- [Spec fields](#spec-fields) — `type`, `meta`, `config`, `derived`, `actions`
- [Config field options](#config-field-options) — all field types with every property
- [Derived (computed) properties](#derived-computed-properties)
- [Actions (methods)](#actions-methods)
- [Hidden / persistent-only fields](#hidden--persistent-only-fields)
- [Custom config UI (`configComponent`)](#custom-config-ui-configcomponent)
- [The View component](#the-view-component)
- [Registration](#registration)
- [Sizing & widget meta](#sizing--widget-meta)
- [Backend overrides (`factory.applyOverrides`)](#backend-overrides-factoryapplyoverrides)
- [Common patterns](#common-patterns)

## The four files

A typical widget folder:

```
widgets/MyWidget/
├── MyWidget.svelte.ts      ← defineWidget({...}) — needs .svelte.ts for $state runes
├── View.svelte             ← The renderer; receives WidgetProps<typeof MyWidget>
└── WidgetConfig.svelte     ← OPTIONAL: custom config UI (replaces auto-form)
```

Plus one line in the shared `widgets/register.ts`:

```ts
widgetFactory.register(MyWidget, { view: MyWidgetView, configComponent: MyWidgetConfig /* optional */ });
```

The class file ends in `.svelte.ts` because `defineWidget()` uses `$state` internally — that rune is only valid in Svelte-preprocessed files. Import it as `./MyWidget.svelte` (drop the `.ts`).

## Spec fields

```ts
defineWidget({
  type:       "metric",                // required — backend widget_type_id
  meta:       { ... },                 // required — title, icon, sizing
  config:     { ... },                 // required — persisted state schema
  derived:    { ... },                 // optional — computed read-only props
  actions:    { ... },                 // optional — methods that mutate state
})
```

Note: `view` and `configComponent` are NOT in the spec. They're supplied at registration to avoid a TypeScript cycle (the View references the widget's type via `typeof MyWidget`; if the widget also referenced the View, neither would resolve).

## Config field options

`config` is a record of named fields. The field type determines the form control and the TS type of the persisted value. Every field has these common properties:

| Property | Type | Purpose |
|---|---|---|
| `type` | `"string" \| "textarea" \| "number" \| "boolean" \| "select"` | Discriminator |
| `default` | matches type | Initial value & fallback when hydration data is missing |
| `label?` | `string` | Form label. Falls back to the field key. |
| `group?` | `string` | Visual group heading in the form |
| `hidden?` | `boolean` | Persist but don't render a form control |

> **This schema is also the contract for seeding.** When a dashboard is *seeded* (a `dashboard.yaml` for `dash-init`, or a `DashboardClient.createDashboard` payload), each widget's `config` blob is hydrated against this schema on load — a value whose type doesn't match its field falls back to that field's `default` (so a seeded `"7"` for a `number` field renders `0`). Match keys + types exactly; `select` values must be one of the option values verbatim. The seeding *mechanism* + auth are platform context — see `aoh-knowledge`'s `dash.md` → "Seeding a dashboard".

### `string`

```ts
name: {
  type: "string",
  default: "Friend",
  label: "Name",
  placeholder: "Who are you greeting?",  // optional
  group: "Content",
}
```

Renders as a single-line `<input type="text">`.

### `textarea`

```ts
description: {
  type: "textarea",
  default: "",
  label: "Description",
  placeholder: "Optional notes...",  // optional
  group: "Content",
}
```

Renders as a multi-line `<textarea>` with vertical resize.

### `number`

```ts
step: {
  type: "number",
  default: 1,
  label: "Step",
  min: 1,                    // optional — passed to <input min>
  max: 100,                  // optional
  step: 1,                   // optional — input step
  placeholder: "default",    // optional
  group: "Behavior",
}
```

Renders as `<input type="number">`. Empty inputs are coerced to `0`.

### `boolean`

```ts
showSeconds: {
  type: "boolean",
  default: true,
  label: "Show Seconds",
  group: "Display",
}
```

Renders as a checkbox.

### `select`

```ts
format: {
  type: "select",
  default: "number",
  label: "Format",
  options: [
    { label: "Number", value: "number" },
    { label: "Currency (USD)", value: "currency" },
    { label: "Percent", value: "percent" },
  ],
  group: "Formatting",
}
```

Options can be plain `string[]` (the string is both label and value) OR `{ label, value }[]` when you need display/value separation. `default` is matched against the value.

Note: at the TS level the field exposes `string` (not the union of option values). If you need narrower typing inside `derived`/`actions`, narrow with a type guard:

```ts
if (s.format === "currency") { /* TS knows it's "currency" here */ }
```

## Derived (computed) properties

`derived` exposes read-only computed values on the model. Each function receives the reactive state and returns the computed value. **The function re-runs every time the property is read**, so Svelte's tracking picks up dependencies automatically — no `$derived` needed.

```ts
defineWidget({
  config: {
    value:      { type: "number", default: 0, label: "Value" },
    comparison: { type: "number", default: 0, label: "Previous" },
  },
  derived: {
    delta: (s) => s.value - s.comparison,
    direction: (s) => {
      if (s.value > s.comparison) return "up" as const;
      if (s.value < s.comparison) return "down" as const;
      return "flat" as const;
    },
    percent: (s) =>
      s.comparison === 0 ? 0 : ((s.value - s.comparison) / Math.abs(s.comparison)) * 100,
  },
});
```

In the View: `model.delta`, `model.direction`, `model.percent` — fully typed (`number`, `"up" | "down" | "flat"`, `number`).

**Cross-references between derived:** a derived function only receives the state, not other deriveds. If you want `direction` to reference `delta`, inline the expression: `const d = s.value - s.comparison;`. Most derived chains are short enough that this isn't a real problem.

**Side effects in derived:** functionally allowed (the function is just JS) but **don't do this**. If you need to react to changes, use `$effect` in the View. If you find yourself wanting `setInterval` in a derived, you want a View-level `$effect` (see the Clock example below).

## Actions (methods)

`actions` exposes methods on the model. Each function receives the reactive state and is invoked as `model.methodName()`.

```ts
defineWidget({
  config: {
    step:  { type: "number", default: 1, label: "Step", min: 1, max: 100 },
    count: { type: "number", default: 0, hidden: true },
  },
  actions: {
    increment: (s) => { s.count += s.step; },
    decrement: (s) => { s.count -= s.step; },
    reset:     (s) => { s.count = 0; },
  },
});
```

Actions take no arguments (other than the injected state). If you want a parameterized action like `setTo(n: number)`, just inline the assignment in the View (`oninput={(e) => model.count = Number(e.currentTarget.value)}`) — every config field has a setter on the model. Actions are best for named, multi-step state changes that you'd reuse or want to read as a verb.

## Hidden / persistent-only fields

Set `hidden: true` on a config field to persist it without rendering a form control. Use this for state the **widget itself** mutates (like a counter's `count`) — values that need to survive a reload but shouldn't be hand-edited.

```ts
count: { type: "number", default: 0, hidden: true },  // no label needed
```

Hidden fields:
- Still need a `type` and `default` (used for runtime validation on hydration and initial value)
- Don't need a `label` (and shouldn't have one — they don't render)
- Are mutable from `actions` (`s.count = ...`) and from the View (`model.count = ...`)

This is also the right pattern for **state that's set by the View, not the user** — e.g. a chart widget storing its zoom level, or a clock storing the last tick.

## Custom config UI (`configComponent`)

When the auto-generated form isn't enough, supply a custom Svelte component as `configComponent`. The picker uses it instead of `ConfigPanel`.

```svelte
<!-- Clock/WidgetConfig.svelte -->
<script lang="ts">
  import { ClockWidget } from "./ClockWidget.svelte";

  let { model }: { model: InstanceType<typeof ClockWidget> } = $props();
</script>

<div class="flex flex-col gap-4 text-sm">
  <label>
    <span>Time Format</span>
    <div class="flex gap-2">
      <button onclick={() => (model.format = "12h")}
        class:selected={model.format === "12h"}>12 Hour</button>
      <button onclick={() => (model.format = "24h")}
        class:selected={model.format === "24h"}>24 Hour</button>
    </div>
  </label>

  <label>
    <span>Show Seconds</span>
    <input type="checkbox"
      checked={model.showSeconds}
      onchange={(e) => (model.showSeconds = e.currentTarget.checked)} />
  </label>
</div>
```

Then register the widget with both:

```ts
widgetFactory.register(ClockWidget, { view: ClockView, configComponent: ClockWidgetConfig });
```

**Key points:**
- The custom component receives the same live `model` that the View uses, so edits appear immediately in the rendered widget.
- The `config` schema is still required — it defines what `model.format` and `model.showSeconds` look like, sets defaults, and handles hydration on import.
- Mix and match: a custom component can render `<ConfigPanel schema={schema} model={model} />` for some fields and custom UI for others, but in practice it's simpler to go fully custom when you reach for this.

When to use a custom component:
- Conditional fields (show "currency code" only when `format === "currency"`)
- Dynamic options (e.g. fetched from an API)
- Specialized inputs (color pickers, sliders with markers, key-binding pickers)
- Custom layouts (tabs, accordions, side-by-side previews)

## The View component

The View receives `model` (the live widget instance) and `context` (`{ id, edit }`). Type the props with `WidgetProps<typeof MyWidget>`.

```svelte
<script lang="ts">
  import type { WidgetProps } from "@mssfoobar/dash-web-sdk/renderer";
  import { CounterWidget } from "./CounterWidget.svelte";

  let { model, context }: WidgetProps<typeof CounterWidget> = $props();
</script>

<div>
  <span>{model.count}</span>
  {#if !context.edit}
    <button onclick={() => model.increment()}>+</button>
  {/if}
</div>
```

`context.edit` tells you whether the dashboard is in edit mode — useful for hiding interactive controls so the user can drag/resize without firing clicks.

`context.id` is the widget instance's UUID. Useful as a key when you need to bind external state to a specific instance (e.g. a chart library that needs an ID).

### Reacting to state changes in the View

Reads inside the template track reactively for free. For side effects (timers, network calls, third-party library updates), use `$effect`:

```svelte
<script lang="ts">
  let { model, context }: WidgetProps<typeof ClockWidget> = $props();

  let time = $state("");

  function updateTime() { /* compute from model.format / model.showSeconds */ }

  $effect(() => {
    // Re-run when these change
    void model.format;
    void model.showSeconds;
    updateTime();
  });

  onMount(() => {
    updateTime();
    const interval = setInterval(updateTime, 1000);
    return () => clearInterval(interval);
  });
</script>
```

The `void model.format` reads are deliberate dependency reads — Svelte tracks them so the effect re-runs when the format changes.

## Registration

All widget registration lives in one file (`widgets/register.ts`). Import the widget class, its View, and any custom config component, then call `widgetFactory.register(widget, { view, configComponent? })`.

```ts
import { widgetFactory } from "@mssfoobar/dash-web-sdk/renderer";

import { CounterWidget } from "./Counter/CounterWidget.svelte";
import CounterView from "./Counter/View.svelte";

import { ClockWidget } from "./Clock/ClockWidget.svelte";
import ClockView from "./Clock/View.svelte";
import ClockConfig from "./Clock/WidgetConfig.svelte";

import { MetricWidget } from "./Metric/MetricWidget.svelte";
import MetricView from "./Metric/View.svelte";

widgetFactory.register(CounterWidget, { view: CounterView });
widgetFactory.register(ClockWidget,   { view: ClockView, configComponent: ClockConfig });
widgetFactory.register(MetricWidget,  { view: MetricView });
```

The factory is a singleton — the same instance the SDK uses internally. Make sure your pages get it via the constants module that side-effect imports register.ts:

```ts
// widget.constants.ts
import { widgetFactory } from "@mssfoobar/dash-web-sdk/renderer";
import "$lib/your-module/widgets/register";  // ← side-effect import

export const factory = widgetFactory;
```

Pages then `import { factory } from "$lib/your-module/constants/widget.constants"` and pass it to `<DashboardCanvas>` and `<WidgetPicker>`.

## Sizing & widget meta

```ts
meta: {
  title:      "Metric",          // required — display name in palette
  icon:       "trending-up",     // required — Lucide icon name in kebab-case
  minWidth:   8,                 // required — min dashboard columns
  minHeight:  4,                 // required — min dashboard rows
  maxWidth:   24,                // optional — defaults to 999
  maxHeight:  10,                // optional — defaults to 999
  limit:      1,                 // optional — max instances per dashboard
  enabled:    true,              // optional — visible in palette
  category:   undefined,         // optional — set by backend overrides, not here
}
```

### The 42-column dashboard

A dashboard is **42 columns wide** with square auto-sized cells (defaults; override via `<DashboardCanvas config={{ column: ... }}>` if you really need a different layout). Plan widget positions on this scale:

| Layout | Widths | Columns (left-edge) |
|---|---|---|
| Full width | 42 | 0 |
| 2 equal columns | 21 + 21 | 0, 21 |
| 3 equal columns | 14 + 14 + 14 | 0, 14, 28 |
| 2 columns (2:1) | 28 + 14 | 0, 28 |

### Size each archetype to its content

A widget spawns at its **min** size, so the min must be the size where the widget *already looks right* — every content region visible, nothing clipped. Don't set a tiny min and rely on the user resizing, or on responsive tricks, to rescue it. The single most common sizing bug is a stat card whose `minHeight` fits the value but **clips its footer** — count the regions and give each room.

Rule of thumb: cells are square, so 1 row ≈ 1 column-width tall. A region costs roughly: a label row ≈ 2 rows, a big value ≈ 3 rows, each footer line ≈ 1.5 rows, plus `p-4` padding ≈ 2 rows. Add them up.

| Archetype (see `widget-style.md`) | min cols × rows | Why that size |
|---|---|---|
| Compact stat (value only) | 8 × 4 | label + value + padding |
| **Stat card** (label + badge + value + 2-line footer) | 10 × 5 | all four regions, footer not clipped |
| List / feed | 14 × 10 | header + ~5 readable rows |
| Gauge / status breakdown (bars / donut) | 12 × 8 | label + a few bars or a dial + legend |
| **Chart card** | 20 × 12 | header + a chart tall enough to read |
| **Data-table block** | 28 × 14 | header + ~6 rows + pagination footer |

These are starting points, not gospel — but they're calibrated against square cells, where rows are surprisingly tall. **A stat card at `minHeight: 7` looks broken**: the value floats near the top, the footer pins to the bottom (`mt-auto`), and a dead gap yawns between them. `5` is the sweet spot. Verify by rendering — the most common sizing mistake is setting heights from intuition rather than looking.

Two more rules the grid enforces:
- **Set a `maxHeight`** on cards that shouldn't stretch (stat cards: `maxHeight: 7`). Without it, a user can drag a stat card to 20 rows and strand the number in the middle. Charts and tables can stretch freely (omit `maxHeight`).
- **`minWidth`s of widgets you intend to sit side-by-side must sum to ≤ the column count (42).** Four stat cards at `minWidth: 11` total 44 > 42, so the grid drops one to the next row no matter how you place them. Keep a row of four at `minWidth: 10`.

### Container queries inside widgets

Tailwind's `sm:`/`md:`/`lg:` breakpoints respond to **viewport**, not widget size, so they're useless inside widgets. See `widget-style.md` for the full guide on container queries, the widget archetypes, and charts.

Quick version — the dashboard sets `container-type: inline-size` on every widget cell, so use `@container` queries in a `<style>` block:

```svelte
<span class="widget-title text-xs">{model.label}</span>

<style>
  @container (min-width: 400px) {
    .widget-title { font-size: 0.875rem; }   /* text-sm */
  }
  @container (min-width: 550px) {
    .widget-title { font-size: 1rem; }       /* text-base */
  }
</style>
```

### Limit

Use `limit` for widgets that should appear at most once (or N times) per dashboard — e.g. a "Page Header" widget that only makes sense once. `<DashboardCanvas>` fires `onlimitreached` when the user tries to drop a clone beyond the limit; the pages catch it and toast a message.

## Backend overrides (`factory.applyOverrides`)

A widget's file-declared `meta` is the default. **Backend `WidgetType` records can override** any subset of fields (title, icon, sizing, enabled flag, category). This lets ops rename or disable widgets without a redeploy.

Pages apply overrides on mount:

```ts
if (data.widget_types?.length) {
  factory.applyOverrides(data.widget_types, data.categories);
}
```

Overrides come from `client.listWidgetTypes()` (server-loaded). The merge is simple: backend value wins if defined, file value falls back. Category names resolve from `client.listCategories()`.

To get a widget into the backend so an admin can override it, call `syncWidgetTypes(factory, client, data.widget_types ?? [])` on the create/edit pages — it pushes every file-registered widget that isn't yet in the backend. A 409 (already exists) is treated as success.

## Common patterns

### Persisted ephemeral state (counter, scroll position)

```ts
config: {
  count: { type: "number", default: 0, hidden: true },
},
actions: {
  increment: (s) => { s.count += 1; },
},
```

### Schema-driven select with computed display

```ts
config: {
  format: { type: "select", default: "number", label: "Format",
    options: [{ label: "Number", value: "number" }, { label: "Currency", value: "currency" }] },
  value: { type: "number", default: 0, label: "Value" },
  precision: { type: "number", default: 2, label: "Decimal Places", min: 0, max: 6 },
},
derived: {
  formatted: (s) => {
    const opts = { minimumFractionDigits: s.precision, maximumFractionDigits: s.precision };
    if (s.format === "currency") {
      return new Intl.NumberFormat(undefined, { style: "currency", currency: "USD", ...opts }).format(s.value);
    }
    return new Intl.NumberFormat(undefined, opts).format(s.value);
  },
},
```

### Tier-2 custom config (Clock)

See the "Custom config UI" section above for the canonical pattern — a segmented format toggle plus a checkbox, with the model accessed directly via `model.format` / `model.showSeconds`.

### Self-updating widget (Clock)

Internal interval lives in the View (`onMount` → `setInterval`), not in `derived`. The View reads `model.format` inside an `$effect` to refresh on config change.

### Computed indicator (Metric direction)

`direction: (s) => s.value > s.comparison ? "up" as const : s.value < s.comparison ? "down" as const : "flat" as const` — use `as const` if you want narrow string-literal types in the View.

### Validation on hydration

`hydrate()` is generated for you. It validates each value against the field type (e.g. `number` requires `typeof === "number"`, `select` requires the value to be in `options`). Invalid values from the backend are silently ignored and the default is kept — no need to write defensive checks.
