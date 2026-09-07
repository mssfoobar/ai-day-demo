# IAN — In-App Notification

## What It Does

IAN provides in-app notifications — messages, alerts, and badges that appear inside the
web application. IAN delivers messages directly to users' browsers in real-time via RTUS,
and owns the per-user inbox + per-recipient `READ`/`UNREAD` state.

## When to call IAN directly vs. go through UNH

Most application code should **not** call IAN's `POST /messages` directly.
The canonical AOH notification flow is:

```
app  ──►  UNH  ──►  IAN  ──►  RTUS user-value map "ian"  ──►  browser SSE
```

UNH owns role-resolution (distribution lists with `internal_role_id`), template
rendering, and fan-out. IAN owns the delivery + inbox. If your trigger is a user
or system event and your audience is "every user in role X" — use UNH with a
distribution list, not a hand-rolled AAS member-lookup + IAN POST in your
service. See [`unh.md`](./unh.md) → "When to use UNH vs. call IAN /
SMTP / FCM directly" for the decision table.

Call IAN directly only when:

- You are writing the UNH custom-channel shim that bridges UNH → IAN (you *are*
  the delivery driver).
- You are sending to a **specific known user-id set** that you already hold and
  there is no role/template story — e.g. a per-document collaborator list
  computed inside the service. Even here, UNH with a dynamically-populated
  distribution list is usually cleaner once a second send-site exists.
- You are the IAN service or its tests.

If you find yourself writing an AAS "list users with role X" client just so you
can call IAN, stop — you are building UNH's distribution-list feature inside
your service.

## What IAN does for browsers (the consumer side)

Whether the publisher is UNH-via-custom-channel or IAN-direct, the *consumer*
side is the same: the web app subscribes to RTUS user-value map `"ian"` via
RTUS-SEH and renders messages in its own notification panel. The "Browser SSE
subscription (canonical pattern)" section below is the contract for that side
regardless of who publishes.

### What IAN already provides (so you don't reinvent it)

IAN is a complete in-app notification backend. Before designing a custom solution
for any of the following, check whether IAN already covers it — it almost
certainly does:

- **Per-user inbox, server-side and durable.** Every message is stored in `ian-db`
  with the list of `receiver_ids`, so each user has their own persistent inbox.
  Browser refresh, device change, multi-tab — the inbox is the same because it
  lives in Postgres, not in the client.
- **Per-user read/unread tracking, server-side.** Each (message, recipient)
  combination carries its own `READ` / `UNREAD` status in IAN-DB. Marking a
  message read in one tab/device updates the server, so other tabs and other
  devices see the new state. Do NOT build a session-scoped or local-storage read
  store — IAN already has this and yours will drift.
- **Unread badge / count.** `GET /users/{id}/messages/unread-count` returns the
  user's current unread count straight from the server. Cheap, correct,
  consistent across tabs.
- **List + pagination.** `GET /users/{id}/messages` is paginated (page, size)
  with optional `unread_only` filter. `GET /users/{id}/quick-access` provides
  cursor pagination for fast "load more" scrolls in a panel.
- **Mark-read endpoints.** `PUT /users/{id}/messages/{message_id}/update-status`
  for a single message and `PATCH /users/{id}/messages/mark-all-as-read` for
  "Mark all as read."
- **Real-time delivery to connected browsers.** IAN publishes every message to
  the `ian` RTUS map; any browser subscribed via RTUS-SEH receives the new
  message instantly without polling.
- **Targeted delivery.** A single `POST /messages` accepts an array of
  `receiver_ids`, fans the message out, and tracks per-recipient state. You do
  not iterate users from the caller.
- **Cross-route attention.** Because the inbox is server-side and the unread
  count is a single API call, a notification bell anywhere in the web app shows
  the right count regardless of which route the user is on.

If your proposal contains phrases like *"session-scoped read state"*, *"no
backend persistence for now"*, *"local unread store"*, or *"v1 simplification of
notification panel"* — stop. That state is what IAN already provides; you're
about to build a half-working local version of a service that exists.

## Architecture

```
┌──────────────┐   REST    ┌──────────┐   RTUS    ┌──────────┐
│ Your Backend │─────────►│ IAN-App  │──────────►│ RTUS-PMS │
│  or UNH-App  │          │ (API)    │  publish   │ (map:ian)│
└──────────────┘          └────┬─────┘           └──────────┘
                               │                       │
                          ┌────▼─────┐           ┌─────▼─────┐
                          │  IAN-DB  │           │ RTUS-SEH  │
                          │(Postgres)│           │  (SSE)    │
                          └──────────┘           └─────┬─────┘
                               ▲                       │ SSE
                         REST  │                       │
                               │      ┌────────────────┘
                          ┌────┴──────▼────┐
                          │  Your Web App  │
                          │  (browser)     │
                          └────────────────┘
```

The web app talks to IAN-App over REST for message CRUD (list, mark-read,
unread-count) and subscribes to the `ian` RTUS map via RTUS-SEH for
real-time delivery. There is no platform-shipped notification UI component
to drop in — the consuming app builds its own notification panel against
the IAN-App REST API and the RTUS SSE stream.

## Components

| Component | Image | Purpose |
|-----------|-------|---------|
| **ian-db** | `postgres:17.0` | Message storage and read status tracking |
| **ian-app** | `ghcr.io/mssfoobar/ian/ian-app` | Notification API and RTUS publishing |

## Key Concepts

### Message lifecycle

1. Backend creates a message via IAN API
2. IAN stores it in the database with per-user status tracking
3. IAN publishes to RTUS map `"ian"` for real-time delivery
4. Connected browsers receive the message instantly via SSE
5. Users can mark messages as read or deleted

### Message statuses

| Status | Meaning |
|--------|---------|
| `UNREAD` | New message, not yet seen |
| `READ` | User has seen/acknowledged the message |

### Message structure

Each message has:
- **title** — Short headline
- **body** — Detailed content
- **ref_link** — Deep link to related content (e.g., `https://example.com/report/123`)
- **icon_id** — Icon identifier for UI rendering
- **sender_id** — Who sent the message (any string, e.g., `"system"` or `"incident-svc"`)
- **receiver_ids** — Target recipient user IDs (array of Keycloak `sub` UUIDs, on send)
- **tenant_id** — Tenant context as a **UUID** (the AAS-resolved tenant UUID,
  not the tenant `name` like `"development"`). Posting `tenant_id: "development"`
  returns `400 "invalid UUID length: 11"`.

  **Where to source the UUID, in order of preference:**

  1. **From the request** — for request-driven sends, take it from the caller's
     `active_tenant.tenant_id` claim, or from the persisted row's tenant column
     (same value, written by the AOH audit columns at insert). This is the
     correct path for "user did X, notify role Y" flows.
  2. **From the UNH `data` map** — when calling IAN through UNH, pass
     `tenant_id` in the send's `data` payload and reference it as
     `{{tenant_id}}` in the template. There is no `{{api.distribution.tenant_id}}`
     special variable — the substitution must come from the data map you control.
  3. **Hardcoded UUID** — only for scheduled/admin-context sends where there is
     no request and no per-tenant data. *Not* the default; do not encode
     `IAN_TENANT_ID_<TENANT>` env maps as a substitute for steps 1–2.

### Real-time delivery

IAN uses RTUS map named `"ian"` for instant delivery:
- When a message is created, IAN publishes it to RTUS as a **user-value map**
  entry keyed by recipient user ID (not a flat map). Each user only sees
  their own slot.
- Your web app subscribes to the `ian` RTUS map via RTUS-SEH (SSE) and
  renders the messages in its own notification panel.

### Browser SSE subscription (canonical pattern)

The subscription is a **`UserBasedSSEClientConfig`** — not Map, not JsonMap,
not Topic. Picking the wrong variant routes to the wrong RTUS-SEH URL and
either gets dropped immediately (broken pipe) or returns no events. The URL
the SDK builds is `${RTUS_SEH}/tenants/{tenantId}/uservaluemaps/ian/users/{userId}`.

Reference implementation: `/Users/nyan/Documents/GitHub/ian/web/src/routes/(private)/+layout.svelte`.

```ts
import { SSESubscribeClient } from "@mssfoobar/sse-client";
import type { Message } from "$lib/aoh/ian/types/types";

type IanSSEEvent = { unread_count: number; message: Message };

const sseClient = new SSESubscribeClient<IanSSEEvent>({
    domainURL: PUBLIC_RTUS_SEH_URL,     // http://rtus-seh.${DEV_DOMAIN}
    tenantId: claims.active_tenant.tenant_id,
    mapName: "ian",                     // map name, not topic — see note below
    userId: claims.sub,                 // ← presence of `userId` selects UserBased
    init: false,                        // true would replay stored values on connect
    maxReconnectAttempts: 10,
    baseReconnectDelay: 1000,
    eventHandlers: {
        Added:   (data) => onArrival(data.message, data.unread_count),
        Updated: (data) => onArrival(data.message, data.unread_count),
    },
});
sseClient.connect();
```

Notes:
- Event payload is `{ unread_count, message }` — take `unread_count` as
  the server-of-truth for the badge so it doesn't drift from optimistic
  client updates.
- Use the typed `eventHandlers.Added` / `eventHandlers.Updated` callbacks,
  NOT `onAnyEvent`. The user-value map emits typed Hazelcast events.
- The `userId` field, not `eventId` or `isJsonBasedMap`, is what
  discriminates the SDK config into `UserBasedSSEClientConfig`. Setting
  `eventId: ""` (even empty string) would route to `/topics/ian` — wrong.

### RTUS integration

When including IAN, you must configure RTUS-SEH to allow the consuming web
app's cookies and origins so it can subscribe to the `ian` map:
- Add the web app's access-token cookie name to RTUS-SEH cookie names
- Add the web app's external URL to RTUS-SEH CORS origin list

### UNH integration

IAN can receive notifications from UNH — when UNH sends a notification, it can also
trigger an in-app message via IAN. This gives you both external delivery (email/push)
and in-app delivery from a single notification API call.

## API Endpoints (IAN-App)

All user-scoped endpoints require the user ID in the path. List endpoints support
`unread_only` filter and `sort` query params.

### Messages

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `POST` | `/messages` | Send notification to one or more users (receiver_ids) |

### User Messages (page-based pagination)

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/users/{id}/messages` | List messages for user (paginated: page, size) |
| `GET` | `/users/{id}/messages/unread-count` | Get unread count. Response: `{ "data": { "total": N }, "sent_at": "..." }` — the field is `total`, NOT `unread_count`. |
| `PUT` | `/users/{id}/messages/{message_id}/update-status` | Update message status (READ/UNREAD) |
| `PATCH` | `/users/{id}/messages/mark-all-as-read` | Mark all messages as read |

### Quick Access (cursor-based pagination)

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/users/{id}/quick-access` | List messages with cursor pagination (cursor, limit) |

### Health

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/livez` | Liveness check |
| `GET` | `/readyz` | Readiness check |

## Data Model

```
message (as returned by API):
  message_id: UUID
  title: string
  body: string
  ref_link: string (deep link URL)
  icon_id: string
  sender_id: string
  tenant_id: string
  created_at: timestamp (RFC3339)
  message_status: string (READ | UNREAD) — per-user status

send message request:
  title: string
  body: string
  ref_link: string
  icon_id: string
  sender_id: string
  tenant_id: string
  receiver_ids: array of user ID strings
```

## Access

| URL | What |
|-----|------|
| `http://ian.${DEV_DOMAIN}` | IAN App API |
| `http://ian.${DEV_DOMAIN}/swagger-ui/index.html` | Swagger UI |

## Dependencies

- **iams** (Keycloak for auth)
- **unh** (notification source — IAN receives from UNH)
- **rtus** (real-time delivery via map `"ian"`)
