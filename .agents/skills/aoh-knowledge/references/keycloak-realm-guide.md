# Keycloak Realm Configuration Guide

> **Scope:** this guide covers **platform-bootstrap** realm configuration —
> editing `realm-import.json` to extend the realm itself (clients, claim
> mappers, user-profile policy, the small set of platform realm roles). It is
> NOT how applications add roles, scopes, or resources. **Application-level
> authorization (roles, scopes, resources, group memberships, user-to-role
> assignments) goes through the AAS API, not through this file.** See
> `services/iams.md` → "Authorization is via AAS, not Keycloak" for the
> application path. Touch this guide only when you have a genuine platform
> bootstrap concern (e.g. registering a new OIDC client for a new app, adding
> a custom claim mapper) — not for adding `field-reporter`-style app roles.

## Bootstrap Asset

The `assets/realm-import.json` is a ready-to-use Keycloak realm config. Copy it to
`infra/iams/keycloak/realm-import.json`. It already includes:

- **Admin user** (`${DEV_USER}`, UUID: `f67cb8f5-0645-444d-a4bd-c61aaf1b2db0`)
- **SDS service account** (`service-account-sds`, with realm-management roles)
- **Core clients:** `iams` (public), `web` (public template), `sds` (confidential)
- **Built-in Keycloak clients:** account, account-console, admin-cli, broker, realm-management, security-admin-console
- **Realm roles:** system-admin, tenant-admin, default-roles-aoh, etc.
- **Client scopes:** `aoh_default_scope` (with `all_tenants` and `active_tenant` claim mappers) + all standard OIDC scopes

The file uses Keycloak env var substitution: `${DEFAULT_REALM}` (resolves to "aoh"),
`${DEV_DOMAIN}`, `${DEV_USER}`, `${DEV_PASSWORD}`.

## Customize by extending the `aoh` realm

Platform-level extensions (new OIDC clients, new claim mappers) go into the
existing `compose/iams/keycloak/realm-import.json`. Don't add a second realm
JSON — the `aoh` realm already configures `aoh_default_scope` (sub,
realm_access.roles, preferred_username, active_tenant, all_tenants claims),
user-profile policy, and tenant seeding via `iams-init`. A parallel realm
inherits none of that.

> Application roles (`field-reporter`, `operations-team`, etc.) are NOT
> realm roles and DO NOT belong in this file. Create them via AAS:
> `POST /admin/tenants/{tenantId}/roles`. Realm roles are reserved for
> platform-wide concerns like `system-admin` / `tenant-admin`, which the
> bootstrap already defines.

### Adding a seed user

Append to `users`. The `active_tenant` attribute is a single-element array
wrapping a JSON-serialized object; `aoh_default_scope` parses it into a nested
claim at token issuance.

```json
{
    "username": "<username>",
    "enabled": true,
    "emailVerified": true,
    "firstName": "<first>",
    "lastName": "<last>",
    "email": "<email>",
    "credentials": [{ "type": "password", "value": "${DEV_PASSWORD}", "temporary": false }],
    "realmRoles": ["default-roles-aoh"],
    "attributes": {
        "active_tenant": ["{\"tenant_id\":\"<tenant>\",\"tenant_name\":\"<label>\",\"roles\":[\"<app-role>\"]}"]
    }
}
```

Gotchas:

- `realmRoles` must include `default-roles-aoh`, or login fails with
  "Account is not fully set up."
- **Do NOT add app-level roles** (`field-reporter`, `operations-team`, etc.) to
  `realmRoles`. Only platform realm roles like `system-admin` / `tenant-admin`
  belong here — and most projects don't need to add either. App roles are AAS
  tenant roles, assigned via `project-aas-init` (see `services/iams.md` →
  "Project-level AAS bootstrap (reproducibility pattern)").
- The `active_tenant.roles` value above seeds the JWT shape on first login.
  AAS-managed role assignments via `project-aas-init` are what drive runtime
  authorization checks — the static seed is just a starting placeholder.
- Empty `credentials: []` is valid for non-interactive service-account users.

### Re-importing after edits

> ⚠️ **`kc.sh start --import-realm` is skip-if-exists.**
> Editing `realm-import.json` and running `docker compose up -d --force-recreate iams-keycloak` does **NOT** apply your changes — Keycloak logs `Realm 'aoh' already exists. Import skipped` and keeps the version already persisted in `iams-db`. Forgetting this wastes hours.

To apply changes to a running stack:

```bash
cd compose
docker compose down -v iams-db && docker compose up -d
```

This drops the `aoh_iams-db-volume` Postgres volume, which forces Keycloak to re-import the realm from `realm-import.json` on next start.

**Side effect:** the IAMS Postgres is shared with AAS, so AAS state is also wiped — `project-aas-init` will re-run from `compose/iams/init/project-aas/roles.yaml` and re-seed your tenant roles, groups, assignments, etc. That's a *feature*: it proves your `realm-import.json` + `roles.yaml` are a complete reproducible boot, which is what the project authz contract requires (a fresh `down -v && up -d` MUST reproduce all external-system state).

> 🛠 **Debugging-only escape hatch.**
> You *can* PATCH the running Keycloak via its admin REST API (`POST /admin/realms/aoh/clients`, `POST /admin/realms/aoh/roles`, etc., with a master-realm `admin-cli` bearer token) when you're mid-debug and don't want to wait through a cold boot. **Do not** use this as a way to customize the realm. The running Keycloak's state is ephemeral — the next `down -v` wipes it, and any change not also reflected in `realm-import.json` is lost. CI and fresh-clone bootstraps will be wrong. If you find yourself reaching for the admin API, the bug is that you need to update `realm-import.json` and reboot anyway; the admin API just postpones that.

## Adding Clients for Supporting Services

When supporting AOH services are included, add their Keycloak client to the `clients` array in `realm-import.json`. Use the templates below.

> 🚫 **Do not register a per-app Keycloak client for a new AOH web app you're scaffolding in this dev realm.**
> All AOH web apps share the bundled `web` client (`clientId: "web"`), which has wildcard `redirectUris: ["*"]` and `webOrigins: ["+"]`. Any subdomain under `${DEV_DOMAIN}` works with zero realm edits. The `aoh-compose` web template ships with `IAM_CLIENT_ID: web` for this reason. Wildcard redirects are a dev convenience and a prod security hole, so per-app web clients belong in the **prod deployment realm config**, not in this dev `realm-import.json`. If you find yourself adding a `web-base`/`my-app-web` client here, stop — you don't need it.

The section below covers two other cases that DO require entries in this realm:
1. **Supporting AOH services** (rtus, ian, etc.) — each has its own platform-defined client that may or may not be bundled, see the table.
2. **Confidential service clients** — backends that exchange tokens directly with Keycloak (service-to-service or service-account flows). Not OIDC end-user logins.

### Which services need which clients

| clientId | Type | Name | baseUrl pattern |
|----------|------|------|-----------------|
| ptmgr | Confidential | Push Token Manager | ptmgr.${DEV_DOMAIN} |

> **Modules delivered as SDKs get no dedicated Keycloak client here.** DASH, RTUS, UNH, Form, and GIS no longer ship a `*-web` console container — consumer apps embed them via the module SDK (`@mssfoobar/dash-web-sdk`, `@mssfoobar/unh-web-sdk`, `@mssfoobar/form-web-sdk`, `@mssfoobar/gis-web-sdk`; real-time via `@mssfoobar/sse-client`) and log in through the shared `web` client like any other AOH web app. Their backends need no dedicated public client either: `unh` and `gis` forward the bearer to `iams-aas`, `form` uses the bundled `sds` service-account client, and `dash-app`/`rtus-pms`/`rtus-seh` validate JWTs against the realm directly. The old `gis` / `unh-web` / `unh-app` / `form-web` clients have accordingly been dropped from `realm-import.json`. If a specific consumer genuinely needs a per-module client, it's the `aoh-*-integration` skill's job to register it — don't pre-bake it here.

Note: `iams`, `web`, `sds`, `ian`, and `ptmgr` are already in the bootstrap — don't duplicate them.

### Public Client Template

For web frontends using OIDC authorization code flow. Replace `<service-name>`, `<Human Readable Name>`, and `<service>`:

```json
{
    "clientId": "<service-name>",
    "name": "<Human Readable Name>",
    "description": "",
    "baseUrl": "http://<service>-web.${DEV_DOMAIN}",
    "enabled": true,
    "clientAuthenticatorType": "client-secret",
    "secret": "${DEV_PASSWORD}",
    "redirectUris": [
        "http://<service>-web.${DEV_DOMAIN}/*",
        "http://${DEV_DOMAIN}:5173/*",
        "http://${DEV_DOMAIN}:4173/*"
    ],
    "webOrigins": ["+"],
    "standardFlowEnabled": true,
    "directAccessGrantsEnabled": true,
    "serviceAccountsEnabled": false,
    "publicClient": true,
    "frontchannelLogout": true,
    "protocol": "openid-connect",
    "attributes": {
        "oidc.ciba.grant.enabled": false,
        "backchannel.logout.session.required": "true",
        "post.logout.redirect.uris": "+",
        "display.on.consent.screen": "false",
        "oauth2.device.authorization.grant.enabled": "false",
        "backchannel.logout.revoke.offline.tokens": "false"
    },
    "fullScopeAllowed": true,
    "protocolMappers": [
        {
            "name": "Client IP Address",
            "protocol": "openid-connect",
            "protocolMapper": "oidc-usersessionmodel-note-mapper",
            "consentRequired": false,
            "config": {
                "user.session.note": "clientAddress",
                "id.token.claim": "true",
                "introspection.token.claim": "true",
                "userinfo.token.claim": "true",
                "access.token.claim": "true",
                "claim.name": "clientAddress",
                "jsonType.label": "String"
            }
        },
        {
            "name": "Client ID",
            "protocol": "openid-connect",
            "protocolMapper": "oidc-usersessionmodel-note-mapper",
            "consentRequired": false,
            "config": {
                "user.session.note": "client_id",
                "id.token.claim": "true",
                "introspection.token.claim": "true",
                "userinfo.token.claim": "true",
                "access.token.claim": "true",
                "claim.name": "client_id",
                "jsonType.label": "String"
            }
        },
        {
            "name": "Client Host",
            "protocol": "openid-connect",
            "protocolMapper": "oidc-usersessionmodel-note-mapper",
            "consentRequired": false,
            "config": {
                "user.session.note": "clientHost",
                "id.token.claim": "true",
                "introspection.token.claim": "true",
                "userinfo.token.claim": "true",
                "access.token.claim": "true",
                "claim.name": "clientHost",
                "jsonType.label": "String"
            }
        }
    ],
    "defaultClientScopes": [
        "web-origins", "acr", "profile", "roles", "basic", "aoh_default_scope", "email"
    ],
    "optionalClientScopes": [
        "address", "phone", "offline_access", "microprofile-jwt"
    ]
}
```

### Confidential Client Template

For backend services needing machine-to-machine auth. Same as public template but change:

```json
{
    "publicClient": false,
    "serviceAccountsEnabled": true,
    "attributes": {
        "realm_client": "false",
        "oidc.ciba.grant.enabled": "false",
        "backchannel.logout.session.required": "true",
        "post.logout.redirect.uris": "+",
        "display.on.consent.screen": "false",
        "oauth2.device.authorization.grant.enabled": "false",
        "backchannel.logout.revoke.offline.tokens": "false"
    }
}
```

Also add a service-account user to the `users` array:

```json
{
    "username": "service-account-<clientId>",
    "emailVerified": false,
    "enabled": true,
    "serviceAccountClientId": "<clientId>",
    "credentials": [],
    "realmRoles": ["system-admin", "default-roles-aoh"],
    "groups": []
}
```

## IAMS Init Postman Collection

The `assets/iams-aas-init.postman_collection.json` initializes IAMS AAS (not Keycloak itself).
Copy it to `infra/iams/init/iams-aas-init.postman_collection.json`.

**What it does:**
1. Resets admin password in Keycloak
2. Creates a "development" tenant in AAS (if none exists)
3. Assigns the admin user to the tenant

**Why it's needed:** The realm-import.json sets up Keycloak (realm, users, clients, roles).
The Postman collection sets up AAS (tenants, memberships). Services that use AAS for
authorization (UNH, PTMGR, AMM) require a tenant to exist.

**Compose service definition** (already included in `references/services/iams.md`):
```yaml
iams-init:
    image: postman/newman
    command:
        - run
        - /etc/newman/iams-aas-init.postman_collection.json
        - --env-var
        - IAMS_AAS_ENDPOINT=http://iams-aas:8080
        - --env-var
        - IAM_ENDPOINT=http://iams-keycloak:8080
        - --env-var
        - IAM_REALM=aoh
        - --env-var
        - DEV_USER=${DEV_USER}
        - --env-var
        - DEV_PASSWORD=${DEV_PASSWORD}
    volumes:
        - ../iams/init:/etc/newman
    depends_on:
        iams-keycloak:
            condition: service_healthy
        iams-aas:
            condition: service_healthy
    restart: on-failure
    deploy:
        restart_policy:
            delay: 20s
```
