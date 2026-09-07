## ADDED Requirements

### Requirement: Read API for field units

The service SHALL expose a read-only HTTP API for field units under `/v1/`, with no
`/api` prefix. It SHALL NOT expose any write operation in this change.

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

#### Scenario: No write verbs are served
- **WHEN** a client issues `POST`, `PUT`, `PATCH` or `DELETE` against `/v1/units`
- **THEN** the response status is 405

### Requirement: Responses use the AOH success envelope

Every 2xx response body SHALL be the AOH success envelope — `data`, `message`, and
`sent_at` — and SHALL NOT return a bare array or a bare object at the top level.

#### Scenario: Envelope shape
- **WHEN** any successful response is inspected
- **THEN** it has a `data` key, a `message` key, and a `sent_at` key
- **AND** `sent_at` is an ISO 8601 timestamp

### Requirement: Health probes

The service SHALL expose `GET /livez` and `GET /readyz` at the root, unauthenticated.
`readyz` SHALL report failure when the database is unreachable, so that a broken
dependency is visible rather than surfacing as a 500 on a data route.

#### Scenario: Liveness while running
- **WHEN** the process is running
- **THEN** `GET /livez` returns 200

#### Scenario: Readiness with a reachable database
- **WHEN** the database is reachable
- **THEN** `GET /readyz` returns 200

#### Scenario: Readiness with an unreachable database
- **WHEN** the database is not reachable
- **THEN** `GET /readyz` returns a non-2xx status

### Requirement: Persisted unit model

A field unit SHALL be persisted in a `dispatch` schema carrying the AOH mandatory
columns (`id`, `created_at`, `updated_at`, `created_by`, `updated_by`, `tenant_id`,
`occ_lock`), plus a human-readable `unit_code` unique per tenant. Crew members SHALL be
stored as rows related to their unit, not as an opaque blob.

#### Scenario: Human-readable code is unique per tenant
- **WHEN** two units in the same tenant are given the same `unit_code`
- **THEN** the database rejects the second one

#### Scenario: Mandatory columns are present
- **WHEN** the `unit` table is inspected
- **THEN** it has `id`, `created_at`, `updated_at`, `created_by`, `updated_by`, `tenant_id` and `occ_lock`, all NOT NULL

### Requirement: Seed data is idempotent

The service SHALL ship checked-in seed data covering the baseline roster, and applying it
repeatedly SHALL converge on the same rows rather than duplicating them.

#### Scenario: Re-running the seed
- **WHEN** the seed is applied twice against the same database
- **THEN** the unit count is the same after the second run as after the first

#### Scenario: Seeded roster covers the status vocabulary
- **WHEN** the seeded units are listed
- **THEN** at least one unit has each of `Available`, `En route` and `Idle`
