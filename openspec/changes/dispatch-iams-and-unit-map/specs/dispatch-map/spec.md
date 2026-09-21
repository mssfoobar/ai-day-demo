## ADDED Requirements

### Requirement: The application has a map surface

The application SHALL offer a map page at `/aoh/dispatch/map`, inside the `(private)` route
group, alongside the console rather than inside it, rendering the tenant's field units on a
geospatial canvas. It SHALL be built on
`@mssfoobar/gis-web-sdk` — the map, its engine provider, its entity providers and its
panels — and SHALL NOT hand-roll a map renderer, a tile client, or an entity layer. Its
chrome (page header, buttons, cards, badges) SHALL come from `@mssfoobar/ui`. Every state
below is drawn in `openspec/changes/dispatch-iams-and-unit-map/design/dispatch-map-mock.html`.

#### Scenario: The map renders the roster
- **WHEN** an operator with either application role opens `/aoh/dispatch/map`
- **THEN** the map canvas renders base tiles and one marker per positioned field unit in their tenant
- **AND** each marker is labelled with its unit's call sign

#### Scenario: The surface is SDK-composed
- **WHEN** the map page source is reviewed
- **THEN** the map, its engine, its entity layers and its panels are components imported from `@mssfoobar/gis-web-sdk`
- **AND** the surrounding chrome is imported from `@mssfoobar/ui` subpaths, with no hand-rolled equivalents

#### Scenario: Reaching the map from the console
- **WHEN** a signed-in operator is on the console
- **THEN** a **Map** entry is present in the sidebar navigation and reaches this page

### Requirement: The map page does not server-render

The map route SHALL opt out of server-side rendering, because the Cesium engine touches
browser globals at module initialisation.

#### Scenario: SSR is off for the route
- **WHEN** the map route's `+page.ts` is inspected
- **THEN** it exports `ssr = false`

#### Scenario: The server does not attempt to render the engine
- **WHEN** the map page is requested
- **THEN** the server response contains no rendered map markup and the server logs no engine or WebGL error

### Requirement: Cesium runtime assets are served by the app

The app SHALL serve Cesium's runtime assets — workers, textures, third-party shims and
widget files — from its own origin under a stable base URL, because the engine fetches
them lazily at runtime and renders an entirely black canvas without them.

#### Scenario: Assets resolve
- **WHEN** the map page is opened and the browser's network log is inspected
- **THEN** no request under the Cesium base URL answers 404
- **AND** the canvas renders tiles rather than remaining black

#### Scenario: Copied assets are not committed or linted
- **WHEN** the repository and the frontend lint configuration are inspected
- **THEN** the copied Cesium asset directory is ignored by both

### Requirement: Field units update live on the map

The map SHALL reflect a unit's movement without a page reload, by subscribing to the RTUS
map named `gis` through the SDK's live feed, authorised by the operator's session cookie.
Feature code SHALL NOT hand-roll an `EventSource`; the SDK's `@mssfoobar/sse-client`-based
subscription is the only live-update path. The page SHALL NOT also fetch an initial entity
list of its own, so there is no second source of entity state to reconcile.

#### Scenario: A position change appears without reload
- **WHEN** a unit's position changes while an operator has the map open
- **THEN** that unit's marker moves to the new position within a few seconds
- **AND** the operator performs no reload or refresh

#### Scenario: A new unit appears without reload
- **WHEN** a dispatcher creates a positioned unit while another operator has the map open
- **THEN** a marker for it appears on that operator's map

#### Scenario: A deleted unit disappears without reload
- **WHEN** a dispatcher deletes a unit while another operator has the map open
- **THEN** its marker is removed from that operator's map

#### Scenario: Entity state has a single source
- **WHEN** the map page's load function and component tree are reviewed
- **THEN** entity state comes only from the SDK's subscription, and the page issues no separate entity list request

#### Scenario: The subscription is authorised by the session cookie
- **WHEN** the browser opens the live feed
- **THEN** the request carries the `web_auth_session_id` cookie and no bearer token

### Requirement: Selection is shared between the map and the console

Selecting a unit on the map SHALL open that unit in the console's detail pane, and
selecting a unit in the console SHALL move the map's camera to it when the map is on
screen. At most one unit SHALL be selected at a time.

#### Scenario: Map to console
- **WHEN** an operator clicks a unit's marker
- **THEN** the console's detail view for that unit is shown
- **AND** the unit's identity in the detail view matches the marker clicked

#### Scenario: Console to map
- **WHEN** an operator selects a unit in the console and opens the map
- **THEN** the map's camera is centred on that unit's position

#### Scenario: Selecting a unit with no position
- **WHEN** an operator selects an un-positioned unit
- **THEN** the map camera does not move and the page states that the unit has no position

### Requirement: Un-positioned units are accounted for, not hidden

Units without a position cannot be drawn. The map SHALL state how many of the tenant's
units are not shown, so a dispatcher never mistakes an incomplete map for the whole roster.
The count SHALL be derived from unit data rather than from the entities on the map, so a
pending projection cannot make the roster look smaller than it is.

#### Scenario: The count is visible
- **WHEN** some of the tenant's units have no position
- **THEN** the page shows how many units are not on the map

#### Scenario: A fully positioned roster says so
- **WHEN** every unit in the tenant has a position
- **THEN** no un-positioned count is shown

#### Scenario: No positioned units at all
- **WHEN** no unit in the tenant has a position
- **THEN** the map renders its base layer with an explicit empty state rather than an apparently broken canvas

### Requirement: The map degrades when the live feed is unavailable

If the live feed cannot be reached or is not configured, the map SHALL still render its
base layers and SHALL tell the operator that positions are not live, rather than failing
to render or silently showing stale data as current.

#### Scenario: Live feed unreachable
- **WHEN** `rtus-seh` is unreachable and the map page is opened
- **THEN** the page renders the base map
- **AND** it shows a notice that live positions are unavailable

#### Scenario: The notice is operational in tone
- **WHEN** that notice is shown
- **THEN** its wording is sentence case and operational, and it names no stack trace, host name, or raw exception text

### Requirement: Reaching the map requires a session

The map page SHALL be reachable only by a signed-in operator, on the same terms as the
rest of the console.

#### Scenario: Unauthenticated access
- **WHEN** a visitor with no session opens `/aoh/dispatch/map`
- **THEN** they are redirected to the sign-in flow and no map or entity data is served

#### Scenario: A viewer may view the map
- **WHEN** an operator holding only `dispatch-viewer` opens the map
- **THEN** the map renders, and it offers no control that would change a unit
