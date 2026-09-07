# AOH Platform Overview

AOH (Agil Ops Hub) is a modular platform purpose-built for developing Command and Control
(C2) applications — systems used by operators to monitor situations, coordinate resources,
and respond to events in real-time. Think operations centres, emergency dispatch, defence
command systems, and incident management platforms.

It provides a set of shared services that handle cross-cutting concerns common to C2
systems — authentication, real-time situational awareness, geospatial tracking, notifications,
file management, and workflow orchestration — so product teams can focus
on domain logic rather than infrastructure.

## Architecture

```
                         ┌─────────────────────────────────┐
                         │          Traefik (infra)         │
                         │   Reverse proxy & API gateway    │
                         │   Port 80 — all HTTP traffic     │
                         └──────────┬──────────────────────┘
                                    │
              ┌─────────────────────┼─────────────────────┐
              │                     │                     │
     ┌────────▼────────┐   ┌───────▼───────┐    ┌───────▼────────┐
     │   Your App(s)   │   │  AOH Services │    │   AOH Web UIs  │
     │ (Go, SvelteKit) │   │  (backends)   │    │  (SvelteKit)   │
     └────────┬────────┘   └───────┬───────┘    └───────┬────────┘
              │                     │                     │
              └─────────────────────┼─────────────────────┘
                                    │
                         ┌──────────▼──────────┐
                         │   IAMS (Keycloak)   │
                         │  Auth & AuthZ core  │
                         └─────────────────────┘
```

All HTTP traffic routes through **Traefik** using host-based routing. Each service gets
its own subdomain: `<service>.127.0.0.1.nip.io` in local dev.

**IAMS** (Keycloak + AAS) is the auth backbone — nearly every service depends on it for
JWT validation, role-based access, and tenant context.

## Tech Stack

| Layer | Technology | Notes |
|-------|-----------|-------|
| Reverse proxy | Traefik v3.6.1 | Host-based routing, CORS middleware, Docker labels |
| Auth | Keycloak (OIDC) | OpenID Connect, realm `aoh`, multi-tenant |
| Authorization | AAS (custom) | Attribute-based access, tenant-scoped roles |
| Backend services | Java (Spring Boot) / Go (Chi) | Most AOH services are Spring Boot; new services use Go |
| Frontend services | SvelteKit (Svelte 5) | Runes syntax, OIDC via openid-client v6 |
| Databases | PostgreSQL 17 | Per-service databases |
| Cache | Valkey (Redis-compatible) | Used by SDS for session data |
| Real-time | Hazelcast + SSE | RTUS manages pub/sub; browsers connect via SSE |
| Object storage | MinIO (S3-compatible) | Opt-in for AMM; AMM defaults to a local volume |
| Container images | `ghcr.io/mssfoobar/<module>/<component>` | All AOH images in GitHub Container Registry |
| Workflow engine | Temporal | Used by WFE for durable workflow orchestration |

## Service Categories

### Core (always present)

- **infra** — Traefik reverse proxy. Required by everything.
- **iams** — Keycloak + AAS. Authentication and authorization.
- **sds** — Session data store. Included automatically with IAMS.

### Real-time & Geospatial

- **rtus** — Real-time pub/sub and SSE delivery to browsers.
- **gis** — Geospatial data, map overlays, entity tracking.

### Notifications

- **unh** — Multi-channel notifications (email, push, SMS, custom).
- **ian** — In-app notifications with real-time delivery via RTUS.
- **ptmgr** — Firebase Cloud Messaging token management.

### Data Management

- **amm** — File upload/download on a local volume by default (MinIO opt-in), optional virus scanning.
- **dash** — Configurable widget dashboards (owns its own tags).

### Auxiliary systems

**WFE** is a standalone product with its own infrastructure stack. It is not part of the
shared AOH service stack but extends C2 system capabilities:
- **wfe** — BPMN 2.0 workflow orchestration via Temporal

## Routing Model

Traefik uses Docker labels for service discovery. Each service declares:

```yaml
labels:
    - traefik.enable=true
    - traefik.http.routers.<name>.rule=Host(`<name>.${DEV_DOMAIN}`)
    - traefik.http.routers.<name>.entrypoints=web
    - traefik.http.routers.<name>.middlewares=permissive-headers@docker
    - traefik.http.services.<name>.loadBalancer.server.port=8080
```

- **DEV_DOMAIN** defaults to `127.0.0.1.nip.io` (resolves to localhost)
- **Entry points**: `web` (port 80) for HTTP, `sdstcp` (port 5333) for SDS TCP
- **CORS**: A global `permissive-headers` middleware allows all origins in dev. RTUS-SEH
  uses its own stricter CORS for SSE credential handling.

## Multi-tenancy

AOH supports multi-tenancy at the platform level:

- Keycloak tokens include an `active_tenant` claim
- AAS enforces tenant-scoped roles and permissions
- Each service filters data by tenant ID
- The platform bootstraps a Keycloak realm (`aoh`, via realm-import) and an AAS tenant (`development`, via `iams-init`) — **reuse both** when scaffolding new services or writing change tasks. Do NOT create per-service realms or tenants for local development — that's a deployment/onboarding concern, not a per-service one.
- **Authorization flows through AAS, not Keycloak.** AAS sits in front of Keycloak as the single authorization control plane. Anything an application needs — roles, scopes, resources, group memberships, user-to-role assignments — goes through the AAS API (`POST /admin/tenants/{tenantId}/roles`, etc.). The `active_tenant.roles` JWT claim is populated from AAS tenant role assignments. Applications MUST NOT create realm roles, realm scopes, or modify realm-import.json — those are platform-bootstrap concerns, not application concerns.
- **Where things go:** OIDC clients (backend confidential, frontend PKCE) → the `aoh` realm. Application roles, scopes, resources, group memberships, seed users with their role assignments → AAS via the `development` tenant.

## Image Tagging

All AOH images follow: `ghcr.io/mssfoobar/<module>/<component>:<tag>`

Tags use environment variables: `<MODULE>_<COMPONENT>_TAG=${GLOBAL_TAG}`
This allows upgrading all services at once by changing `GLOBAL_TAG`.
