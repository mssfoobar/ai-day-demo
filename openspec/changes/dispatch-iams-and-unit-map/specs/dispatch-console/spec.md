## ADDED Requirements

### Requirement: The console lives under the private route group

The console SHALL move from `/units` to `/aoh/dispatch/units`, inside SvelteKit's
`(private)` route group, matching the AOH `[project]/[module]` page-routing convention it
was exempted from while the app had no authentication. It SHALL be reachable from a
sidebar navigation entry alongside the map.

#### Scenario: The console answers on its module route
- **WHEN** a signed-in operator opens `/aoh/dispatch/units`
- **THEN** the console renders

#### Scenario: The old route no longer serves the console
- **WHEN** a visitor opens `/units`
- **THEN** they do not receive the console at that path

#### Scenario: Navigation lists both surfaces
- **WHEN** a signed-in operator views the sidebar
- **THEN** it carries an entry for the console and an entry for the map, and both navigate without a full page load

### Requirement: The console identifies the signed-in operator

The console SHALL show who is signed in and offer a sign-out control, composed from
`@mssfoobar/ui` primitives.

#### Scenario: The operator's identity is visible
- **WHEN** a signed-in operator opens the console
- **THEN** the header shows their display name or username

#### Scenario: Sign-out is reachable from the console
- **WHEN** the operator activates the sign-out control
- **THEN** their session ends and they are returned to the sign-in flow

### Requirement: The detail pane shows the unit's last known position

The detail pane SHALL show a selected unit's last known position and the time of its fix,
in the same sectioned layout as its other information. A unit with no position SHALL say
so explicitly rather than rendering an empty section.

#### Scenario: A positioned unit shows its fix
- **WHEN** an operator selects a unit that has a position
- **THEN** the pane shows its coordinates and the time of the fix, rendered with the console's existing recency wording

#### Scenario: An un-positioned unit states so
- **WHEN** an operator selects a unit with no position
- **THEN** the position section states that no position has been reported

#### Scenario: Position is visible on the map from the detail pane
- **WHEN** a positioned unit is selected
- **THEN** the pane offers a control that opens that unit on the map

### Requirement: An unseeded roster explains itself

The console SHALL render an empty roster as an explicit state that says why, not as a bare
empty list — the two are indistinguishable to an operator. This state is reachable because a
tenant is seeded by its first dispatcher request, so anyone signing in before a dispatcher
has — a viewer on a fresh stack, most likely — sees no units.

#### Scenario: A viewer on a fresh stack
- **WHEN** a viewer opens the console against a tenant that has never been seeded
- **THEN** the roster area states that there are no units yet and that a dispatcher signing in will populate it
- **AND** no error state is shown, because nothing has failed

#### Scenario: The state clears once seeding has happened
- **WHEN** a dispatcher has since made a request and the viewer reloads
- **THEN** the roster renders normally

## MODIFIED Requirements

### Requirement: Detail pane is sectioned

The detail pane SHALL present a selected unit's information grouped into labelled
sections — overview, position, assignment, crew, and capabilities — rather than one flat
list of fields.

#### Scenario: Sections are present for an assigned unit
- **WHEN** a unit with an assignment and crew is selected
- **THEN** the pane shows its overview fields, its position, its assignment, and its crew members with their roles

#### Scenario: An unassigned unit states so explicitly
- **WHEN** a unit with no assignment is selected
- **THEN** the assignment section says the unit is unassigned rather than rendering an empty section

#### Scenario: A unit with no capabilities omits nothing silently
- **WHEN** a unit has no capability tags
- **THEN** the capabilities section is either absent or states that none are recorded

#### Scenario: A unit with no position omits nothing silently
- **WHEN** a unit has no position
- **THEN** the position section states that none has been reported

### Requirement: The console degrades when the service is unavailable

Because the roster is fetched, the console SHALL show an explicit error state when the
dispatch service cannot be reached, rather than rendering an empty list as though the
roster were empty. It SHALL distinguish an unreachable service from a refusal by the
service, so an operator is not told to retry something that will never succeed.

#### Scenario: Service unreachable
- **WHEN** the console is loaded and the dispatch service cannot be reached
- **THEN** the page shows an error state explaining that units could not be loaded
- **AND** it does not present an empty units list as a successful result

#### Scenario: Error state is operational in tone
- **WHEN** the error state is shown
- **THEN** its wording is sentence case and operational, and it names no stack trace or raw exception text

#### Scenario: The service refuses the caller's token
- **WHEN** the dispatch service answers 401 for the console's request
- **THEN** the operator is returned to the sign-in flow rather than shown a service-unavailable message

#### Scenario: The service refuses the caller's role
- **WHEN** the dispatch service answers 403 for an action the operator attempted
- **THEN** the console shows a permission-denied message naming the action, and offers no retry

## REMOVED Requirements

### Requirement: The application has no authentication

**Reason**: Superseded. The workshop baseline was specified as unauthenticated so it would
run with no containers; this change introduces IAMS, which every other AOH application
already depends on, and which `gis-service` and `rtus-seh` require in order to authorise
the map's entity feed. The requirement's underlying observation — that the scaffold's
top-level `await` OIDC discovery makes auth all-or-nothing — still holds, and is the
reason the layer returns whole rather than partially.

**Migration**: The console is now reachable only after signing in through the `aoh` realm,
and the stack requires `iams-keycloak`, `iams-aas`, `sds-server` and `valkey` to be
running. `/` leads to the sign-in flow rather than directly to the console. Operators need
a realm account that is a member of the `development` tenant and carries
`dispatch-viewer` or `dispatch-dispatcher`; the seeded accounts cover both. See
`dispatch-access-control`.
