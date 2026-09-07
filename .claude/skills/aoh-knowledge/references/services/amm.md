# AMM — Attachment Management Module

## What It Does

AMM handles file upload, download, storage, and management for the AOH platform. For local
development it stores files on a **local volume by default** (`STORAGE_TYPE=local`); MinIO
(S3-compatible) object storage is opt-in. It can optionally scan uploads for viruses (off by
default).

Use AMM when your application needs file attachments, document storage, or media management.

## Architecture

```
┌──────────────┐   REST      ┌──────────┐   S3 API   ┌──────────┐
│ Your Backend │───────────►│ AMM-App  │───────────►│  MinIO   │
│  or Frontend │  multipart  │ (API)    │            │(S3 store)│
└──────────────┘             └────┬─────┘            └──────────┘
                                  │
                    ┌─────────────┼─────────────┐
                    │             │             │
               ┌────▼─────┐ ┌────▼─────┐ ┌────▼────────┐
               │  AMM-DB  │ │Gotenberg │ │ClamAV+C-ICAP│
               │(Postgres)│ │(PDF prev)│ │(virus scan) │
               └──────────┘ └──────────┘ └─────────────┘
                                          (optional)
```

> Storage defaults to a local volume; the MinIO/S3 path shown above is opt-in.

## Components

| Component | Image | Purpose |
|-----------|-------|---------|
| **amm-db** | `postgres:17.0` | File metadata, access records, tags |
| **amm-app** | `ghcr.io/mssfoobar/amm/amm-app` | File CRUD API, upload/download handling |
| **amm-minio** | `minio/minio` | S3-compatible object storage (opt-in) |
| **amm-minio-init** | `minio/mc` | Bucket initialization (runs once, opt-in) |
| **amm-gotenberg** | `gotenberg/gotenberg:8` | PDF/document preview generation |
| **clamav** | `clamav/clamav` | Virus scanning engine (optional) |
| **amm-c-icap** | Custom | ICAP proxy for ClamAV integration (optional) |

## Key Concepts

### Storage model

- Files are stored on a **local volume by default** (`STORAGE_TYPE=local`,
  `STORAGE_PATH=/file/data`). MinIO (S3-compatible) is opt-in — set `STORAGE_TYPE=minio`
  and enable the `infra/minio-compose.yml` include; use it when you want S3 semantics or
  production parity.
- Metadata (filename, size, type, tags) is stored in PostgreSQL
- When backed by MinIO, objects have a 7-day expiration policy
- Access via pre-signed URLs or the AMM API

### Local volume ownership (podman vs docker)

With `STORAGE_TYPE=local`, amm-app writes uploads under `STORAGE_PATH` (`/file/data`)
as a **non-root** user, so the mounted volume must be writable by that user — otherwise
**every upload fails with a permission error** that's easy to misread as an app bug.
This is a `STORAGE_TYPE=local` concern only; MinIO/S3 has no host-volume ownership issue.

- **Podman (rootless — the default/recommended runtime):** append the `:U` flag so
  Podman recursively chowns the volume to the container's UID/GID:

  ```yaml
  volumes:
      - amm-file-volume:/file/data:U
  ```

- **Docker:** Docker has **no `:U`** option and will reject it — leave it off. A fresh
  **named** volume instead inherits the image's ownership of `/file/data` on first
  mount, so it usually works as-is. If uploads still hit `permission denied` (the
  volume was created empty, or pre-existed root-owned), recreate it so it re-inherits:

  ```bash
  docker volume rm aoh_amm-file-volume    # then `compose up` recreates it
  ```

  or chown it once to the amm-app user (find its uid/gid with
  `docker compose exec amm-app id`):

  ```bash
  docker run --rm -v aoh_amm-file-volume:/data alpine chown -R <uid>:<gid> /data
  ```

### File upload flow

1. Client sends multipart POST to AMM API
2. AMM validates file type and size
3. AMM stores the file (local volume by default; MinIO when enabled)
4. (Optional) AMM sends file to ClamAV for virus scanning
5. AMM generates preview via Gotenberg (for PDFs/documents)
6. AMM saves metadata to PostgreSQL
7. Returns file ID and metadata

### Checksum validation

Uploads require a SHA256 checksum generated from request metadata (description, tags, module,
entity_type, entity_id). Keys are sorted alphabetically, formatted as `key=value;key=value`,
then SHA256 hashed. Controlled by `CHECKSUM_ENABLED` env var (default: false).

### Module & entity association

Attachments are organized by `module` (required on upload) and optionally `entity_type` +
`entity_id`. This allows querying all attachments for a specific module or entity.

### Data access control

AMM uses AAS-based authorization with two levels:
- **Functional access**: resource (`amm_attachment` or `amm_admin`) + scope (`view` or `edit`)
- **Data access**: per-attachment resource_name + scope — controls who can view/edit specific files

These are IAMS-AAS resources of type `data-access-control` (AMM's
`IAMS_DATA_ACCESS_RESOURCE_TYPE`). AMM matches the **literal** resource name, so seed
them **exactly** — `amm_attachment` / `amm_admin`, underscores, not dots — in your
project's `roles.yaml` and grant the user the scope; without a matching permission AMM
silently denies upload/download. (`aoh-compose` ships no AMM-specific authz — it defers
here; see the `iams` reference for the `roles.yaml` schema.)

### Secure downloads

Two-step download flow:
1. `GET /attachments/{id}/download-id` — generates a time-limited download ID
2. `GET /downloads/{id}` — downloads the file using that ID

### Logical vs physical delete

- **Logical delete** (`DELETE /attachments/{id}/logical-delete`) — soft delete, can be restored
- **Physical delete** (`DELETE /attachments/{id}/physical-delete`) — permanent delete
- **Restore** (`POST /attachments/{id}/restore`) — restore a logically deleted file (requires `amm_admin`)

### Preview generation

Gotenberg generates PDF previews for uploaded documents. This enables thumbnail/preview
rendering without downloading the full file.

### No dedicated Keycloak client

AMM doesn't have its own Keycloak client. It validates JWT tokens directly against the
Keycloak realm URL.

### Object expiry

When backed by MinIO, objects have a 7-day expiration policy. Virus scanning is disabled by
default — enable it with `VIRUS_SCANNER_ENABLED=true` plus the `infra/icap-compose.yml`
include.

## API Endpoints (AMM-App)

### Upload & CRUD

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `POST` | `/attachments` | Upload file(s) (multipart/form-data, requires checksum) |
| `GET` | `/attachments/{id}` | Get attachment metadata |
| `PUT` | `/attachments/{id}` | Update metadata (description, tags, data access) |

### Download

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/attachments/{id}/download-id` | Generate secure download ID (time-limited) |
| `GET` | `/downloads/{id}` | Download file using download ID |

### Listing

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/attachments/modules/{module_id}` | List attachments by module (paginated) |
| `GET` | `/attachments/modules/{module_id}/entities/{entity_id}` | List by module + entity (paginated) |

### Delete & Restore

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `DELETE` | `/attachments/{id}/logical-delete` | Soft delete (can be restored) |
| `DELETE` | `/attachments/{id}/physical-delete` | Permanent delete |
| `POST` | `/attachments/{id}/restore` | Restore a soft-deleted attachment (admin) |

### Health

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/health/livez` | Liveness check |
| `GET` | `/health/readyz` | Readiness check |

## Data Model

```
attachment:
  id: UUID
  file_name: string
  file_size: int64
  mime_type: string
  description: string
  module: string
  entity_type: string
  entity_id: UUID
  tenant_id: string
  created_at, updated_at: timestamp (RFC3339)
  created_by, updated_by: string

data_access:
  attachment_id: UUID (FK)
  resource_name: string
  scope: string (view, edit)

attachment_tag:
  attachment_id: UUID (FK)
  tag_id: UUID (opaque tag identifier supplied by the caller)

download_id:
  download_id: UUID
  expires_at: timestamp (RFC3339)
  download_path: string
```

## Access

| URL | What |
|-----|------|
| `http://amm.${DEV_DOMAIN}` | AMM App API |
| `http://amm.${DEV_DOMAIN}/swagger-ui/index.html` | Swagger UI |
| `http://amm-minio.${DEV_DOMAIN}` | MinIO web console (only when MinIO is enabled) |

## Dependencies

- **iams** (Keycloak for JWT validation)
- Optional: ClamAV + C-ICAP for virus scanning
