## ADDED Requirements

### Requirement: Add a unit from the console

The console SHALL offer an **Add unit** action that opens a form for the unit's editable
fields (call sign, unit code, status, type, station, sector, radio channel, shift,
capabilities) and creates the unit through the service on submit.

#### Scenario: Successful add
- **WHEN** an operator submits the form with valid values
- **THEN** the new unit appears in the list, is selected, and a success toast names it
- **AND** the form closes

#### Scenario: Add rejected by validation
- **WHEN** an operator submits with a blank required field or a duplicate unit code
- **THEN** the form stays open and shows the field-level message the service returned
- **AND** no unit is added

### Requirement: Edit a selected unit

With a unit selected, the console SHALL offer an **Edit** action that opens the same form
prefilled, and replaces the unit through the service on submit, echoing the unit's
`occ_lock`.

#### Scenario: Successful edit
- **WHEN** an operator changes the status and submits
- **THEN** the row and detail pane show the new status, and a success toast confirms it

#### Scenario: Edit of a unit changed elsewhere
- **WHEN** the unit was changed since the operator loaded it and they submit
- **THEN** the console shows a conflict message asking them to reload and retry
- **AND** the operator's edit is not applied

### Requirement: Delete a selected unit

With a unit selected, the console SHALL offer a **Delete** action behind a confirmation
that names the unit, and removes it through the service on confirm.

#### Scenario: Confirmed delete
- **WHEN** an operator confirms the deletion
- **THEN** the unit disappears from the list, the detail pane returns to its unselected state, and a toast confirms it

#### Scenario: Cancelled delete
- **WHEN** an operator dismisses the confirmation
- **THEN** nothing changes

### Requirement: Writes stay same-origin

Add, edit and delete SHALL be submitted to the SvelteKit server (form actions), which calls
the service. The browser SHALL NOT call the dispatch service directly.

#### Scenario: Write path
- **WHEN** an operator submits any of the three forms
- **THEN** the browser's request goes to its own origin and the service is called by the server

## REMOVED Requirements

### Requirement: No command actions in the baseline

**Reason**: Superseded — the console now carries add, edit and delete.

**Migration**: None needed; the read-only console is a strict subset of the new one.
