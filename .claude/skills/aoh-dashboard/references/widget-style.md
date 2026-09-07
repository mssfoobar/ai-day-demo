# Widget visual style — the shadcn-block design language

Widgets should read like **shadcn dashboard blocks**: a stat card with a label, a trend badge, a big tabular number and a two-line footer; a chart card with a titled header and a range switcher; a data-table block with status badges and a pagination footer. Polished, structured, consistent — not flat ad-hoc text.

The whole point of this file: a dashboard looks unprofessional when every widget invents its own type scale, spacing, and chrome. Pull from one shared component kit and one type scale and the dashboard reads as a single designed surface.

> **This reverses the old "austere" guidance.** Earlier versions of this skill banned badges, pills, and decorated footers. That produced flat, low-information widgets. We now lean *into* the shadcn vocabulary — badges, structured headers/footers, status colour — because that's what makes a dashboard scannable and finished.

## The widget cell *is* your Card — never nest another

The dashboard already wraps every widget in a card: the cell renders `.dash-widget-card` with the card background, border, radius and shadow (it's the shadcn `Card` shell). Your job is to fill that shell with structured content — **don't render your own `<Card>` inside a widget**, or you get a double border (a card in a card).

So replicate shadcn's `CardHeader` / `CardContent` / `CardFooter` *regions* as plain flex sections inside your root div — the structure and spacing, without the outer `<Card>` wrapper:

```svelte
<div class="@container flex h-full w-full flex-col overflow-hidden p-4 gap-3">
  <header class="flex items-start justify-between gap-2 shrink-0"> … </header>
  <div class="flex-1 min-h-0"> … </div>                <!-- content -->
  <footer class="shrink-0"> … </footer>
</div>
```

If you want the cell card itself to match shadcn exactly (radius, border tint, shadow), retheme `.dash-widget-card` once in your app's global stylesheet — see *Override hooks* below. That fixes the chrome for every widget at once.

## Build from the component kit — don't hand-roll

`@mssfoobar/ui` ships the full shadcn-svelte primitive set. Reach for these instead of bespoke markup — that's most of what keeps widgets consistent.

| Content | Component | Import |
|---|---|---|
| Status / trend / category / count | `Badge` | `@mssfoobar/ui/badge` |
| Tabular data | `Table` (+ `Pagination`) | `@mssfoobar/ui/table`, `/pagination` |
| Range / view switch | `ToggleGroup` or `Tabs` | `@mssfoobar/ui/toggle-group`, `/tabs` |
| Row actions menu | `DropdownMenu` | `@mssfoobar/ui/dropdown-menu` |
| Inline edit cell | `Select` | `@mssfoobar/ui/select` |
| Dividers | `Separator` | `@mssfoobar/ui/separator` |
| Loading placeholder | `Skeleton` | `@mssfoobar/ui/skeleton` |
| Hover detail | `Tooltip` | `@mssfoobar/ui/tooltip` |
| Any plotted chart | LayerChart | `layerchart` |
| Simple magnitude bars | plain CSS bar list | — |

## Badges (now core)

```svelte
<script lang="ts">
  import { Badge } from "@mssfoobar/ui/badge";
  import { TrendingUp } from "@lucide/svelte";
</script>

<!-- trend delta, shadcn SectionCard style: outline pill, icon + signed % -->
<Badge variant="outline" color={delta >= 0 ? "success" : "destructive"}>
  <TrendingUp class="size-3.5" />
  {delta >= 0 ? "+" : ""}{delta.toFixed(1)}%
</Badge>

<!-- status pill -->
<Badge variant="soft" color="success">Done</Badge>
<Badge variant="soft" color="warning">In Process</Badge>
```

- **Props:** `size` (`sm`/`md`/`lg`, default `md`), `variant` (`solid`/`soft`/`outline`/`surface`, default `soft`), `color` (`default`/`info`/`success`/`warning`/`destructive`).
- **Map colour to meaning**, not decoration: `success` = good / healthy / up-is-good, `destructive` = critical / down-is-bad, `warning` = degraded, `info` = neutral category. The Badge owns its own token colours — never pass raw Tailwind colour classes.
- **Direction vs. goodness can differ.** For "response time", a *downward* delta is *good* — colour by goodness (`success`), keep the arrow pointing down. Don't reflexively colour all up-arrows green.

## Widget archetypes

Most widgets are one of these. Each is described by its anatomy + the key snippet — adapt freely, they're patterns not rigid templates.

### Stat card — *shadcn SectionCard*

The workhorse. Anatomy, top to bottom:
1. **Header row** — label (left), trend `Badge` (right).
2. **Value** — one big number.
3. **Footer** — line 1: a short takeaway with a trend icon; line 2: muted context.

```svelte
<div class="@container flex h-full w-full flex-col overflow-hidden p-4 gap-1">
  <header class="flex items-start justify-between gap-2 shrink-0">
    <span class="text-sm text-muted-foreground truncate">{model.label}</span>
    <Badge variant="outline" color={delta >= 0 ? "success" : "destructive"}>
      <TrendingUp class="size-3.5" />{delta >= 0 ? "+" : ""}{delta.toFixed(1)}%
    </Badge>
  </header>

  <span class="font-semibold tabular-nums tracking-tight text-foreground truncate
               text-3xl @[14rem]:text-4xl @[20rem]:text-5xl">
    {model.formatted}
  </span>

  <footer class="mt-auto shrink-0 pt-2">
    <div class="flex items-center gap-1.5 text-sm font-medium text-foreground">
      Trending up this month <TrendingUp class="size-4" />
    </div>
    <div class="text-sm text-muted-foreground truncate">Visitors for the last 6 months</div>
  </footer>
</div>
```

Notes:
- Label is **normal case**, `text-sm text-muted-foreground` — *not* `uppercase tracking-widest`. Uppercased micro-labels read as dated and noisy; shadcn doesn't do it.
- `mt-auto` pins the footer to the bottom so cards in a row align even when values differ in height.

### Chart card — *shadcn area-chart block*

Anatomy: a header (title + muted description stacked on the left, a `ToggleGroup`/`Tabs` range switch on the right), then the chart filling the rest.

```svelte
<div class="@container flex h-full w-full flex-col overflow-hidden p-4 gap-3">
  <header class="flex items-start justify-between gap-2 shrink-0">
    <div class="min-w-0">
      <div class="text-base font-semibold truncate">Total Visitors</div>
      <div class="text-sm text-muted-foreground truncate">Total for the last 3 months</div>
    </div>
    <ToggleGroup.Root type="single" bind:value={range} size="sm">
      <ToggleGroup.Item value="90d">Last 3 months</ToggleGroup.Item>
      <ToggleGroup.Item value="30d">Last 30 days</ToggleGroup.Item>
      <ToggleGroup.Item value="7d">Last 7 days</ToggleGroup.Item>
    </ToggleGroup.Root>
  </header>
  <div bind:this={chartEl} class="flex-1 min-h-0"></div>
</div>
```

See *Charts* below for the LayerChart pattern that matches the shadcn gradient-area look.

### Data-table block — *shadcn data-table*

Anatomy: optional toolbar (`Tabs` + action buttons), a `Table` body, a footer with selection count, rows-per-page `Select`, and `Pagination`.

- **Status cells** are badges with an icon (`CircleCheck` for done, `Loader` for in-process), not plain text.
- **Numeric columns** are right-aligned and `tabular-nums`.
- **Editable cells** use an inline `Select`; **row actions** use a trailing `DropdownMenu` (the `⋮` button).
- Keep the table body in a `flex-1 min-h-0 overflow-auto` region so the footer stays pinned.

```svelte
<Table.Row>
  <Table.Cell class="font-medium truncate">{row.header}</Table.Cell>
  <Table.Cell><Badge variant="soft" color="default">{row.type}</Badge></Table.Cell>
  <Table.Cell>
    <Badge variant="soft" color={row.done ? "success" : "warning"}>
      {#if row.done}<CircleCheck class="size-3.5" />{:else}<Loader class="size-3.5" />{/if}
      {row.status}
    </Badge>
  </Table.Cell>
  <Table.Cell class="text-right tabular-nums">{row.target}</Table.Cell>
</Table.Row>
```

### List / feed

Anatomy: a scrollable column of flat rows — each `icon + primary (truncate) + muted secondary + trailing badge/number`. Divide rows with `Separator` or `divide-y divide-border`; wrap the column in `ScrollArea` (or `flex-1 overflow-auto min-h-0`). No bordered sub-cards per row — the flat row *is* the unit (see *No card-in-card*).

## Type scale & spacing

One scale, every widget. This is the single biggest lever on "looks consistent".

| Role | Classes |
|---|---|
| Widget / field label | `text-sm text-muted-foreground` (normal case) |
| Big value | `text-2xl`–`text-5xl` (scale up with `@container`) `font-semibold tabular-nums tracking-tight` |
| Section / card title | `text-base font-semibold` |
| Body, table cells, rows | `text-sm` |
| Footer / meta | `text-sm text-muted-foreground` (or `text-xs` when tight) |
| Any number | `tabular-nums`; right-align in tables |

- **Padding:** `p-4` is the default card-region padding (matches shadcn's roomier feel). Drop to `p-3` only when a widget is genuinely small. Internal gaps: `gap-2`/`gap-3`.
- **Weight:** values and titles are `font-semibold` — not `font-bold` (too heavy) and not `font-normal` (no hierarchy).
- **Truncate** every label, title, and table cell that holds a variable-length string.

## Colour & tokens

- **Semantic tokens only.** Never raw Tailwind colours (`text-red-500`, `bg-amber-500`) — they don't switch with the theme. Use `text-foreground`, `text-muted-foreground`, and the `Badge` `color` prop for status.
- **Charts** use the chart palette tokens `--chart-1` … `--chart-5` (available as `text-chart-1` / `bg-chart-1` etc.). Pull series colours from these so charts theme with the rest of the app.

## The three data states — always render them

A data-driven widget (anything bound to a backend — see SKILL.md "The data behind the widgets") has three states. shadcn blocks render all three; so should you:

- **Loading** — a `Skeleton` shaped like the content (a wide bar where the value goes, a few lines where rows go), *not* the word "Loading…". A skeleton that matches the final layout avoids the jarring reflow when data lands.
- **Error** — a short muted line (`text-sm text-muted-foreground`), e.g. "Feed unavailable". Don't dump the raw error.
- **Empty** — a centered muted hint when the backend returns nothing, so a real-but-empty widget doesn't look broken.

```svelte
{#if feed.loading}
  <Skeleton class="h-9 w-28" />
{:else if feed.error}
  <span class="text-sm text-muted-foreground">Feed unavailable</span>
{:else if feed.metrics}
  …
{/if}
```

## Charts (LayerChart)

Use **LayerChart** for plotted charts (line, area, bar, pie/donut, scatter). It's Svelte-native and is what shadcn-svelte's own chart components are built on, so it themes with the rest of the app and matches the reference blocks out of the box. Don't reach for a second charting library — one chart vocabulary across the dashboard is half of what makes it feel consistent.

> **Not every "chart" needs LayerChart.** A short labeled magnitude comparison (incident counts by severity, a top-5 list) is a **bar list** — a few flex rows with a `bg-*` track and a `tabular-nums` value, like the SeverityBreakdown widget. That's lighter, gives you per-row colour control, and reads cleanly. Save LayerChart for charts with axes, curves, or arcs.

LayerChart composes primitives inside a `<Chart>` context — `<Svg>` plus `Area`, `Line`/`Spline`, `Bars`, `Axis`, `Grid`, `LinearGradient`, `Highlight`, `Pie`/`Arc`, `Tooltip`. A donut (sized off the smaller of width/height so it fits any cell):

```svelte
<script lang="ts">
  import { Chart, Svg, Pie, Arc } from "layerchart";
  let cw = $state(0), ch = $state(0);
  const outerR = $derived(Math.max(0, Math.min(cw, ch) * 0.36));   // fits tall or wide cells
  // Series colours: pull from the chart palette tokens so they theme with the app.
  const COLORS = ["var(--color-chart-1)", "var(--color-chart-2)", "var(--color-chart-3)"];
</script>

<div class="relative min-h-0 w-full flex-1" bind:clientWidth={cw} bind:clientHeight={ch}>
  <Chart {data} x="value">
    <Svg center>
      <Pie sort={null} let:arcs>
        {#each arcs as arc, i (arc.data.label)}
          <Arc startAngle={arc.startAngle} endAngle={arc.endAngle} padAngle={arc.padAngle}
               outerRadius={outerR} innerRadius={outerR * 0.62} cornerRadius={2}
               fill={COLORS[i]} />
        {/each}
      </Pie>
    </Svg>
  </Chart>
</div>
```

For a trend (area/line), compose `<Axis placement="bottom"/>` + `<Axis placement="left"/>` + `<Area>`/`<Spline>` inside `<Svg>`, fill the area with a `<LinearGradient>` from `--color-chart-1` to transparent, and keep the chart container `flex-1 min-h-0` so it tracks the cell. Theme via the chart tokens (`--color-chart-1` … `--color-chart-5`) and `fill-muted`/`stroke-border` utilities rather than hard-coded colours — never raw hex.

> **Pin the LayerChart version.** This pattern targets the primitive composition API. Check the installed major (the repo currently uses **v1**) before copying snippets from elsewhere — v2 introduced higher-level `BarChart`/`AreaChart` components with a different shape. Match what's installed.

If several widgets share a chart shape, lift the option/setup into a small helper in your module and reuse it — don't repeat the theming in every widget.

## Container queries inside widgets

Tailwind's `sm:`/`md:`/`lg:` fire on **viewport** width, not widget width, so they're useless here. The dashboard sets `container-type: inline-size` on every cell, so use `@container` (utility form `@[20rem]:text-4xl`, or a `@container` block in `<style>`):

```svelte
<span class="value text-3xl @[14rem]:text-4xl @[20rem]:text-5xl">{model.value}</span>
```

| Container width | Show |
|---|---|
| < 22rem | Label + the one essential number. Hide secondary metrics/columns. |
| 22–32rem | Add the trend badge, secondary metrics, more columns. |
| > 32rem | Full layout — footer context, all columns, full timestamps. |

For elements inside imported components, scope a `:global()` to your widget root: `:global(.my-widget .col-extra) { display: none; }`.

## Override hooks for the SDK renderer chrome

Every renderer component exposes a stable `dash-*` class on its root. Retheme by writing CSS against those classes in your app's global stylesheet (`src/app.css`) — no SDK fork, no `!important`. This is also how you make the cell card match shadcn exactly.

| Hook | Element | Use for |
|---|---|---|
| `.dash-grid` | `<DashboardCanvas>` root | Dashboard surface colour, empty state |
| `.dash-widget-card` | each widget cell | **Card chrome: bg, border, radius, shadow, hover, selection ring** |
| `.dash-widget-picker` | `<WidgetPicker>` root | Sidebar bg, width, border |
| `.dash-widget-picker-tile` | draggable palette tile | Tile hover / padding / icon colour |
| `.dash-config-panel` | `<ConfigPanel>` root | Form spacing, label colours |
| `.dash-error-widget` | `<ErrorWidget>` root | Error placeholder theming |

```css
/* src/app.css — make every widget cell a shadcn Card */
.dash-widget-card {
  background-color: hsl(var(--card));
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  box-shadow: var(--shadow-sm);
}
```

The full hook list (picker header/search/category/confirm bar, drop placeholder, resize gate) and the SDK's default classes live in `node_modules/@mssfoobar/dash-web-sdk/src/renderer/dash-renderer.css` — read it directly for exact element shapes, or `@import "@mssfoobar/dash-web-sdk/dash-renderer.css"` to get the layout rules plus empty override blocks to fill in.

## Development tip

To keep a work-in-progress widget out of the picker while you iterate, set `enabled: false` on its `meta` (the picker only shows `meta.enabled !== false`). Flip it back when ready — don't rely on a naming convention.
