# Workshop user stories

Paste-ready briefs for the three workshop exercises. Each block is self-contained: copy the
whole exercise, story through out-of-scope, so the acceptance criteria and the scope boundary
travel with it. The full brief, including hints on what already exists in the codebase, is in
`WORKSHOP.md`.

---

## Exercise 1 — Show the incidents on the units page

**As a** dispatcher,
**I want to** see every incident the fleet is working on one panel,
**so that** I can tell what is happening across the shift without clicking each unit in
turn, and can send a unit to an incident from there.

**Frontend only.** The assignment routes ship built.

### Acceptance criteria

1. The units page carries an **Incidents** panel listing every active incident across the
   roster, derived from the units that carry an assignment: priority, incident code,
   title, description, location and the unit working it.
2. The panel is ordered by priority, `P1` first, and its heading carries the count.
3. Clicking a row selects that incident's unit.
4. With an available unit selected, **Dispatch** opens a form with five required fields
   (incident code, title, description, priority, location) and an optional longitude and
   latitude. The unit turns **En route** and joins the panel without a page reload.
5. On an assigned unit the same control reads **Stand down**; the unit returns to
   **Available** and leaves the panel.
6. A stale `occ_lock` comes back as a 409 and is shown in the form.
7. An empty list shows `No active incidents.`

### API

Already built, no service work:

| Method | Path | Body |
|---|---|---|
| POST | `/v1/units/{unit_code}/assignment` | `{ incident_code, title, description, priority, location, point?, occ_lock }` |
| DELETE | `/v1/units/{unit_code}/assignment?occ_lock=N` | none |

## Exercise 2 — See a unit's location without leaving the units page

**As a** dispatcher,
**I want to** see where the selected unit is on a small map in the detail pane,
**so that** I can place it at a glance without losing the roster, my filters and my
selection to a trip to the map page.

**Frontend only.** No service work.

### Acceptance criteria

1. With a positioned unit selected, the detail pane shows a small map centred on that
   unit, under the Position section, with the unit marked.
2. The existing coordinates and fix time stay; the map is an addition, not a replacement.
3. Selecting a different unit re-centres the map.
4. A unit with no position keeps `No position reported.` and renders no map frame.
5. **Show on map** still works and still goes to the full map page.
6. The units page still server-renders.

### API

None. `unit.position` is already on the wire. Render it with `LocationMapDisplay`
(`@mssfoobar/gis-web-sdk/location-map-display`), which takes `position`, `zoom` and
`map_xyz_url`. It draws no marker itself. Its children are laid over the map centre, where
`position` is, so pass the marker as children. Its root has no size, so give the frame a
height and `[&>div]:h-full`. Otherwise the map overflows the frame and the unit is clipped
out of view. The marker is fixed to the frame, not to the map, so also put
`pointer-events-none` on the frame. That keeps the map a static preview, so a drag or a
scroll cannot move the map out from under the marker.

Note: that component pulls in Cesium, which touches browser globals at module init, so it
must be imported dynamically in the browser. A top-level import breaks SSR of the units
page.

## Exercise 3 — Put the incidents on the map

**As a** dispatcher,
**I want to** see where the incidents actually are on the map,
**so that** I can judge which unit is closest to one without reading addresses off a list.

**Frontend only.** No service work.

### Acceptance criteria

1. Every incident whose assignment carries a `point` gets a marker on the map, alongside
   the field-unit markers.
2. Clicking a marker highlights it and opens a panel with the incident: priority, code,
   title, description, location and the unit working it.
3. The panel carries a placeholder picture where a photo of the scene would go.
4. Clicking the unit named in the panel selects it.
5. An incident with no coordinates is counted, never silently dropped.
6. Clicking empty space clears the panel.

### API

None. `unit.assignment.point` is already on the wire, and the map page already loads the
roster. Render the markers with `MapSingleEntityProvider`, which takes a host-supplied
entity and needs no RTUS feed.

## Ground rules that apply to every exercise

- No new libraries.
- Copy the existing create / edit / delete slice: form component, form action, client call,
  then handler → service → repo.
- Keep optimistic concurrency: every write echoes `occ_lock`; a stale write is a 409
  `DISPATCH_UNIT_STALE`.
- Errors follow the AOH contract; new codes are `DISPATCH_*` in `internal/service/errors.go`.
- Remove the placeholder for the exercise you built and flip its `complete` flag in
  `apps/dispatch-web/src/lib/aoh/dispatch/workshop.ts`.
- `pnpm verify` must stay green.
