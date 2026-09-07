## Context

`baseline-dispatch-console` shipped a container-free frontend whose roster is hardcoded
behind `listUnits()`. This change replaces that source with a Go + PostgreSQL service and
fills the console out into something that reads like a real operations surface.

Environment facts that shaped the approach, verified rather than assumed:

- **Go 1.25.6 and Docker 29.1.3 are installed. Python is not** (`python`/`python3`
  resolve to the Windows Store stub). `aoh-go-init`'s scaffold is a Python script, so it
  **cannot be run here** — the service is hand-written to the same architecture instead.
  See D1.
- `github.com/mssfoobar/ops-hub` is reachable with the developer's existing git
  credentials, and `packages/aoh-golib/v0.3.0` is a published tag, so the shared library
  is usable as a normal module dependency.
- The workshop is unauthenticated by decision. No IAMS, no SDS, no Traefik.

## Goals / Non-Goals

**Goals:**

- A real backend the console reads from, conformant to the AOH wire contract.
- A unit model rich enough that the console is worth looking at.
- Keep the browser talking only to its own origin — no CORS, no service URL in the bundle.
- Keep the seam: the console page should not learn that data now arrives over HTTP.

**Non-Goals:**

- Authentication, authorization, tenancy enforcement.
- Writes, real-time updates, pagination, a map.
- End-to-end tests (removed by decision — see D6).

## Runtime dependencies

| Dependency | Role | Owned by |
|---|---|---|
| `postgres:16` | The `dispatch` schema — the service's own database. The only container in the stack. | This change — added under `compose/` |
| `apps/dispatch-svc` | The Go service. Run natively with `go run`, not composed. | This change |
| `apps/dispatch-web` | The SvelteKit console. Run natively with `pnpm dev`. | Existing — modified by this change |

Deliberately absent: `iams-keycloak`, `iams-aas`, `sds-server`, `valkey`, Traefik, RTUS.

## API surface

| Method | Path | Auth posture | Description | Fronting gateway route |
|---|---|---|---|---|
| GET | /v1/units | Unauthenticated | List all field units. | n/a — read server-side by SvelteKit |
| GET | /v1/units/{unit_code} | Unauthenticated | One unit by its human-readable code. | n/a |
| GET | /livez | Unauthenticated | Liveness. | n/a |
| GET | /readyz | Unauthenticated | Readiness; fails when the database is unreachable. | n/a |

There is no `/aoh/gateway/...` route: the gateway was removed with the auth layer, and it
attaches a bearer token this service would not read. The SvelteKit server calls the
service directly (D3).

Success bodies are the AOH envelope — `{ data, message, sent_at }` — via
`aohhttp.Response`. Errors use `aoherr` + `aohhttp.Render`, so no 4xx/5xx has an empty
body.

### Unit resource

```jsonc
{
  "unit_code": "FU-101",
  "call_sign": "Alpha-1",
  "status": "Available",           // Available | En route | Idle
  "unit_type": "Ambulance",
  "station": "Marina Bay Station 4",
  "sector": "Sector 4 · Marina Bay",
  "radio_channel": "TAC-2",
  "shift": "Day · 07:00-19:00",
  "capabilities": ["ALS", "Water rescue"],
  "crew": [{ "name": "J. Tan", "role": "Paramedic" }],
  "assignment": {                   // omitted entirely when unassigned
    "incident_code": "INC-2841",
    "title": "Cardiac arrest · Raffles Quay",
    "priority": "P1",
    "location": "12 Raffles Quay",
    "since": "2026-09-07T13:12:00Z"
  },
  "last_contact": "2026-09-07T13:41:00Z"
}
```

`snake_case` on the wire (Go/AOH convention); the SvelteKit client maps to `camelCase` at
the boundary so the Svelte components keep TS-idiomatic names.

## Data model

Schema `dispatch`, configurable. Table names singular per `aoh-conventions/database.md`.

**`dispatch.unit`** — mandatory AOH columns (`id uuid` PK, `created_at`, `updated_at`,
`created_by`, `updated_by`, `tenant_id text`, `occ_lock int`), plus `unit_code text`,
`call_sign text`, `status text`, `unit_type text`, `station text`, `sector text`,
`radio_channel text`, `shift text`, `capabilities text[]`, `last_contact timestamptz`,
and the nullable assignment columns `assignment_incident_code`, `assignment_title`,
`assignment_priority`, `assignment_location`, `assignment_since`.

- `UNIQUE (unit_code, tenant_id)` — the human-readable id, per convention.
- `CHECK (status IN ('Available','En route','Idle'))` — this is where the closed status
  vocabulary now lives, replacing the frontend TypeScript union.
- Assignment is embedded rather than a table: it is 1:0..1, has no identity of its own
  here (the real incident lives in another system), and a join buys nothing. A single
  `CHECK` keeps the columns all-null or all-non-null so a half-populated assignment cannot
  exist.

**`dispatch.unit_crew`** — mandatory columns plus `unit_id uuid` FK → `unit(id)`
`ON DELETE CASCADE`, `name text`, `role text`, `sort_order int`. A child table rather than
JSONB because crew are enumerable rows the spec asks to be stored "as rows, not an opaque
blob", and ordering matters for display.

## UI / Design System

Mockup: `openspec/changes/dispatch-units-service/design/units-console-mock.html`
(clickable — selection, the summary tiles, the sectioned detail pane, an unassigned unit,
the error state, and a light/dark toggle).

### 1. Page archetypes

Unchanged in kind: a **Main List** (narrowed) beside a **Details** pane, now with a
summary strip above. Still not a Dashboard archetype — the tiles are inline metrics on a
list/detail page, which is the catalogue's explicit exclusion from "dashboards belong to
DASH".

### 2. Components from `@mssfoobar/ui`

Verified against the installed `@mssfoobar/ui@1.1.0` `exports`.

| Surface element | Primitive | Subpath |
|---|---|---|
| Summary tiles | `Card`, `CardContent` | `@mssfoobar/ui/card` |
| Pane shells | `Card`, `CardHeader`, `CardTitle`, `CardContent` | `@mssfoobar/ui/card` |
| Unit row | `Button` with `variant="ghost"` | `@mssfoobar/ui/button` |
| Row / section dividers | `Separator` | `@mssfoobar/ui/separator` |
| Status + priority + capability chips | `Badge` with `variant="soft"` | `@mssfoobar/ui/badge` |
| Error state | `Card` with `class="border-destructive"` | `@mssfoobar/ui/card` |
| Icons | per-icon from `@lucide/svelte/icons/<name>` | `@lucide/svelte` |

`Badge` splits shape from palette: `variant` is the shape, `color` the palette. Status
colours are unchanged (`success` / `info` / `default`); assignment priority uses
`destructive` for `P1`, `warning` for `P2`, `default` otherwise; capability chips use
`variant="outline"`.

There is no inline `Alert` primitive in `@mssfoobar/ui` — the error state composes a
`Card` with `border-destructive` and a `text-destructive` title, which is the documented
substitute.

### 3. Composition for anything not directly exported

- **Summary tile** — `Card` + `CardContent` holding a large tabular-nums figure over a
  muted label. Not a bespoke widget; the same pattern as the kit's `MetricCard`.
- **Unit row** — as before, a full-width `Button variant="ghost"`, now two-line: call sign
  + id on the first line, assignment-or-station on the second, with a type icon leading and
  a status `Badge` trailing.
- **Selected row** — `aria-current="true"` **plus** `bg-accent text-accent-foreground`.
  The attribute alone is invisible: `variant="ghost"` styles no `aria-current` state. This
  was a real defect in the baseline and is called out so it is not re-introduced.
- **Detail sections** — a `CardContent` per section with a small uppercase muted heading
  and a `Separator` between, each body a description list.

### 4. Theming surface

Light and dark both supported; `app.html` carries `class="dark"` so the first paint is
already dark (the baseline flashed light before hydration). No new semantic tokens; no raw
hex. **`src/app.css`'s `@source './'` must stay** — without it Tailwind does not scan this
app and any app-only utility silently renders as nothing.

### 5. Widgets

None. No dashboard widget, no `defineWidget`, no grid slot.

### 6. Copy and tone

| Slot | String |
|---|---|
| Page title | `Dispatch console` |
| Page description | `Field units and their current status.` |
| Summary labels | `Total units`, `Available`, `En route`, `Idle` |
| Units pane title | `Units` |
| Detail pane title | `Unit detail` (none selected) / the call sign |
| Empty-state line | `Select a unit to see its details.` |
| Section headings | `Overview`, `Assignment`, `Crew`, `Capabilities` |
| Unassigned line | `Not currently assigned.` |
| No capabilities | `None recorded.` |
| Error title / line | `Units unavailable` / `Could not reach the dispatch service. Check that it is running, then reload.` |
| Field labels | `Call sign`, `Unit ID`, `Status`, `Type`, `Station`, `Sector`, `Radio`, `Shift`, `Last contact` |

### 7. Accessibility

As the baseline, plus: summary tiles are plain text (no role games); section headings are
real headings; the error state is a `role="alert"` region so it is announced. Status and
priority remain conveyed by badge **text**, never colour alone.

### 8. Open design questions

Listed under Open Questions.

### 9. Iteration 2 — interactivity (facilitator feedback: "too plain, make it sharper")

Applied after the first cut shipped. All of it is client-side state over the same
server-loaded roster; nothing fetches, navigates, or mutates. Composition stays inside
`@mssfoobar/ui`:

| Behaviour | Primitive | Notes |
|---|---|---|
| Search across call sign, id, type, station, sector, radio, incident, crew, capabilities | `Input` (`@mssfoobar/ui/input`) | `/` focuses it from anywhere; `Esc` clears. Every whitespace term must match. |
| Status filter | `ToggleGroup type="multiple"` (`@mssfoobar/ui/toggle-group`) | The three status **summary tiles are the toggles**. Counts stay fleet-wide while filtering — the summary is about the fleet, the list about what you are looking at. An empty selection means "no filter", never "nothing". |
| Sort | `Select` (`@mssfoobar/ui/select`) | Call sign / Status (En route → Available → Idle) / Priority (P1 first, unassigned last) / Last contact (newest first). Ties fall back to call sign so order never flickers. |
| Keyboard navigation | native `<ul>` keydown | `↑ ↓ Home End` move selection through the **visible** list and move focus with it; `Enter`/`Space` still come from `Button`. A key-hint strip sits under the list. |
| Live recency | `$effect` + 30 s interval | "3 min ago" labels re-render without refetching. A fresh/ageing/stale dot (<10 / <30 / ≥30 min) sits under each status badge. |
| Full-height console | layout only | `h-dvh` flex column; each pane is a `Card` with an internal `ScrollArea` given a definite height by `flex-1 min-h-0` — the reason the earlier `ScrollArea` was removed (no definite height) no longer applies. |
| Sharper rows | composition | A 4 px status **rail** (`bg-success` / `bg-info-strong` / muted), the type icon, priority chip on assigned units, incident on the second line. Status is still conveyed by the badge **text** — the rail and dots are reinforcement, never the only signal. |
| Sharper selection | composition | `aria-current` + `bg-accent` + an inset 1 px `--ring` outline via `shadow-[inset_0_0_0_1px_var(--ring)]`. |
| Empty-filter state | `Card` + `Button` | `No units match.` with a `Clear filters` action, distinct from the pre-selection empty state. |

Pure logic (`filters.ts`, `format.ts`) is separated from rendering and unit-tested (28
tests): search semantics, each sort order, fleet-wide counts, recency buckets, and that
the rail classes are semantic tokens rather than raw colours.

**Keycaps** (`<kbd class="kbd">`) are typography on semantic tokens in a scoped `<style>`;
`@mssfoobar/ui` ships no keycap primitive, and this is not a re-implementation of one.

Deliberately not done: persisting selection in the URL (the spec keeps selection
client-side and non-navigating), tabs in the detail pane (a dispatcher wants everything
visible at once), and any polling (RTUS is the AOH answer when live data arrives).

## Decisions

**D1 — Hand-write the Go service to the `aoh-go-init` architecture instead of running the
scaffold.** The scaffold is `scripts/scaffold.py` and Python is not installed on this
machine, so the skill cannot execute. Rather than install a toolchain the workshop does
not otherwise need, the service is written by hand to the same shape: `cmd/server`,
`internal/{config,handler,service,repo}`, chi router, sqlx, Viper, `aoh-golib` for the
envelope and logging, `/livez` + `/readyz`, layered so the repo is swappable. The cost is
that the scaffold's extras — mockery config, swag annotations, its Makefile and Dockerfile
— are not reproduced; those are additive and can come from a later real scaffold run.
*This deviation is the main thing a reviewer should sanity-check.*

**D2 — `unit_code`, not a UUID, is what the API and the URL use.** The database keeps a
synthetic `uuid` primary key per convention, but `FU-101` is what a dispatcher says out
loud and what the console shows, so it is the resource identifier in `/v1/units/{code}`.
The convention explicitly provides for this with a uniqueness constraint per tenant.

**D3 — The SvelteKit server calls the service; the browser never does.** A `+page.server.ts`
load hits `DISPATCH_SVC_URL`. This keeps the browser same-origin (no CORS middleware to
write, no service URL in the bundle), mirrors the BFF role the AOH gateway normally plays,
and means the service does not need to care about browser origins at all. Alternative —
fetching from client code — was rejected on those three counts.

**D4 — `listUnits()` survives as the seam, now server-only.** It moves to
`src/lib/aoh/dispatch/units.server.ts`, becomes `async`, and performs the HTTP call and
the snake→camel mapping. The console page's server load calls it and nothing else. The
`.server.ts` suffix makes accidental client import a build error rather than a leak.

**D5 — Assignment is embedded in `unit`, crew is a child table.** Rationale in Data model
above. The asymmetry is deliberate: assignment is 1:0..1 with no local identity, crew is
1:N and ordered.

**D6 — Remove the Playwright layer entirely.** *Facilitator-confirmed.* The suite is
deleted rather than trimmed, along with its config, dependency and turbo task. This is a
real loss of coverage — it is what caught the invisible selected row — and it is accepted
so the workshop repo stays small. Unit tests, type-checking, lint and build remain on both
apps, and the Go service gets handler and service tests.

**D7 — The service owns the status vocabulary now.** A `CHECK` constraint replaces the
frontend TypeScript union as the enforcement point, because values now originate in the
database. The frontend keeps `UnitStatus` as documentation and guards unknown values when
decoding rather than trusting the wire.

## Risks / Trade-offs

- **[The workshop is no longer container-free — the console needs the service, which
  needs Postgres]** → accepted deliberately; it is the point of the change. `compose/`
  holds exactly one service and the README states the three commands.
- **[Hand-written service drifts from what `aoh-go-init` would have produced]** → D1
  documents the deviation; a later scaffold run on a machine with Python can diff against
  it.
- **[Deleting the E2E layer removes the guard that caught a real UI defect]** → D6;
  mitigated only partially by unit tests. Stated plainly rather than papered over.
- **[`aoh-golib` is a private module — a contributor without repo access cannot build the
  service]** → the app README documents it alongside the existing `~/.npmrc` requirement.
  It is the second credential the workshop needs.
- **[Seed data drifting from the frontend's expectations]** → the seed is checked in and
  idempotent, and the service is the single source; nothing in the frontend hardcodes
  units any more.

## Migration Plan

Nothing is deployed, so this is a developer-workflow change. Order matters:

1. `docker compose up -d postgres` (or `podman compose ...`) — the only container.
2. Migrations + seed applied on service start.
3. `go run ./cmd/server` in `apps/dispatch-svc`.
4. `pnpm dev` in `apps/dispatch-web`.

Rollback is reverting the PR; the frontend's previous commit still renders a hardcoded
roster with no containers at all.

Archive `baseline-dispatch-console` **before** this change, so its capabilities exist in
`openspec/specs/` for these deltas to modify.

## Open Questions

- **Whether `tenant_id` should be populated with anything meaningful.** Every row carries
  it per convention, but with no auth there is no tenant to derive. It is currently a
  fixed placeholder. If the workshop later adds IAMS, this is where multi-tenancy starts.
- **Whether the console should poll.** Status is exactly the kind of data that goes stale
  on screen. Polling is the wrong answer in AOH (RTUS exists for this), so nothing polls
  today and the data is as fresh as the last page load. Flagged rather than silently
  accepted.
- **Seed realism.** The seeded incidents and crew names are invented. If the workshop has
  a scenario script, the seed should match it.
