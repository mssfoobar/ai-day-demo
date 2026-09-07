# AOH UI kit — Svelte sources

Worked examples of AOH page archetypes and shell components, written in Svelte 5
against **`@mssfoobar/ui`** (the canonical AOH component package — shadcn-svelte
primitives + Bits UI + Tailwind v4). Lift any file into an AOH SvelteKit app and
modify in place; the imports already resolve against the published package.

## Files

| File                                    | Archetype / role                                   |
| --------------------------------------- | -------------------------------------------------- |
| `AppShell.svelte`                       | `SidebarProvider` + `Sidebar` + main-content slot  |
| `Sidebar.svelte`                        | AOH-flavoured sidebar (Platform / Tenants groups)  |
| `Navbar.svelte`                         | Breadcrumb header with search + global actions     |
| `Dashboard.svelte`                      | Dashboard archetype (KPI row + charts + table)     |
| `MainListPage.svelte`                   | Main List archetype (header + DataTable + dialogs) |
| `IncidentDetails.svelte`                | Details archetype (header + impact + timeline)     |
| `widgets/MetricCard.svelte`             | KPI tile bound to `@mssfoobar/ui` `Card`           |
| `widgets/LineChart.svelte`              | SVG line chart, AOH chart palette                  |
| `widgets/BarChart.svelte`               | SVG bar chart, AOH chart palette                   |
| `widgets/DonutChart.svelte`             | SVG donut chart, AOH chart palette                 |

## Prerequisites in the host app

The kit relies on the package being installed and its stylesheet loaded. See
`apps/<*-web>` in any AOH frontend, or use the `aoh-web-init` skill to scaffold
one. Minimum wire-up:

```ini
# .npmrc
@mssfoobar:registry=https://npm.pkg.github.com
```

```sh
pnpm add @mssfoobar/ui @lucide/svelte
```

```svelte
<!-- src/routes/+layout.svelte -->
<script lang="ts">
  import '@mssfoobar/ui/styles/app.css';
  let { children } = $props();
</script>

{@render children()}
```

Tailwind v4 picks up the package's compiled components automatically via its
`@source` declarations — no extra Tailwind config needed.

## Lifting a screen

```svelte
<!-- src/routes/(private)/dashboard/+page.svelte -->
<script lang="ts">
  import { goto } from '$app/navigation';
  import AppShell from '$lib/aoh/screens/AppShell.svelte';
  import Navbar from '$lib/aoh/screens/Navbar.svelte';
  import Dashboard from '$lib/aoh/screens/Dashboard.svelte';
</script>

<AppShell active="dashboard">
  <Navbar breadcrumbs={[{ label: 'Dashboards' }, { label: 'Incident Management' }]} />
  <Dashboard onOpenIncident={(inc) => goto(`/incidents/${inc.id}`)} />
</AppShell>
```

`AppShell` wraps the sidebar in `SidebarProvider` and renders `Sidebar.svelte`
inside it. To customize the navigation, edit `Sidebar.svelte` directly or pass
a `groups={…}` prop to `AppShell`. `Sidebar.svelte` is also exported standalone,
but **you must render it inside a `SidebarProvider`** — the Provider has to
wrap both the sidebar and the main content for flex layout to work.

## When extending the kit

When you build a new screen or widget, prefer composing from `@mssfoobar/ui`
primitives over hand-rolling new HTML — that's the whole point of the kit.
Reach for the smallest primitive that fits (`Card` + `Badge` + `Button` + a
Lucide icon is almost always the answer), and only introduce custom SVG / CSS
when the package genuinely doesn't ship the primitive you need (charts being
the only common case today). New tokens go in
`apps/<*-web>/src/app.css` under `@layer aoh-theme`; don't restyle from raw
hex.
