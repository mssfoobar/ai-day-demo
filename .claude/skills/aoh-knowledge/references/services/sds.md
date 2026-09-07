# SDS — Session Data Store

> 🛑 **SDS is MANDATORY for every AOH web application.** It is the platform's
> standard session backend; cookie-only auth is a fallback meant for emergency
> diagnostics, NOT a deployable mode. Every AOH web app — local dev, CI, and
> production — sets `SDS_URL` to a reachable `sds-server` and stores tokens
> server-side. See "Why SDS is mandatory" below for the rationale; see
> `aoh-web-init` for the env wiring (`SDS_URL=tcp://sds-server:5333` in
> compose, `tcp://127.0.0.1:5333` for native `pnpm dev`).

## What It Does

SDS provides server-side session data storage for AOH web applications. It manages two
types of stores: **authenticated stores** (tied to Keycloak JWT sessions) and **temporary
stores** (short-lived, no auth required). Each store holds arbitrary key-value pairs.

SDS is automatically included when you include IAMS — the `sds-server` + `valkey`
services are part of the IAMS compose include, so every project that boots IAMS
already has SDS. There is no "skip SDS" mode.

## Why SDS is mandatory

The cookie-only fallback path (no `SDS_URL`) exists in `auth.ts` for emergency
diagnostics — e.g. SDS is down and you need to confirm Keycloak still works —
but it is **not a posture any AOH service is allowed to ship in.** Four
load-bearing reasons:

1. **Tokens never reach the browser.** With SDS, only a session ID cookie is
   set; the access and refresh tokens stay in `valkey` behind the SvelteKit
   server. With the cookie-only fallback, both tokens are HTTP-only cookies
   served to the browser — anyone who exfiltrates them (via an XSS that finds
   a `document.cookie`-adjacent surface, a sloppy CSP, or a logged response
   header) has a usable bearer token until expiry.
2. **Server-side refresh.** `sds-server`'s background job (`CRON_INTERVAL=10s`)
   refreshes tokens against Keycloak before they expire, so the browser only
   ever sees a session ID. Cookie-only mode requires the SvelteKit server to
   refresh on every protected request, doubling the call volume to Keycloak
   and racing across concurrent tabs.
3. **Logout is replay-safe.** `authSessionDestroy` removes the tokens from
   Valkey, so even if a captured session cookie is replayed it resolves to a
   dead store. Cookie-only logout can only clear cookies in the current
   browser — leaked tokens stay valid until natural expiry.
4. **RTUS-SEH and other server-side consumers read SDS directly.** Adding
   real-time features later (RTUS subscriptions, server-pushed events)
   requires SDS to be the source of truth for the session. Building on the
   cookie-only flow means re-doing the auth wiring when you add RTUS.

If you find yourself reasoning about whether SDS is necessary, the answer is
yes — every time, in every environment.

## Architecture

```
Browser ──► SvelteKit App ──► SDS Server ──► Valkey (Redis-compatible)
                                   │
                              Keycloak
                           (token refresh/validation)
```

## Components

| Component | Image | Purpose |
|-----------|-------|---------|
| **valkey** | `valkey/valkey:8.1` | Redis-compatible in-memory cache |
| **sds-server** | `ghcr.io/mssfoobar/sds/sds-server` | HTTP + TCP API for session CRUD |

## Key Concepts

### Dual protocol

SDS exposes two interfaces:

| Protocol | Port | Used by |
|----------|------|---------|
| HTTP | 5080 | Backend services, RTUS-SEH |
| TCP | 5333 | JavaScript clients (via Traefik TCP entry point) |

### Store types

**Authenticated store** (`auth-store`) — Created with a JWT (accessToken + refreshToken).
SDS manages token lifecycle: `GET .../accessToken` returns a valid token, refreshing it
automatically if needed. Used for user login sessions — the store ID becomes the session
cookie value.

**Temporary store** (`temp-store`) — Short-lived store with no authentication required.
Created with a simple POST. Used for ephemeral data like CSRF tokens or pre-login state.

Both store types support arbitrary key-value pairs (string keys, string values).

### Session lifecycle

1. User logs in via Keycloak
2. SvelteKit app creates an authenticated store in SDS with the access + refresh tokens
3. Store ID is saved in a cookie (e.g., `web_auth_session_id`)
4. Subsequent requests include the session cookie
5. Backend retrieves session data or a valid access token from SDS by store ID
6. RTUS-SEH reads session data from SDS for auth context on SSE connections

### Auth

SDS uses a **confidential** Keycloak client (`sds-client`) with a service account.

## API Endpoints

### Authenticated Store

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `POST` | `/session/auth-store` | Create store (body: `{ accessToken, refreshToken }`) |
| `DELETE` | `/session/auth-store/{store_id}` | Delete store |
| `GET` | `/session/auth-store/{store_id}/accessToken` | Get valid access token (auto-refreshes if expired) |
| `GET` | `/session/auth-store/{store_id}/keys` | Get all key-value pairs |
| `PUT` | `/session/auth-store/{store_id}/keys` | Upsert key-value pairs (body: `{ key: value, ... }`) |
| `DELETE` | `/session/auth-store/{store_id}/keys` | Delete all key-value pairs |
| `GET` | `/session/auth-store/{store_id}/keys/{key}` | Get value for specific key |
| `PUT` | `/session/auth-store/{store_id}/keys/{key}` | Update value for specific key (body: `{ value }`) |
| `DELETE` | `/session/auth-store/{store_id}/keys/{key}` | Delete specific key-value pair |

### Temporary Store

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `POST` | `/session/temp-store` | Create store (no body needed) |
| `DELETE` | `/session/temp-store/{store_id}` | Delete store |
| `GET` | `/session/temp-store/{store_id}/keys` | Get all key-value pairs |
| `PUT` | `/session/temp-store/{store_id}/keys` | Upsert key-value pairs (body: `{ key: value, ... }`) |
| `DELETE` | `/session/temp-store/{store_id}/keys` | Delete all key-value pairs |
| `GET` | `/session/temp-store/{store_id}/keys/{key}` | Get value for specific key |
| `PUT` | `/session/temp-store/{store_id}/keys/{key}` | Update value for specific key (body: `{ value }`) |
| `DELETE` | `/session/temp-store/{store_id}/keys/{key}` | Delete specific key-value pair |

## Data Model

```
Store creation response:
  id: string (store ID — used as session cookie value)

Session data (per key-value pair):
  id: string
  data: { key: value, ... } (string map)
  createdAt: string (timestamp)
  accessedAt: string (timestamp)
  expiresAt: string (timestamp)

Auth store creation request (JWT):
  accessToken: string
  refreshToken: string
```

## Access

| URL | What |
|-----|------|
| `http://sds.${DEV_DOMAIN}` | SDS HTTP API (via Traefik) |
| `http://sds.${DEV_DOMAIN}/swagger-ui/index.html` | Swagger UI |
| Port 5333 (via Traefik TCP) | SDS TCP interface |

## Dependencies

- **iams** (Keycloak — for token validation and refresh via service account)
