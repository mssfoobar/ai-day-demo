## ADDED Requirements

### Requirement: Status summary

The console SHALL show, above the two panes, a summary of the fleet by status — a total
and a count per status value — so a dispatcher can read availability without counting
rows.

#### Scenario: Counts reflect the roster
- **WHEN** the console is loaded
- **THEN** the total equals the number of units rendered in the list
- **AND** each status count equals the number of units carrying that status

#### Scenario: Summary is inline, not a dashboard
- **WHEN** the summary is rendered
- **THEN** it is part of the console page composed from `@mssfoobar/ui` primitives, not a separate dashboard surface

### Requirement: Unit rows carry operational context

Each row in the units list SHALL show, in addition to call sign, identifier and status, a
second line giving the unit's current context — its active assignment when it has one, and
its station otherwise — so the list is scannable without opening each unit.

#### Scenario: An assigned unit shows its assignment
- **WHEN** a unit has an active assignment
- **THEN** its row shows that assignment's reference or title

#### Scenario: An unassigned unit shows its station
- **WHEN** a unit has no active assignment
- **THEN** its row shows its station instead

### Requirement: Detail pane is sectioned

The detail pane SHALL present a selected unit's information grouped into labelled
sections — overview, assignment, crew, and capabilities — rather than one flat list of
fields.

#### Scenario: Sections are present for an assigned unit
- **WHEN** a unit with an assignment and crew is selected
- **THEN** the pane shows its overview fields, its assignment, and its crew members with their roles

#### Scenario: An unassigned unit states so explicitly
- **WHEN** a unit with no assignment is selected
- **THEN** the assignment section says the unit is unassigned rather than rendering an empty section

#### Scenario: A unit with no capabilities omits nothing silently
- **WHEN** a unit has no capability tags
- **THEN** the capabilities section is either absent or states that none are recorded

### Requirement: The console degrades when the service is unavailable

Because the roster is now fetched, the console SHALL show an explicit error state when
the dispatch service cannot be reached, rather than rendering an empty list as though the
fleet were empty.

#### Scenario: Service unreachable
- **WHEN** the console is loaded and the dispatch service cannot be reached
- **THEN** the page shows an error state explaining that units could not be loaded
- **AND** it does not present an empty units list as a successful result

#### Scenario: Error state is operational in tone
- **WHEN** the error state is shown
- **THEN** its wording is sentence case and operational, and it names no stack trace or raw exception text
