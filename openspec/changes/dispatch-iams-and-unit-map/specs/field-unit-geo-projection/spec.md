## ADDED Requirements

### Requirement: A field unit carries an optional last-known position

A field unit SHALL carry a last-known position — a longitude, a latitude, and the time the
fix was taken — or no position at all. A partially populated position SHALL be impossible:
the three values are set together or not at all. Coordinates SHALL be validated as
longitude in `[-180, 180]` and latitude in `[-90, 90]`.

#### Scenario: A positioned unit reports its fix
- **WHEN** a client reads a unit that has a position
- **THEN** the unit's `position` object carries `lon`, `lat` and `at`
- **AND** `at` is an ISO 8601 timestamp

#### Scenario: An un-positioned unit omits the key
- **WHEN** a client reads a unit with no position
- **THEN** the `position` key is absent from the unit object, not present as an empty object or nulls

#### Scenario: A half-populated position is rejected by the database
- **WHEN** a row is written with a longitude but no latitude
- **THEN** the database rejects it

#### Scenario: Out-of-range coordinates are rejected
- **WHEN** a write supplies a latitude of `91` or a longitude of `-181`
- **THEN** the response status is 400 and the error detail names the offending field

### Requirement: Every unit write enqueues a GIS projection in the same transaction

`dispatch-svc` SHALL mirror field units into `gis-service` through a transactional outbox:
each create, replace and delete SHALL write an outbox row inside the **same database
transaction** as the unit change, so the unit row and the pending projection can never
disagree. The service SHALL NOT call `gis-service` inline on the request path.

#### Scenario: A create enqueues an upsert
- **WHEN** a unit is created
- **THEN** exactly one outbox row for that `unit_code` with an upsert intent exists after the transaction commits

#### Scenario: A failed unit write enqueues nothing
- **WHEN** a unit write is rolled back
- **THEN** no outbox row for it remains

#### Scenario: GIS being down does not fail the write
- **WHEN** `gis-service` is unreachable and a dispatcher creates a unit
- **THEN** the write succeeds with its normal status
- **AND** the unit is present in `GET /v1/units`

### Requirement: The projection worker upserts a geo-entity per field unit

A background worker SHALL drain the outbox and write each unit into `gis-service` as a
geo-entity with `entity_id` equal to the unit's `unit_code`, `entity_type` `track`,
`geojson.geometry` a Point of `[lon, lat]`, and `geojson.properties` carrying `kind`
`field-unit` plus the unit's call sign and status. Delivery SHALL be retried until it
succeeds and SHALL be idempotent — replaying an outbox row SHALL NOT create a second
entity.

#### Scenario: A positioned unit appears in GIS
- **WHEN** a unit with a position is created and the worker has drained the outbox
- **THEN** `GET /geoentity/entity_id/{unit_code}` on `gis-service` returns an entity
- **AND** its `entity_type` is `track` and its `geojson.properties.kind` is `field-unit`

#### Scenario: Moving a unit moves its entity
- **WHEN** a unit's position is changed and the outbox drains
- **THEN** the entity's coordinates equal the unit's new longitude and latitude

#### Scenario: Replay does not duplicate
- **WHEN** the same outbox row is delivered twice
- **THEN** `gis-service` holds exactly one entity for that `unit_code`

#### Scenario: Delivery resumes after an outage
- **WHEN** `gis-service` is unavailable at the time of a write and becomes available afterwards
- **THEN** the entity reflects the write without any operator action

#### Scenario: An un-positioned unit has no entity
- **WHEN** a unit has no position
- **THEN** no geo-entity exists for its `unit_code`

### Requirement: Deleting a unit removes its geo-entity

Deleting a field unit SHALL enqueue a delete projection, and the worker SHALL remove the
corresponding geo-entity from `gis-service`.

#### Scenario: The entity goes with the unit
- **WHEN** a dispatcher deletes a unit and the outbox drains
- **THEN** `GET /geoentity/entity_id/{unit_code}` on `gis-service` returns 404

#### Scenario: Deleting an entity that is already gone is not an error
- **WHEN** a delete projection is delivered for a `unit_code` with no entity in GIS
- **THEN** the worker treats it as done and does not retry indefinitely

### Requirement: Entity changes reach browsers over RTUS

The projection SHALL rely on `gis-service`'s own publication to the RTUS map named `gis`;
`dispatch-svc` SHALL NOT publish to RTUS itself.

#### Scenario: A position change is observable on the RTUS map
- **WHEN** a unit's position changes and the outbox drains
- **THEN** a subscriber to the `gis` RTUS map receives an update whose key identifies that unit's entity

#### Scenario: The dispatch service holds no RTUS credentials
- **WHEN** `dispatch-svc`'s configuration is reviewed
- **THEN** it names `gis-service` as its only downstream and carries no RTUS endpoint

### Requirement: The projection is scoped to the caller's tenant

Every projected geo-entity SHALL be written under the tenant of the unit it mirrors, so a
map renders only entities the operator's tenant owns.

#### Scenario: An entity is visible to its own tenant
- **WHEN** a unit belonging to a tenant is projected and an operator of that tenant reads `GET /geoentity/entity_id/{unit_code}`
- **THEN** the entity is returned

#### Scenario: An entity is not visible to another tenant
- **WHEN** an operator of a different tenant reads the same entity id
- **THEN** the entity is not returned to them
