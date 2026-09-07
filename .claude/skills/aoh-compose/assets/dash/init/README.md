# dash-init — optional dashboard seed

A one-shot that provisions dashboards into `dash-service` **before first use**, from
`dashboard.yaml`. It mirrors the `project-aas-init` pattern: read desired state
from YAML, reconcile against dash-service, idempotently.

## Do you need this?

**No, by default.** Dashboards are normally created by an admin in the UI at
runtime (dash-service persists them per tenant). Seeding is only for when a
dashboard must exist up front — e.g. a default operations dashboard every user
sees on login, reproducible on a fresh `down -v && up`.

- **Admin creates dashboards at runtime** → you don't need this. Leave the
  `dash-init` block in `../compose.yml` commented out (you may delete this
  `init/` dir too).
- **You want dashboards seeded beforehand** → customize `dashboard.yaml`, enable
  the seed (below).

## Enabling the seed

`dash-init` ships **commented out** in `../compose.yml`, so it's off by default.
Uncomment the whole `dash-init` block there to enable it. Once uncommented it
runs on **every** `compose up` (no profile), so the seed is reproducible on a
fresh `down -v && up` — the same way `project-aas-init` runs unconditionally:

```bash
cd compose
podman compose up -d dash-init   # or: docker compose ...; or just `compose up`
```

It exits 0 when done. Re-running converges (existing dashboards are left as-is),
so running it on every up is harmless.

## How it works

- **Auth:** password grant as the dev user. dash-service scopes by the token's
  `active_tenant` claim, which only interactive grants carry — a
  `client_credentials` service-account token's membership shows in `all_tenants`
  but `active_tenant` stays `null`, so it can't create a tenant-scoped
  dashboard. The dev user is a member of the seed tenant via `iams-init`.
- **Reconcile:** registers `widget_types` (409 = already present), then creates
  each dashboard by name (resolve-by-name → create on miss; dash-service returns
  `500 "sql: no rows in result set"`, not 404, when absent — handled).

## Editing `dashboard.yaml`

`widget_type_id`s MUST match the `defineWidget` types your consumer app
registers, and each `config` blob must match that widget's config schema. See
the **aoh-dashboard** skill for widget authoring and the 42-column grid.

> Requires `dash-service` to have run its startup schema migration — the asset gates
> `dash-service` on `dash-db`'s healthcheck so the tables exist before seeding.
