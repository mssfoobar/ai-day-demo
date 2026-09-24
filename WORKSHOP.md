# Workshop exercises

Three features are **stubbed but not built**. Each is visible in the running console, so
you can see exactly where it lands before you write a line. Pick one, build it, remove its
placeholder.

All three are **frontend only**: everything they need already exists in the service,
which ships complete and has no placeholder routes.

| # | Feature | Where the placeholder is |
|---|---|---|
| 1 | Show the incidents on the units page | a dashed **Incidents** panel sketched below the roster. Frontend only: the assignment routes ship built |
| 2 | See a unit's location on the units page | a dashed **Location** box sketched under Position in the detail pane. Frontend only |
| 3 | Put the incidents on the map | a dashed **Incidents** card sketched over the map canvas. Frontend only |

In the console, everything workshop-related floats over the UI rather than being part of it:
the floating button at the bottom right opens a dialog listing what is still to build, each
with its story. **Show me where** selects a unit and draws a dashed sketch of the missing
control, outlined, exactly where it goes — one exercise at a time; choosing another swaps it.
Clicking the sketch opens the same dialog on that exercise. When you finish one, flip its
`complete` flag in `apps/dispatch-web/src/lib/aoh/dispatch/workshop.ts` and it drops off the
list.

Every route is behind a bearer token, so an unauthenticated `curl` answers **401** before
it reaches anything. Fetch a token first — `scope=openid` is not optional, the service
validates through Keycloak's userinfo endpoint which rejects a token issued without it:

```sh
TOKEN=$(curl -fsS -X POST \
  "http://iams-keycloak.127.0.0.1.nip.io/realms/aoh/protocol/openid-connect/token" \
  -d grant_type=password -d client_id=web -d scope=openid \
  -d username=admin -d password='P@ssw0rd' \
  | node -e "let d='';process.stdin.on('data',c=>d+=c).on('end',()=>console.log(JSON.parse(d).access_token))")

curl -i http://localhost:8081/v1/units -H "Authorization: Bearer $TOKEN"
```

## Ground rules

- **No new libraries.** Everything you need is already here: `@mssfoobar/ui` primitives,
  SvelteKit form actions with `use:enhance`, chi + sqlx + `aoh-golib` on the Go side, and a
  PostgreSQL schema that already carries every column exercise 3 needs.
- **Copy the existing slice.** Create / edit / delete is the worked example: a form
  component (`UnitForm.svelte`), a form action in `+page.server.ts`, a client call in
  `units.server.ts`, then handler → service → repo in `apps/dispatch-svc`. Follow that shape.
- **Keep optimistic concurrency.** Every write echoes the unit's `occ_lock`; a stale write
  is a 409 `DISPATCH_UNIT_STALE`, never a silent overwrite.
- **Errors follow the contract.** Validation failures list every bad field in `details`
  via `aoherr.FieldDetail`; new error codes are `DISPATCH_*` in `internal/service/errors.go`.
- **Placeholders go when the feature arrives.** Delete the `Workshop exercise N` sketch
  for the exercise you built, and mark it `complete` in `workshop.ts`. When all three are done, delete `workshop.ts` and `ExerciseDialog.svelte`.
- `pnpm verify` must stay green.

Each exercise is written as a user story with acceptance criteria. Anything not listed is
out of scope — resist adding it.

---

## Exercise 1 — Show the incidents on the units page

> **As a** dispatcher, **I want to** see every incident the fleet is working on one panel,
> **so that** I can tell what is happening across the shift without clicking each unit in
> turn, and can send a unit to an incident from there.

**Frontend only.** The service side ships **built**: `POST` and
`DELETE /v1/units/{unit_code}/assignment` are real handlers with validation, the
`occ_lock` guard and the status rule. Nothing in `apps/dispatch-svc` needs touching. The
exercise is the console.

### Acceptance criteria

1. The units page carries an **Incidents** panel listing every active incident across the
   roster, derived from the units that carry an assignment. Each row shows the priority
   badge, incident code, title, description, location and the unit working it.
2. The panel is ordered by priority, `P1` first, and its heading carries the count.
3. Clicking a row selects that incident's unit, so the detail pane follows the panel.
4. With an available unit selected, **Dispatch** opens a form with five required fields
   (incident code, title, description, priority, location) and an optional longitude and
   latitude. Submitting `POST`s to the assignment route; the unit turns **En route** and
   appears in the panel without a page reload.
5. On an assigned unit the same control reads **Stand down** and `DELETE`s the assignment.
   The unit returns to **Available** and leaves the panel.
6. A stale `occ_lock` comes back as a 409 and is shown in the form, exactly as edit does
   today. Never re-read and retry silently.
7. An empty fleet-wide incident list shows `No active incidents.` rather than an empty box.

### What already exists

- **The whole service side.** `POST /v1/units/{unit_code}/assignment` takes
  `{ incident_code, title, description, priority, location, point?: { lon, lat }, occ_lock }`
  and returns the updated unit; `DELETE .../assignment?occ_lock=N` stands it down. Both
  answer 400 with per-field `details`, 409 `DISPATCH_UNIT_STALE` on a stale lock and 409
  `DISPATCH_UNIT_NOT_ASSIGNED` when standing down a unit that is not on an incident.
- The status rule is the service's: dispatch means **En route**, stand down means
  **Available**. Do not send a status.
- `Assignment` in `types.ts` already carries `description` and the optional `point`, and
  `UnitDetail.svelte` already renders them. Read from `unit.assignment`.
- **Writes go through form actions, never a client fetch** (`apps/dispatch-web/AGENTS.md`).
  `+page.server.ts` already has `create` / `update` / `delete`; add `dispatch` and
  `standDown` beside them, and a client call in `units.server.ts` next to the existing
  ones.
- `UnitForm.svelte` is the Sheet form with field errors, `use:enhance` and a hidden
  `occLock`. Copy it and cut it down.
- `filters.ts` is where the deriving belongs: a pure `activeIncidents(units)` with no DOM,
  rendered by a component. Put the behaviour in the pure module first.

### Out of scope

Choosing from a list of incidents that exist independently of units, more than one unit
per incident, incidents on the map, and notifying anyone.

---

## Exercise 2 — See a unit's location without leaving the units page

> **As a** dispatcher, **I want to** see where the selected unit is on a small map in the
> detail pane, **so that** I can place it at a glance without losing the roster, my
> filters and my selection to a trip to the map page.

**Frontend only.** The coordinates are already in the pane; today they are two numbers and
a link that navigates away. This exercise turns them into a map.

### Acceptance criteria

1. With a positioned unit selected, the detail pane shows a small map centred on that
   unit, under the Position section, with the unit marked.
2. The existing coordinates and fix time stay. The map is an addition, not a replacement:
   a dispatcher reads coordinates out loud over the radio.
3. Selecting a different unit re-centres the map on it.
4. A unit with no position keeps today's `No position reported.` and renders no map frame
   at all. An empty map centred on null island is worse than no map.
5. The **Show on map** link still works and still goes to the full map page.
6. The units page still server-renders. If it starts failing with a `window`/`document`
   error at startup, see the trap below.

### What already exists

- `unit.position` is `{ lon, lat, at }` and already typed in `types.ts`.
- **`LocationMapDisplay`** (`@mssfoobar/gis-web-sdk/location-map-display`) is the component
  for exactly this: one location, its own tiles, no RTUS. Props are
  `position: [lon, lat]`, `zoom: number` and `map_xyz_url: string`.
- It draws **no marker of its own**. The camera centres on `position`, and whatever you
  pass as its children is laid over the centre of the map, so the children are the marker.
- That marker is **fixed to the frame, not to the map**. The map still pans and zooms, so
  after one drag the marker points at the wrong place. Put `pointer-events-none` on the
  frame. The map then ignores the mouse and stays a static preview, and exploring is what
  **Show on map** is for.
- Its root element has **no size**. Give the frame a height and make the component fill it,
  for example
  `class="pointer-events-none relative h-40 overflow-hidden [&>div]:h-full"` on the
  wrapper. Without that, the map renders taller than the frame and is clipped. The unit then
  sits below the visible area while the marker stays in the middle of the frame.
- The tile URL the map page uses is `https://tile.openstreetmap.org/{z}/{x}/{y}.png`
  (`OSM_URL` in `map/+page.svelte`). Lift it somewhere both pages can read rather than
  typing it twice.
- `UnitDetail.svelte` already has the Position section and the dashed placeholder box to
  replace.

### The trap

`LocationMapDisplay` imports the **Cesium** engine, and Cesium touches browser globals at
module init. The map page gets away with a plain import because its `+page.ts` sets
`ssr = false`. The units page server-renders and must keep doing so, so a top-level
`import` of this component will break every render of the roster.

Load it in the browser only: import it dynamically inside the component that shows it, and
render the frame once it has resolved. `{#await import('…')}` or an `onMount` import both
work. Do **not** fix this by turning SSR off for the units page.

### Out of scope

Panning and zooming (the inline map is a static preview, so turn them off rather than
leave them on), a layer switcher, showing more than the selected unit, the incident's
location (that is exercise 3, on the map page), and replacing the map page.

---

## Exercise 3 — Put the incidents on the map

> **As a** dispatcher, **I want to** see where the incidents actually are on the map,
> **so that** I can judge which unit is closest to one without reading addresses off a
> list.

**Frontend only.** The incident's coordinates are already on the wire, and the map page
already loads the roster. No service change, no new request.

### Acceptance criteria

1. Every incident whose assignment carries a `point` gets a marker on the map, alongside
   the field-unit markers.
2. Clicking a marker highlights it and opens a panel showing the incident: priority,
   code, title, description, location, and the unit working it.
3. The panel carries a **placeholder picture** where a photo of the scene would go. A
   `Skeleton` or a captioned grey box is the right level of effort; do not add a binary
   asset to the repo for it.
4. Clicking the unit named in the panel selects it, exactly as clicking a unit marker does.
5. An incident with no coordinates is counted somewhere ("2 not shown"), never silently
   dropped, the same way the unit layer handles unpositioned units.
6. Clicking empty space clears the panel.

### What already exists

- `unit.assignment.point` is `{ lon, lat }` and already on the wire, already typed in
  `types.ts`, and the seeded incidents carry real coordinates.
- **`MapSingleEntityProvider`** (`@mssfoobar/gis-web-sdk/single-entity-provider`) is the
  component for this. It renders a host-supplied entity **with no RTUS feed**, which is
  exactly what an incident is: the map's live entity stream carries field units only.
  Import it from its subpath as a default export, like every other SDK component here.
  - Give the entity a `geojson.properties.kind` or the engine cannot build a pick proxy
    and the marker will not be clickable.
  - Entities rendered this way are **not** part of a `MapEntityLayerProvider`, so they do
    not appear in the **Layers** control. That is the documented trade-off, not a bug.
- The unit-marker snippet in `map/+page.svelte` is the worked example of a clickable
  marker, and of the render-purity rule that goes with it.
- The `{#if units.length === 0}` / `{:else if positioned.length === 0}` block is the
  worked example of the "counted, not dropped" empty states.

### The rule not to break

`apps/dispatch-web/AGENTS.md`: **entity state has one source, the SDK's RTUS
subscription**, and the roster the map loads is for counts only. Incidents are a
*separate* set of host-owned markers, so rendering them from the roster is fine. Merging
them into the field-unit entity stream, or feeding unit positions from the roster, is the
thing that rule forbids.

### Out of scope

Editing an incident from the map, clustering, routing a unit to an incident, drawing the
unit-to-incident line, and a real photograph.

## Done looks like

- The placeholder for your exercise is gone from the console, the page, and `Routes`.
- `pnpm verify` is green.
- `pnpm reset && pnpm start`, then sign in as `admin` — the first dispatcher request is
  what seeds the roster — and walk the acceptance criteria against a clean stack,
  including the restart in the last criterion.
