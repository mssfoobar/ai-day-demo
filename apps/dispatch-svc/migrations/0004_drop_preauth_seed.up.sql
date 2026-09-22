-- Remove the pre-auth baseline roster.
--
-- `0002_seed.up.sql` inserted five units under the placeholder tenant `'workshop'`,
-- which predates authentication. Reads are now scoped to the caller's
-- `active_tenant.tenant_id`, so no token can ever see those rows — they would
-- linger invisibly, holding `unit_code`s and confusing anyone reading the table.
--
-- They are deleted rather than re-tenanted, and no positions are added here, because
-- a committed migration cannot know the tenant id (AAS assigns it at stack-up, which
-- is why project-aas's bootstrap has to look it up by name) and because rows written
-- in SQL bypass the GIS outbox and would be permanently absent from the map.
-- Seeding is service behaviour now: the first request from a `dispatch-dispatcher`
-- seeds that caller's tenant, through the ordinary write path (design.md D12).
--
-- Crew cascades from the unit delete (unit_crew.unit_id ON DELETE CASCADE).

DELETE FROM {{SCHEMA}}.unit WHERE tenant_id = 'workshop';
