# Database Conventions

Schema design, SQL naming, and mandatory column rules for any AOH service
that owns a database.

## Naming

- **Schema name** = the singular domain name, matching the primary
  entity table (e.g. `incident`, `notification`, `user`). Only use a
  short abbreviation (e.g. `unh`) when the full domain name exceeds
  ~12 characters or is otherwise awkward. Schema name MUST be
  configurable — projects may share a database with external services
  whose schemas could clash.
- **Table names**: singular, no plural `s`. Example: `notification`, NOT
  `notifications`.
- **Views**: prefix with `v_`. Example: `v_notification_template`. Lets
  DB UIs sort views together.
- **Association tables**: concatenate with **double** underscore
  (`<table1>__<table2>`, e.g. `user__notification`). Single underscore
  is ambiguous when source table names contain underscores (e.g.
  `user_info` + `notification` would collide with `user` +
  `info_notification`).

## Mandatory columns

Every entity table MUST have these columns, all `NOT NULL`:

| Column | Type | Default | Purpose |
|--------|------|---------|---------|
| `id` | `uuid` | `gen_random_uuid()` (or `uuid.NewV7()` from app) | Primary key. Synthetic UUIDs avoid a class of reference errors and are better for security. |
| `created_at` | `timestamp with time zone` | `now()` | UTC create timestamp |
| `updated_at` | `timestamp with time zone` | `now()` | UTC update timestamp; maintain via DB trigger |
| `created_by` | `text` |  | User reference — no FK constraint (cross-service reference) |
| `updated_by` | `text` |  | User reference — no FK constraint |
| `tenant_id` | `text` |  | Tenant reference. **TEXT, not UUID** — no FK (cross-service) |
| `occ_lock` | `int` | `0` | Optimistic concurrency lock. UPDATE queries MUST match this value; bump on each UPDATE. Acts as a version number. |

Notes:

- `tenant_id` is **`text`, not `uuid`** — by org-wide convention. AAS
  tenant IDs are server-assigned UUIDs but stored as text in service
  schemas to keep the cross-service reference loose.
- **No foreign keys across services.** FK constraints only apply within
  a single service's schema. Cross-service references (`created_by`,
  `tenant_id`, etc.) are unconstrained text.
- For human-readable IDs (e.g. `INC-20240607-0001`), add a separate
  column with a `UNIQUE` constraint across `(<column>, tenant_id)`.
- The `occ_lock` is an integer that MUST be referenced in the WHERE
  clause of UPDATE statements; if the row's `occ_lock` has changed
  since the read, the UPDATE affects 0 rows and the application should
  return a conflict error (HTTP 409).
