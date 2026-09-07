# DASH — Dashboard Service

## What It Does

DASH provides configurable, widget-based dashboards for the AOH platform. Users can
create custom dashboards with different widget types, arrange them, and share configurations.

**In AOH, any dashboard is a DASH surface — use DASH for it, not a bespoke page.** This holds whether the dashboard is user-customizable (users arrange their own widgets) or a fixed, dev-authored layout (a situation picture, KPI overview, ops board) that you author as widgets and seed once via the REST API. For the "should I use DASH or build it in-app?" decision criteria, see `service-selection-criteria.md` → DASH.

## Architecture

```
┌───────────────────┐   REST    ┌──────────────┐
│ Consumer app      │─────────►│ dash-service │
│ (built with       │          │ (API)        │
│  @mssfoobar/      │          └──────┬───────┘
│   dash-web-sdk)   │                 │
└───────────────────┘          ┌──────▼───────┐
                               │ DASH-DB      │
                               │ (Postgres)   │
                               └──────────────┘
```

dash-service is a layered Go service (handler → service → repo), re-homed into
the ops-hub monorepo. It migrates its schema on boot (golang-migrate; it also
issues `CREATE SCHEMA IF NOT EXISTS`).

DASH owns tags natively — tag definitions live in DASH-DB and are served by
dash-service's own `/tag` endpoints. There is no separate tag service.

DASH no longer ships a standalone dashboard management UI. Consumer apps embed
dashboard functionality by installing `@mssfoobar/dash-web-sdk` and building their
own management pages on top of it (see DASH-SDK below).

## Components

| Component | Image | Purpose |
|-----------|-------|---------|
| **dash-db** | `postgres:17.0` | Dashboard configs, widgets, favourites |
| **dash-service** | `ghcr.io/mssfoobar/ops-hub/dash-service` (v3.0.0) | Dashboard CRUD API |

## Key Concepts

### Dashboards

A dashboard is a named container for widgets:
- Users create dashboards and add widgets to them
- Dashboards can be tagged (tags are owned by DASH itself) for categorization
- Dashboards can be shared across users
- Users can favourite dashboards for quick access

### Widgets

Widgets are the building blocks of a dashboard:
- Each widget has a **type** (chart, table, map, etc.)
- Widget types define the available configuration options
- Widgets store their config (data source, filters, display options)
- Layout position and size are part of the widget config

### Categories

Widgets are organized by categories for the widget picker UI.

### Widget types

Widget types define what kinds of widgets are available:
- Each type has size constraints (min/max width and height)
- Types can be enabled/disabled via a toggle
- Types belong to a category
- Types have a configurable instance limit
- Types have an icon for the widget picker UI

### Shared config

Shared configs allow reusable widget configurations across widgets:
- Tied to a specific widget type
- Stores config as binary data (byte array)
- Can be referenced by multiple widgets via `shared_config_id`

### DASH-SDK

DASH ships a TypeScript SDK that consumer apps install and build dashboard
management UI on top of. Install with:

```bash
npm install @mssfoobar/dash-web-sdk
```

The SDK provides:

- `DashboardClient` — typed HTTP client for the dash-service REST endpoints listed
  below; methods return a `Result<T>` discriminated union rather than throwing.
- `<Grid>` — drag-and-drop widget canvas (GridStack-backed, 42-column by default).
- `<WidgetPicker>` — sidebar palette with schema-driven config UI.
- `defineWidget()` — API for authoring custom widget types (config schema,
  derived values, actions, optional custom config component).

For how to *use* the SDK — building dashboard list / create / edit pages and
authoring widgets — see the **`aoh-dashboard`** skill. This service reference
owns "what DASH exposes" plus how dashboards get *seeded* (below); consumer-side
UI patterns belong in `aoh-dashboard`.

## Seeding a dashboard

When a dashboard must exist before a user ever opens the UI — a default ops
board, an openspec-defined situation picture — you *seed* it. Three mechanisms;
choose by whether the dashboard must be **reproducible** on a fresh stack:

| Mechanism | Use when | Reproducible on `down -v && up`? | Owned by |
|---|---|---|---|
| **`dash-init` one-shot** | A dashboard must exist up front and survive a stack rebuild (openspec changes, default boards). Declarative `compose/dash/init/dashboard.yaml`, reconciled on every `compose up`. | **Yes** | `aoh-compose` (the `dash/init` reconciler) |
| **`DashboardClient` (SDK)** | Ad-hoc, interactive creation — "make me a dashboard now." A one-off script or `+server.ts` call. | No — manual run + borrowed token | `aoh-dashboard` (SDK usage) |
| **JSON import / hand-build in UI** | Exploratory, one-off layouts. | No | dash UI |

**Default to `dash-init` for anything that must be reproducible** (openspec, a
board every user sees on login). It mirrors `project-aas-init`: register the
widget types (`409 = exists`), then create each dashboard by name (resolve-by-name
→ create-on-miss, existing left as-is), so a fresh stack reconverges with no
human step. Enable it by uncommenting the `dash-init` block in
`compose/dash/compose.yml` and editing `compose/dash/init/dashboard.yaml`.

**Auth (every mechanism): a password grant as a real user — there is no
service-account path for creating dashboards.** dash-service scopes every write by
the token's `active_tenant` claim, and only interactive grants (password /
auth-code) carry it; a `client_credentials` service-account token validates but
has `active_tenant = null`, so it CANNOT create a tenant-scoped dashboard.
`dash-init` authenticates as the dev user seeded by `iams-init` (a member of the
seed tenant).

The **widget side** of seeding — which `widget_type_id`s and `config` blobs go in
the seed so they match your `defineWidget` widgets — is owned by **`aoh-dashboard`**
(it authors the widgets the seed references). This reference owns the
*mechanism + auth* context; `aoh-dashboard` owns the *widget config
contract*.

## API Endpoints (dash-service)

> **Auth required on every endpoint.** All routes below validate
> `Authorization: Bearer <jwt>` against the configured Keycloak realm
> (`IAM_URL`). There is NO unauthenticated path — calls without a token
> return `401`. dash-service is a normal RS256 JWT validator, so any token
> the realm issues passes validation. But **creating a tenant-scoped
> dashboard/widget needs an `active_tenant`-bearing token** — i.e. an
> interactive (password / auth-code) grant. A `client_credentials`
> service-account token validates but has `active_tenant = null` and so
> cannot create tenant-scoped resources. For the seeding decision +
> auth, see **Seeding a dashboard** above.
>
> Tenant scoping: dash-service reads `active_tenant` off the JWT and uses
> that to scope reads/writes — you NEVER pass `tenant_id` in the URL or
> body of dashboard/widget/category calls.

All list endpoints support pagination (`page`, `size`, `sort` query params).

### Dashboards

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/dashboard` | List dashboards (paginated) |
| `POST` | `/dashboard` | Create dashboard (with optional inline widgets) |
| `GET` | `/dashboard/id/{id}` | Get dashboard by ID (includes widgets, tags) |
| `PATCH` | `/dashboard/id/{id}` | Update dashboard by ID |
| `DELETE` | `/dashboard/id/{id}` | Delete dashboard by ID |
| `GET` | `/dashboard/name/{name}` | Get dashboard by name (includes widgets, tags) |
| `PATCH` | `/dashboard/name/{name}` | Update dashboard by name |
| `DELETE` | `/dashboard/name/{name}` | Delete dashboard by name |
| `GET` | `/dashboard/user_id/{id}` | List dashboards with user's favourite status (paginated). Supports `?name=` (partial-match filter) and `?favourite=true` (only favourited dashboards). |

### Categories

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/category` | List categories (paginated) |
| `POST` | `/category` | Create category |
| `GET` | `/category/id/{id}` | Get category by ID |
| `PATCH` | `/category/id/{id}` | Update category by ID |
| `DELETE` | `/category/id/{id}` | Delete category by ID |
| `GET` | `/category/name/{name}` | Get category by name |
| `PATCH` | `/category/name/{name}` | Update category by name |
| `DELETE` | `/category/name/{name}` | Delete category by name |

### Widgets

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/widget` | List widgets (paginated) |
| `POST` | `/widget` | Create widget |
| `PUT` | `/widget` | Upsert widget (create or update) |
| `GET` | `/widget/id/{id}` | Get widget by ID |
| `PATCH` | `/widget/id/{id}` | Update widget by ID |
| `DELETE` | `/widget/id/{id}` | Delete widget by ID |

### Widget Types

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/widget/type` | List widget types (paginated, filterable by `enabled`) |
| `POST` | `/widget/type` | Create widget type |
| `GET` | `/widget/type/id/{id}` | Get widget type by ID or name |
| `PATCH` | `/widget/type/id/{id}` | Update widget type by ID |
| `DELETE` | `/widget/type/id/{id}` | Delete widget type by ID |
| `PATCH` | `/widget/type/id/{id}/toggle-enabled` | Toggle widget type enabled status |
| `PATCH` | `/widget/type/name/{name}` | Update widget type by name |
| `DELETE` | `/widget/type/name/{name}` | Delete widget type by name |

### Shared Configs

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/widget/shared_config` | List shared configs (paginated) |
| `POST` | `/widget/shared_config` | Create shared config |
| `GET` | `/widget/shared_config/id/{id}` | Get shared config by ID |
| `PATCH` | `/widget/shared_config/id/{id}` | Update shared config by ID |
| `DELETE` | `/widget/shared_config/id/{id}` | Delete shared config by ID |
| `PATCH` | `/widget/shared_config/name/{name}` | Update shared config by name |
| `DELETE` | `/widget/shared_config/name/{name}` | Delete shared config by name |

### Dashboard Tag Mappings

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/dashboard/tag` | List all dashboard-tag mappings (paginated) |
| `POST` | `/dashboard/tag` | Create single dashboard-tag mapping |
| `GET` | `/dashboard/tag/id/{id}` | Get mapping by ID |
| `PATCH` | `/dashboard/tag/id/{id}` | Update mapping by ID |
| `DELETE` | `/dashboard/tag/id/{id}` | Delete mapping by ID |
| `GET` | `/dashboard/tag/dashboard_id/{id}` | Get all tag mappings for a dashboard |
| `POST` | `/dashboard/tag/dashboard_id/{id}` | Create multiple tag mappings (body: array of tag IDs) |
| `DELETE` | `/dashboard/tag/dashboard_id/{id}` | Delete all tag mappings for a dashboard |

### Favourites

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/favourite` | List favourites (paginated) |
| `POST` | `/favourite` | Create favourite (requires user_id + dashboard_id) |
| `DELETE` | `/favourite` | Delete favourite by composite key (user_id + dashboard_id in body) |
| `GET` | `/favourite/id/{id}` | Get favourite by ID |
| `PATCH` | `/favourite/id/{id}` | Update favourite by ID |
| `DELETE` | `/favourite/id/{id}` | Delete favourite by ID |

### Health

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/health/live` | Liveness check |
| `GET` | `/health/ready` | Readiness check |

## Data Model

```
dashboard:
  id: string (UUID)
  name: string
  description: string
  occ_lock: integer
  tenant_id: string
  created_at, updated_at: string (timestamp)
  created_by, updated_by: string

  # When fetched by ID/name, includes:
  favourite: boolean
  tags: array of { id, text, description, occ_lock }
  widgets: array of { id, widget_type_id, config, row, column, width, height, path, occ_lock }
  # occ_lock per widget is load-bearing — the dash backend rejects widget upserts
  # whose lock value doesn't match the row's current value (set_occ_lock_and_timestamp
  # trigger). Clients must echo it back on save or the update will fail.

dashboard_tag_mapping:
  id: string (UUID)
  dashboard_id: string (FK)
  tag_id: string (FK to DASH's own tag table)
  occ_lock: integer
  tenant_id: string
  created_at, updated_at: string
  created_by, updated_by: string

widget:
  id: string (UUID)
  dashboard_id: string (FK)
  widget_type_id: string (FK)
  shared_config_id: string (FK, nullable)
  config: byte array (binary config data)
  row: integer (>= 0)
  column: integer (>= 0)
  width: integer (>= 1)
  height: integer (>= 1)
  occ_lock: integer
  tenant_id: string
  created_at, updated_at: string
  created_by, updated_by: string

widget_type:
  id: string (TEXT, not UUID — can be a readable name)
  name: string
  category_id: string (FK)
  icon: string
  enabled: boolean
  limit: integer
  min_width, max_width: integer
  min_height, max_height: integer
  occ_lock: integer
  tenant_id: string
  created_at, updated_at: string
  created_by, updated_by: string

category:
  id: string (UUID)
  name: string
  description: string
  occ_lock: integer
  tenant_id: string
  created_at, updated_at: string
  created_by, updated_by: string

favourite:
  id: string (UUID)
  user_id: string (required)
  dashboard_id: string (FK, required)
  occ_lock: integer
  tenant_id: string
  created_at, updated_at: string
  created_by, updated_by: string

shared_config:
  id: string (UUID)
  name: string
  description: string
  widget_type_id: string (FK)
  config: byte array (binary config data)
  occ_lock: integer
  tenant_id: string
  created_at, updated_at: string
  created_by, updated_by: string
```

## Access

| URL | What |
|-----|------|
| `http://dash.${DEV_DOMAIN}` | dash-service API |
| `http://dash.${DEV_DOMAIN}/swagger-ui/index.html` | Swagger UI |
| (consumer app's own routes) | Dashboard UI — built with `@mssfoobar/dash-web-sdk` in your SvelteKit app |

## Dependencies

- **iams** (Keycloak for auth)
