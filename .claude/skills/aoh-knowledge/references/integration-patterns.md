# AOH Integration Patterns

Common patterns for integrating with AOH services from your application code.

## 1. Authentication & Authorization (IAMS)

> **Load-bearing rule — application roles live in AAS, not the Keycloak realm.**
> The full taxonomy (realm-level vs AAS tenant roles), JWT claim shape
> (`active_tenant.roles`, exposed in Go as `getTenantRoles(ctx)`), and project-level
> bootstrap pattern (`compose/iams/init/project-aas/roles.yaml`, laid down by
> `aoh-compose`) live in `services/iams.md` → "Authorization is via AAS, not
> Keycloak" and "Project-level AAS bootstrap (reproducibility pattern)". Read it
> before adding anything role-related; do not infer from generic OIDC priors.

### OIDC login flow (web frontends — SvelteKit)

```
Browser -> Your SvelteKit App -> Keycloak (redirect) -> Browser (login page)
Browser -> Keycloak (credentials) -> Your App (callback with auth code)
Your App -> Keycloak (exchange code for tokens) -> access + refresh tokens
Your App -> SDS (store tokens) -> session ID cookie returned to browser
```

- Use `openid-client` v6 for OIDC
- Keycloak discovery URL: `http://iams-keycloak.${DEV_DOMAIN}/realms/aoh/.well-known/openid-configuration`
- Client ID: your service's public client name (e.g., `gis`, `ian`)
- Tokens include: `sub` (user ID), `active_tenant`, realm roles
- **Tokens are stored in SDS, not in browser cookies.** Only a session ID
  cookie reaches the browser; the access/refresh tokens live in Valkey
  behind `sds-server`. `SDS_URL` is required on every AOH web app — see
  `services/sds.md` → "Why SDS is mandatory" for the rationale and
  `aoh-web-init` for the env wiring. The cookie-only fallback in `auth.ts`
  is for emergency diagnostics, not a deployable mode.

### JWT validation (Go backends)

```
Incoming request -> Extract Bearer token from Authorization header
-> Validate JWT signature against Keycloak JWKS endpoint
-> Check token expiry, issuer, audience
-> Extract claims: sub, active_tenant, realm_access.roles
```

- JWKS URL: `http://iams-keycloak:8080/realms/aoh/protocol/openid-connect/certs`
- Issuer: `http://iams-keycloak.${DEV_DOMAIN}/realms/aoh`
- **Realm-level (platform-bootstrap) roles only**: `system-admin`, `tenant-admin`. Application roles (e.g. `field-reporter`) are NOT realm roles — they are AAS tenant roles surfaced into the `active_tenant.roles` claim. See the callout at the top of this section.

### Authorization via AAS

AAS provides tenant-scoped access control (RBAC, UBAC, GBAC) on top of Keycloak roles.
All AAS endpoints use the `/admin/` prefix:

```
Your Backend -> AAS API (http://iams-aas:8080)
  GET  /admin/tenants                      # List tenants
  GET  /admin/tenants/{id}/memberships     # List tenant members
  GET  /admin/tenants/{tenantId}/users/{userId}/resources/{resourceId}/scopes/{scopeId}/evaluate
                                           # Check permission -> returns "PERMIT" or "DENY"
  GET  /admin/tenants/{tenantId}/users/{userId}/resources-by-name/{resourceName}/scopes-by-name/{scopeName}/evaluate
                                           # Check permission by name -> "PERMIT" or "DENY"
  GET  /admin/tenants/{tenantId}/users/{userId}/resources
                                           # List all accessible resources for user
```

- Permission evaluation returns `"PERMIT"` or `"DENY"` (not boolean)
- Can evaluate by resource/scope ID or by name
- AAS uses the same Keycloak token for caller identity
- For basic role checks, read `active_tenant.roles` from the JWT directly (no AAS call needed)
- For reproducible AAS setup — creating tenant roles, scopes, resources, and user-to-role assignments on every fresh stack-up — see `services/iams.md` → "Project-level AAS bootstrap (reproducibility pattern)". A starter Newman collection ships with `aoh-compose`.

### Service-to-service auth — TWO postures, pick by what the target validates

Backend-to-backend calls in AOH have **two** auth postures. Pick by what
claim the target service validates, NOT by what's convenient:

**A. Propagate the inbound user bearer (default for request-driven calls).**

When the call is driven by an HTTP request from the user (e.g. the Field
Reporter's `POST /v1/incidents` triggers a side-effect call to UNH),
capture the inbound `Authorization: Bearer <jwt>` header in the handler
and forward the same token string on the downstream call. The user JWT
carries `active_tenant`, the realm roles, and the resolved AAS roles —
which is exactly what the target needs.

```
Browser -> Your Backend (Bearer <user-jwt>)
   handler captures bearerToken := strings.TrimPrefix(authHeader, "Bearer ")
   -> Downstream service (Authorization: Bearer <same user-jwt>)
```

Gotcha: if the downstream call runs in a post-commit goroutine derived
from `context.Background()` (so a cancelled request does not cancel the
goroutine), capture the token STRING by value BEFORE detaching from the
request context — once the request closes, the header is no longer
accessible.

**Required** for any target service that validates `active_tenant`:
- **UNH** — rejects `client_credentials` tokens as `"invalid jwt"` because they
  don't carry `active_tenant`. The `/admin/*` AND `/notification/send/*`
  endpoints both fail without it.
- **IAMS-AAS admin endpoints** — `active_tenant`-aware.
- Any AOH service whose authorization layer projects from
  `active_tenant.roles` (i.e. anything using the standard AOH middleware).

**B. Mint a service-account token via `client_credentials` (system-context only).**

Use when there's no inbound user (cron jobs, startup tasks, daemons) AND
the target accepts a tokenless or signature-only call. Examples in AOH:
the initial RTUS topic-publish endpoint, IAN's `POST /messages` from
inside the compose network, and SDS calls between services. These
endpoints validate the bearer's signature only — `active_tenant` is not
required.

```
Your Backend -> Keycloak token endpoint
  POST /realms/aoh/protocol/openid-connect/token
  Body: grant_type=client_credentials&client_id=<id>&client_secret=<secret>
-> Service-account JWT (NO active_tenant claim)
-> Call target service with Bearer token
```

If you find yourself reaching for `client_credentials` to call UNH —
stop. Posture A is the one you want.

Used by posture A: `incident-svc → UNH` (in the in-app notification
flow), any service-to-service call you trigger from a user request.

Used by posture B: SDS internal calls, PTMGR token storage, scheduled
sends with no inbound request.

## 2. Real-time Updates (RTUS)

### Publishing data (backend -> RTUS)

Your backend publishes to RTUS-PMS via REST. All data endpoints are tenant-scoped:

```
# Topics (broadcast to all subscribers)
POST http://rtus-pms:8080/tenants/{tenantId}/topics/{topicName}
Body: <json or text payload>
Content-Type: application/json

# Maps (key-value with change notifications, text/JSON values)
POST http://rtus-pms:8080/tenants/{tenantId}/maps/{mapName}/keys/{key}
Body: <value>
Content-Type: application/json

# JSON Maps (key-value, JSON only, supports per-entry access control)
POST http://rtus-pms:8080/tenants/{tenantId}/json-maps/{mapName}/keys/{key}
Body: <json value>
Content-Type: application/json

# Batch operations (Maps and JSON Maps)
POST http://rtus-pms:8080/tenants/{tenantId}/maps/{mapName}/batch
Body: { "key1": "value1", "key2": "value2" }

# User Value Maps (keyed by user ID, for per-user data)
POST http://rtus-pms:8080/tenants/{tenantId}/uservaluemaps/{mapName}/users/{userId}
Body: <value>
```

- **Topics**: Fire-and-forget broadcast. Oldest events dropped when capacity (100) is full.
- **Maps**: Persistent key-value (text). Subscribers get change events (Added, Updated, Removed).
- **JSON Maps**: Like Maps but JSON-only. Supports fine-grained per-entry access control.
- **User Value Maps**: Like Maps but keyed by user ID. For per-user data delivery.

### Subscribing (browser -> RTUS-SEH via SSE)

Frontend connects to RTUS-SEH for Server-Sent Events. SSE access is controlled via
admin SSE config endpoints (`/admin/tenants/{tenantId}/maps/{mapName}/sse`, etc.,
plus `/admin/tenants/{tenantId}/topics/{topic}/sse` for topics) which set
role-based and fine-grained access rules.

> Use `@mssfoobar/sse-client` (1.2.0+) for the browser-side connection — it ships
> the topic/user-map/json-map URL strategies, reconnect backoff,
> refresh-before-reconnect, and heartbeat timeout. Do NOT hand-roll an
> `EventSource` wrapper. See `services/rtus.md` → "SSE client SDK".

- Auth: `withCredentials: true` sends the platform-shared
  `web_auth_session_id` cookie (SDS-flow) or a `<x>_access_token` cookie
  (legacy flow). rtus-seh resolves the session via SDS or reads the JWT
  cookie directly.
- **Cookie prefix defaults to `web`** so the auth cookie name matches the
  seeded `rtus.session-id.cookienames` on rtus-seh out of the box. A per-app
  prefix is allowed but requires updating rtus-seh's config in the same
  change. See `services/rtus.md` → "Cookie prefix convention".
- Each web app that uses RTUS must be added to the RTUS-SEH CORS origin list.
- **Initial-fetch + SSE dedup**: pages that combine `GET /list` with a live
  topic of the same domain MUST deduplicate by `id` before prepending —
  otherwise a race produces duplicate keys and Svelte 5 throws
  `each_key_duplicate`, breaking the table render. See `services/rtus.md`
  → "Initial-fetch + SSE dedup (consumer rule)".

### RTUS naming conventions

- GIS uses JSON Map name `"gis"` for entity positions
- IAN uses User Value Map name `"ian"` for per-user notification delivery
- Your service should use its own map/topic name

## 3. Notifications (UNH + IAN)

### Sending a notification via UNH

UNH uses a template-based approach. First create channels, distribution lists, and
templates via the admin API, then send by template ID:

```
POST http://unh:8080/notification/send/template/{template_id}
Headers: Authorization: Bearer <jwt>
Body: {
    "data": { "username": "John", "link": "https://..." }
}
```

The `data` map provides values for template variable substitution. The template itself
defines which channels to use (email, push, SMS, custom) and which distribution list
to send to.

- Admin APIs at `/admin/email_channel`, `/admin/push_channel`, `/admin/custom_channel`,
  `/admin/distribution_list`, `/admin/notification_template`
- Distribution lists support: internal users/groups/roles + external emails/phones
- Send response includes per-channel results and any unresolved distribution entries

### In-app notifications via IAN

IAN stores messages in its database and pushes them to connected browsers via RTUS:

```
POST http://ian:8080/messages
Headers: Authorization: Bearer <jwt>
Body: {
    "title": "Alert",
    "body": "Something happened",
    "ref_link": "https://example.com/details/123",
    "icon_id": "alert-icon",
    "sender_id": "system",
    "tenant_id": "<tenant-uuid>",
    "receiver_ids": ["user-uuid-1", "user-uuid-2"]
}
```

- Message statuses: `READ`, `UNREAD` (strings, not integers)
- User endpoints are scoped: `GET /users/{id}/messages`, `GET /users/{id}/messages/unread-count`
- Update status: `PUT /users/{id}/messages/{message_id}/update-status`
- Mark all read: `PATCH /users/{id}/messages/mark-all-as-read`
- Frontend receives real-time delivery via RTUS-SEH SSE
- There is no platform-shipped notification panel UI — the consuming web app builds its own panel against the IAN-App REST API and the `ian` RTUS map subscription

## 4. File Management (AMM)

### Upload a file

```
POST http://amm:8080/attachments
Headers: Authorization: Bearer <jwt>
Content-Type: multipart/form-data
Body: file + metadata (module, entity_type, entity_id, description, tags)
```

Uploads require a module name. Optional checksum validation (SHA256 of sorted metadata
key-value pairs) when `CHECKSUM_ENABLED=true`.

### Download a file (2-step)

```
# Step 1: Generate a time-limited download ID
GET http://amm:8080/attachments/{id}/download-id
Headers: Authorization: Bearer <jwt>
-> Returns { download_id: "..." }

# Step 2: Download using the download ID
GET http://amm:8080/downloads/{download_id}
```

### Delete

```
# Soft delete (can be restored)
DELETE http://amm:8080/attachments/{id}/logical-delete

# Permanent delete
DELETE http://amm:8080/attachments/{id}/physical-delete

# Restore soft-deleted (requires amm_admin)
POST http://amm:8080/attachments/{id}/restore
```

- Storage: local volume by default (`STORAGE_TYPE=local`); MinIO (S3-compatible, bucket per tenant, 7-day expiry) opt-in
- PDF preview: Generated by Gotenberg
- Virus scanning: Optional (ClamAV + C-ICAP), disabled by default
- Data access control: functional (amm_attachment/amm_admin) + per-attachment resource/scope

## 5. Geospatial Data (GIS)

### CRUD operations

```
# Create entity
POST http://gis-service:8080/geoentity
Headers: Authorization: Bearer <jwt>
Body: {
    "geojson": {
        "type": "Feature",
        "geometry": { "type": "Point", "coordinates": [103.8, 1.35] },
        "properties": { "name": "Asset 1", "kind": "vehicle" }
    },
    "entity_id": "vehicle-001",
    "entity_type": "track"
}

# Upsert entity (create or update by ID)
PUT http://gis-service:8080/geoentity
Body: { "id": "<uuid>", "geojson": { ... }, "entity_id": "...", "entity_type": "..." }

# Partial update (requires occ_lock)
PATCH http://gis-service:8080/geoentity/id/{id}
Body: { "occ_lock": 1, "geojson": { ... } }

# Query by kind
GET http://gis-service:8080/geoentity/kind/{kind}

# List paginated (0-indexed pages)
GET http://gis-service:8080/geoentity?page=0&size=10&sort=name,asc
```

- Entity types: `static`, `track`, `geofence`, `annotation`
- `kind` in GeoJSON properties groups entities (e.g., "aircraft", "vehicle")
- Real-time position updates published to RTUS JSON Map `"gis"` via a transactional outbox worker
- The GIS map UI ships as the `@mssfoobar/gis-web-sdk` Svelte SDK (Cesium/MapLibre), embedded in a host app — there is no standalone GIS web service
- Batch operations: `POST/PUT /geoentity/batch`

## 6. Session Management (SDS)

### How SDS works

SDS is transparent to most developers — it's used by web frontends automatically.
It provides two store types:

```
# Authenticated store (tied to Keycloak JWT, manages token refresh)
POST http://sds:5080/session/auth-store
Body: { "accessToken": "...", "refreshToken": "..." }
-> Returns { "id": "<store_id>" }  (saved as session cookie)

# Get a valid access token (auto-refreshes if expired)
GET http://sds:5080/session/auth-store/{store_id}/accessToken

# Store/retrieve arbitrary key-value data in the session
PUT http://sds:5080/session/auth-store/{store_id}/keys
Body: { "active_tenant": "...", "preferences": "..." }

GET http://sds:5080/session/auth-store/{store_id}/keys

# Temporary store (short-lived, no auth — for CSRF tokens, pre-login state)
POST http://sds:5080/session/temp-store
```

- HTTP API on port 5080, TCP on port 5333
- Backed by Valkey (Redis-compatible)
- Keycloak client: `sds-client` (confidential)
- RTUS-SEH reads session data from SDS for auth context

## 7. Workflow Orchestration (WFE)

### Starting a workflow from your backend

```
# Start a workflow execution from a published template
POST http://wfm:8080/v1/workflow
Headers: Authorization: Bearer <jwt>
Body: {
    "template_id": "<published-template-uuid>",
    "metadata": {
        "incident_id": "INC-001",
        "triggered_by": "user-uuid"
    }
}
-> Returns { "data": { "workflow_id": "<uuid>" } }
```

- `template_id` must reference a published template (`editable: false`)
- `metadata` is a free-form key-value map accessible inside activities via
  `temporal.ContextValue(ctx)` from `aoh-golib/temporal`

### Monitoring workflow progress

```
# Poll workflow event history (paginated)
GET http://wfm:8080/v1/workflow/{workflow_id}?page=1&size=10&sort=timestamp,desc

# Get the most recent event to check current status
GET http://wfm:8080/v1/workflow/{workflow_id}?page=1&size=1&sort=timestamp,desc
```

Key terminal event types: `WorkflowExecutionCompleted`, `WorkflowExecutionFailed`,
`WorkflowExecutionTerminated`, `WorkflowExecutionCanceled`

### Signalling a paused workflow (form/recover tasks)

```
# Submit form data to unblock a waiting FormTask
POST http://wfm:8080/v1/workflow/{workflow_id}/activity_name/{activity_name}
Headers: Authorization: Bearer <jwt>
Body: { "data": "<form field values>" }
```

`activity_name` is the node name in the BPMN diagram (e.g., `"ApprovalForm"`).

### Template lifecycle

```
# Save a draft (editable, can be updated)
PUT http://wfm:8080/v1/workflow_template/save
Body: { "name": "...", "workflow_json": {...}, "designer_json": {...} }

# Publish (locks the template — required before it can be executed)
PUT http://wfm:8080/v1/workflow_template/publish
Body: { same as save }
```

## 8. Common Patterns

### Internal service URLs (container-to-container)

Services call each other using Docker Compose service names:

| Target | Internal URL |
|--------|-------------|
| Keycloak | `http://iams-keycloak:8080` |
| AAS | `http://iams-aas:8080` |
| RTUS PMS | `http://rtus-pms:8080` |
| RTUS SEH | `http://rtus-seh:8080` |
| GIS Service | `http://gis-service:8080` |
| SDS Server | `http://sds:5080` |
| UNH Service | `http://unh-service:8080` |
| IAN App | `http://ian:8080` |
| AMM App | `http://amm:8080` |

### External URLs (browser-accessible via Traefik)

| Target | External URL |
|--------|-------------|
| Keycloak | `http://iams-keycloak.${DEV_DOMAIN}` |
| IAMS Web | `http://iams-web.${DEV_DOMAIN}` |
| RTUS SEH (SSE) | `http://rtus-seh.${DEV_DOMAIN}` |
| MinIO Console (opt-in) | `http://amm-minio.${DEV_DOMAIN}` |
| Traefik Dashboard | `http://traefik.${DEV_DOMAIN}` |

### Default credentials

- **Dev user**: `admin` / `P@ssw0rd`
- **DEV_DOMAIN**: `127.0.0.1.nip.io`
- All service databases use `${DEV_USER}` / `${DEV_PASSWORD}`
