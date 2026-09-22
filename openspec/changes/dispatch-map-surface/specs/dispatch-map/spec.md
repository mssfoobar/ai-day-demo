## ADDED Requirements

### Requirement: The application has a map surface

The application SHALL offer a map page at `/map` rendering a geospatial canvas with base
tiles. It SHALL be built on `@mssfoobar/gis-web-sdk` — the map, its engine provider and its
layer panel — and SHALL NOT hand-roll a map renderer or a tile client. Its surrounding
chrome SHALL come from `@mssfoobar/ui`. Every state below is drawn in
`openspec/changes/dispatch-map-surface/design/map-mock.html`.

#### Scenario: The map renders
- **WHEN** a visitor opens `/map`
- **THEN** the canvas renders base tiles
- **AND** the visitor can pan and zoom

#### Scenario: The surface is SDK-composed
- **WHEN** the map page source is reviewed
- **THEN** the map, its engine and its layer panel are components imported from `@mssfoobar/gis-web-sdk`
- **AND** any surrounding chrome is imported from `@mssfoobar/ui` subpaths, with no hand-rolled equivalents

#### Scenario: The layer panel lists the base layer
- **WHEN** an operator opens the layer panel
- **THEN** the OpenStreetMap base layer is listed and can be toggled

### Requirement: The map carries no application data

This change integrates the map module and stops there. The map SHALL NOT render field
units, SHALL NOT call `dispatch-svc`, and SHALL NOT require `gis-service` or RTUS to be
running. The page SHALL say that it carries no entities yet, so an operator can tell a
correctly integrated empty map from a broken one.

#### Scenario: No application data is fetched
- **WHEN** the map page is opened and the browser's network log is inspected
- **THEN** no request is made to the dispatch service, to `gis-service`, or to RTUS

#### Scenario: The empty map explains itself
- **WHEN** an operator opens the map
- **THEN** the page states that no entities are shown yet and names wiring field units to the map as the next step
- **AND** no error state is shown, because nothing has failed

#### Scenario: The map runs on the baseline stack
- **WHEN** the map is opened with only the existing PostgreSQL container running
- **THEN** it renders normally
- **AND** no additional container is required

### Requirement: The map requires no authentication

The map SHALL be reachable without signing in, on the same terms as the rest of this
application. The GIS SDK gates authentication on `IAM_URL`; leaving it unset SHALL be the
supported posture here rather than a workaround, and the page SHALL NOT attempt an OIDC
discovery, a token fetch, or a session lookup.

#### Scenario: An unauthenticated visitor reaches the map
- **WHEN** a visitor with no session opens `/map`
- **THEN** the map renders
- **AND** they are not redirected to a sign-in flow

#### Scenario: No identity provider is contacted
- **WHEN** the map page is opened with no identity provider running
- **THEN** the page renders and the server logs no discovery or token error

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
widget files — from its own origin under a stable base URL, because the engine fetches them
lazily at runtime and renders an entirely black canvas without them. The copied assets SHALL
be ignored by version control and by the linter.

#### Scenario: Assets resolve
- **WHEN** the map page is opened and the browser's network log is inspected
- **THEN** no request under the Cesium base URL answers 404
- **AND** the canvas renders tiles rather than remaining black

#### Scenario: Assets are copied in both dev and build
- **WHEN** the dev server is started, and separately when a production build is run
- **THEN** the Cesium asset directory is present under the app's static root in both cases

#### Scenario: Copied assets are not committed or linted
- **WHEN** the repository and the frontend lint configuration are inspected
- **THEN** the copied Cesium asset directory is ignored by both

### Requirement: The map follows the console's theme

The map SHALL repaint with the application's existing dark-mode store rather than owning a
theme of its own, so switching the console's theme switches the map's.

#### Scenario: The map follows a theme change
- **WHEN** an operator switches the application's theme while the map is open
- **THEN** the map's palette changes with it

#### Scenario: The theme store is the app's existing one
- **WHEN** the root layout is reviewed
- **THEN** `<GisProvider>` is fed the store the app's `ThemeProvider` already owns, and no second theme store is introduced
