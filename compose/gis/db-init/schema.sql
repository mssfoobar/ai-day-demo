-- gis-service runs its own table migrations on startup (golang-migrate) but does
-- NOT issue CREATE SCHEMA — it only sets search_path from SQL_SCHEMA_NAME (the
-- legacy "preliquibase" step created the schema out-of-band). This init script
-- creates the schema namespace so the app has somewhere to put its tables. Do
-- NOT add table definitions here — they would race the in-app migrations.
CREATE SCHEMA IF NOT EXISTS "gis";
