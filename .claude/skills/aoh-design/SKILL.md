---
name: aoh-design
description: >
  Generate AOH-branded UI — production code or throwaway mocks — for AGIL Ops Hub,
  ST Engineering's operations-hub platform. Bundles AOH design tokens, Geist + Lucide
  assets, and a Svelte source kit consuming `@mssfoobar/ui` (sidebar, navbar, dashboard
  widgets, data tables, dialogs) with specimen cards. Use whenever the developer is
  mocking, designing, or implementing any AOH UI surface — dashboard widgets,
  list/detail pages, splash/error states, slides, or operator-console screens —
  including unbranded asks like "a mock", "a screen", or "a design" when the project
  is clearly AOH. **Also load during openspec workflows (`/opsx:propose`,
  `/opsx:apply`, `/openspec-propose`, `openspec new change`, or edits to
  `openspec/changes/<change>/proposal.md`, `design.md`, `specs/**/spec.md`,
  `tasks.md`) whenever the change touches a user-facing surface** — UI-relevant
  artifacts must stay consistent with `@mssfoobar/ui`, semantic tokens, page
  archetypes, and dark-mode + tenant theming, not invented locally.
allowed-tools: Read Write Edit Glob Grep Bash
---

# AOH Design

This skill scaffolds AOH-branded UI. It ships three layers:

1. **Tokens** — `assets/colors_and_type.css` is the contract. Semantic CSS variables (light + dark via `.dark` / `[data-theme="dark"]`) plus drop-in classes (`.h1`, `.metric-lg`, `.widget-title`, etc.). In production, prefer importing `@mssfoobar/ui/styles/app.css` once in `+layout.svelte` — that ships the canonical token set under `@layer aoh-theme`. The local CSS file is a fallback for throwaway HTML mocks where the npm package isn't available.
2. **Assets to lift** — `assets/fonts/` (Geist variable), `assets/agil-logo.png`, `assets/icons/` (custom AOH glyphs: `status-dot`, `phone-end`, `star-filled`). Lucide ships via `@lucide/svelte` in Svelte code, or `https://unpkg.com/lucide-static@latest/icons/<name>.svg` for static HTML mocks.
3. **Reference UI** — two side-by-side surfaces:
   - `assets/preview/` — runnable specimen HTML cards for colors, type, spacing, and isolated component looks. These are **mocks only** — they don't run `@mssfoobar/ui` (it's a Svelte build-time package). Use them for slides, asset cards, and quick visual references.
   - `assets/ui_kits/aoh/` — **Svelte source kit** that imports from `@mssfoobar/ui` directly (`AppShell.svelte`, `Sidebar.svelte`, `Navbar.svelte`, `Dashboard.svelte`, `MainListPage.svelte`, `IncidentDetails.svelte`, plus `widgets/*.svelte`). Lift any file straight into an AOH SvelteKit app — names, imports, and props are production-faithful.

For the full system spec — content voice, color theory, layering, spacing, shadows, icon usage, caveats — read `references/design-system.md`. It's long; read it when you need the *why*, not just the *what*.

## How to use it

- **Throwaway mocks / slides / prototypes:** copy from `assets/preview/` and `assets/colors_and_type.css` and emit a static HTML file the user can open in a browser. Don't try to render `@mssfoobar/ui` components inline — they're Svelte 5 + Tailwind v4 and need a build step.
- **Production code (or anything that will become production):** lift `.svelte` files out of `assets/ui_kits/aoh/` into the target SvelteKit app, then customize. The imports already resolve against the published package (`@mssfoobar/ui/button`, `@mssfoobar/ui/sidebar`, `@mssfoobar/ui/table`, etc.) plus `@lucide/svelte`. Don't reach for hand-rolled HTML or raw shadcn — `@mssfoobar/ui` is the AOH component contract.

## Always-use-`@mssfoobar/ui`-as-a-building-block

When you author a *new* UI component or screen (not in the kit), the first move is to compose from `@mssfoobar/ui` primitives, not to hand-roll one. The reason this rule earns its keep: every primitive in the package already encodes the AOH semantic tokens, focus-ring, dark-mode and tenant-theming hooks, accessibility wiring, and Tailwind v4 build pipeline. Skip that and you'll have to backfill all of it — usually with subtle drift.

A practical rubric for any new UI you produce:

1. **Need a button / badge / card / sheet / dialog / table / sidebar / navbar / select / tabs / tooltip / dropdown / avatar / checkbox / input / textarea / spinner?** It's already in `@mssfoobar/ui`. Import from the matching subpath (`@mssfoobar/ui/button`, `@mssfoobar/ui/dialog` style). The full subpath list is in `references/design-system.md` and verified against `node_modules/@mssfoobar/ui` when present.
2. **Need a "complex" component (a stat tile, an empty state, an alert banner, a page header)?** Build it by composing the primitives above. `MetricCard.svelte` in the kit is a worked example — it's just `Card + CardHeader + CardContent + a Lucide icon + AOH tokens`. Don't invent a new `<div class="rounded shadow border …">`.
3. **Need something the package genuinely doesn't ship?** Chart visualisations are the main legitimate gap today (Line / Bar / Donut / Radar). For those, write SVG that consumes the AOH chart palette tokens (`var(--bg-info-strong)`, `var(--bg-success-strong)`, etc.) so the visualisation theming follows the rest of the system. The kit's `widgets/LineChart.svelte`, `BarChart.svelte`, and `DonutChart.svelte` are the pattern to copy.
4. **Need a token that doesn't exist?** Add it under `@layer aoh-theme` in the app's `src/app.css`, both raw (`--brand: …`) and as a Tailwind `--color-brand` alias if it'll be used as a utility. Never paste raw hex into a component.

If you find yourself writing a `.svelte` file with no `from '@mssfoobar/ui/…'` imports and a non-trivial chunk of layout, stop — you're almost certainly drifting from the design system.

## If the user invokes this skill without specifics, ask:

1. **What surface** — dashboard widget, list page, details page, splash/error state, slide, etc.?
2. **Light or dark mode** — AOH supports both; light is default.
3. **Tenant context** — generic AOH or branded for a specific tenant (TTSH, NUH, SGH)?
4. **Fidelity** — production-faithful Svelte (must compile against `@mssfoobar/ui`) or throwaway HTML mock (specimen cards + tokens are enough)?

Then act as an expert designer who outputs Svelte code *or* static HTML mocks, depending on the answer.

## Key reminders (the stuff that gets forgotten most)

- **`@mssfoobar/ui` is the implementation contract.** Production Svelte code imports from it; do not hand-roll a `Button.svelte` lookalike. If you can't run against the registry, scaffold local stand-ins that match the *exact* `@mssfoobar/ui` API surface (see `apps/incident-web/src/lib/aoh/ui/` for the pattern) and document the swap in a README — never invent a divergent API.
- **Geist** for display & UI; **Inter** for body. Tabular numerals everywhere numeric.
- **Zinc** is the dominant neutral; **Slate** is the canvas background. The product is **flat** — no gradients, no glass, no blur (except chart area-fills).
- **Lucide** icons at 24×24, 2 px stroke, `currentColor`. In Svelte: `import { Icon } from '@lucide/svelte'`. Never emoji in product UI.
- **Tone** is operational and direct. Sentence case headings. Verb-first buttons. No marketing voice.
- **Radii**: 6 for buttons/inputs, 8 for cards/modals, 12 for widgets — all baked into `@mssfoobar/ui` defaults; don't override unless the design system changes.
- **Shadows**: 4 steps (`sm`/`md`/`lg`/`xl`). Default widget shadow = `md`; hover bumps to `lg` with a 1 px upward translate. Use the `--shadow-*` tokens from `@mssfoobar/ui`.
- **Focus ring**: 2 px solid `var(--ring)`, 2 px offset. Already wired into every focusable primitive.
- **Chart palette** (in order): Blue, Green, Amber, Red, Violet, Teal, Cyan — exposed as `var(--bg-info-strong)`, `var(--bg-success-strong)`, `var(--bg-warning-strong)`, `var(--bg-error-strong)`, etc.

If any of these conflict with what `references/design-system.md` says, the reference doc wins — it's the source of truth and gets updated when the Figma file does.

## Use during openspec proposals

When an openspec change touches a user-facing surface, this skill is part of the design contract — every UI-relevant openspec artifact should be consistent with the AOH system, not invented locally. Engage on `/opsx:propose`, `/opsx:apply`, or any direct edit of `openspec/changes/<change>/proposal.md`, `design.md`, `specs/**/spec.md`, or `tasks.md` where the change has a frontend impact (a new SvelteKit `apps/<*-web>` module, a new page, dashboard widget, dialog, or copy/tone-sensitive empty / error state).

**Sit this skill out** if the change is purely backend (Go service internals, schema-only migration, infra patch with no UI consumer) — there's nothing for the design system to contribute and inserting it just bloats the artifacts.

### What to add to each artifact

- **`proposal.md`** — one sentence under *What Changes* or *Impact* noting the UI surfaces conform to the AOH design system and consume `@mssfoobar/ui`. Do **not** paste tokens, hex codes, or component lists here; proposals stay at the "what / why" level.
- **`design.md`** — this is where the design-system decisions live. Include a *UI / Design System* section that nails down:
  1. **Page archetype(s)** — Main List, Create, Edit, Details, Dashboard, Splash / Error.
  2. **Components from `@mssfoobar/ui`** — name them by subpath (`@mssfoobar/ui/table`'s `DataTable`, `@mssfoobar/ui/sheet`'s `Sheet`, `@mssfoobar/ui/alert-dialog`, …). Don't propose hand-rolled or raw shadcn equivalents.
  3. **Composition for anything not directly exported** — if the screen needs a stat tile, empty state, page header, banner, etc., name the `@mssfoobar/ui` primitives it composes from. This is the design contract that prevents drift.
  4. **Theming surface** — confirm light + dark mode. Note any new semantic tokens that have to be added to `app.css` under `@layer aoh-theme`.
  5. **Widgets**, if any — sizing slot (Metric / Number / Data-table / Area / Bar / Line / Donut / Radar) and dashboard grid position.
  6. **Copy & tone** — list the page title, page description, primary action label, and any empty / error / permission-denied strings, written in AOH voice (sentence case, operational, verb-first buttons, no emoji).
  7. **Accessibility** — focus ring, keyboard nav, `prefers-reduced-motion`, contrast against semantic tokens. Most of this is handled by `@mssfoobar/ui` primitives; only call out deviations.
  8. **Open design questions** — anything the Figma source doesn't answer yet (custom icon, new chart series colour, etc.). Surface these instead of inventing.
- **`specs/<capability>/spec.md`** — requirements that are *visible to a tester* should reference the design system. Examples:
  - "The list page SHALL follow the Main List archetype using `DataTable` from `@mssfoobar/ui/table`, with header, search bar, primary action, filter bar, and pagination."
  - "Empty state SHALL show the AOH empty-state pattern (32 px Lucide glyph + one-sentence supporting line, operational tone), composed from `@mssfoobar/ui/card`."
  - "Destructive actions SHALL use `@mssfoobar/ui/button` with `variant=\"destructive\"` (no emoji, no custom red)."

  Don't write CSS or list hex codes in specs — describe behaviour against the system, not the implementation.
- **`tasks.md`** — implementation tasks should reference `@mssfoobar/ui` imports explicitly so reviewers can verify the design contract held: e.g. "Import `Sidebar` from `@mssfoobar/ui/sidebar` and wire `groups` to the route map", "Import `DataTable` from `@mssfoobar/ui/table` and bind to the query".

### Design checklist before marking a UI proposal apply-ready

Use this as an **authoring** pass on `design.md` and `specs/` while writing. The
**mechanical** post-write version is `lint` §4 in the `aoh-spec-driven` schema
(`openspec/schemas/aoh-spec-driven/templates/lint.md`) — that's what gates
`/opsx:apply`. Keep the two in sync: if you add or change a check here, mirror
it in §4 (and vice versa).

- [ ] Page archetype(s) named.
- [ ] Every component named comes from `@mssfoobar/ui` (by subpath), not raw shadcn or hand-rolled.
- [ ] New composites are explicitly described as combinations of `@mssfoobar/ui` primitives, not new bespoke widgets.
- [ ] Semantic tokens cover light + dark; no raw hex outside `@layer aoh-theme`.
- [ ] Copy is operational (sentence case, verb-first buttons, no emoji, no marketing voice).
- [ ] Accessibility (focus ring, keyboard, reduced-motion) called out only where it deviates from the package default.
- [ ] Tenant theming / branding surface considered.
- [ ] Open design questions are listed, not silently decided.

If anything is missing, flag it in `design.md` under *Open Questions* rather than invent a value — the design system is the source of truth, and silently improvising drifts the product away from it.
