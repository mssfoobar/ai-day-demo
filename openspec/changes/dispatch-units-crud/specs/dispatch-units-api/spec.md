## ADDED Requirements

### Requirement: Create a field unit

`POST /v1/units` SHALL create a unit from a JSON body carrying `unit_code`, `call_sign`,
`status`, `unit_type`, `station`, `sector`, `radio_channel` and `shift`. `capabilities` is
optional and defaults to empty. Crew and assignment are not settable in this change.

#### Scenario: Valid create
- **WHEN** a client posts a body with every required field and a status in the vocabulary
- **THEN** the response status is 201
- **AND** the body is the AOH success envelope with `data` as the created unit, including `occ_lock` 0

#### Scenario: Missing or blank required field
- **WHEN** a client posts a body missing `call_sign`
- **THEN** the response status is 400
- **AND** the error payload carries `errorCode` `DISPATCH_UNIT_INVALID` and a detail naming `call_sign`

#### Scenario: Status outside the vocabulary
- **WHEN** a client posts `status` `Out of service`
- **THEN** the response status is 400 and the detail names `status`

#### Scenario: Duplicate unit code
- **WHEN** a client posts a `unit_code` that already exists for the tenant
- **THEN** the response status is 409
- **AND** the error payload carries `errorCode` `DISPATCH_UNIT_CODE_TAKEN`

### Requirement: Replace a field unit

`PUT /v1/units/{unit_code}` SHALL replace the unit's editable fields (everything a create
accepts except `unit_code`) and SHALL require the caller's `occ_lock`.

#### Scenario: Valid replace with current occ_lock
- **WHEN** a client puts a valid body whose `occ_lock` matches the stored value
- **THEN** the response status is 200
- **AND** `data` is the updated unit with `occ_lock` incremented by one and `last_contact` set to the time of the write

#### Scenario: Stale occ_lock
- **WHEN** a client puts a body whose `occ_lock` does not match the stored value
- **THEN** the response status is 409
- **AND** the error payload carries `errorCode` `DISPATCH_UNIT_STALE`
- **AND** the stored unit is unchanged

#### Scenario: Missing occ_lock
- **WHEN** a client puts a body without `occ_lock`
- **THEN** the response status is 400

#### Scenario: Unknown unit
- **WHEN** a client puts to a `unit_code` that does not exist
- **THEN** the response status is 404

### Requirement: Delete a field unit

`DELETE /v1/units/{unit_code}?occ_lock=N` SHALL remove the unit and its crew rows, and
SHALL require the caller's `occ_lock` as a query parameter.

#### Scenario: Valid delete
- **WHEN** a client deletes with the current `occ_lock`
- **THEN** the response status is 204 with no body
- **AND** a subsequent `GET /v1/units/{unit_code}` returns 404

#### Scenario: Stale or missing occ_lock on delete
- **WHEN** a client deletes with a mismatched `occ_lock`, or none
- **THEN** the response status is 409 or 400 respectively
- **AND** the unit still exists

### Requirement: Writes persist

A successful write SHALL be durable: it SHALL survive a service restart and be visible to
a fresh client.

#### Scenario: Restart after a write
- **WHEN** a unit is created, then the service process is stopped and started again
- **THEN** `GET /v1/units/{unit_code}` returns the created unit

### Requirement: The unit resource carries `occ_lock`

Every unit returned by the API SHALL include its integer `occ_lock`, so a client can echo
it on the next write.

#### Scenario: occ_lock on read
- **WHEN** a client lists or fetches units
- **THEN** each unit has an integer `occ_lock` field

## MODIFIED Requirements

### Requirement: Read API for field units

The service SHALL expose an HTTP API for field units under `/v1/`, with no `/api` prefix.
Reads SHALL be `GET /v1/units` and `GET /v1/units/{unit_code}`; writes are specified by
the requirements above. Verbs with no defined meaning SHALL return 405.

#### Scenario: Listing units
- **WHEN** a client issues `GET /v1/units`
- **THEN** the response status is 200
- **AND** the body is the AOH success envelope with `data` as an array of units

#### Scenario: Fetching one unit
- **WHEN** a client issues `GET /v1/units/{unit_code}` for a unit that exists
- **THEN** the response status is 200
- **AND** the body's `data` is that single unit object

#### Scenario: Fetching a unit that does not exist
- **WHEN** a client issues `GET /v1/units/{unit_code}` for an unknown code
- **THEN** the response status is 404
- **AND** the body is a structured error payload, never an empty body

#### Scenario: Undefined verbs are rejected
- **WHEN** a client issues `PATCH /v1/units/{unit_code}`, or `PUT` / `DELETE` against the collection `/v1/units`
- **THEN** the response status is 405

## REMOVED Requirements

### Requirement: No write verbs are served

**Reason**: Superseded — create, replace and delete are now defined above.

**Migration**: Clients that relied on 405 for every non-GET verb should expect 201 / 200 /
204 for the defined writes; `PATCH` and collection-level `PUT`/`DELETE` still return 405.
