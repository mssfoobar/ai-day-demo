# AOH Design System

The **AGIL Ops Hub (AOH)** design system. AOH is an enterprise **operations hub** platform built by ST Engineering for mission‑critical command‑centre style workflows — dashboards, incident management, distribution lists, channel settings, system alerts, and operator console screens. Component implementation lives in **`@mssfoobar/ui`** — the internal npm package every AOH frontend installs. It mirrors the shadcn primitive surface (Radix + Tailwind underneath) but is the canonical source of truth: extend or theme through `@mssfoobar/ui`, not by lifting raw shadcn.

## What's in this system

- **Foundations** — Radix color primitives (30 scales) + a semantic-color layer; Inter & Geist type stacks; an 8‑px spacing grid; a 4-step shadow elevation system.
- **Components** — exported from `@mssfoobar/ui` (Svelte 5 + Tailwind v4, verified against the installed package). Each ships under its own subpath: `@mssfoobar/ui/alert-dialog`, `/avatar`, `/badge`, `/bottom-sheet`, `/breadcrumb`, `/button`, `/calendar`, `/card`, `/checkbox`, `/combobox`, `/command`, `/date-picker`, `/date-range-picker`, `/drawer`, `/dropdown-menu`, `/input`, `/label`, `/menubar`, `/navbar`, `/pagination`, `/popover`, `/range-calendar`, `/scroll-area`, `/select`, `/separator`, `/sheet`, `/sidebar`, `/skeleton`, `/slider`, `/spinner`, `/table` (TanStack-backed `DataTable` with checkbox / actions / pagination helpers), `/tabs`, `/textarea`, `/toast`, `/toggle`, `/toggle-group`, `/tooltip`. Plus the `cn` utility from `@mssfoobar/ui/utils`. Chart primitives are **not** exported — write SVG against the AOH chart-palette tokens (see `assets/ui_kits/aoh/widgets/`).

  > 🛑 **Common shadcn-svelte primitives NOT shipped in `@mssfoobar/ui`** — do not reference these when proposing designs or specs; they will not resolve at import time:
  >
  > - `Alert` / `AlertTitle` / `AlertDescription` (i.e. the plain inline alert banner). **Not shipped.** `alert-dialog` exists but is a *modal* — different component. For an inline destructive banner, compose from `Card` with `class="border-destructive"` + `CardTitle class="text-destructive"`; for a confirmation modal, use `AlertDialog` with the action button's `variant="destructive"`.
  > - `Form` / `FormField` / `FormItem` / `FormLabel` / `FormControl` / `FormMessage` (Formsnap wrapper). **Not shipped.** Compose forms directly from `Label` + `Input` / `Textarea` / `Select` and a sibling `<p class="text-destructive text-sm">` for field errors. Use Svelte 5 runes for state.
  > - `HoverCard`, `ContextMenu`, `Resizable`, `Accordion`, `Collapsible`, `Carousel`, `Sonner` (use `/toast` instead — re-exports svelte-sonner). **Not shipped.**
  >
  > Verify against the installed version before authoring: `cat apps/<app>/node_modules/@mssfoobar/ui/package.json | jq .exports`. If a primitive you need is missing, either compose from primitives that *are* there or open an upstream request — do not silently hand-roll raw HTML + Tailwind, which bypasses theme tokens and dark mode.
- **Patterns** — Layering & background system, form structure, layout structure (Main / Create / Edit / Details page archetypes), splash/error states.
- **Widgets** — dashboard widget kit (Area / Bar / Line / Donut / Radar / Metric / Data‑table / Number‑indicator), with documented sizing grid.
- **Icons** — full Lucide set (~1,200 glyphs) plus a small set of AOH custom icons (status‑dot, phone‑end, filled star).

## Sources

This system was reconstructed from a Figma file the user supplied:

- **Figma file:** the AOH UI component library — 66 pages, 336 frames
  - Foundations: `/Thumbnail`, `/Colors`, `/Typography`, `/Spacing-Grid`, `/Icons`, `/Shadows-Blurs`
  - Components: 50+ component pages, one per `@mssfoobar/ui` primitive
  - Patterns: `/Design-Patterns`, `/Layering-Background`, `/Form-Structure`, `/Common-Components`, `/Layout-Structure`, `/Splash-Page-States`, `/Widgets`

The pseudocode JSX in the .fig was used as design source of truth; values match Figma node properties (colors, spacing, type, instance overrides). Treat `@mssfoobar/ui` as the runtime implementation of what the Figma file describes.

## Index

```
aoh-design/
├── SKILL.md                       entry point — read first
├── references/
│   └── design-system.md           you are here — full spec
└── assets/
    ├── colors_and_type.css        :root vars for color + type tokens, + semantic classes
    │                              (fallback for HTML mocks; in production prefer
    │                              `import '@mssfoobar/ui/styles/app.css'`)
    ├── fonts/                     Geist variable (Inter & Geist Mono loaded from web fonts)
    ├── agil-logo.png              AGIL brand mark
    ├── icons/                     custom AOH glyphs (status-dot, phone-end, star-filled)
    ├── preview/                   specimen HTML cards — colors, type, spacing, components
    │                              (mocks-only; do not run @mssfoobar/ui)
    └── ui_kits/
        └── aoh/                   Svelte source kit — lifts into a SvelteKit app
            ├── AppShell.svelte    SidebarProvider + Sidebar + main slot
            ├── Sidebar.svelte     uses @mssfoobar/ui/sidebar
            ├── Navbar.svelte      uses @mssfoobar/ui/navbar
            ├── Dashboard.svelte   Dashboard archetype
            ├── MainListPage.svelte  Main List archetype (uses @mssfoobar/ui/table)
            ├── IncidentDetails.svelte  Details archetype
            ├── widgets/
            │   ├── MetricCard.svelte
            │   ├── LineChart.svelte
            │   ├── BarChart.svelte
            │   └── DonutChart.svelte
            ├── index.ts           re-exports
            └── README.md          lift-into-app instructions
```

CONTENT FUNDAMENTALS, VISUAL FOUNDATIONS, and ICONOGRAPHY follow below.

---

## CONTENT FUNDAMENTALS

AOH copy is **operational, calm and direct**. It's the voice of a console an on‑shift operator reads at 3 a.m. Avoid marketing flourish; avoid playful hedging.

**Tone.** Neutral and matter-of-fact. Confident without being chipper. Reads like an internal tool, not a SaaS landing page. The default register is "report what is true, then offer the next action."

**Voice.** Mostly _you_-implicit (imperative). "Configure distribution lists for sending message notifications." — not "We help you configure your lists." Avoid first person plural ("we / our"). Avoid second-person flattery ("you've got this").

**Casing.** Sentence case for headings, page titles, table headers and buttons (`Distribution Lists`, `Create new`, `Last modified`). Reserve Title Case only for product/brand nouns (AGIL Ops Hub, AOH). Never ALLCAPS body copy; small uppercase reserved for table column-headers when wanted (with `0.05em` letter-spacing).

**Punctuation & length.**
- Page titles: 1–3 words ("Main Page", "Distribution Lists", "Incident Management Dashboard").
- Page descriptions: one sentence, ends in a period. _"Configure distribution lists for sending message notifications."_
- Empty/error supporting text: one or two short sentences. _"The site is currently down for maintenance."_ _"We apologize for any inconveniences caused. We should be back shortly."_
- Permission-denied: _"Access to this page is restricted. For assistance with access, please contact your administrator."_

**Numbers, dates, IDs.** Numerals everywhere (no spelled-out integers). Date format `MM/DD/YYYY HH:mm:ss` in tables. Counts are plain (`7`, `0`). Selection summaries match: `"0 of 3 row(s) selected."`.

**Buttons & actions.** Verb-first, short. `Create`, `Delete`, `Save changes`, `Cancel`, `Apply filters`, `Reset`, `Export CSV`, `Acknowledge`, `Escalate`. Destructive actions use the verb plainly (`Delete`) — the colour does the work, not the copy.

**Emoji.** **No.** Body copy and UI never use emoji. Section headers _inside the design docs_ in the Figma file occasionally use a single leading emoji (📊 Widget Component Specifications, 🎨 Color System Foundation) as a category marker — that's a docs convention, not a product convention. Product UI is emoji-free.

**Examples to imitate**
- Page title + supporting: _"Main Page" → "The typical layout of the Main Page includes the following components: Header & Description, Search Bar & Action Buttons, Table & Pagination."_
- Notification preference: _"Configure distribution lists for sending message notifications."_
- Maintenance state: _"The site is currently down for maintenance. We apologize for any inconveniences caused. We should be back shortly."_

**Avoid**
- Exclamation marks in product copy (allowed sparingly in splash states: _"Stay tuned for exciting updates!"_ but never in operational screens).
- Latin filler ("lorem ipsum") — placeholder text in the kit is realistic ops content.
- Cute synonyms ("oops", "uh-oh"). When something fails, name it.

---

## VISUAL FOUNDATIONS

### Color

A two-tier system: **Radix primitives** at the bottom, **semantic tokens** mapped on top.

- **Primitives.** All 30 Radix Color scales are available (Gray, Zinc, Neutral, Stone, Red, Orange, Amber, Yellow, Lime, Green, Emerald, Teal, Cyan, Sky, Blue, Indigo, Violet, Purple, Fuchsia, Pink, Rose — light & dark). The dominant neutrals are **Zinc** (`#18181B`, `#71717A`, `#E4E4E7`, `#F4F4F5`) and **Slate** for backgrounds (`#F1F5F9`, `#F8FAFC`).
- **Semantic tokens** (the `@mssfoobar/ui` contract): `--background`, `--foreground`, `--card`, `--popover`, `--primary`, `--primary-foreground`, `--secondary`, `--muted`, `--muted-foreground`, `--accent`, `--destructive`, `--border`, `--input`, `--ring`. Light + dark variants.
- **Brand / chart palette** (from widget guidelines):
  - Primary Blue `#3B82F6`
  - Success Green `#10B981`
  - Warning Amber `#F59E0B`
  - Danger Red `#EF4444`
  - Info Cyan `#06B6D4`
  - Purple `#8B5CF6`
  - Teal `#14B8A6`
- **Destructive** uses Red‑600 `#DC2626` for borders/buttons and Red‑500 `#EF4444` for chart series.
- **Focus ring** is Sky‑600 `#0284C7`, 2 px solid, 2 px offset.

### Typography

- **Display & UI body:** Geist (Regular 400, Medium 500, SemiBold 600, Bold 700, ExtraBold 800).
- **Body / fallback:** Inter (Regular through Black) — used wherever Geist isn't loaded.
- **Mono / numerics in dense tables:** Geist Mono is implied; the Figma file uses Geist with `tabular-nums`.
- **Type scale** (matches Tailwind defaults; `@mssfoobar/ui` uses the same tokens):

| Token       | Size | Line ht | Role                          |
|-------------|------|---------|-------------------------------|
| text-xs     | 12   | 16      | Helper, microcopy             |
| text-sm     | 14   | 20      | Body small, table cells       |
| text-base   | 16   | 24      | Paragraph                     |
| text-lg     | 18   | 28/30   | Large body, widget title (600)|
| text-xl     | 20   | 32      | H4                            |
| text-2xl    | 24   | 38      | H3, medium metric (600)       |
| text-3xl    | 30   | 42      | H2                            |
| text-4xl    | 36   | 40      | H1 / page title section       |
| text-5xl    | 48   | 72      | Display (700)                 |

### Spacing & grid

- **Base unit 8 px.** Common values: 8, 12, 16, 24, 32, 48, 64.
- **Widget gaps:** 24 px standard, 16 px mobile.
- **Content padding:** 16 mobile / 20 tablet / 24 desktop / 32 desktop‑XL.
- **Sidebar:** 256 px expanded, 56 px collapsed.
- **Page padding:** 80 px on the main canvas; widget rails sit at 16/18 px inset.

### Backgrounds & layering

- Surface is flat — **no gradients, no glass / blur, no textures**. (One exception: chart fills below an area‑chart line use a 80 → 0 % gradient of the series colour.)
- Canvas: `#FAFAFA` / `slate‑50` (`#F1F5F9`) for dashboard rails; **Widget surface: `#FFFFFF`**; muted backgrounds: `#F1F5F9`.
- Dark mode canvas: `#03070F` (near-black slate-950) on splash pages; `#18181B` (zinc-900) for surfaces.

### Animation

- **Restrained.** Hover lift is a `translateY(-1px)` with the shadow stepping from "default" to "hover".
- **Tooltip / dropdown** appearance uses default Radix transitions (fast fade + small scale).
- **Loading** uses skeleton blocks (gray placeholders matching final layout) and 2 px‑stroke primary‑blue spinners.
- **Respect `prefers-reduced-motion`.** No bounce, no parallax, no decorative auto-animation.

### Interactive states

- **Hover:** widgets gain elevation (shadow `sm` → `md`) and a 1 px upward translate. Buttons darken background by ~10 %.
- **Focus:** 2 px solid `#3B82F6` outline with 2 px offset. Keyboard accessibility is required.
- **Press / active:** background steps one shade darker; no scale-down.
- **Disabled:** 50 % opacity; cursor `not-allowed`.

### Borders, corners, shadows

- **Border:** 1 px solid; default colour `#E4E4E7` (zinc‑200) or `#E2E8F0` (slate‑200).
- **Radius scale:** 4 (`sm`), 6 (default — buttons, inputs), 8 (cards, modals), 12 (widget surfaces), 16 (large containers).
- **Shadow elevation** (4 steps, matching the Figma `/Shadows-Blurs` page):

| Token  | Box-shadow                                                                 | Use                          |
|--------|----------------------------------------------------------------------------|------------------------------|
| sm     | `0 1px 2px 0 rgba(16,24,40,.05)`                                           | Inputs, low cards            |
| md     | `0 1px 3px 0 rgba(0,0,0,.1), 0 1px 2px -1px rgba(0,0,0,.1)`                | Default widgets              |
| lg     | `0 4px 6px -1px rgba(0,0,0,.1), 0 2px 4px -2px rgba(0,0,0,.1)`             | Hover, popovers              |
| xl     | `0 10px 15px -3px rgba(0,0,0,.1), 0 4px 6px -2px rgba(0,0,0,.05)`          | Modals, tooltips, dropdowns  |

### Cards

A standard card = `bg-card` + 1 px border + radius 8 + shadow `sm` + 16 px padding. A widget = `bg-card` + radius 12 + shadow `md` + 24 px padding. No coloured left-border accents.

### Transparency & blur

- No backdrop‑filter blur in product UI.
- Overlays on modal/sheet are flat `rgba(0,0,0,.5)`.
- Inner shadows are not used.

### Imagery

- The kit ships almost no photography. Splash pages use a single line‑art / 3D access‑denied illustration; otherwise content is structural. Imagery, where used in dashboards, is **map tiles** (sober, low-saturation, no overlay tinting).

---

## ICONOGRAPHY

The icon system is **[Lucide](https://lucide.dev)**, 24 × 24, 2 px stroke, rounded line caps & joins, currentColor.

- Verified by counting instances: the Figma file references **1,210 instances** of the Lucide icon component, and the icon-set page name literally lists Lucide glyph names (`chevron-down`, `calendar`, `tag`, `bold`, `gallery-vertical`, `terminal-square`, `log-out`, `chevrons-left`, `chevrons-right`, `star`, `shovel`, etc.).
- A **small custom set** sits alongside Lucide for AOH-specific glyphs: `status-dot` (solid 10×10 circle inside a 24×24 box), `phone-end` (telephony hangup), `star_filled` (filled variant of Lucide's outline star). These live in `assets/icons/` (relative to the skill root).
- **Loading**: the Lucide `loader-circle` (a.k.a. `loader-animate`) rotates at 2 px stroke, primary‑blue, used in spinners and the inline busy state on buttons.

**Usage**
- Default colour: `currentColor` so icons inherit text colour in any context.
- Size scale: 14 (xs in tags / badges), 16 (in inputs), 20 (in 36‑px buttons), 24 (default standalone), 32 (empty states), 48+ (splash).
- **Avoid emoji entirely** in product UI. The handful of emoji that appear in the Figma file are inside design-doc category headers and never ship to users.
- **Unicode chevrons / arrows** are not used as nav glyphs — always Lucide.

CDN reference (no codebase install required): `https://unpkg.com/lucide-static@latest/icons/<name>.svg`. The cards in this repo link Lucide via the static SVG CDN; the AOH UI kit uses `lucide-react` from unpkg.

---

## How to use

- **Quick visuals / mocks:** lift HTML out of `assets/preview/` and use `assets/colors_and_type.css` for tokens. These specimen cards are intentionally framework-free so they open straight in a browser. Don't try to embed `@mssfoobar/ui` here — it's a Svelte 5 + Tailwind v4 package and needs a build step.
- **Production code:** lift `.svelte` files out of `assets/ui_kits/aoh/` into a SvelteKit app. The imports already resolve against `@mssfoobar/ui` and `@lucide/svelte`. In the host app, install the packages and `import '@mssfoobar/ui/styles/app.css'` once in `+layout.svelte` — that ships the canonical token set under `@layer aoh-theme`, and Tailwind v4 picks up the package's components automatically via its `@source` declarations.

## Composing new components

When the kit doesn't ship the exact thing you need, compose it from `@mssfoobar/ui` primitives instead of hand-rolling. The package already encodes the semantic tokens, focus ring, dark mode, tenant theming hooks, accessibility, and Tailwind build wiring — skipping it means rebuilding all of that.

- **Stat tile, page header, banner, empty state** → `Card` + `Badge` + `Button` + a Lucide icon. `widgets/MetricCard.svelte` shows the pattern.
- **Confirmation flow** → `AlertDialog` from `@mssfoobar/ui/alert-dialog` with `destructive` variant on the action button.
- **Side panel / drawer** → `Sheet` from `@mssfoobar/ui/sheet` or `Drawer` from `@mssfoobar/ui/drawer`.
- **Tabular data** → `DataTable` from `@mssfoobar/ui/table` (TanStack-backed; helpers `createCheckboxColumn` and `createActionsColumn` cover the common cases).
- **Charts** → SVG, AOH chart palette tokens (`var(--bg-info-strong)`, `var(--bg-success-strong)`, etc.). `widgets/LineChart.svelte`, `BarChart.svelte`, `DonutChart.svelte` are the templates.

If you find yourself writing a `.svelte` file with no `from '@mssfoobar/ui/…'` imports and a non-trivial chunk of layout, stop and reach for a primitive.

## Caveats noted by the agent

- **Geist webfonts** are loaded from the Vercel CDN via `@font-face` since the user did not supply the .woff2 binaries. Inter is loaded from Google Fonts.
- The Figma file's pseudocode resolves most tokens but not every Radix variable alias — semantic dark-mode values for less-common tokens (e.g. `--chart-5`, `--sidebar-ring`) were inferred from the Tailwind/Radix defaults `@mssfoobar/ui` ships with.
- The `assets/preview/` specimen cards are intentionally HTML-only — they're for slides, asset cards, and quick visual references, not for production composition. The Svelte source kit in `assets/ui_kits/aoh/` is the production-faithful path.
