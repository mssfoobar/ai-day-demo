## ADDED Requirements

### Requirement: Reads and writes are scoped to the caller's tenant

Every `/v1/units` operation SHALL be confined to the tenant named by the caller's
`active_tenant.tenant_id` claim. A unit belonging to another tenant SHALL be invisible and
unwritable — indistinguishable from one that does not exist.

> Verifiable without standing up a second identity: insert a row carrying a different
> `tenant_id` directly with `psql`, then exercise the API with the seeded operator's token.
> The predicate under test is the `WHERE tenant_id = $n` clause, not Keycloak.

#### Scenario: A list returns only the caller's tenant
- **WHEN** a unit row belonging to another tenant exists and a caller lists units
- **THEN** that row is absent from the response
- **AND** the caller's own units are all present

#### Scenario: Another tenant's unit is not found
- **WHEN** a caller issues `GET /v1/units/{unit_code}` for a unit row that exists only under another `tenant_id`
- **THEN** the response status is 404

#### Scenario: Another tenant's unit cannot be written
- **WHEN** a caller with the dispatcher role issues `PUT` or `DELETE` against a unit row that exists only under another `tenant_id`
- **THEN** the response status is 404
- **AND** that row is unchanged

#### Scenario: The same unit code may exist in two tenants
- **WHEN** a unit row with a given `unit_code` exists under another `tenant_id` and a dispatcher creates a unit with that same `unit_code`
- **THEN** the create succeeds
- **AND** both rows exist, one per tenant

### Requirement: Audit columns carry the caller's identity

`created_by` and `updated_by` SHALL be set from the caller's `sub` claim, and `tenant_id`
from `active_tenant.tenant_id`. The literals `'system'` and `'workshop'` SHALL no longer be
written by the service on a caller-initiated write.

#### Scenario: A create records its author
- **WHEN** a dispatcher creates a unit
- **THEN** the stored row's `created_by` and `updated_by` equal that caller's `sub`
- **AND** its `tenant_id` equals that caller's `active_tenant.tenant_id`

#### Scenario: An edit records the editor
- **WHEN** a dispatcher replaces a unit created earlier
- **THEN** the row's `updated_by` is that caller's `sub`
- **AND** its `created_by` is unchanged from the create

#### Scenario: Identity is not taken from the request body
- **WHEN** a write body includes `created_by`, `updated_by` or `tenant_id`
- **THEN** those values are ignored and the token's values are stored

### Requirement: The unit resource carries `position`

A unit returned by the API SHALL include a `position` object — `lon`, `lat` and `at` —
when it has one, and SHALL omit the key entirely when it does not. `position` SHALL be
settable on create and replace by a caller with the dispatcher role.

#### Scenario: Position on read
- **WHEN** a client lists or fetches a unit that has a position
- **THEN** that unit carries a `position` with numeric `lon` and `lat` and an ISO 8601 `at`

#### Scenario: Position omitted on read
- **WHEN** a client fetches a unit with no position
- **THEN** the `position` key is absent

#### Scenario: Setting a position
- **WHEN** a dispatcher creates or replaces a unit with a valid `position`
- **THEN** the response's `data` carries that position
- **AND** a subsequent read returns it

#### Scenario: Clearing a position
- **WHEN** a dispatcher replaces a unit with no `position` in the body
- **THEN** the stored unit has no position and subsequent reads omit the key

#### Scenario: Position changes bump the version
- **WHEN** a position is set or cleared through a replace
- **THEN** the unit's `occ_lock` increments by one, exactly as any other replace

## MODIFIED Requirements

### Requirement: Persisted unit model

A field unit SHALL be persisted in a `dispatch` schema carrying the AOH mandatory
columns (`id`, `created_at`, `updated_at`, `created_by`, `updated_by`, `tenant_id`,
`occ_lock`), plus a human-readable `unit_code` unique per tenant, plus an optional
position held as `position_lon`, `position_lat` and `position_at` constrained all-or-
nothing and to valid coordinate ranges. Crew members SHALL be stored as rows related to
their unit, not as an opaque blob. The schema SHALL additionally carry an outbox table
holding pending GIS projections, written in the same transaction as the unit change it
describes.

#### Scenario: Human-readable code is unique per tenant
- **WHEN** two units in the same tenant are given the same `unit_code`
- **THEN** the database rejects the second one

#### Scenario: Mandatory columns are present
- **WHEN** the `unit` table is inspected
- **THEN** it has `id`, `created_at`, `updated_at`, `created_by`, `updated_by`, `tenant_id` and `occ_lock`, all NOT NULL

#### Scenario: Position columns are all-or-nothing
- **WHEN** the `unit` table is inspected
- **THEN** a constraint requires `position_lon`, `position_lat` and `position_at` to be all null or all non-null
- **AND** a constraint restricts `position_lon` to `[-180, 180]` and `position_lat` to `[-90, 90]`

#### Scenario: The outbox table exists alongside the unit table
- **WHEN** the schema is inspected
- **THEN** an outbox table is present carrying, per row, the target `unit_code`, the projection intent, the payload, and its delivery state

### Requirement: Seed data is idempotent

`POST /v1/units/seed` SHALL write the baseline roster into the caller's tenant and SHALL
converge on the same rows when applied repeatedly rather than duplicating them. It SHALL
require the `dispatch-dispatcher` role. The seed SHALL be owned by the service rather than by
a SQL migration or an external script, because the tenant the rows belong to is assigned by
AAS at stack-up time and is not knowable to a committed migration, because rows inserted
directly bypass the GIS projection, and because crew and assignment are not expressible
through the write API — so anything outside the service would have to reach around it for
exactly the two shapes it withholds. Seeded units SHALL carry positions, crew and
assignments, and SHALL enqueue the same projections any other write enqueues. A migration
SHALL remove the pre-auth seeded rows that predate this change.

#### Scenario: Re-running the seed
- **WHEN** the seed endpoint is called twice against the same stack
- **THEN** the second call succeeds
- **AND** the unit count, crew count and assignment count are the same after it as after the first

#### Scenario: Seeding requires the dispatcher role
- **WHEN** a caller holding only `dispatch-viewer` calls the seed endpoint
- **THEN** the response status is 403 and no row is written

#### Scenario: Seeded rows belong to the caller's tenant
- **WHEN** a dispatcher seeds and then lists units
- **THEN** the seeded roster is returned
- **AND** the stored rows carry that caller's `sub` in `created_by` and their tenant in `tenant_id`, not the pre-auth placeholders

#### Scenario: Seeded roster covers the status vocabulary
- **WHEN** the seeded units are listed
- **THEN** at least one unit has each of `Available`, `En route` and `Idle`

#### Scenario: Seeded units carry crew and an assignment
- **WHEN** the seeded units are listed
- **THEN** at least one carries crew members with their roles
- **AND** at least one carries an assignment, so the console's assigned-unit state and workshop Exercise 1 have a starting point

#### Scenario: Seeded units carry positions
- **WHEN** the seeded units are listed
- **THEN** at least one carries a `position`, one carries none, and every seeded position is within valid coordinate ranges

#### Scenario: The pre-auth rows are gone
- **WHEN** the database is inspected after migration
- **THEN** no unit row remains under the pre-auth placeholder tenant

#### Scenario: Seeded units reach the map
- **WHEN** the stack is brought up fresh, the seed endpoint is called, and the outbox drains
- **THEN** each seeded, positioned unit has a corresponding geo-entity in `gis-service`
