## ADDED Requirements

### Requirement: The application has a map surface

The application SHALL offer a map page at `/aoh/dispatch/map`, inside the `(private)` route
group, alongside the console rather than inside it. It SHALL be built on
`@mssfoobar/gis-web-sdk` — the map, its engine provider and its panels — and SHALL NOT
hand-roll a map renderer or a tile client. Its chrome (page header, buttons, cards, badges)
SHALL come from `@mssfoobar/ui`. Every state below is drawn in
`openspec/changes/dispatch-iams-and-unit-map/design/dispatch-map-mock.html`.

#### Scenario: The map renders
- **WHEN** an operator with either application role opens `/aoh/dispatch/map`
- **THEN** the map canvas renders base tiles and its layer panel
- **AND** the operator can pan and zoom

#### Scenario: The surface is SDK-composed
- **WHEN** the map page source is reviewed
- **THEN** the map, its engine and its panels are components imported from `@mssfoobar/gis-web-sdk`
- **AND** the surrounding chrome is imported from `@mssfoobar/ui` subpaths, with no hand-rolled equivalents

#### Scenario: Reaching the map from the console
- **WHEN** a signed-in operator is on the console
- **THEN** a **Map** entry is present in the sidebar navigation and reaches this page

### Requirement: The map carries no dispatch data

This change integrates the map module and stops there. The map SHALL NOT render field units,
SHALL NOT read the dispatch service, and the console SHALL NOT drive the map's camera or
selection. Wiring the two together is deliberately left as a workshop exercise, and the page
SHALL say so rather than presenting an empty map as a fault.

#### Scenario: No field units are drawn
- **WHEN** the map is open and the tenant has units
- **THEN** no unit appears on the map
- **AND** the page issues no request to the dispatch service

#### Scenario: The empty map explains itself
- **WHEN** an operator opens the map
- **THEN** the page states that no entities are being published yet and names wiring field units to the map as the next step
- **AND** no error state is shown, because nothing has failed

#### Scenario: The console and the map do not interact
- **WHEN** an operator selects a unit in the console and opens the map
- **THEN** the map's camera is unchanged
- **AND** the console offers no control that claims to show a unit on the map

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

### Requirement: The live feed is connected even though nothing publishes to it

The map SHALL subscribe to the RTUS map named `gis` through the SDK's live feed, authorised
by the operator's session cookie, even though no entity is published to it yet. Connecting it
now is what proves the hardest part of the integration — the session cookie reaching
`rtus-seh` across origins — rather than deferring that discovery to the exercise that adds
entities. Feature code SHALL NOT hand-roll an `EventSource`; the SDK's
`@mssfoobar/sse-client`-based subscription is the only live-update path.

#### Scenario: The subscription is established
- **WHEN** an operator opens the map
- **THEN** the browser opens the live feed to `rtus-seh` and the connection is accepted
- **AND** it is not rejected as unauthorised

#### Scenario: The subscription is authorised by the session cookie
- **WHEN** the browser opens the live feed
- **THEN** the request carries the `web_auth_session_id` cookie and no bearer token

#### Scenario: The feed delivers nothing yet
- **WHEN** the subscription is open
- **THEN** no entity is received, because nothing publishes to the `gis` map in this change
- **AND** the map continues to render its base layers

### Requirement: The map degrades when the live feed is unavailable

If the live feed cannot be reached or is not configured, the map SHALL still render its
base layers and SHALL tell the operator that live updates are unavailable, rather than
failing to render.

#### Scenario: Live feed unreachable
- **WHEN** `rtus-seh` is unreachable and the map page is opened
- **THEN** the page renders the base map
- **AND** it shows a notice that live updates are unavailable

#### Scenario: The notice is operational in tone
- **WHEN** that notice is shown
- **THEN** its wording is sentence case and operational, and it names no stack trace, host name, or raw exception text

### Requirement: Reaching the map requires a session

The map page SHALL be reachable only by a signed-in operator, on the same terms as the
rest of the console.

#### Scenario: Unauthenticated access
- **WHEN** a visitor with no session opens `/aoh/dispatch/map`
- **THEN** they are redirected to the sign-in flow and no map data is served

#### Scenario: A viewer may view the map
- **WHEN** an operator holding only `dispatch-viewer` opens the map
- **THEN** the map renders, on the same terms as for a dispatcher
