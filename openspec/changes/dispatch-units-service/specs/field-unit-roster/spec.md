## MODIFIED Requirements

### Requirement: Field unit shape

A **field unit** SHALL be represented by a typed record carrying the operator-visible
attributes below. The type SHALL be exported so later changes can widen it without
rewriting consumers.

| Attribute | Meaning |
| --- | --- |
| **call sign** | The human-readable name an operator uses to address the unit. |
| **identifier** | A stable, unique machine key (`unit_code`). |
| **status** | One of the closed status vocabulary. |
| **unit type** | What kind of resource it is (e.g. ambulance, fire engine, patrol car). |
| **station** | The unit's home base. |
| **sector** | The area it is currently working. |
| **radio channel** | The talkgroup an operator hails it on. |
| **shift** | The duty period it is currently working. |
| **capabilities** | Zero or more tags describing what it can do (e.g. `ALS`). |
| **crew** | Zero or more people, each with a name and a role. |
| **assignment** | The incident it is currently committed to, or nothing. |
| **last contact** | When the unit was last heard from. |

#### Scenario: Every unit carries the core attributes
- **WHEN** any unit in the roster is inspected
- **THEN** it has a non-empty call sign, a non-empty identifier, a status, a unit type, a station, and a last-contact time

#### Scenario: Identifiers are unique
- **WHEN** the roster is loaded
- **THEN** no two units share the same identifier

#### Scenario: An unassigned unit has no assignment
- **WHEN** a unit is not committed to an incident
- **THEN** its assignment is absent rather than an empty placeholder object

### Requirement: Baseline roster is served by the dispatch service

The roster SHALL be read from the `dispatch-svc` API rather than from hardcoded in-app
data. The read SHALL happen on the SvelteKit **server**, so the browser never contacts
the service directly and no service URL reaches the browser bundle.

#### Scenario: Roster comes from the service
- **WHEN** the console is loaded and the service is reachable
- **THEN** the units rendered are those the service returns

#### Scenario: The browser does not call the service
- **WHEN** the console is loaded
- **THEN** the browser issues no request to the dispatch service's origin

#### Scenario: Roster is deterministic for a given database
- **WHEN** the console is loaded twice against the same seeded database
- **THEN** the same units are rendered, in the same order

### Requirement: Roster is read through an accessor

The console SHALL obtain units through an exported accessor rather than constructing
HTTP calls inline, so the transport can change without editing the page.

#### Scenario: Console does not build requests itself
- **WHEN** the console's server load function is reviewed
- **THEN** it obtains units by calling the roster accessor and contains no URL or fetch wiring of its own

## REMOVED Requirements

### Requirement: Status vocabulary is closed

**Reason**: The vocabulary itself is unchanged (`Available`, `En route`, `Idle`), but it is
no longer enforced by a TypeScript union at compile time — the values now originate in the
database, so a compile-time check on the frontend type cannot constrain them. The
constraint moves to the persistence layer and to a runtime guard when decoding the
service response.

**Migration**: The closed vocabulary is now asserted by a database `CHECK` constraint on
`unit.status` (see `dispatch-units-api`) and by the frontend rejecting a unit whose status
is outside the known set. The `UnitStatus` TypeScript type remains as documentation of the
expected values.

### Requirement: Baseline roster is hardcoded and deterministic

**Reason**: Superseded by "Baseline roster is served by the dispatch service" above — the
roster is no longer hardcoded in the app.

**Migration**: The same five units are now checked-in seed data applied to the `dispatch`
schema, so the roster remains identical for every attendee against a freshly seeded
database.
