-- Position on a field unit, plus the GIS projection outbox and the per-tenant seed marker.
--
-- `{{SCHEMA}}` is substituted at runtime from SQL_SCHEMA_NAME (default `dispatch`).
-- Additive: every column is nullable or defaulted, so a rolled-back binary ignores them.

-- --------------------------------------------------------------------------- --
-- Last known position.
--
-- Held here, not in gis-service: dispatch-svc owns the position and GIS holds a
-- mirror (design.md D7), so "where is this unit" never needs a network hop.
-- `position_at` is the time the *fix* was taken and moves independently of
-- `last_contact` — a unit can be heard from without reporting a location (D11).
-- --------------------------------------------------------------------------- --
ALTER TABLE {{SCHEMA}}.unit
    ADD COLUMN IF NOT EXISTS position_lon double precision,
    ADD COLUMN IF NOT EXISTS position_lat double precision,
    ADD COLUMN IF NOT EXISTS position_at  timestamptz;

ALTER TABLE {{SCHEMA}}.unit
    DROP CONSTRAINT IF EXISTS unit_position_all_or_nothing;
ALTER TABLE {{SCHEMA}}.unit
    ADD CONSTRAINT unit_position_all_or_nothing CHECK (
        (position_lon IS NULL AND position_lat IS NULL AND position_at IS NULL)
        OR
        (position_lon IS NOT NULL AND position_lat IS NOT NULL AND position_at IS NOT NULL)
    );

-- Range checks pass on NULL (a CHECK is satisfied when it evaluates to unknown),
-- so they compose with the all-or-nothing rule above rather than fighting it.
ALTER TABLE {{SCHEMA}}.unit
    DROP CONSTRAINT IF EXISTS unit_position_lon_range;
ALTER TABLE {{SCHEMA}}.unit
    ADD CONSTRAINT unit_position_lon_range CHECK (position_lon BETWEEN -180 AND 180);

ALTER TABLE {{SCHEMA}}.unit
    DROP CONSTRAINT IF EXISTS unit_position_lat_range;
ALTER TABLE {{SCHEMA}}.unit
    ADD CONSTRAINT unit_position_lat_range CHECK (position_lat BETWEEN -90 AND 90);

-- --------------------------------------------------------------------------- --
-- GIS projection outbox.
--
-- A row is written in the SAME transaction as the unit change it describes, so
-- the unit row and its pending projection can never disagree (design.md D6). The
-- worker drains it; delivery is at-least-once and `PUT /geoentity` is an upsert,
-- which makes replay a no-op.
--
-- `intent` is derived from the unit's position AFTER the write, not from the HTTP
-- verb (D6a): no position — never had one, cleared, or deleted — means `delete`.
--
-- The row deliberately carries NO credential. The projection travels on the
-- writing operator's own bearer, held in memory for the life of that delivery
-- (D2a); a token in this table would be a persisted credential and would expire
-- anyway.
-- --------------------------------------------------------------------------- --
CREATE TABLE IF NOT EXISTS {{SCHEMA}}.gis_outbox (
    -- AOH mandatory columns.
    id          uuid        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    created_by  text        NOT NULL,
    updated_by  text        NOT NULL,
    tenant_id   text        NOT NULL,
    occ_lock    int         NOT NULL DEFAULT 0,

    unit_code    text  NOT NULL,
    intent       text  NOT NULL,
    payload      jsonb NOT NULL,
    attempts     int   NOT NULL DEFAULT 0,
    last_error   text,
    delivered_at timestamptz,

    CONSTRAINT gis_outbox_intent_vocabulary CHECK (intent IN ('upsert', 'delete'))
);

DROP TRIGGER IF EXISTS gis_outbox_set_updated_at ON {{SCHEMA}}.gis_outbox;
CREATE TRIGGER gis_outbox_set_updated_at
    BEFORE UPDATE ON {{SCHEMA}}.gis_outbox
    FOR EACH ROW EXECUTE FUNCTION {{SCHEMA}}.set_updated_at();

-- Pending rows are the ones anyone ever looks for — an operator diagnosing a
-- stranded projection, and the tests asserting one was enqueued.
CREATE INDEX IF NOT EXISTS gis_outbox_pending_idx
    ON {{SCHEMA}}.gis_outbox (tenant_id, unit_code)
    WHERE delivered_at IS NULL;

-- --------------------------------------------------------------------------- --
-- Per-tenant seed marker.
--
-- The rule is "seed once per tenant, ever" — NOT "seed when the tenant has no
-- units". Deleting units is a legitimate operator action and must not resurrect
-- the roster (design.md D12). A ledger table like this one and `schema_migration`
-- carries no audit columns: it records that an event happened, it is not an
-- entity anyone reads back.
-- --------------------------------------------------------------------------- --
CREATE TABLE IF NOT EXISTS {{SCHEMA}}.tenant_seed (
    tenant_id text        NOT NULL PRIMARY KEY,
    seeded_at timestamptz NOT NULL DEFAULT now(),
    seeded_by text        NOT NULL
);
