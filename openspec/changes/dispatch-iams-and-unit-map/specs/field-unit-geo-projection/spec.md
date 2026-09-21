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
each create, replace and delete — **and the one-time tenant seed**, which is the largest
batch the outbox will ever carry — SHALL write an outbox row inside the **same database
transaction** as the unit change, so the unit row and the pending projection can never
disagree. The service SHALL NOT call `gis-service` inline on the request path.

The intent recorded SHALL be derived from the unit's position **after** the write, not from
the HTTP verb: a unit that has a position enqueues an `upsert`; a unit that has none —
whether it never had one, had it cleared by a replace, or was deleted — enqueues a `delete`.

#### Scenario: A create of a positioned unit enqueues an upsert
- **WHEN** a unit with a position is created
- **THEN** exactly one outbox row for that `unit_code` with an upsert intent exists after the transaction commits

#### Scenario: A create of an un-positioned unit enqueues a delete
- **WHEN** a unit with no position is created
- **THEN** the outbox row for that `unit_code` carries a delete intent, not an upsert

#### Scenario: Clearing a position enqueues a delete
- **WHEN** a replace removes a unit's position
- **THEN** the outbox row for that `unit_code` carries a delete intent
- **AND** after the worker drains, no geo-entity exists for that `unit_code`

#### Scenario: A failed unit write enqueues nothing
- **WHEN** a unit write is rolled back
- **THEN** no outbox row for it remains

#### Scenario: GIS being down does not fail the write
- **WHEN** `gis-service` is unreachable and a dispatcher creates a unit
- **THEN** the write succeeds with its normal status
- **AND** the unit is present in `GET /v1/units`

### Requirement: The projection worker upserts a geo-entity per field unit

A background worker SHALL drain the outbox. For a unit that **has** a position it SHALL
write a geo-entity with `entity_id` equal to the unit's `unit_code`, `entity_type` `track`,
`geojson.geometry` a Point of `[lon, lat]`, and `geojson.properties` carrying `kind`
`field-unit` plus the unit's call sign and status. Delivery SHALL be retried, bounded by the
remaining lifetime of the token it carries, and SHALL be idempotent — replaying an outbox row
SHALL NOT create a second entity. A row still undelivered when that token expires SHALL
remain pending and inspectable rather than being dropped or silently retried with another
operator's credential.

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

#### Scenario: Delivery resumes after a brief outage
- **WHEN** `gis-service` is briefly unavailable at the time of a write and becomes available while the writing operator's token is still valid
- **THEN** the entity reflects the write with no operator action

#### Scenario: A longer outage leaves the row pending and visible
- **WHEN** `gis-service` is unavailable for longer than the writing operator's token lifetime
- **THEN** the outbox row is still present and marked undelivered, with its attempt count and last error recorded
- **AND** an operator re-saving that unit through the API enqueues a fresh row and delivers it on a fresh token

#### Scenario: An un-positioned unit has no entity
- **WHEN** a unit has no position and the outbox has drained
- **THEN** no geo-entity exists for its `unit_code`
- **AND** no stale marker remains at a position it previously held

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
- **THEN** `gis-service` is its only downstream for entity data, and it carries no RTUS endpoint, topic name or credential

### Requirement: The projection is written in the tenant of the operator who caused it

Because the worker carries the operator's own token, every projected geo-entity SHALL land
in that operator's tenant — which is, by construction, the tenant of the unit they wrote.
The worker SHALL NOT hold a tenant-independent credential, and SHALL NOT project a unit it
has no operator token for.

#### Scenario: The entity lands in the writing operator's tenant
- **WHEN** an operator writes a unit and the outbox drains
- **THEN** reading `GET /geoentity/entity_id/{unit_code}` with that same operator's token returns the entity

#### Scenario: No tenant-independent credential exists
- **WHEN** `internal/config` and the projection worker's source are reviewed
- **THEN** they contain no client secret, service-account credential, or client-credentials grant
