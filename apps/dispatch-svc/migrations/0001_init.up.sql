-- Field-unit schema for dispatch-svc.
--
-- `{{SCHEMA}}` is substituted at runtime from SQL_SCHEMA_NAME (default `dispatch`).
-- The schema name is configurable per aoh-conventions: a project may share a database
-- with services whose schemas would otherwise clash.

CREATE SCHEMA IF NOT EXISTS {{SCHEMA}};

-- Maintains updated_at on every UPDATE, per aoh-conventions (trigger, not app code).
CREATE OR REPLACE FUNCTION {{SCHEMA}}.set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS {{SCHEMA}}.unit (
    -- AOH mandatory columns.
    id          uuid        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    created_by  text        NOT NULL DEFAULT 'system',
    updated_by  text        NOT NULL DEFAULT 'system',
    tenant_id   text        NOT NULL DEFAULT 'workshop',
    occ_lock    int         NOT NULL DEFAULT 0,

    -- The human-readable identifier a dispatcher actually says out loud. Kept
    -- alongside the synthetic uuid per aoh-conventions, unique within a tenant.
    unit_code     text NOT NULL,
    call_sign     text NOT NULL,
    status        text NOT NULL,
    unit_type     text NOT NULL,
    station       text NOT NULL,
    sector        text NOT NULL,
    radio_channel text NOT NULL,
    shift         text NOT NULL,
    capabilities  text[] NOT NULL DEFAULT '{}',
    last_contact  timestamptz NOT NULL DEFAULT now(),

    -- Current assignment. Embedded rather than a child table: it is 1:0..1, has no
    -- identity of its own here (the incident of record lives in another system), and a
    -- join would buy nothing. The CHECK below keeps the group all-null or all-non-null
    -- so a half-populated assignment cannot exist.
    assignment_incident_code text,
    assignment_title         text,
    assignment_priority      text,
    assignment_location      text,
    assignment_since         timestamptz,

    CONSTRAINT unit_code_unique_per_tenant UNIQUE (unit_code, tenant_id),

    -- The closed status vocabulary lives here now. The frontend used to enforce it with
    -- a TypeScript union, which cannot constrain values that originate in the database.
    CONSTRAINT unit_status_vocabulary
        CHECK (status IN ('Available', 'En route', 'Idle')),

    CONSTRAINT unit_assignment_all_or_nothing CHECK (
        (assignment_incident_code IS NULL
            AND assignment_title    IS NULL
            AND assignment_priority IS NULL
            AND assignment_location IS NULL
            AND assignment_since    IS NULL)
        OR
        (assignment_incident_code IS NOT NULL
            AND assignment_title    IS NOT NULL
            AND assignment_priority IS NOT NULL
            AND assignment_location IS NOT NULL
            AND assignment_since    IS NOT NULL)
    )
);

DROP TRIGGER IF EXISTS unit_set_updated_at ON {{SCHEMA}}.unit;
CREATE TRIGGER unit_set_updated_at
    BEFORE UPDATE ON {{SCHEMA}}.unit
    FOR EACH ROW EXECUTE FUNCTION {{SCHEMA}}.set_updated_at();

-- Crew are enumerable rows with a display order, not an opaque blob.
CREATE TABLE IF NOT EXISTS {{SCHEMA}}.unit_crew (
    id         uuid        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    created_by text        NOT NULL DEFAULT 'system',
    updated_by text        NOT NULL DEFAULT 'system',
    tenant_id  text        NOT NULL DEFAULT 'workshop',
    occ_lock   int         NOT NULL DEFAULT 0,

    -- FK is in-schema, which is where aoh-conventions permits them.
    unit_id    uuid NOT NULL REFERENCES {{SCHEMA}}.unit (id) ON DELETE CASCADE,
    name       text NOT NULL,
    role       text NOT NULL,
    sort_order int  NOT NULL DEFAULT 0,

    CONSTRAINT unit_crew_unique_per_unit UNIQUE (unit_id, name)
);

DROP TRIGGER IF EXISTS unit_crew_set_updated_at ON {{SCHEMA}}.unit_crew;
CREATE TRIGGER unit_crew_set_updated_at
    BEFORE UPDATE ON {{SCHEMA}}.unit_crew
    FOR EACH ROW EXECUTE FUNCTION {{SCHEMA}}.set_updated_at();

CREATE INDEX IF NOT EXISTS unit_crew_unit_id_idx ON {{SCHEMA}}.unit_crew (unit_id);
