# Form

## What It Does

Form is a full-stack form builder and submission management system for the AOH platform.
Administrators design dynamic forms visually using a drag-and-drop interface (powered by
SurveyJS); end-users fill and submit those forms with full lifecycle management including
drafts, versioning, state machine workflows, and audit logging.

Use Form when you need configurable, structured data collection with role-based access
control over who can submit and who can advance a submission through workflow states.

## Architecture

```
┌──────────────┐   /aoh/gateway   ┌──────────────┐
│  host app    │─────────────────►│ form-service │
│ (form SDK)   │                  │  (REST API)  │
└──────────────┘                  └──────┬───────┘
                                         │
                                  ┌──────▼───────┐   S3/API  ┌──────────┐
                                  │  form-db     │──────────►│  AMM     │
                                  │ (PostgreSQL) │           │(files)   │
                                  └──────────────┘           └──────────┘
```

The UI is no longer a pair of standalone SPAs — a host app composes the form
SDK trio (see *Frontend / SDK integration*) and routes it through its shared
`/aoh/gateway` BFF to `form-service`.

## Components

| Component | Image | Purpose |
|-----------|-------|---------|
| **form-db** | `postgres:17.0` | Stores form definitions, versions, submissions, themes, audit logs |
| **form-service** | `ghcr.io/mssfoobar/ops-hub/form-service` (v1.0.0) | Layered Go REST API (handler → service → repo) on chi + sqlx + Viper; handles form management, versioning, submissions |

## Key Concepts

### Form Lifecycle

```
Form Created
     │
     ▼
Form Draft ──(edit freely)──▶ Form Draft
     │
     │ publish
     ▼
Form Version (immutable)
     │
     │ mark as current
     ▼
Form Published Version  ◄── only version accepting live submissions
     │
     │ further edits? create a new draft
     ▼
Form Draft
```

- **Form** — top-level entity, scoped per tenant, name must be unique within a tenant
- **Form Draft** — editable work-in-progress; only the creator can edit it, multiple drafts
  can exist per form simultaneously
- **Form Version** — immutable snapshot created by publishing a draft; cannot be modified
- **Form Published Version** — pointer to the one active version accepting submissions; only
  one per form at a time

### Submission Lifecycle

```
User Opens Form (Published Version)
      │
      ▼
Submission Draft ──(autosave every 10s)──▶ Submission Draft
      │
      │ submit
      ▼
Submission (Initial State)
      │
      │ role-permitted state transitions
      ▼
Submission (State N)  ──▶  any configured state
```

- **Submission** — permanently linked to the exact Form Version it was submitted against
- **Submission Draft** — in-progress save; autosaved every 10 s by default, resumable
- **State machine** — forms define configurable states and transitions; state changes are
  controlled by RBAC (which roles can trigger which transitions)

### Schema-Driven Storage

When a form is published, form-service parses the SurveyJS JSON schema and automatically
generates a dedicated PostgreSQL table to store typed submission responses, named
`ft_<first 18 chars of the form UUID, dashes removed>`. Nested elements
(paneldynamic, matrix, multipletext) get child tables under it. On subsequent
publishes, new columns are added for new questions — existing columns are never
dropped, and a type change is rejected outright, preserving historical data.

Column types follow the question's `inputType`:

| Question | Column | |
|---|---|---|
| `date` | `DATE` | zone-free: a date has no instant, so shifting it between zones would move it a day |
| `time` | `TIME` | zone-free: Postgres deprecates `time with time zone`, and a bare clock has no date to resolve DST against |
| `datetime-local` | `TIMESTAMPTZ` | an instant, so the moment is unambiguous to anything querying these tables directly |

**A datetime with no offset means UTC**, consistently on all three paths: the
pool pins `timezone=UTC` (otherwise `::timestamptz` would resolve a zone-less
literal in whatever timezone the deployment's session happened to use), the read
path returns datetime answers as RFC3339 instants (`2026-07-01T10:30:00Z`), and
migration `0002` reads legacy naive columns that way when converting them.

> Tables generated before `datetime-local` became zone-aware hold naive
> `TIMESTAMP` columns. `0002_normalize_timestamps` converts them on the first
> boot of form-service ≥ 1.1.0, under an `ACCESS EXCLUSIVE` lock.

### AMM Integration

Form integrates the AMM attachment-management service for image attachments embedded in form
layouts and for file upload questions. Attachment IDs are stored in `form_draft.image_ids` and
`form_version.image_ids`. Stale attachments are cleaned up automatically by form-service's
background task (`stale_duration: 12h` by default).

### Configuration

form-service reads a mounted `config.yaml` — blocks `admin_roles`, `log`, `http`,
`sql`, `iams`, and `amm` — with environment overrides for secrets:
`SQL_USER` / `SQL_PASSWORD`, `IAMS_KEYCLOAK_CLIENT_ID` / `IAMS_KEYCLOAK_CLIENT_SECRET`,
and `AMM_KEYCLOAK_CLIENT_ID` / `AMM_KEYCLOAK_CLIENT_USERNAME` / `AMM_KEYCLOAK_CLIENT_PASSWORD`.

It migrates on boot: `repo.RunMigrations` applies pending golang-migrate
migrations from `schema/` (`SCHEMA_DIR`, else `/app/schema`, else `./schema`)
before serving, creating the `search_path` schema first if it is missing.

### SurveyJS Licence Key

`PUBLIC_SURVEYJS_KEY` must be set for the form builder UI to work. Leave blank for
evaluation/development (functionality may be limited). Set `SURVEYJS_KEY=` in `.env`.

### Access Control

- **form_admin** role (configurable via `admin_roles` in `config.yaml`) — required for
  admin operations (manage forms, publish, manage themes)
- State transition permissions are defined per-state inside the form's SurveyJS JSON schema
- Field-level visibility and editability can be set per-role, per-state directly in the form
  schema (no backend config change needed)

## API Endpoints (form-service)

All endpoints require `Authorization: Bearer {token}`. Tenant + user context are extracted
from JWT claims.

**Every path below is served under a `/v1` prefix** — `POST /v1/forms`, and so on.
The same paths are still served unversioned for compatibility, but they respond
with `Deprecation: true`, a `Link: </v1…>; rel="successor-version"` header and a
`Warning` header, and **will be removed in form-service v2**. Write new callers
against `/v1`. The liveness and readiness probes are deliberately unversioned.

`@mssfoobar/form-client` addresses `/v1` for you — it inserts the version segment
per module, so `form` and `theme` requests carry it and `amm` (a different service,
with no version prefix) does not.

### Forms

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `POST` | `/forms` | Create a form (also creates an initial empty draft) |
| `GET` | `/forms` | List forms (paginated; filters: name, is_published) |
| `GET` | `/forms/{form_id}` | Get form details |
| `DELETE` | `/forms/{form_id}` | Delete form (and all its drafts, versions, submissions) |

### Form Drafts

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `POST` | `/forms/{form_id}/drafts` | Create a new draft for a form |
| `GET` | `/forms/{form_id}/drafts` | List drafts for a form |
| `GET` | `/forms/{form_id}/drafts/{draft_id}` | Get draft |
| `PUT` | `/forms/{form_id}/drafts/{draft_id}` | Update draft (name, form_json, theme) |
| `DELETE` | `/forms/{form_id}/drafts/{draft_id}` | Delete draft |
| `POST` | `/forms/{form_id}/drafts/{draft_id}/publish` | Publish draft → creates a Form Version |

### Form Versions

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/forms/{form_id}/versions` | List versions for a form |
| `GET` | `/forms/{form_id}/versions/{version_id}` | Get version |
| `POST` | `/forms/{form_id}/versions/{version_id}/publish` | Mark version as current (published) |
| `DELETE` | `/forms/{form_id}/versions/{version_id}` | Delete version (if not current) |

### Submissions

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `POST` | `/forms/{form_id}/submissions` | Create a submission against the current published version |
| `GET` | `/forms/{form_id}/submissions` | List submissions (paginated) |
| `GET` | `/forms/{form_id}/submissions/{submission_id}` | Get submission |
| `PUT` | `/forms/{form_id}/submissions/{submission_id}/state` | Trigger a state transition |
| `DELETE` | `/forms/{form_id}/submissions/{submission_id}` | Soft-delete a submission |

### Submission Drafts

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `PUT` | `/forms/{form_id}/submissions/{submission_id}/draft` | Save / update submission draft (autosave target) |
| `GET` | `/forms/{form_id}/submissions/{submission_id}/draft` | Get current user's submission draft |

### Themes

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `POST` | `/themes` | Create a theme |
| `GET` | `/themes` | List themes (paginated) |
| `GET` | `/themes/{theme_id}` | Get theme |
| `PUT` | `/themes/{theme_id}` | Update theme |
| `DELETE` | `/themes/{theme_id}` | Delete theme |

### Audit Logs

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/audit-logs` | List audit log entries (paginated; filter by entity_id, entity_type) |

### Health

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/health` | Liveness check |

## Data Model

```
form:
  id: UUID
  name: string (unique per tenant)
  tenant_id: string
  created_at, updated_at: timestamp
  created_by, updated_by: string

form_draft:
  id: UUID
  form_id: UUID (FK → form)
  name: string (nullable)
  description: string (nullable)
  form_json: JSONB          # SurveyJS schema
  theme_json: JSONB         # SurveyJS theme JSON
  image_ids: []string       # AMM attachment IDs
  created_by, updated_by: string

form_version:              # immutable after creation
  id: UUID
  form_id: UUID
  name: string
  description: string
  form_json: JSONB
  theme_json: JSONB
  changelog: JSONB          # property-level diff from previous version
  image_ids: []string

form_published_version:    # one per form, pointer to current version
  form_id: UUID
  version_id: UUID

submission:
  id: UUID
  form_version_id: UUID
  current_state: string     # as defined in form_json state machine
  tenant_id: string
  deleted_at: timestamp (nullable for soft delete)

submission_draft:          # one per (submission, user)
  id: UUID
  submission_id: UUID
  user_id: UUID
  data: JSONB               # in-progress form responses

submission_data_{version_id}:  # dynamically generated per form version
  id: UUID
  submission_id: UUID
  {question_key}: typed column  # generated from SurveyJS schema
```

## Access

| URL | What |
|-----|------|
| `http://form.${DEV_DOMAIN}` | form-service REST API |
| `http://form.${DEV_DOMAIN}/swagger-ui/index.html` | Swagger UI |

The designer ("Form Studio") and submission UIs no longer ship as standalone
deployed apps — they're the embeddable SDK trio a host app composes (see
*Frontend / SDK integration*).

## Dependencies

- **iams** (Keycloak for JWT auth; AAS for RBAC)
- **amm** (file/image attachments embedded in forms and form questions)

## Frontend / SDK integration

The standalone designer / submission SPAs are decommissioned. The form
authoring + fill/submit UI now ships as an embeddable frontend following the
standard `types ← client ← web` trio (AOH-7762):

- **`@mssfoobar/form-types`** (`modules/form/types`) — the wire DTOs.
- **`@mssfoobar/form-client`** (`modules/form/client`) — the typed REST clients
  (`FormClient` admin / `FormSubmitterClient` read + submit / `AmmClient`
  attachments) routed through the host's shared `/aoh/gateway` BFF.
- **`@mssfoobar/form-web-sdk`** (`modules/form/web`) — the SurveyJS authoring
  Creator + read/submit renderer.

This is how a host like `apps/reference-host` (and wfe Form-task steps) embeds
forms directly. To wire it into a host SvelteKit app, use the
**`aoh-form-integration`** skill.
