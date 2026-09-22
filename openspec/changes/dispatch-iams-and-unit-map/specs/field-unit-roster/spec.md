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
| **position** | Where the unit was last located — a longitude, a latitude, and the **fix time** the location was reported — or nothing. |
| **last contact** | When the unit was last heard from. Distinct from the fix time: a unit can be heard from without reporting a position. |

#### Scenario: Every unit carries the core attributes
- **WHEN** any unit in the roster is inspected
- **THEN** it has a non-empty call sign, a non-empty identifier, a status, a unit type, a station, and a last-contact time

#### Scenario: Identifiers are unique
- **WHEN** the roster is loaded
- **THEN** no two units share the same identifier

#### Scenario: An unassigned unit has no assignment
- **WHEN** a unit is not committed to an incident
- **THEN** its assignment is absent rather than an empty placeholder object

#### Scenario: An un-positioned unit has no position
- **WHEN** a unit has never reported a location
- **THEN** its position is absent rather than a zero coordinate or an empty placeholder object

#### Scenario: A positioned unit carries its fix time
- **WHEN** a unit has a position
- **THEN** that position carries a longitude, a latitude, and the time the fix was taken

#### Scenario: Position and last contact are independent
- **WHEN** a unit is edited without reporting a new location
- **THEN** its last-contact time advances and its fix time does not
