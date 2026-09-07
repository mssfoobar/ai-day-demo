## ADDED Requirements

### Requirement: Field unit shape

A **field unit** SHALL be represented by a typed record carrying exactly three
operator-visible attributes: a **call sign** (the human-readable name an operator uses
to address the unit), an **identifier** (a stable, unique machine key), and a
**status**. The type SHALL be exported so later changes can widen it without rewriting
consumers.

#### Scenario: Every unit carries all three attributes
- **WHEN** any unit in the roster is inspected
- **THEN** it has a non-empty call sign, a non-empty identifier, and a status

#### Scenario: Identifiers are unique
- **WHEN** the roster is loaded
- **THEN** no two units share the same identifier

### Requirement: Status vocabulary is closed

A field unit's status SHALL be exactly one of `Available`, `En route`, or `Idle`. The
status type SHALL be a closed union, so that adding a status is a deliberate,
type-checked change rather than a free-form string edit.

#### Scenario: Status is constrained at compile time
- **WHEN** a unit is declared with a status outside the three permitted values
- **THEN** `pnpm check-types` fails

#### Scenario: Displayed status matches the vocabulary
- **WHEN** a unit's status is rendered in the console
- **THEN** the displayed label is exactly `Available`, `En route`, or `Idle`

### Requirement: Baseline roster is hardcoded and deterministic

The baseline roster SHALL contain 4–5 units defined as checked-in, hardcoded data in the
app. It SHALL NOT be generated randomly, read from an environment variable, or fetched
over the network, so that every attendee starts from a byte-identical state.

#### Scenario: Roster is identical across attendees and runs
- **WHEN** two attendees load the console from the same commit, or one attendee reloads it
- **THEN** both see the same units, with the same call signs, identifiers, statuses, and ordering

#### Scenario: Roster covers the full status vocabulary
- **WHEN** the roster is loaded
- **THEN** at least one unit has status `Available`, at least one has `En route`, and at least one has `Idle`

#### Scenario: Roster size
- **WHEN** the roster is loaded
- **THEN** it contains no fewer than 4 and no more than 5 units

### Requirement: Roster is read through an accessor

The console SHALL read the roster through an exported accessor function rather than
importing the data array directly, so that a later change can swap the hardcoded source
for a service-backed one without editing the console page.

#### Scenario: Console does not import the data array
- **WHEN** the console page source is reviewed
- **THEN** it obtains units by calling the roster accessor and does not reference the underlying hardcoded array

#### Scenario: Accessor returns the full roster
- **WHEN** the accessor is called
- **THEN** it returns every unit in the roster, in the roster's declared order
