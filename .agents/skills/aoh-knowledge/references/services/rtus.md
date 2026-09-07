# RTUS — Real-time Update Service

## What It Does

RTUS provides real-time data distribution across the AOH platform. Backend services
publish data to RTUS, and browser clients receive updates instantly via Server-Sent
Events (SSE).

It's the backbone for any feature that needs live updates — map positions, notifications,
dashboard widgets, status changes.

## Architecture

```
┌──────────────┐    publish     ┌───────────┐
│ Your Backend │──────────────►│ RTUS-PMS  │ (Point Management Service)
│              │   REST API     │ Hazelcast │ (pub/sub engine)
└──────────────┘                └─────┬─────┘
                                      │ Hazelcast internal
                                      │
┌──────────────┐    SSE         ┌─────▼─────┐
│   Browser    │◄──────────────│ RTUS-SEH  │ (SSE Handler)
│              │   EventSource  │           │
└──────────────┘                └───────────┘
```

## Components

| Component | Image | Purpose |
|-----------|-------|---------|
| **rtus-db** | `postgres:17.0` | Metadata storage (topics, maps, subscriptions) |
| **rtus-pms** | `ghcr.io/mssfoobar/rtus/rtus-pms` | Pub/sub management — topics, maps, publishing (Spring Boot) |
| **rtus-seh** | `ghcr.io/mssfoobar/rtus/rtus-seh` | SSE delivery to browsers (Spring Boot) |

## Key Concepts

### Data Structures

RTUS has four types of data structures:

| Type | Model | Key | Use case |
|------|-------|-----|---------|
| **Map** | Key-value store (text values) | Any string | Entity state, positions — any key |
| **JSON Map** | Key-value store (JSON values) | Any string | Structured data with fine-grained access per entry |
| **User Value Map** | Key-value store | User ID | Per-user state (e.g., user-specific notifications) |
| **Topic** | Message stream | N/A | Events, alerts, log streams |

**Map** — General key-value store. Values are text or JSON strings. Clients subscribe
to receive change notifications (Added, Updated, Removed, Cleared). Subscribe with
`init=true` to get current state on connect.

**JSON Map** — Like Map but enforces `application/json` content type. Supports
fine-grained access control per entry via `resourceType` and `valuePath` in SSE config
(e.g., GIS uses this for per-entity access). Used by GIS for entity sync.

**User Value Map** — Like Map but each key is a User ID. Clients subscribe to a specific
user ID's value — useful for delivering per-user data (e.g., IAN uses this). Subscribe
with `init=true` to get the current value.

**Topic** — Pub/sub channel. Events are transient: default capacity 100, default TTL 300
seconds. Oldest events drop when full. Clients can specify `lastEventId` on subscribe to
resume from a point.

### Hazelcast

RTUS uses Hazelcast as its in-memory data grid for pub/sub:
- Topics are Hazelcast distributed topics
- Maps are Hazelcast distributed maps
- Supports clustering for horizontal scaling
- `KUBERNETES_SERVICE_HOST=true` is set as a flag for Hazelcast's discovery mode

> ⚠️ **Gotcha — `rtus.clustername` must match between rtus-pms and rtus-seh.**
> Hazelcast only forms a cluster between members that share the same cluster
> name. If rtus-pms sets `rtus.clustername=aoh_rtus` and rtus-seh leaves it
> default (or sets a different value), SEH will spin up its own one-node
> cluster, never see PMS's topics/maps, and SSE subscribers will get nothing
> — usually with no obvious error in either log. Always set the same
> `rtus.clustername` on both services. The committed `aoh-compose` template
> uses `aoh_rtus`.

### SSE (Server-Sent Events)

RTUS-SEH delivers events to browsers using the SSE protocol:
- Connection is long-lived (HTTP streaming)
- Connection lifecycle, reconnection-with-backoff, and heartbeat detection are
  handled by `@mssfoobar/sse-client` — do NOT hand-roll an `EventSource`
  wrapper in feature code. See "SSE client SDK" below.
- Auth: browser-side cookies travel automatically (`withCredentials: true`).
  rtus-seh resolves the cookie to a JWT and authorises the subscription. Two
  paths, in priority order:
  1. **SDS session-id cookie (default for SDS-flow apps)** — rtus-seh reads
     `web_auth_session_id` from the request, calls SDS's
     `authSessionGetAccessToken(sid)` to mint a JWT, then validates +
     authorises. This is the platform's preferred path; tokens never reach
     the browser.
  2. **Access-token cookie (legacy)** — rtus-seh reads one of
     `web_access_token`, `ian_access_token`, `gis_access_token` directly
     from the cookie. Used by older cookie-flow apps where the JWT lives
     in a browser-readable cookie.

The runtime config keys on `rtus-seh` are `rtus.session-id.cookienames` and
`rtus.access-token.cookienames`; both accept comma-separated lists. The
committed `aoh-compose` template wires them to platform-standard values.

### Cookie prefix convention

rtus-seh reads the browser's session-id cookie by **exact name** — there is no
prefix-matching. The list of names it accepts is set by the
`rtus.session-id.cookienames` property (env
`RTUS_SESSION-ID_COOKIENAMES`), a comma-separated list. The committed
`aoh-compose` template seeds it with `web_auth_session_id`.

This drives the platform default: **`aoh-web-init` ships
`PUBLIC_COOKIE_PREFIX=web` so the resulting cookie name
(`web_auth_session_id`) matches the seeded rtus-seh config without any
extra wiring.** Every web app under `${DEV_DOMAIN}` shares the parent
domain, so the same cookie covers every app — that's the SSO model.

If a project genuinely needs a per-app cookie name (e.g. for an isolation
requirement, or staging two web apps with different session lifecycles),
both halves must move together:

1. Set `PUBLIC_COOKIE_PREFIX=<my-app>` in that app's `.env` and compose
   entry.
2. Update `compose/rtus/compose.yml` so
   `rtus.session-id.cookienames` includes the new value (comma-separated).
   The same goes for `rtus.access-token.cookienames` if the app is
   still on the legacy cookie-flow.

Use the platform default unless you have a concrete reason — and when you
deviate, the rtus config edit is part of the change, not a discovery left
for a future contributor.

### SSE client SDK

The platform ships `@mssfoobar/sse-client` (currently 1.2.0) — every AOH web
app subscribes via this SDK, not raw `EventSource`. It encapsulates the URL
strategy, reconnect backoff, refresh-before-reconnect, and the
heartbeat-timeout liveness signal.

Four URL strategies, discriminated by config fields:

| Strategy | Discriminator | URL shape |
|---|---|---|
| `UserBased` | `userId` present | `…/tenants/<tid>/uservaluemaps/<map>/users/<uid>` |
| `Topic` | `eventId` present (empty string OK; non-empty becomes `lastEventId`) | `…/tenants/<tid>/topics/<topicName>` |
| `JsonMap` | `isJsonBasedMap: true` | `…/tenants/<tid>/json-maps/<map>` |
| `MapBased` | (default) | `…/tenants/<tid>/maps/<map>` |

Note the field repurposing: `mapName` carries the topic name for the topic
strategy. `domainURL` is the rtus-seh origin (`http://rtus-seh.${DEV_DOMAIN}`
under Traefik).

**Common trap**: the discriminator fields are checked in priority order. If
you accidentally set `eventId: ""` (even empty string) you'll get the
`Topic` strategy regardless of `mapName` semantics — your subscription will
hit `/topics/<map>` and either get no events or an immediate disconnect.
When subscribing to a map, omit `eventId` entirely.

### Pick the right strategy per upstream service

| Consuming events from | Use | Why |
|---|---|---|
| **IAN** (`POST /messages` → in-app notification fan-out) | `UserBased` with `userId: claims.sub` | IAN publishes per-recipient into a user-value map; only the owner sees their slot. Setting `Map` instead routes to a tenant-wide stream that drops on its first emit. See `aoh-knowledge/services/ian.md` → "Browser SSE subscription". |
| **Incident-svc `incident.created` topic** (or any post-commit broadcast on a topic) | `Topic` with `eventId: ""` (no replay) | One-to-many broadcast; every authorised subscriber sees the same event. The triage-list pattern in `incident-web` is the worked example. |
| **GIS entity-position maps** (generic key/value, e.g. `entities`, `tracks`) | `MapBased` | Tenant-wide map of strings/JSON without per-user filtering. |
| **JSON-structured maps** where the value MUST be parsed JSON (some GIS layers, custom widgets) | `JsonMap` with `isJsonBasedMap: true` | Same URL family as Map but RTUS parses the value as JSON server-side. Only use when the upstream service explicitly publishes via `/json-maps/...`. |

If you're unsure which one a service uses, the fastest signal is the
service's `RTUS_MAP_NAME` env (or equivalent) plus the publish HTTP call —
search the service repo for the URL it POSTs to: `/tenants/{T}/topics/...`
vs `/tenants/{T}/maps/...` vs `/tenants/{T}/uservaluemaps/...` tells you
which strategy the consumer must mirror.

Required callback wiring for the canonical AOH integration:

- `onConnected` → flip UI status to `live`
- `OnReconnecting` → flip UI status to `reconnecting`
- `onHeartbeatTimeout` (set `heartbeatTimeout: 15_000` since rtus-seh emits
  keepalive every 5s) → flip status to `reconnecting`
- `onDisconnected` (SDK exhausted retries) → flip status to `offline`
- `onAnyEvent(name, data)` → drop `keepalive` events, route everything else
  into the local store

The SDK's `refreshUrl` defaults to `/aoh/api/auth/refresh` — the SvelteKit
endpoint that refreshes the SDS session before each reconnect. Leave it
alone unless the app has a non-standard refresh route.

### CORS for SSE

RTUS-SEH has its own CORS middleware (not the global `permissive-headers`) because SSE
with credentials requires explicit origin whitelisting:

- Each web frontend that uses RTUS must be added to the SEH CORS origin list.
- The `aoh-compose` RTUS template seeds the platform's known dev origins
  (`gis-web`, etc.). Add new web-app origins to that list.

### Initial-fetch + SSE dedup (consumer rule)

A subtle correctness rule that bites pages combining an initial `GET /list`
with a live SSE topic of the same domain: **deduplicate by `id` before
prepending an SSE event into the rendered list.** Without this filter, an
event that races the initial fetch produces two rows with the same key, and
Svelte 5's keyed `{#each}` throws `each_key_duplicate` — the table stops
rendering and looks "stuck on a stale row" with no obvious cause in the
console.

The fix is one line at the prepend site:

```ts
rows = [incoming, ...rows.filter((r) => r.id !== incoming.id)].slice(0, size);
```

Apply it in every page that consumes a live topic alongside a paginated list.

### Database

PostgreSQL with `wal_level=logical` for CDC (Change Data Capture) compatibility.

## API Endpoints (RTUS-PMS)

All data endpoints are tenant-scoped: `/tenants/{tenantId}/...`. Map names must not
contain `@`. Values can be `application/json` or `text/plain` (JSON Maps require JSON).

### Topics

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `POST` | `/tenants/{tenantId}/topics/{name}` | Publish data to topic (overwrites oldest if full) |

### Maps

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `POST` | `/tenants/{tenantId}/maps/{mapName}/keys/{key}` | Add/update value (upsert) |
| `DELETE` | `/tenants/{tenantId}/maps/{mapName}/keys/{key}` | Delete value by key |
| `GET` | `/tenants/{tenantId}/maps/{mapName}/keys` | Get all keys |
| `POST` | `/tenants/{tenantId}/maps/{mapName}/batch` | Batch add/update (JSON object as key-value pairs) |
| `DELETE` | `/tenants/{tenantId}/maps/{mapName}/batch` | Batch delete (body: array of keys) |
| `DELETE` | `/tenants/{tenantId}/maps/{mapName}` | Clear all values |

### JSON Maps

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `POST` | `/tenants/{tenantId}/json-maps/{mapName}/keys/{key}` | Add/update JSON value (upsert) |
| `DELETE` | `/tenants/{tenantId}/json-maps/{mapName}/keys/{key}` | Delete value by key |
| `GET` | `/tenants/{tenantId}/json-maps/{mapName}/keys` | Get all keys |
| `POST` | `/tenants/{tenantId}/json-maps/{mapName}/batch` | Batch add/update |
| `DELETE` | `/tenants/{tenantId}/json-maps/{mapName}/batch` | Batch delete (body: array of keys) |
| `DELETE` | `/tenants/{tenantId}/json-maps/{mapName}` | Clear all values |

### User Value Maps

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `POST` | `/tenants/{tenantId}/uservaluemaps/{mapName}/users/{userId}` | Add/update user's value |
| `DELETE` | `/tenants/{tenantId}/uservaluemaps/{mapName}/user/{userId}` | Delete user's value |
| `DELETE` | `/tenants/{tenantId}/uservaluemaps/{mapName}` | Clear all values |

### SSE Config (Admin — requires Bearer token)

Manage SSE access control per data structure. Config supports `roles` (tenant role
whitelist), `resource`/`scope` (fine-grained RBAC via AAS), and `protected` flag.
JSON Map config also supports `resourceType` and `valuePath` for per-entry access control.

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/admin/tenants/{tenantId}/topics/sse` | List all topic SSE configs |
| `GET` | `/admin/tenants/{tenantId}/topics/{name}/sse` | Get topic SSE config |
| `POST` | `/admin/tenants/{tenantId}/topics/{name}/sse` | Create/update topic SSE config |
| `DELETE` | `/admin/tenants/{tenantId}/topics/{name}/sse` | Delete topic SSE config |
| `GET` | `/admin/tenants/{tenantId}/maps/sse` | List all map SSE configs |
| `GET` | `/admin/tenants/{tenantId}/maps/{mapName}/sse` | Get map SSE config |
| `POST` | `/admin/tenants/{tenantId}/maps/{mapName}/sse` | Create/update map SSE config |
| `DELETE` | `/admin/tenants/{tenantId}/maps/{mapName}/sse` | Delete map SSE config |
| `GET` | `/admin/tenants/{tenantId}/json-maps/sse` | List all JSON map SSE configs |
| `GET` | `/admin/tenants/{tenantId}/json-maps/{mapName}/sse` | Get JSON map SSE config |
| `POST` | `/admin/tenants/{tenantId}/json-maps/{mapName}/sse` | Create/update JSON map SSE config |
| `DELETE` | `/admin/tenants/{tenantId}/json-maps/{mapName}/sse` | Delete JSON map SSE config |

## Which Services Use RTUS?

| Service | RTUS usage | Data structure | Name |
|---------|------------|---------------|------|
| **GIS** | JSON Map — real-time entity positions | JSON Map | `gis` |
| **IAN** | User Value Map — per-user notification delivery | User Value Map | `ian` |
| **Your service** | Define your own topics/maps/json-maps | Your choice | Your choice |

## Access

| URL | What |
|-----|------|
| `http://rtus-pms.${DEV_DOMAIN}` | PMS API (topic/map management) |
| `http://rtus-pms.${DEV_DOMAIN}/swagger-ui/index.html` | Swagger UI |
| `http://rtus-seh.${DEV_DOMAIN}` | SEH (SSE endpoint for browsers) |

## Dependencies

- **iams** (Keycloak + AAS for auth)
- **sds** (session data for SEH auth — included via IAMS)
