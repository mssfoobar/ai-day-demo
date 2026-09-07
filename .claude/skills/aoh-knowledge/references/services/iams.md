# IAMS — Identity & Access Management System

## What It Does

IAMS is the authentication and authorization backbone of AOH. It provides:

- **Authentication** via Keycloak (OpenID Connect with PKCE)
- **Authorization** via AAS (Authorization and Admin Service) — tenant-scoped RBAC, UBAC, GBAC
- **User management** via IAMS-Web UI
- **Tenant initialization** via IAMS-Init (Newman/Postman runner)
- **Custom Keycloak extensions** — password management, TOTP, inactivity/expiry disabling,
  user security info

Nearly every AOH service depends on IAMS for JWT validation and role/permission checks.

## Architecture

```
                    ┌───────────────┐
                    │  iams-keycloak │◄── OIDC provider, JWT issuer
                    │  (Keycloak)    │    Realm: "aoh"
                    └───────┬───────┘
                            │
              ┌─────────────┼─────────────┐
              │             │             │
     ┌────────▼──────┐ ┌───▼────┐ ┌──────▼──────┐
     │   iams-aas    │ │iams-web│ │  iams-init  │
     │ (AuthZ API)   │ │ (UI)   │ │  (setup)    │
     └───────────────┘ └────────┘ └─────────────┘
              │
     ┌────────▼──────┐
     │   iams-db     │
     │ (PostgreSQL)  │
     └───────────────┘
```

## Components

| Component | Image | Purpose |
|-----------|-------|---------|
| **iams-db** | `postgres:17.0` | Keycloak's backing database (also supports MS SQL, Oracle) |
| **iams-keycloak** | `ghcr.io/mssfoobar/iams/iams-keycloak` | OIDC provider, token issuer, custom extensions |
| **iams-aas** | `ghcr.io/mssfoobar/iams/iams-aas` | Tenant-scoped authorization API (Spring Boot) |
| **iams-web** | `ghcr.io/mssfoobar/iams/iams-web` | User/role management UI (SvelteKit) |
| **iams-init** | `postman/newman` | Runs once to create default tenant and admin membership |

## Key Concepts

### Keycloak Realm

AOH uses a single realm called `aoh`. It contains:

- **Clients**: one per service in production (public for frontends, confidential
  for backends). **In local dev, most changes add no new client** — frontends
  reuse the bundled `web` PKCE client, and a backend that only validates bearer
  tokens (userinfo/JWKS) needs none. Register a new client only for a *confidential*
  (client-credentials / introspection) backend. See "Reuse the bootstrapped … realm" below.
- **Realm roles** (platform-bootstrap only): `system-admin`, `tenant-admin`,
  `default-roles-aoh`. **Application roles live in AAS, not the realm — see
  "Authorization is via AAS, not Keycloak" below.**
- **Client scopes**: `aoh_default_scope` with custom claim mappers
- **Claim mappers**: `all_tenants` (list of tenant memberships with roles) and
  `active_tenant` (current tenant context with tenant_id, tenant_name, roles).
  Note the raw JSON key is **`tenant_id`** (not `id`); aoh-golib's Go struct
  field is `Id` (json-tagged `tenant_id`), surfaced via `getTenantID(ctx)`. A
  script/seed decoding the JWT directly must read `active_tenant.tenant_id`.

### Client Types

**Public clients** — Used by web frontends (Authorization Code Flow with PKCE):
- `publicClient: true`, no client secret
- Examples: `iams`, `gis`, `ian`, `dash`, `unh-web`

**Confidential clients** — Used by backend services (Client Credentials Grant):
- `publicClient: false`, `serviceAccountsEnabled: true`
- Requires a client secret and a `service-account-<clientId>` user
- Must include `openid` scope when requesting tokens
- Examples: `sds-client`, `unh-app`, `ptmgr`

### Authentication Flow (PKCE)

AOH Web Base uses Authorization Code Flow with PKCE:

1. Web App Backend creates `code_verifier` and derives `code_challenge` (S256)
2. Redirects user to Keycloak with `code_challenge`, `client_id`, `redirect_uri`
3. User authenticates at Keycloak login screen (may include TOTP)
4. Keycloak redirects back with `authorization_code`
5. Web App Backend exchanges `authorization_code` + `code_verifier` for tokens
6. **Access token** and **refresh token** stored in browser cookies (ID token is discarded)
7. Subsequent requests include access token in cookies; backend forwards as
   `Authorization: Bearer {token}` to microservices

Configuration: `IAM_URL` (Keycloak realm URL) and `IAM_CLIENT_ID` in `.env`.

### Token Lifecycle

- **Access token**: Short-lived (default 5 min), used as Bearer token for API access.
  Web Base auto-refreshes before expiry.
- **Refresh token**: Longer-lived (default 30 min idle, 10 hour max SSO session).
  Extended on each use up to SSO Session Max.
- **ID token**: Discarded by Web Base (info available in access token).

### Token Validation

Three methods to validate access tokens:

1. **Online — Userinfo endpoint**: `GET /realms/{realm}/protocol/openid-connect/userinfo`.
   Returns 200 if valid, 401 if invalid. Requires `openid` scope in token.

2. **Online — Token introspection**: `POST /realms/{realm}/protocol/openid-connect/token/introspect`.
   Confidential clients only (requires client_id + client_secret). Returns `{"active": true/false}`
   with full claims. Recommended for backend services that may receive service account tokens.

3. **Offline validation**: Decode JWT, verify RS256 signature against realm public key
   (from `GET /realms/{realm}`), check `exp` claim. Most efficient (no HTTP round trip).

### Domain Model

The AAS domain model hierarchy:

- **Realm** → top-level container (typically one: "aoh")
- **Tenant** → organization within a realm (unique name per realm)
- **User** → belongs to realm, can be member of multiple tenants
- **Role** → per-tenant, assignable to users or groups (unique name per tenant)
- **Group** → per-tenant collection of users, supports sub-groups
- **Resource** → per-tenant protected entity (unique name per tenant)
- **Scope** → per-tenant action on a resource (e.g., "view", "edit", "delete")

### Authorization Models

AAS supports three access control models:

- **RBAC (Role-based)** — permission granted to roles; users with those roles get access.
  Roles can be marked as `required` (user must have that specific role).
- **UBAC (User-based)** — permission granted directly to specific users.
- **GBAC (Group-based)** — permission granted to groups; group members get access.
  Supports `extendChildren` to inherit permissions to sub-groups.

Authorization check flow: backend calls AAS evaluate endpoint → returns `"PERMIT"` or
`"DENY"`. The `active_tenant` JWT claim carries the user's roles for basic role checks
without calling AAS.

> ⚠️ **The JWT carries `active_tenant.roles` only. It does NOT carry a resolved
> `active_tenant.permissions` array** (verified against `iams-aas` v1.3.7). Code or
> specs that assume the JWT has a `permissions` claim will silently produce 403s.
> Three valid ways to gate by permission in a service:
>
> 1. **Project roles → permissions in the service** using a static map that mirrors
>    `compose/iams/init/project-aas/roles.yaml`. Cheapest, no extra hops, but the
>    map is duplicated. Recommended for small role sets.
> 2. **POST to AAS `/evaluate`** for an authoritative check. Adds a round trip per
>    request; cache aggressively if used on hot paths.
> 3. **Gate by role name** at the handler/middleware (`getTenantRoles(ctx)`). Same
>    as option 1 without the explicit projection — fine for coarse "this endpoint
>    is for field-reporters" checks. Not fine for fine-grained per-resource ACLs.
>
> Avoid writing middleware that reads `claim.ActiveTenant.Permissions` and `403`s
> on empty — that path will reject every legitimate user.

### Custom Keycloak Extensions

IAMS Keycloak includes custom REST endpoints:

**Password management:**
- `POST /realms/{realm}/password-resources/validate-password` — validate current password
- `POST /realms/{realm}/password-resources/reset-password` — update password (requires
  currentPassword + newPassword)

**TOTP management:**
- `GET /admin/realms/{realm}/totp-resources/generate` — get QR code + encoded secret
- `POST /admin/realms/{realm}/totp-resources/register` — register TOTP credential
- `POST /admin/realms/{realm}/totp-resources/verify` — verify OTP code

**User security info:**
- `GET /realms/{realm}/user-security-resources/users` — list user security details
  (lastLogin, passwordExpiry, TOTP status, etc.). Paginated, admin-only.

**Account lifecycle (for CronJobs):**
- `GET /realms/{realm}/user-security-resources/inactivity-disabling` — disable inactive
  accounts (threshold configurable via SPI env var, default 90 days)
- `GET /realms/{realm}/user-security-resources/backfill-last-login` — backfill last login
  timestamps from event store
- `GET /realms/{realm}/user-security-resources/password-expiry-disabling` — disable accounts
  with expired passwords beyond grace period

**Event listener:**
- `last-login-tracker` — records `lastLogin` and `lastLoginTimestamp` user attributes on
  each login event.

### Concurrent Session Limiting

Keycloak supports limiting concurrent user sessions via the **User Session Count Limiter**
authenticator. Configurable per authentication flow (browser, direct grant) with options to
deny new sessions or terminate the oldest session.

### IAMS Init

A one-shot Newman runner that:
1. Resets the admin password in Keycloak
2. Creates a "development" tenant in AAS
3. Assigns the admin user to that tenant

Required for services that use AAS (UNH, PTMGR, AMM, etc.).

**Reuse the bootstrapped `aoh` realm AND `development` tenant for local
development.** New services and changes MUST NOT create their own realm or
tenant during dev/test setup — reuse what `iams-keycloak` (realm `aoh`, imported
from `keycloak/realm-import.json`) and `iams-init` (AAS tenant `development`,
created via the Newman collection) already provision.

- **Realm** (`aoh`): OIDC clients (backend confidential, frontend PKCE), claim
  mappers, the user-profile policy, and the small set of platform-bootstrap
  realm roles (`system-admin`, `tenant-admin`, `default-roles-aoh`) live here.
  Applications only ever ADD OIDC clients to this realm; they do NOT add realm
  roles, realm scopes, or modify the user-profile policy.
- **Tenant** (`development`): all seed users for local dev are added as
  members of this tenant. The `active_tenant` claim in dev JWTs already points
  at it. **All application-level authorization** (roles, scopes, resources,
  group memberships, user-to-role assignments) lives in this tenant via the
  AAS API — see "Authorization model" below.

### Authorization is via AAS, not Keycloak

AAS is the platform's authorization control plane. It sits in front of Keycloak
and exposes the API that applications use to define and enforce access:

- **Roles** — created per tenant via `POST /admin/tenants/{tenantId}/roles`.
  Application roles like `field-reporter`, `operations-team`,
  `incidents-reader` are AAS tenant roles, NOT Keycloak realm roles. They are
  surfaced into JWTs as the `active_tenant.roles` claim.
- **Scopes & resources** — created per tenant via the AAS resource/scope
  endpoints (see "AAS — Resources & Scopes" below). Used for fine-grained
  permission checks on entities.
- **Group memberships and user-to-role assignments** — managed via AAS
  endpoints, not Keycloak's user-management UI.

Realm roles (`system-admin`, `tenant-admin`) are platform-bootstrap concerns
maintained by the AOH platform team via `realm-import.json` — applications
should not add to them.

### Project-level AAS bootstrap (reproducibility pattern)

The platform ships `compose/iams/init/iams-aas-init.postman_collection.json` —
a Newman runner that creates the `development` tenant and seeds the admin
user. Projects that need their own AAS roles, groups, resources, scopes, or
user-to-role assignments **own a sibling artifact** that runs after the
platform's, kept idempotent so it converges on the same end state on every
fresh stack-up.

The project-owned artifact is a **small Python script + a YAML data file**,
packaged as a tiny container that the project's compose runs once on
startup. (A Postman collection works too and was the historical pattern,
but YAML diffs review better than Postman JSON and the script can model
the full AAS authz vocabulary — RBAC + GBAC + UBAC + resources + scopes —
in one file.)

**File layout** — `aoh-compose` lays this down automatically:

```
compose/iams/init/
├── iams-aas-init.postman_collection.json   # platform-shipped — do NOT modify
└── project-aas/                            # project-owned, shipped by aoh-compose
    ├── README.md
    ├── Dockerfile
    ├── requirements.txt
    ├── bootstrap.py                        # do NOT edit — generic reconciler
    └── roles.yaml                          # this is what developers edit
```

`aoh-compose` lays down the full template AND wires the `project-aas-init`
compose service into `compose/iams/compose.yml`. Developers only ever edit
`roles.yaml` — never the script, never the Dockerfile, never the compose
service definition.

**`roles.yaml` schema** — single canonical file modeling every AAS authz
concept; sections are kept (possibly empty) so the file teaches the schema:

```yaml
tenant: development

roles:           # RBAC role definitions
  - { name: <role>, description: <text>, required: false }

assignments:     # user → role
  - { username: <user>, roles: [<role>, ...] }

tenant_admins:   # full admins of THIS tenant (scoped to `tenant:` above, not all
  - <username>   # tenants the user belongs to); member added first if needed

groups:          # GBAC group definitions (with optional hierarchy)
  - { name: <group>,
      parent: <parent-group-or-omit>,
      extend_children: false,
      roles: [<role>, ...],
      members: [<username>, ...] }

resources:       # AAS protected entities
  - { name: <resource>, type: <type>, displayName: <label> }

scopes:          # AAS actions
  - { name: <scope> }

resource_scopes: # which scopes apply to which resource
  - { resource: <resource>, scopes: [<scope>, ...] }

permissions:     # RBAC + GBAC + UBAC compose here
  - { resource: <resource>, scope: <scope>,
      roles:  [<role>, ...],
      groups: [<group>, ...],
      users:  [<username>, ...] }
```

**`bootstrap.py` contract** — walks the sections in dependency order:

```
roles → groups → resources → scopes → resource_scopes
      → assignments → tenant_admins → permissions
```

Each step is **GET-then-create** (idempotent: re-runs are no-ops if state
matches) except `permissions`, which is **PUT-the-whole-array** (YAML is
source of truth — removing a name revokes the grant on next run).

**Compose service** — `aoh-compose` adds this to `compose/iams/compose.yml`
automatically (developers do not wire it manually):

```yaml
project-aas-init:
  build:
    context: ./init/project-aas
  environment:
    IAMS_AAS_ENDPOINT: http://iams-aas:8080
    IAM_ENDPOINT: http://iams-keycloak:8080
    IAM_REALM: aoh
    IAM_CLIENT_ID: iams
    DEV_USER: ${DEV_USER}
    DEV_PASSWORD: ${DEV_PASSWORD}
  depends_on:
    iams-init:
      condition: service_completed_successfully
  restart: on-failure
```

`depends_on: iams-init` with `condition: service_completed_successfully`
guarantees the platform's `development` tenant exists before the project
bootstrap runs.

**Reproducibility surface** — what lives where for a fresh
`docker compose up`:

| Concern | Reproducible via |
|---------|------------------|
| OIDC client registration | `realm-import.json` (Keycloak first-start import) |
| Seed user accounts | `realm-import.json` (`users` array) |
| Realm-level claim mappers, user-profile policy | `realm-import.json` |
| The `development` AAS tenant itself | platform's `iams-aas-init` Newman collection |
| **All application authz state** (roles, groups, resources, scopes, permissions, assignments) | **`compose/iams/init/project-aas/roles.yaml` + `bootstrap.py`** |

Keycloak-level concerns stay in JSON. Tenant-level (AAS) concerns stay in
YAML driven by the script. Both are committed to the repo and both run on
every fresh stack-up.

**Scope of this artifact**

This bootstrap is **development-only**. Production AAS configuration is
owned by the deployment pipeline — different tenant name, different role
matrix, different secrets management. Agents recommending edits to
`roles.yaml` for production state are wrong; redirect to the deployment
pipeline's source of truth instead.

**What agents MUST do when adding application authz**

1. Edit `compose/iams/init/project-aas/roles.yaml` only. Never edit
   `realm-import.json` to add application roles, and never edit
   `iams-aas-init.postman_collection.json` (upstream-owned).
2. Append the new role / group / resource / scope / grant to the
   appropriate section of `roles.yaml`. Do not invent new top-level keys.
3. In the change's `tasks.md`, the verification step MUST obtain a token
   for an affected user and confirm the JWT `active_tenant.roles` claim
   contains the expected role. Anything less leaves silent 403s lurking.
4. If a service REQUIRES the role at startup (e.g. an integration test
   that authenticates as a `field-reporter`), that service's compose entry
   MUST `depends_on: project-aas-init` with
   `condition: service_completed_successfully`.

## API Endpoints

### Keycloak (Standard OIDC)

| Endpoint | Purpose |
|----------|---------|
| `GET /realms/aoh/.well-known/openid-configuration` | OIDC discovery |
| `GET /realms/aoh/protocol/openid-connect/certs` | JWKS (public keys for JWT validation) |
| `POST /realms/aoh/protocol/openid-connect/token` | Token exchange (auth code, client credentials, refresh) |
| `GET /realms/aoh/protocol/openid-connect/userinfo` | User info / online token validation |
| `POST /realms/aoh/protocol/openid-connect/token/introspect` | Token introspection (confidential clients) |
| `POST /realms/aoh/protocol/openid-connect/logout` | End session |
| `GET /realms/aoh` | Realm info (includes public_key for offline validation) |

### AAS — Users

All AAS endpoints use the `/admin/` prefix. Pagination via `first` (offset) and `max`
(defaults to 100) query params.

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/admin/users` | List users (searchable by username, name, email) |
| `POST` | `/admin/users` | Create user |
| `GET` | `/admin/users/count` | Get user count |
| `GET` | `/admin/users/{userId}` | Get user by UUID |
| `PUT` | `/admin/users/{userId}` | Update user |
| `DELETE` | `/admin/users/{userId}` | Delete user |
| `PUT` | `/admin/users/{userId}/reset-password` | Set new password for user |
| `GET` | `/admin/users/{userId}/tenants/memberships` | List user's tenant memberships |

### AAS — Tenants

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/admin/tenants` | List tenants (searchable) |
| `POST` | `/admin/tenants` | Create tenant |
| `GET` | `/admin/tenants/count` | Get tenant count |
| `GET` | `/admin/tenants/{tenantId}` | Get tenant |
| `DELETE` | `/admin/tenants/{tenantId}` | Delete tenant |

### AAS — Tenant Memberships

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/admin/tenants/{tenantId}/memberships` | List tenant members (searchable) |
| `GET` | `/admin/tenants/{tenantId}/memberships/count` | Get member count |
| `POST` | `/admin/tenants/{tenantId}/memberships/{userId}` | Add user to tenant |
| `DELETE` | `/admin/tenants/{tenantId}/memberships/{userId}` | Remove user from tenant |
| `GET` | `/admin/tenants/{tenantId}/nonmemberships` | List non-members (searchable) |
| `GET` | `/admin/tenants/{tenantId}/nonmemberships/count` | Get non-member count |

### AAS — Tenant Roles

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/admin/tenants/{tenantId}/roles` | List roles (searchable) |
| `POST` | `/admin/tenants/{tenantId}/roles` | Create role |
| `GET` | `/admin/tenants/{tenantId}/roles/{role-name}` | Get role by name |
| `GET` | `/admin/tenants/{tenantId}/roles-by-id/{role-id}` | Get role by ID |
| `PUT` | `/admin/tenants/{tenantId}/roles/{role-name}` | Update role |
| `DELETE` | `/admin/tenants/{tenantId}/roles/{role-name}` | Delete role |
| `GET` | `/admin/tenants/{tenantId}/roles/{role-name}/users` | List users with role |
| `POST` | `/admin/tenants/{tenantId}/roles/{role-name}/users` | Assign users to role |
| `DELETE` | `/admin/tenants/{tenantId}/roles/{role-name}/users` | Unassign users from role |
| `GET` | `/admin/tenants/{tenantId}/roles/{role-name}/users/available` | List users without role |
| `GET` | `/admin/tenants/{tenantId}/users/{userId}/roles` | List user's roles |
| `POST` | `/admin/tenants/{tenantId}/users/{userId}/roles` | Assign roles to user |
| `DELETE` | `/admin/tenants/{tenantId}/users/{userId}/roles` | Unassign roles from user |
| `GET` | `/admin/tenants/{tenantId}/users/{userId}/roles/available` | List assignable roles |

### AAS — Tenant Groups

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/admin/tenants/{tenantId}/groups` | List top-level groups |
| `POST` | `/admin/tenants/{tenantId}/groups` | Create group |
| `GET` | `/admin/tenants/{tenantId}/groups/{groupId}` | Get group |
| `PUT` | `/admin/tenants/{tenantId}/groups/{groupId}` | Update group (name only) |
| `DELETE` | `/admin/tenants/{tenantId}/groups/{groupId}` | Delete group |
| `GET` | `/admin/tenants/{tenantId}/groups/{groupId}/children` | List child groups |
| `POST` | `/admin/tenants/{tenantId}/groups/{groupId}/children` | Create child group |
| `GET` | `/admin/tenants/{tenantId}/groups/{groupId}/members` | List group members |
| `PUT` | `/admin/tenants/{tenantId}/users/{userId}/groups/{groupId}` | Add user to group |
| `DELETE` | `/admin/tenants/{tenantId}/users/{userId}/groups/{groupId}` | Remove user from group |
| `GET` | `/admin/tenants/{tenantId}/users/{userId}/groups` | List user's groups |
| `GET` | `/admin/tenants/{tenantId}/groups/{groupId}/roles` | List group roles |
| `POST` | `/admin/tenants/{tenantId}/groups/{groupId}/roles` | Assign roles to group |
| `DELETE` | `/admin/tenants/{tenantId}/groups/{groupId}/roles` | Unassign roles from group |
| `GET` | `/admin/tenants/{tenantId}/groups/{groupId}/roles/available` | List assignable roles |

### AAS — Resources & Scopes

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/admin/tenants/{tenantId}/resources` | List resources (filter by name, type) |
| `POST` | `/admin/tenants/{tenantId}/resources` | Create resource |
| `GET` | `/admin/tenants/{tenantId}/resources/{resourceId}` | Get resource |
| `PUT` | `/admin/tenants/{tenantId}/resources/{resourceId}` | Update resource |
| `DELETE` | `/admin/tenants/{tenantId}/resources/{resourceId}` | Delete resource |
| `GET` | `/admin/tenants/{tenantId}/resources/{resourceId}/scopes` | List resource scopes |
| `PUT` | `/admin/tenants/{tenantId}/resources/{resourceId}/scopes/{scopeId}` | Add scope to resource |
| `DELETE` | `/admin/tenants/{tenantId}/resources/{resourceId}/scopes/{scopeId}` | Remove scope from resource |
| `GET` | `/admin/tenants/{tenantId}/scopes` | List all tenant scopes |
| `POST` | `/admin/tenants/{tenantId}/scopes` | Create scope |
| `PUT` | `/admin/tenants/{tenantId}/scopes/{scopeId}` | Update scope |
| `DELETE` | `/admin/tenants/{tenantId}/scopes/{scopeId}` | Delete scope |
| `GET` | `/admin/tenants/{tenantId}/scopes/{scopeId}/resources` | List resources with scope |

### AAS — Resource Permissions

Manage who (users/roles/groups) has access to resource+scope combinations.

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/admin/tenants/{tenantId}/resources/{resourceId}/scopes/{scopeId}/permissions/users` | List user permissions |
| `PUT` | `/admin/tenants/{tenantId}/resources/{resourceId}/scopes/{scopeId}/permissions/users` | Update user permissions (array of userIds) |
| `GET` | `/admin/tenants/{tenantId}/resources/{resourceId}/scopes/{scopeId}/permissions/roles` | List role permissions |
| `PUT` | `/admin/tenants/{tenantId}/resources/{resourceId}/scopes/{scopeId}/permissions/roles` | Update role permissions (array of RoleDefinition) |
| `GET` | `/admin/tenants/{tenantId}/resources/{resourceId}/scopes/{scopeId}/permissions/groups` | List group permissions |
| `PUT` | `/admin/tenants/{tenantId}/resources/{resourceId}/scopes/{scopeId}/permissions/groups` | Update group permissions (array of GroupDefinition) |

### AAS — Permission Evaluation

Check if a user has access to a specific resource+scope. Returns `"PERMIT"` or `"DENY"`.

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/admin/tenants/{tenantId}/users/{userId}/resources/{resourceId}/scopes/{scopeId}/evaluate` | Evaluate user access (by IDs) |
| `GET` | `/admin/tenants/{tenantId}/users/{userId}/resources-by-name/{resourceName}/scopes-by-name/{scopeName}/evaluate` | Evaluate user access (by names) |
| `GET` | `/admin/tenants/{tenantId}/users/{userId}/roles/{role-name}/resources/{resourceId}/scopes/{scopeId}/evaluate` | Evaluate user+role access (by IDs) |
| `GET` | `/admin/tenants/{tenantId}/users/{userId}/roles/{role-name}/resources-by-name/{resourceName}/scopes-by-name/{scopeName}/evaluate` | Evaluate user+role access (by names) |
| `GET` | `/admin/tenants/{tenantId}/users/{userId}/resources` | List all accessible resources for user |
| `GET` | `/admin/tenants/{tenantId}/users/{userId}/resources-by-type/{resourceType}` | List accessible resources by type |
| `GET` | `/admin/tenants/{tenantId}/users/{userId}/resources/{resourceId}/scopes` | List user's allowed scopes on resource (by ID) |
| `GET` | `/admin/tenants/{tenantId}/users/{userId}/resources-by-name/{resourceName}/scopes` | List user's allowed scopes on resource (by name) |

### AAS — Admin Management

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/admin/sys-admin` | Get system admin role definition |
| `GET` | `/admin/sys-admin/users` | List system admins |
| `POST` | `/admin/sys-admin/users/{userId}` | Assign system admin role |
| `DELETE` | `/admin/sys-admin/users/{userId}` | Unassign system admin role |
| `GET` | `/admin/tenants/{tenantId}/tenant-admin/users` | List tenant admins |
| `GET` | `/admin/tenants/{tenantId}/tenant-admin/users/count` | Get tenant admin count |
| `POST` | `/admin/tenants/{tenantId}/tenant-admin/users/{userId}` | Assign tenant admin |
| `DELETE` | `/admin/tenants/{tenantId}/tenant-admin/users/{userId}` | Unassign tenant admin |

### AAS — Realm

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/admin/realm/password-policy` | Get configured password policy |

## Access

| URL | What |
|-----|------|
| `http://iams-keycloak.${DEV_DOMAIN}` | Keycloak admin console |
| `http://iams-aas.${DEV_DOMAIN}` | AAS API |
| `http://iams-aas.${DEV_DOMAIN}/swagger-ui/index.html` | Swagger UI |
| `http://iams-web.${DEV_DOMAIN}` | User management UI |

## Dependencies

- **sds** (session data store — included automatically)
