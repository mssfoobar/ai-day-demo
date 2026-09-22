-- Incident detail on a unit's assignment: what the incident is, and where it is.
--
-- `{{SCHEMA}}` is substituted at runtime from SQL_SCHEMA_NAME (default `dispatch`).
-- Additive: every column is nullable or backfilled, so a rolled-back binary ignores them.

ALTER TABLE {{SCHEMA}}.unit
    ADD COLUMN IF NOT EXISTS assignment_description text,
    ADD COLUMN IF NOT EXISTS assignment_lon         double precision,
    ADD COLUMN IF NOT EXISTS assignment_lat         double precision;

-- Assignments written before this migration have no description, and the rule below
-- would reject them. The title is the only sentence on the row to carry over.
UPDATE {{SCHEMA}}.unit
   SET assignment_description = assignment_title
 WHERE assignment_incident_code IS NOT NULL
   AND assignment_description IS NULL;

-- Description joins the all-or-nothing group: an incident nobody described is a
-- half-populated assignment, which is what this rule exists to forbid.
ALTER TABLE {{SCHEMA}}.unit
    DROP CONSTRAINT IF EXISTS unit_assignment_all_or_nothing;
ALTER TABLE {{SCHEMA}}.unit
    ADD CONSTRAINT unit_assignment_all_or_nothing CHECK (
        (assignment_incident_code IS NULL
            AND assignment_title       IS NULL
            AND assignment_description IS NULL
            AND assignment_priority    IS NULL
            AND assignment_location    IS NULL
            AND assignment_since       IS NULL)
        OR
        (assignment_incident_code IS NOT NULL
            AND assignment_title       IS NOT NULL
            AND assignment_description IS NOT NULL
            AND assignment_priority    IS NOT NULL
            AND assignment_location    IS NOT NULL
            AND assignment_since       IS NOT NULL)
    );

-- The priority vocabulary belongs here for the same reason the status one does: a
-- TypeScript union cannot constrain values that originate in the database.
ALTER TABLE {{SCHEMA}}.unit
    DROP CONSTRAINT IF EXISTS unit_assignment_priority_vocabulary;
ALTER TABLE {{SCHEMA}}.unit
    ADD CONSTRAINT unit_assignment_priority_vocabulary CHECK (
        assignment_priority IS NULL OR assignment_priority IN ('P1', 'P2', 'P3')
    );

-- The point is a pair, and optional within an assignment: a call is taken by address
-- and may be resolved to a coordinate later, or never.
ALTER TABLE {{SCHEMA}}.unit
    DROP CONSTRAINT IF EXISTS unit_assignment_point_all_or_nothing;
ALTER TABLE {{SCHEMA}}.unit
    ADD CONSTRAINT unit_assignment_point_all_or_nothing CHECK (
        (assignment_lon IS NULL AND assignment_lat IS NULL)
        OR
        (assignment_lon IS NOT NULL AND assignment_lat IS NOT NULL)
    );

-- A point with no assignment to hang off would never be read.
ALTER TABLE {{SCHEMA}}.unit
    DROP CONSTRAINT IF EXISTS unit_assignment_point_needs_assignment;
ALTER TABLE {{SCHEMA}}.unit
    ADD CONSTRAINT unit_assignment_point_needs_assignment CHECK (
        assignment_lon IS NULL OR assignment_incident_code IS NOT NULL
    );

-- Range checks pass on NULL, so they compose with the pair rule above.
ALTER TABLE {{SCHEMA}}.unit
    DROP CONSTRAINT IF EXISTS unit_assignment_lon_range;
ALTER TABLE {{SCHEMA}}.unit
    ADD CONSTRAINT unit_assignment_lon_range CHECK (assignment_lon BETWEEN -180 AND 180);

ALTER TABLE {{SCHEMA}}.unit
    DROP CONSTRAINT IF EXISTS unit_assignment_lat_range;
ALTER TABLE {{SCHEMA}}.unit
    ADD CONSTRAINT unit_assignment_lat_range CHECK (assignment_lat BETWEEN -90 AND 90);
