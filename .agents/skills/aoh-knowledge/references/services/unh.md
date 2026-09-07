# UNH — Unified Notification Hub

> **This describes the MIGRATED service** that lives in the ops-hub monorepo at
> `modules/unh/` (Go backend `modules/unh/service`, image
> `ghcr.io/mssfoobar/ops-hub/unh-service`, published SDK
> `@mssfoobar/unh-web-sdk`). The standalone `mssfoobar/unh` + `mssfoobar/unh-web`
> repos (images `ghcr.io/mssfoobar/unh/unh-app` + `unh-web`) are **decommissioned**.
> The migrated service is a clean-room rewrite with a **different contract** from
> the legacy one — if you find an older doc referencing `unh-app`, `unh-web`,
> `APP_PORT`, `{{api.distribution.user_ids}}`, `internal_role_id`, or
> `sms_notification`, it is stale (see "What changed from the legacy service").

## What It Does

UNH is the AOH platform's notification engine. It manages and delivers
notifications across multiple channels — **email** (SMTP), **push** (FCM), and
**custom** channels (a webhook escape hatch) — driven by reusable **templates**,
**distribution lists**, and `{{key}}` variable substitution.

Use UNH when your application needs to send notifications to users outside the
app (email/push) or to fan a templated payload out to any HTTP endpoint.

## When to use UNH

**Default to UNH** for any notification triggered by application logic. The
reason is **distribution lists with IAMS role/group resolution** — UNH resolves
"all users holding role X in tenant T" against AAS at send time and binds the
resolved contact endpoints (emails / phones / FCM tokens) into the message.
Re-implementing role-membership lookup + caching in your own service duplicates
what UNH owns.

| Situation | Use |
|---|---|
| Notify "everyone in role X" by email/push | **UNH** with a `role` distribution-list member |
| Templated send to a saved audience (cron / worker) | **UNH** — keeps the audience declarative |
| One-off send to a specific email/phone you already hold | **UNH** (external `email`/`phone` member) |

## Architecture

```
┌──────────────┐   /v1 REST  ┌───────────────┐
│ Your Backend │────────────►│  unh-service  │──► Email (SMTP, STARTTLS or implicit TLS)
│   or BFF     │             │   (Go API)    │──► Push  (FCM, per-channel service-account key)
└──────────────┘             └──────┬────────┘──► Custom channel → any HTTP endpoint you host
        ▲                           │      └────► IAMS/AAS (resolve role/group/user → contacts, design D5)
        │ @mssfoobar/unh-web-sdk    │
┌───────┴────────┐            ┌─────▼─────┐
│  SvelteKit app │            │ Postgres  │
│ (reference-host│            │  (schema) │
│  or your app)  │            └───────────┘
└────────────────┘
```

There is **no separate web UI service** anymore. The admin/authoring UI ships as
the Svelte SDK `@mssfoobar/unh-web-sdk`, integrated into a host SvelteKit app
(the canonical host is `apps/reference-host`). To wire the SDK into an app, use
the **`aoh-unh-integration`** skill.

## Components

| Component | Image / package | Purpose |
|-----------|-----------------|---------|
| **unh-service** | `ghcr.io/mssfoobar/ops-hub/unh-service` (`modules/unh/service`) | Notification `/v1` API + delivery engine (chi + sqlx + Postgres) |
| **unh-db** | `postgres` (operator-provided / bundled) | Channels, distribution lists, templates |
| **@mssfoobar/unh-types** | npm (`modules/unh/types`) | Wire DTOs mirroring the `/v1` contract |
| **@mssfoobar/unh-client** | npm (`modules/unh/client`) | Transport-only typed REST client (used by the host BFF + SDK) |
| **@mssfoobar/unh-web-sdk** | npm (`modules/unh/web`) | Svelte 5 admin/authoring UI (forms, editors, the IAMS member picker, the built-in Edra email editor) |

## Key Concepts

### Channels

| Channel | Built-in | Delivery | Notes |
|---------|----------|----------|-------|
| **Email** | yes | SMTP | `/v1/admin/email_channel`. Per channel: host/port/`send_from`, `encryption` (`none`\|`starttls`\|`tls`, default `tls`; STARTTLS is required, never opportunistic), `auth_method` (`none`\|`plain`, default `plain`; `plain` requires encryption), username/password when authenticating, optional `ca_cert` (PEM trust scoped to that channel) and `sender_name`. `encryption: none` + `auth_method: none` is the local-relay topology. Password AES-256-GCM encrypted at rest, never returned; `ca_cert` is public and IS returned. |
| **Push** | yes | FCM | `/v1/admin/push_channel`. Each channel carries its **own** FCM `service_account_key` (AES-encrypted at rest) — there is no global FCM credential. |
| **Custom** | yes (mechanism) | webhook → an endpoint you host | `/v1/admin/custom_channel` + per-field parameters. UNH POSTs a mapped JSON body to the endpoint at send time. The escape hatch for SMS, in-app, paging, etc. |

There is **no SMS channel and no `sms_notification`** — implement SMS as a custom
channel.

### Templates

A template bundles per-channel payloads + a distribution-list link. A single
template can populate several channel blocks; UNH attempts each populated channel
on send.

- **email_notification** — `channel_id`, `subject`, `body` (HTML), optional
  `editor_json` (opaque visual-editor source the SDK round-trips; the **`body`
  HTML** is what sends + binds).
- **push_notification** — `channel_id`, `title`, `body`, optional `image_url`.
- **custom_notification** — channel-grouped on read; per-parameter `param_value`
  on write.
- **distribution_list_id** — who to send to.

### Distribution lists & members

A list owns rows in one collapsed table — `distribution_list_member(member_type,
value, delivery_mode)`:

- **member_type** — `user` / `role` / `group` (IAMS references), or `email` /
  `phone` (external literals).
- **value** — the user id / role name / group id, or the literal email/phone.
- **delivery_mode** — `to` / `cc` / `bcc`. Applies to **email + IAMS members**
  (user/role/group resolve to email at send); **`phone` is always `to`** (the
  non-`to`-for-phone rule is enforced service-side). (AOH-7588.)

IAMS members are resolved against **AAS at send time** and **collapse down to
contact endpoints** (email / phone / FCM token). The userid / role-name /
group-name themselves are audience *selectors*, **not** bindable tokens.

Two corrections to what this section used to claim (AOH-8280):

- **Resolution uses UNH's own client-credentials identity, not the caller's
  forwarded bearer.** The recipient set is a property of the template, not of
  whoever pressed send. (The old text said "forwarding the caller's bearer —
  design D9"; D5 supersedes D9.) The caller needs no IAMS privilege at all;
  before this, sending to a role member effectively required Keycloak realm
  administration.
- **The collapse to contact endpoints is CONDITIONAL, not unconditional.** Email
  comes from the user's Keycloak `email`, but **phone and FCM token come from
  optional user-profile attributes** (`phone`, `fcmToken` — camelCase, both
  arrays on the wire). An attribute that the realm does not declare is not even
  returned by the admin API, and a user may simply not have set one. When they are
  absent the channel reports `skipped`, the send stays successful, and nothing
  is reported as unresolved — so a template can look healthy and deliver to
  nobody. Declare the attributes in the realm user profile if you rely on those
  channels.

### Authorization (AOH-8280)

Every `/v1` resource endpoint is gated on **UNH-local tenant roles**, not merely
on a valid bearer. Two capabilities, both **deployment-configured** — no role
name is compiled into the service:

| Config key | Env | Grants |
|---|---|---|
| `authz.admin_roles` | `AUTHZ_ADMIN_ROLES` | the configuration surface |
| `authz.sender_roles` | `AUTHZ_SENDER_ROLES` | the send surface |

Recommended names are `unh_admin` / `unh_sender`. Both **FAIL CLOSED**: an empty
or absent list denies that capability, including sending. Administration
**confers** send, so an administrator needs no second listing.

These are **tenant** roles, created at runtime through AAS — not realm roles from
a realm import, so they are absent from `realm-import.json` exactly as
`Tenant-development` and `form_admin` are. Create them on the tenant (IAMS console
or `POST /admin/tenants/{id}/roles`) and assign them, or a correctly deployed UNH
answers 403 to everything and reads as broken.

Resolution is separate and needs no caller privilege: the `IAMS_KEYCLOAK_CLIENT_ID`
client must be confidential with service accounts enabled, and its service account
needs the realm role `realm-tenant-admin` plus `realm-management`'s `view-users`
and `view-clients` — all three. Without the realm role, `user` members resolve
while every tenant-scoped read (role/group members) 403s. Don't point it at `sds`,
which also carries `manage-users`.

**By path prefix**, and the prefix is the unit — `/v1/admin/` means what it says:

| Prefix | Capability |
|---|---|
| `/v1/admin/*` (channels, custom params, distribution lists + members, templates) | administration |
| `/v1/notification/*` (the templated send) | send — **administration confers it** |

Every method on a resource shares its prefix's requirement; there is no per-method
distinction. So a sender-only caller cannot read templates or distribution lists,
and therefore cannot discover a template id through the API — that suits a
service-triggered send holding the id in its own config, not a human sender-only
console. An administrator who cannot send is not expressible.

A 403 names the missing capability and the config key granting it (never the
configured role names), and every gated operation documents 403 in the OpenAPI.

Health probes and the Swagger UI are unauthenticated and ungated.

**There are no lookup endpoints.** Read APIs return STORED values — a member row is
`{member_type, value}` where the value is a user id, a role name or a group id, and a
template references a channel by id. Resolving those for display is the console's job.

A console does that by calling IAMS-AAS with the signed-in operator's token. **Know the
consequence:** AAS forwards that token to Keycloak, so the directory reads need
`realm-management:view-users`, which in the AOH realm arrives only via `realm-admin`
(composited by `tenant-admin`). Measured: a principal holding only the tenant role
`unh_admin` gets 403 on `/admin/users`, `/admin/users/{id}`,
`/admin/tenants/{t}/roles` and `/admin/tenants/{t}/groups`. So the member picker and
member-name display work for realm administrators and silently yield nothing for
anyone else — `loadIamsDirectory`-style helpers fail soft to an empty list.

Channel names come from the per-channel reads, which are admin-only. A host authorized
only to **send** therefore cannot label a template's channel at all.

Recipient-resolution failure is split by cause: IAMS rejecting UNH's own
credentials → **500** (deployer action, not retryable); IAMS unreachable/5xx/
timeout → **503** (retryable); one member IAMS cannot find → **200** with the
member in `unresolved[]`. A list holding only external `email` members never
calls IAMS, so it still delivers during an outage.

### Template binding (`{{key}}`)

unh-service renders `{{ key }}` placeholders (dotted identifiers) in the email
subject/body, push title/body, and custom-channel parameter values. **Unknown
placeholder → empty string** (never aborts). Two binding sources:

1. **`{{distribution.*}}` — auto, exactly three tokens**, populated from the
   *resolved recipients* of the list as a **comma-joined union over the whole
   list (rendered once, NOT per-recipient)**:
   - `{{distribution.email}}` — every resolved recipient email
   - `{{distribution.phone}}` — every resolved phone
   - `{{distribution.fcm_token}}` — every resolved FCM token

   A multi-value custom param whose value is exactly one `{{distribution.*}}`
   token binds to the JSON **array** instead of the joined string.

2. **Free-form `{{key}}` — caller data**, filled from the send request body
   `{ "data": { … } }`. No declared schema; a string binds directly, an array
   binds comma-joined, an omitted key renders empty.

> **There is no per-recipient personalization** (no `{{recipient.first_name}}`)
> and **no `{{api.distribution.user_ids}}`** — the legacy `api.distribution.*`
> namespace and per-recipient user-id binding do **not** exist in the migrated
> service. Recipients are a union, and IAMS identities collapse to contact
> endpoints. (Full contract: `modules/unh/service/README.md` → "Template
> binding".)

### Auth posture (D5, superseding D9)

unh-service validates a Keycloak bearer with an `active_tenant` claim on every
`/v1` route, then gates it on the UNH-local tenant roles above.

Recipient resolution uses **UNH's own client-credentials identity**
(`IAMS_KEYCLOAK_CLIENT_ID` / `_SECRET`), never the caller's bearer, so the
recipient set is a property of the template rather than of whoever pressed send
and the caller needs no IAMS privilege. Resolution failing systemically now
fails the send (500 for rejected credentials, 503 for an outage) instead of
reporting success with a truncated recipient list.

## Frontend / SDK

The UI is the Svelte SDK `@mssfoobar/unh-web-sdk`, consumed by a host SvelteKit
app through a transparent server-side BFF proxy. **The SDK ships forms / editors
/ managers + the IAMS member picker + a built-in Edra (TipTap v3) email-body
editor (with a host-overridable `bodyEditor` seam) — but NOT list components**;
the host composes lists itself from `@mssfoobar/unh-client`'s `list()` + a
`@mssfoobar/ui` `DataTable`. **To integrate it, use the `aoh-unh-integration`
skill** — it carries the provider mount, the BFF proxy + IAMS picker route, the
stub, the Tailwind `@source` import gotcha, and the page recipes. Canonical
reference: `apps/reference-host` (`src/routes/aoh/unh/`).

## API Endpoints (unh-service)

All entity routes are under **`/v1`** and require a tenant-scoped bearer; health
(`/livez`, `/readyz`) + Swagger (`/swagger-ui/`) are unauthenticated.

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `POST` | `/v1/notification/send/template/{template_id}` | Send a templated notification (body `{ "data": { … } }`) |
| CRUD | `/v1/admin/email_channel[/{id}]` | Email channels |
| CRUD | `/v1/admin/push_channel[/{id}]` | Push channels |
| CRUD | `/v1/admin/custom_channel[/{id}]` (+ nested `/{channel_id}/parameter[/{parameter_id}]`) | Custom channels + parameters |
| CRUD | `/v1/admin/distribution_list[/{id}]` (+ nested `/{list_id}/member[/{member_id}]`) | Distribution lists + members |
| CRUD | `/v1/admin/notification_template[/{id}]` | Templates |

List endpoints are paginated (`page` 1-indexed, `size`) and return the `aohhttp`
envelopes.

## Data Model (summary)

```
email_channel:    id, name, username, password(enc), send_from, host, port,
                  encryption, auth_method, ca_cert, sender_name
                                              + audit/occ_lock/tenant_id
push_channel:     id, name, service_account_key(enc)  + audit/…
custom_channel:   id, name, endpoint  + audit/…
custom_channel_param: id, channel_id, name, description, regexp_validation,
                  is_multi_value  + audit/…
distribution_list: id, name  + audit/…
distribution_list_member: id, distribution_list_id,
                  member_type CHECK(user|role|group|email|phone),
                  value, delivery_mode CHECK(to|cc|bcc)
email_notification:  id(=template id), channel_id, subject, body, editor_json
push_notification:   id(=template id), channel_id, title, body, image_url
custom_notification: id, notification_template_id, custom_channel_param_id, param_value
notification_template: id, name  + audit/…  (+ the per-channel payloads + the
                  notification_distribution link, assembled via v_notification_template)

send request:  { data: { key: value | [values] } }
send response: { channels: [{channel, status, error?, failed_tokens?}],
                 unresolved: [{type, value, reason}] }
```

## What changed from the legacy standalone service

| | legacy `mssfoobar/unh` | migrated `modules/unh/service` |
|---|---|---|
| image | `ghcr.io/mssfoobar/unh/unh-app` | `ghcr.io/mssfoobar/ops-hub/unh-service` |
| UI | `unh-web` service | `@mssfoobar/unh-web-sdk` (host-embedded) |
| API mount | mixed | single `/v1` namespace; health/swagger at root |
| port env | `APP_PORT` | `HTTP_PORT` |
| AES key env | `AES_256_KEY` | `ENCRYPT_AES_256_KEY` |
| recipient binding | `{{api.distribution.user_ids/email_addr/phone_num/fcm_tokens}}` (per-id) | `{{distribution.email/phone/fcm_token}}` (union, 3 tokens) |
| dist members | 5 arrays (`internal_user_id`, …, `external_phone`) + `sms_notification` | one `distribution_list_member(member_type,value,delivery_mode)`; no SMS |
| auth | `unh-app` confidential client (client creds) | `unh` confidential client (client creds) + UNH-local role gate (D5) |
| push cred | global FCM credential | per-channel `service_account_key` |

## Access

| URL | What |
|-----|------|
| `http://unh.${DEV_DOMAIN}` | unh-service `/v1` API |
| `http://unh.${DEV_DOMAIN}/swagger-ui/index.html` | Swagger UI |

(UNH pages are served by the host app, e.g. `apps/reference-host` at `/aoh/unh`.)

## Dependencies

- **iams** (Keycloak for JWT validation and for UNH's own client-credentials
  token; AAS for recipient resolution under that identity, D5).

## See also

- `modules/unh/service/README.md` — service overview + the full template-binding contract.
- **`aoh-unh-integration` skill** — wiring `@mssfoobar/unh-web-sdk` into a host app.
- `unh-openapi.json` (this dir) — the `/v1` OpenAPI. Regenerate from the service's
  `docs/swagger.json` when stale, then **append a trailing newline**: swag emits none,
  and this subtree's `end-of-file-fixer` hook fails without one.
