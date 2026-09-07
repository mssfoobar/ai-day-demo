## ADDED Requirements

### Requirement: The application has no authentication

The application SHALL contain no authentication mechanism. It SHALL NOT perform OIDC
discovery, SHALL NOT redirect to a login page, and SHALL NOT require Keycloak, Traefik,
IAMS, SDS, or any other container to start or to serve any route.

Removing authentication — rather than merely bypassing it for one route — is required
because the web-base scaffold performs OIDC discovery in a top-level `await` during
server startup, which fails every route when no identity provider is reachable.

#### Scenario: Console renders from a clean checkout
- **WHEN** an attendee clones the repo, runs `pnpm install`, runs `pnpm dev`, and opens the console route
- **THEN** the console page renders the units list and the detail pane
- **AND** no login redirect occurs

#### Scenario: No container is required
- **WHEN** the console is loaded with no compose stack running
- **THEN** the page responds with HTTP 200 and reports no connection or authentication error

#### Scenario: The server starts with no identity provider reachable
- **WHEN** the dev server is started with no Keycloak or other identity provider running
- **THEN** the server starts successfully and serves requests
- **AND** no outbound request to an identity provider is attempted at startup or on any request

#### Scenario: Bare root lands on the console
- **WHEN** a visitor opens the site root `/`
- **THEN** they are taken to the console page
- **AND** they are not sent to a login route

#### Scenario: No authentication surface remains
- **WHEN** the application's routes are enumerated
- **THEN** no login, logout, callback, token-refresh, or authenticated-gateway route is present

### Requirement: Two-pane console layout

The console SHALL present a single page laid out as a left **Units** list pane and a
right **unit detail** pane, following the AOH Main List and Details page archetypes.
All visual primitives SHALL be composed from `@mssfoobar/ui`; the page SHALL NOT
hand-roll equivalents of primitives the package already exports.

#### Scenario: Both panes are visible on a desktop viewport
- **WHEN** the console is opened at a viewport of 1280px wide or greater
- **THEN** the units list is visible on the left and the detail pane on the right, both without horizontal page scroll

#### Scenario: Page conforms to the design system
- **WHEN** the console page source is reviewed
- **THEN** every button, card, badge, and list primitive it renders is imported from a `@mssfoobar/ui` subpath
- **AND** no raw hex color and no raw Tailwind colour utility (e.g. `bg-blue-500`) appears in the page source; colours come from the semantic tokens `@mssfoobar/ui` ships

#### Scenario: Dark mode parity
- **WHEN** the console is viewed with the dark theme active
- **THEN** every pane, label, and status indicator remains legible using semantic tokens, with no light-mode-only color

### Requirement: Units list contents

The units list SHALL render every unit in the baseline roster, and for each unit SHALL
display its call sign, its identifier, and a status indicator reflecting that unit's
status.

#### Scenario: All roster units are listed
- **WHEN** the console is opened
- **THEN** the list renders one row per unit in the roster, in the roster's declared order
- **AND** the number of rows is between 4 and 5 inclusive

#### Scenario: Each row shows call sign, ID, and status
- **WHEN** a unit row is inspected
- **THEN** it displays that unit's call sign, its identifier, and a status indicator whose label is exactly `Available`, `En route`, or `Idle`

#### Scenario: Status is distinguishable without relying on color alone
- **WHEN** a unit row's status indicator is rendered
- **THEN** the status is conveyed by a visible text label, not by color alone

### Requirement: Selecting a unit populates the detail pane

Selecting a unit in the list SHALL populate the detail pane with that unit's
information. Selection SHALL be a client-side state change only — it SHALL NOT issue a
network request and SHALL NOT navigate away from the console page.

#### Scenario: Selecting a unit shows its details
- **WHEN** an operator selects a unit in the list
- **THEN** the detail pane displays that unit's call sign, identifier, and status
- **AND** no network request is issued

#### Scenario: Selected unit is visually indicated in the list
- **WHEN** a unit is selected
- **THEN** that unit's row is rendered in a selected state distinct from the unselected rows
- **AND** at most one row is in the selected state at any time

#### Scenario: Switching selection replaces the detail contents
- **WHEN** an operator selects a second unit while a first unit is already selected
- **THEN** the detail pane shows the second unit's information and no residual information from the first

#### Scenario: Keyboard selection
- **WHEN** an operator moves focus to a unit row using the keyboard and activates it with `Enter` or `Space`
- **THEN** that unit is selected and the detail pane updates
- **AND** the focused row shows the design system's focus ring

### Requirement: Initial state before any selection

The console SHALL define a deliberate state for the detail pane before an operator has
selected a unit, so that every attendee sees the same first screen.

#### Scenario: No unit is selected on first load
- **WHEN** the console is opened and no selection has been made
- **THEN** the detail pane shows a prompt to select a unit, in AOH operational voice and sentence case
- **AND** no unit row is rendered in the selected state

### Requirement: No command actions in the baseline

The detail pane SHALL NOT offer any command, dispatch, assignment, or status-mutation
action. The baseline is read-only.

#### Scenario: Detail pane exposes no mutating control
- **WHEN** the detail pane is displayed for a selected unit
- **THEN** it presents no control that would dispatch, reassign, or change the status of that unit
