## Context

`fleet-dispatch-console` is an empty AOH monorepo: Turborepo + pnpm workspaces are
configured and `apps/*` / `packages/*` are globbed, but both directories are empty and
`openspec/specs/` has no capabilities. This change lands the repo's first app.

The consumer is a workshop facilitator. The baseline must reach a rendered screen with
the fewest moving parts an attendee can get wrong: one `pnpm install`, one `pnpm dev`,
no containers, no login, no seeded realm. The workshop is explicitly unauthenticated —
its later exercises add a backend service and dispatch behaviour, not identity.

### Why authentication is removed rather than deferred

This design originally kept the full `aoh-web-init` scaffold and placed the console in
its existing `(public)` route group, expecting the auth machinery to sit dormant. That
does not work, and the reason matters enough to record:

The scaffold's `src/hooks.server.ts` opened with:

```ts
const oidc_config: Configuration = await discovery(new URL(envPrivate.IAM_URL!), …)
```

That is a **top-level await**. It runs when the server module is loaded — before any
routing — so with no identity provider reachable it throws `ECONNREFUSED` and every
route returns HTTP 500, `(public)` routes included. Verified directly:
`GET /units` → `500 {"message":"Internal Error"}` with no containers running.

There is therefore no "auth present but inert" state available. Auth is either running
against a real Keycloak, or removed. The workshop specifies no authentication, so it is
removed.

Platform constraints that still bound the design:

- `@mssfoobar/ui` is the component contract (`aoh-conventions/web.md`, `aoh-design`).
  Visual primitives come from its subpaths; hand-rolled equivalents bypass the semantic
  tokens, dark mode, focus ring, and tenant theming.
- Design tokens ship inside `@mssfoobar/ui/styles/app.css`, which the app's
  `src/app.css` `@import`s as its first line. Apps **must not redeclare** those tokens.
- `pnpm-workspace.yaml` sets `minimumReleaseAge: 10080` (7 days), `trustPolicy:
  no-downgrade`, and `blockExoticSubdeps: true`; a dependency published in the last week
  will not resolve.
- `@mssfoobar/ui` is published to GitHub Packages, so `~/.npmrc` needs a
  `//npm.pkg.github.com/:_authToken=…` entry. This is the one prerequisite the
  container-free design cannot remove.

## Goals / Non-Goals

**Goals:**

- Every attendee reaches an identical rendered console from a clean clone with
  `pnpm install && pnpm dev` and nothing else running.
- The console is a faithful AOH surface — `@mssfoobar/ui` primitives, semantic tokens,
  light and dark parity — so what attendees read resembles production.
- The unit roster sits behind a seam narrow enough that the follow-up change
  (`dispatch-units-service`) can serve it from a Go + PostgreSQL backend without editing
  the console page.
- No authentication code remains to confuse an attendee or fail at boot.

**Non-Goals:**

- Authentication, authorization, sessions, tenancy, or any IAMS/SDS integration.
- Any command, dispatch, or status-mutation action.
- A backend service, an API endpoint, a database, a map, real-time updates, or
  persistence. The backend arrives in `dispatch-units-service`.
- Responsive/mobile layout beyond not breaking. The two-pane layout targets 1280px and
  wider and may stack below that.
- Sidebar / header chrome — see D3.

## Runtime dependencies

None.

| Dependency | Role | Owned by |
|---|---|---|
| *(none)* | No container, database, broker, identity provider, or peer AOH service is required to run or build the baseline. | n/a — this change introduces no runtime dependency |

With the auth layer removed there is no `iams-keycloak`, no `iams-aas`, no `sds-server`,
no `valkey`, and no Traefik. `pnpm dev` is the whole run story, so there is no
`aoh-compose` work and no reproducibility gate in `tasks.md`.

## API surface

None. This change adds no endpoint and calls no backend.

| Method | Path | Auth posture | Description | Fronting gateway route |
|---|---|---|---|---|
| GET | /units | Unauthenticated | The baseline console page. A SvelteKit page route, not an API endpoint. | n/a |

The roster is imported at build time by the page module. There is no fetch, so no
response envelope, no pagination, and no `occ_lock` apply here. The scaffold's `/livez`
and `/readyz` health routes are retained unchanged and are not part of this change's
surface.

## UI / Design System

Mockup: `openspec/changes/baseline-dispatch-console/design/units-console-mock.html`
(clickable — selection, keyboard activation, and the light/dark toggle all work).

### 1. Page archetypes

A **master-detail composition of two archetypes on one route**: the left pane is a
narrowed **Main List**, the right pane a **Details** pane. Deliberately *not* the
Dashboard archetype — see D4.

### 2. Components from `@mssfoobar/ui`

Verified against the installed `@mssfoobar/ui@1.1.0` `exports` field.

| Surface element | Primitive | Subpath |
|---|---|---|
| Page header (title + description) | plain heading + paragraph on semantic tokens | n/a |
| Units pane shell | `Card`, `CardHeader`, `CardTitle`, `CardContent` | `@mssfoobar/ui/card` |
| Each selectable unit row | `Button` with `variant="ghost"` | `@mssfoobar/ui/button` |
| Row dividers | `Separator` | `@mssfoobar/ui/separator` |
| Status indicator | `Badge` with `variant="soft"` | `@mssfoobar/ui/badge` |
| Detail pane shell | `Card`, `CardHeader`, `CardTitle`, `CardContent` | `@mssfoobar/ui/card` |
| Icons | `radio`, `navigation`, `pause-circle`, `mouse-pointer-click` | `@lucide/svelte/icons/<name>` |

`Badge` splits shape from palette (`aoh-conventions/web.md`): `variant` is the shape,
`color` the semantic palette. Writing `variant="success"` silently renders a default
solid badge.

| Status | `color` | Rationale |
|---|---|---|
| `Available` | `success` | Ready to be tasked. |
| `En route` | `info` | Active, in transit — informational, not a warning. |
| `Idle` | `default` | Neutral; deliberately not `warning`, which would imply a fault. |

### 3. Composition for anything not directly exported

- **Selectable unit row** — not an exported primitive. A full-width
  `Button variant="ghost"` containing a two-line stack (call sign, then identifier in
  `--muted-foreground`) plus a trailing `Badge`. Using `Button` rather than a `div` with
  `role="option"` buys the keyboard activation and focus ring from the package.
- **Selected-row treatment** — `aria-current="true"` for assistive tech, **plus** a
  visible change: the row carries `bg-accent text-accent-foreground` and its status icon
  moves from `text-muted-foreground` to `text-foreground`. Both are package-mapped
  semantic tokens, so selection reads in either theme without a bespoke highlight colour.
  The `aria-current` attribute alone is *not* sufficient — `variant="ghost"` styles no
  `aria-current` state, so a row set only via the attribute is pixel-identical to an
  unselected one. `units-console.spec.ts` asserts the rendered difference, not just the
  attribute.
- **No scroll region.** An earlier draft wrapped the list in `ScrollArea`. It was removed:
  `ScrollArea`'s viewport is `h-full`, so with a `max-h-*` and no definite height the
  viewport never overflows, no scrollbar appears, and the root's `overflow-hidden` simply
  clips. With a fixed 4–5 row roster nothing overflows anyway, and the approved mockup has
  no scroll region. Reintroduce it — with a definite height — when the roster becomes
  service-backed and unbounded.
- **Detail field rows** — a description list inside `CardContent`; labels in
  `--muted-foreground` at `--fs-sm`, values in `--foreground`. Identifiers render in
  `font-mono` with tabular numerals.
- **Empty state** — the AOH empty-state pattern: a 32px Lucide `mouse-pointer-click`
  glyph above one supporting sentence, composed from `Card` + `CardContent`.

### 4. Theming surface

Light and dark are both supported and both covered by the mockup's toggle. The root
`+layout.svelte` wraps the app in `ThemeProvider mode="dark"`, so **dark is the default
the workshop sees** — the light path must still be verified, not assumed.

**No new semantic tokens are introduced, and `app.css` is not edited.** Every colour
resolves to a token already shipped by `@mssfoobar/ui/styles/app.css`. No raw hex
appears in any `.svelte` file.

### 5. Widgets

None. This change authors no dashboard widget and occupies no dashboard grid slot.

### 6. Copy and tone

Sentence case, operational, no marketing voice, no emoji.

| Slot | String |
|---|---|
| Page title | `Dispatch console` |
| Page description | `Field units and their current status.` |
| Units pane title | `Units` |
| Detail pane title (no selection) | `Unit detail` |
| Detail pane title (selected) | The unit's call sign |
| Empty-state line | `Select a unit to see its details.` |
| Field labels | `Call sign`, `Unit ID`, `Status` |
| Primary action | None — the baseline has no action. |

### 7. Accessibility

Package defaults handle most of this; only deviations and deliberate choices are noted:

- The units list is an unordered list of `Button`s, natively tabbable, with `Enter` /
  `Space` activating a row. The focus ring comes from the installed `Button`
  (`focus-visible:border-ring` + a 3px `ring-ring/50`) and is not overridden. Note this
  is the shipped primitive's ring, which differs from the 2px-at-2px-offset ring the
  design-system doc describes; the package is the authority for what actually renders.
- Selection is announced via `aria-current="true"` on the selected row's button.
- Status is conveyed by the `Badge`'s **text label**, never by colour alone.
- The detail pane is an `aria-live="polite"` region, so the swap is announced without
  moving focus.
- No animation is introduced, so `prefers-reduced-motion` needs no handling.

### 8. Open design questions

Listed under Open Questions below rather than decided silently.

## Decisions

**D1 — Strip the auth layer from the `aoh-web-init` scaffold rather than re-scaffolding
a plain SvelteKit app.**
The scaffold contributes real value the workshop keeps: `@mssfoobar/ui` wiring, the
`app.css` token import, `ThemeProvider`, the OpenTelemetry bootstrap, the AOH folder
layout, the app-level `AGENTS.md`, and prettier/eslint/vitest/playwright configuration.
Only the auth layer is removed. Alternative: `sv create` a minimal app — rejected because
it discards all of the above and would have to re-derive the design-system wiring by
hand. *Facilitator-confirmed.*

**Removed:** OIDC discovery in `hooks.server.ts`, `(public)/aoh/api/auth/*` (login,
callback, refresh, logout, context), the `(private)` group and its layout,
`AuthProvider`, `auth.ts` and its SDS client, the `iams` module and its API routes, the
gateway proxy, `Headerbar` (its purpose is the signed-in user and logout), `Sidebar` and
`nav.ts` (see D3), and the `authResult` / `clients` fields on `App.Locals`.

**Kept:** `ThemeProvider`, `app.css`, `instrumentation.server.ts`, `+error.svelte` and
the `App.Error` contract, `/livez` and `/readyz`, and the logger.

**D2 — The console is served at `/units`, and `/` redirects to it.**
With the `(private)` / `(public)` groups gone, the route is a plain `src/routes/units/`.
The scaffold's root `+layout.server.ts` redirected `/` to the login flow; that
destination no longer exists, so it now redirects to `/units`. Alternative: mount the
console at `/` — rejected because a named route keeps the redirect explicit and leaves
`/` free to become a landing page later.

**D3 — No sidebar or header chrome in this change.**
`Headerbar` exists to render the signed-in user and a logout control, so it cannot
survive the removal of auth. `Sidebar` + `nav.ts` could survive — `filterByRoles` with
`roles=[]` simply shows ungated entries — but a one-page app gains a nav rail whose only
entry is the page you are on, and `Sidebar`'s footer would render a fabricated "User".
Both are removed; the console renders bare, exactly as the reviewed mockup shows.
Re-introducing a sidebar when the workshop has several pages is a deliberate follow-up,
not something to carry dead.

**D4 — Compose the list from `Button` + `Card` rather than using `DataTable`.**
`DataTable` from `@mssfoobar/ui/table` is the design system's answer for *tabular data*
and was the first candidate. Its selection model is multi-select checkboxes, whereas
this list is single-select and drives a sibling pane; and it brings search, sorting, and
pagination chrome the baseline does not want. The composed alternative uses only
exported primitives, so the design contract holds. If the roster later grows into a
filterable table, moving to `DataTable` is the natural follow-up.

**D5 — The roster is read through an exported accessor, not an imported array.**
`listUnits()` in `src/lib/aoh/dispatch/roster.ts` returns the hardcoded data; the page
calls it and never references the array. This is the seam `dispatch-units-service`
repoints at the Go backend without editing the console page. `listUnits()` returns fresh
objects rather than a shallow array copy, so a caller cannot mutate the baseline for the
next reader.

**D6 — Module code lives at `src/lib/aoh/dispatch/`, not `src/lib/units/`.**
`aoh-conventions/web.md` places feature-module code under `src/lib/aoh/{module}/`. The
module is `dispatch`.

**D7 — Selection state is a Svelte 5 rune in the page component, not a URL parameter.**
`let selectedId = $state<string | null>(null)` keeps the interaction client-side,
satisfying the spec's "no network request, no navigation" requirement. Alternative:
`/units/[id]` with a `load()` — rejected as more machinery than a hardcoded roster earns,
and it would make selection a navigation.

**D8 — No unit is preselected on load.**
An explicit empty state makes the selection behaviour visible as a behaviour, and gives
the workshop a concrete empty-state pattern to point at.

## Risks / Trade-offs

- **[Removing the auth layer diverges from the `aoh-web-init` scaffold, which the skill
  says to copy verbatim]** → the divergence is total and deliberate rather than partial,
  which is the safer of the two: no half-wired auth path remains to behave unexpectedly.
  The removal is enumerated in D1 and in the app README, so restoring it later is a
  re-scaffold of known files rather than an archaeology exercise.
- **[An attendee later follows AOH docs that assume `(private)` / `(public)` groups and
  a gateway]** → the app README states plainly that this app has no auth and no gateway,
  and why.
- **[`minimumReleaseAge: 10080` blocks a freshly published dependency]** → pin the
  offending package to the newest version older than 7 days rather than weakening the
  workspace-level supply-chain setting.
- **[`ThemeProvider mode="dark"` means the light path is never seen and can rot]** → the
  light path is asserted explicitly in the E2E spec, and the mockup ships a toggle.
- **[The `~/.npmrc` GitHub Packages token is still required]** → documented as the first
  item in the app README; it is the only prerequisite the container-free design cannot
  remove.
- **[Tailwind does not scan this app by default, so app-only utility classes render as
  nothing]** → `@import 'tailwindcss'` lives inside the package stylesheet, so v4's
  automatic content detection roots at the package, not the consumer. `src/app.css`
  carries an explicit `@source './'`. This failure is silent — the class reaches the DOM,
  the token resolves, and no rule exists — and it is how the selected-row highlight first
  shipped invisible. Recorded in the app's `AGENTS.md`; the E2E suite now asserts the
  rendered difference rather than the `aria-current` attribute, so a regression fails.

## Migration Plan

Not applicable in the deployment sense — nothing is running today and nothing is
replaced. Rollout: merge the PR; attendees clone, `pnpm install`, `pnpm dev`. Rollback is
reverting the PR, which returns the repo to an empty `apps/`.

The forward path this design keeps cheap:

1. **`dispatch-units-service`** — a Go + PostgreSQL service exposing `/v1/units`;
   `listUnits()` becomes an async call to it. The console page is untouched, which is the
   point of D5.
2. **Command exercise** — actions on the detail pane. No roster or layout change.

## Open Questions

- **Unit ID format.** The mockup and implementation use `FU-101`-style identifiers.
  `dispatch-units-service` will need a primary key; if it uses UUIDs, these become the
  display identifier rather than the key. Deferred to that change — the spec here only
  requires uniqueness and stability.
- **Call sign vocabulary.** Phonetic call signs (`Alpha-1`) are placeholders. If the
  facilitator has a house vocabulary for the workshop scenario, it should replace them
  before the session; nothing depends on the choice.
- **Whether `listUnits()` should become async now.** Keeping it synchronous is simpler to
  read, but `dispatch-units-service` will have to make it `async` and touch the page's
  `const units = listUnits()`. Making it `Promise`-returning now would absorb that churn
  early at the cost of an `await` the baseline does not need. Flagged for the follow-up
  change to decide, since it is that change's cost to pay.
