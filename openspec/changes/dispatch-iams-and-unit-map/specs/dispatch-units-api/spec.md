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

## MODIFIED Requirements

### Requirement: Persisted unit model

A field unit SHALL be persisted in a `dispatch` schema carrying the AOH mandatory
columns (`id`, `created_at`, `updated_at`, `created_by`, `updated_by`, `tenant_id`,
`occ_lock`), plus a human-readable `unit_code` unique per tenant. Crew members SHALL be
stored as rows related to their unit, not as an opaque blob. The schema SHALL additionally
carry a marker table recording which tenants have been seeded.

#### Scenario: Human-readable code is unique per tenant
- **WHEN** two units in the same tenant are given the same `unit_code`
- **THEN** the database rejects the second one

#### Scenario: Mandatory columns are present
- **WHEN** the `unit` table is inspected
- **THEN** it has `id`, `created_at`, `updated_at`, `created_by`, `updated_by`, `tenant_id` and `occ_lock`, all NOT NULL

#### Scenario: The seed marker table exists alongside the unit table
- **WHEN** the schema is inspected
- **THEN** a marker table is present, carrying the AOH mandatory columns and a UNIQUE constraint on `tenant_id`

### Requirement: A tenant is seeded once, on its first dispatcher request

The service SHALL seed a tenant's baseline roster the first time it serves an authenticated
request from a caller holding `dispatch-dispatcher`, before that request is answered, and
SHALL record in the same transaction that it has done so. It SHALL NOT seed again for that
tenant, and SHALL NOT decide whether to seed by checking whether the tenant currently has
units — deleting units is a legitimate operator action and must not resurrect the roster.
Seeding SHALL NOT be triggered by a caller who lacks the dispatcher role, because it is a
write. Seeded rows SHALL carry the triggering caller's `sub` and tenant, and SHALL include
crew and assignments. A migration SHALL remove the pre-auth seeded rows that predate this
change.

#### Scenario: The first dispatcher request seeds the tenant
- **WHEN** a dispatcher makes their first authenticated request against a tenant that has never been seeded
- **THEN** that request is answered with the baseline roster already present
- **AND** the stored rows carry that caller's `sub` in `created_by` and their tenant in `tenant_id`, not the pre-auth placeholders

#### Scenario: Seeding happens once, not once per request
- **WHEN** the same dispatcher makes further requests
- **THEN** no further seeding occurs and the unit, crew and assignment counts are unchanged

#### Scenario: Deleting units does not resurrect them
- **WHEN** a dispatcher deletes every unit in a seeded tenant and then makes another request
- **THEN** the roster stays empty
- **AND** no seeded unit reappears

#### Scenario: A viewer does not trigger seeding
- **WHEN** a caller holding only `dispatch-viewer` makes the first request against an unseeded tenant
- **THEN** no rows are written
- **AND** they are served an empty roster

#### Scenario: Two simultaneous dispatchers seed once between them
- **WHEN** two dispatcher requests against an unseeded tenant are served concurrently
- **THEN** the tenant is seeded exactly once, with no duplicate units or crew members
- **AND** no seeded unit's `occ_lock` is bumped by a second write

#### Scenario: A seed that cannot commit fails loudly
- **WHEN** the seed transaction cannot commit
- **THEN** the triggering request fails with a 5xx carrying a `DISPATCH_*` error code, rather than being answered with an empty roster
- **AND** no marker row survives, so the next dispatcher request attempts the seed again

#### Scenario: Seeded roster covers the status vocabulary
- **WHEN** the seeded units are listed
- **THEN** at least one unit has each of `Available`, `En route` and `Idle`

#### Scenario: Seeded units carry crew and an assignment
- **WHEN** the seeded units are listed
- **THEN** at least one carries crew members with their roles
- **AND** at least one carries an assignment, so the console's assigned-unit state and workshop Exercise 1 have a starting point

#### Scenario: The pre-auth rows are gone
- **WHEN** the database is inspected after migration
- **THEN** no unit row remains under the pre-auth placeholder tenant

