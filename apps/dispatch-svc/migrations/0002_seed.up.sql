-- Baseline roster seed.
--
-- Idempotent by construction: every statement is an UPSERT keyed on the same unique
-- constraints the schema declares, so re-running converges on identical rows rather than
-- duplicating them. The reproducibility gate depends on this.
--
-- `last_contact` is relative to now() so the console never shows a stale-looking fleet
-- weeks after the seed was written.

INSERT INTO {{SCHEMA}}.unit
    (unit_code, call_sign, status, unit_type, station, sector, radio_channel, shift,
     capabilities, last_contact,
     assignment_incident_code, assignment_title, assignment_priority, assignment_location,
     assignment_since)
VALUES
    ('FU-101', 'Alpha-1', 'Available', 'Ambulance', 'Marina Bay Station 4',
     'Sector 4 · Marina Bay', 'TAC-2', 'Day · 07:00-19:00',
     ARRAY['ALS', 'Water rescue'], now() - interval '3 minutes',
     NULL, NULL, NULL, NULL, NULL),

    ('FU-102', 'Alpha-2', 'En route', 'Ambulance', 'Marina Bay Station 4',
     'Sector 2 · Raffles Place', 'TAC-2', 'Day · 07:00-19:00',
     ARRAY['ALS'], now() - interval '1 minute',
     'INC-2841', 'Cardiac arrest', 'P1', '12 Raffles Quay', now() - interval '14 minutes'),

    ('FU-204', 'Bravo-1', 'Idle', 'Fire engine', 'Tanjong Pagar Station 2',
     'Sector 7 · Tanjong Pagar', 'TAC-4', 'Night · 19:00-07:00',
     ARRAY['Hazmat', 'Ladder 30m'], now() - interval '22 minutes',
     NULL, NULL, NULL, NULL, NULL),

    ('FU-205', 'Bravo-2', 'Available', 'Rescue tender', 'Tanjong Pagar Station 2',
     'Sector 7 · Tanjong Pagar', 'TAC-4', 'Day · 07:00-19:00',
     ARRAY['Extrication', 'Rope rescue'], now() - interval '8 minutes',
     NULL, NULL, NULL, NULL, NULL),

    -- Deliberately has NO capabilities: exercises the "none recorded" path in the UI.
    ('FU-311', 'Charlie-1', 'En route', 'Patrol car', 'Central Division',
     'Sector 1 · City Hall', 'TAC-1', 'Day · 07:00-19:00',
     ARRAY[]::text[], now() - interval '2 minutes',
     'INC-2839', 'Traffic obstruction', 'P3', 'Nicoll Highway / Republic Ave',
     now() - interval '35 minutes')
ON CONFLICT (unit_code, tenant_id) DO UPDATE SET
    call_sign                = EXCLUDED.call_sign,
    status                   = EXCLUDED.status,
    unit_type                = EXCLUDED.unit_type,
    station                  = EXCLUDED.station,
    sector                   = EXCLUDED.sector,
    radio_channel            = EXCLUDED.radio_channel,
    shift                    = EXCLUDED.shift,
    capabilities             = EXCLUDED.capabilities,
    last_contact             = EXCLUDED.last_contact,
    assignment_incident_code = EXCLUDED.assignment_incident_code,
    assignment_title         = EXCLUDED.assignment_title,
    assignment_priority      = EXCLUDED.assignment_priority,
    assignment_location      = EXCLUDED.assignment_location,
    assignment_since         = EXCLUDED.assignment_since,
    occ_lock                 = {{SCHEMA}}.unit.occ_lock + 1;

INSERT INTO {{SCHEMA}}.unit_crew (unit_id, name, role, sort_order)
SELECT u.id, c.name, c.role, c.sort_order
FROM (VALUES
    ('FU-101', 'J. Tan',       'Paramedic',    0),
    ('FU-101', 'M. Lim',       'EMT',          1),
    ('FU-102', 'S. Wong',      'Paramedic',    0),
    ('FU-102', 'K. Chua',      'EMT',          1),
    ('FU-204', 'R. Kumar',     'Officer',      0),
    ('FU-204', 'D. Ng',        'Driver',       1),
    ('FU-204', 'A. Rahman',    'Firefighter',  2),
    ('FU-205', 'P. Goh',       'Officer',      0),
    ('FU-205', 'Y. Lee',       'Firefighter',  1),
    ('FU-311', 'L. Fernandez', 'Sergeant',     0)
) AS c(unit_code, name, role, sort_order)
JOIN {{SCHEMA}}.unit u ON u.unit_code = c.unit_code
ON CONFLICT (unit_id, name) DO UPDATE SET
    role       = EXCLUDED.role,
    sort_order = EXCLUDED.sort_order,
    occ_lock   = {{SCHEMA}}.unit_crew.occ_lock + 1;
