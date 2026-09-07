# API Contract

Cross-runtime HTTP wire conventions that apply to every AOH service.
Backend services (currently Go) produce these shapes; frontend code
(SvelteKit web-base) consumes them through the gateway proxy. Both
sides assume this contract — diverging breaks the other side silently.

## Path prefix and versioning

All HTTP endpoints (other than health probes and Swagger UI) are
mounted under **`/v{N}/{resource}`** — no `/api` prefix. `{N}` is
the major version (start at `1`). The frontend gateway proxy strips
`/aoh/gateway/{module}/` and forwards to `${host}/v{N}/...` —
configure this via `basePath: "/v1"` in `gateway.config.ts`.

Examples:

- `POST /v1/incidents`
- `GET  /v1/incidents/{id}`
- `GET  /v1/users?page=3&size=10`

Health probes (`/livez`, `/readyz`) and Swagger UI (`/swagger-ui/`)
are mounted at the root — Kubernetes probes expect them there.

## Liveness and readiness probes

Every service MUST expose:

- `GET /livez` — liveness (process is running)
- `GET /readyz` — readiness (process can serve traffic; e.g. DB
  connection healthy, downstream auth reachable)

Both endpoints are unauthenticated by design — Kubernetes probes them
without credentials. If the framework provides its own conventions
(e.g. Spring Boot's `/actuator/health/liveness` /
`/actuator/health/readiness`), use the framework path; otherwise
default to `/livez` and `/readyz`.

## Web page routing

Frontend page routes are namespaced first by project, then by module:

```
/[project]/[module]/...
```

For AOH apps, the project is `aoh`. Examples:

- View all incidents: `/aoh/incidents/`
- View a specific incident: `/aoh/incidents/inc-20240607-0001`
- Dashboard with query params: `/aoh/dashboard?name=My+First+Dashboard`

In SvelteKit, this maps to `src/routes/(private)/aoh/{module}/`. See
`web.md` for the full SvelteKit routing layout.

## Success response body shape

| Key | Type | Optional | Description |
|-----|------|----------|-------------|
| `data` | `object` or `array` | yes | Response payload — object for single records, array for lists |
| `message` | `string` | yes | Human-readable accompanying message |
| `sent_at` | ISO 8601 string | yes | When the response was sent |

```jsonc
// Example success body
{
  "data": { "id": "01927a2f-...", "name": "Widget" },
  "message": "",
  "sent_at": "2024-08-12T18:32:42Z"
}
```

In Go services, this shape is produced by `aohhttp.Response()` /
`aohhttp.PaginationResponse()` from `aoh-golib/http`. See `go.md` →
"Response envelopes" for the swag annotation pattern that documents this
shape in the OpenAPI spec.

## Error response body shape

Errors are **not** the success envelope. Two error shapes are live, and
which one applies depends only on whether the endpoint has migrated.

**New and migrated error paths use the AOH error contract** —
`{timestamp, trace_id, errorCode, errorMessage, details}`. That contract
is owned end to end by the **`aoh-error-handling`** skill: field formats,
the error-class → HTTP-status map, the `errorCode` vocabulary and its
required format, `trace_id` stamping, log levels, and the BFF and
frontend halves. Consult that skill before designing any error response;
this file deliberately does not restate the shape, because two documents
each specifying one wire contract is how they came to disagree.

**The legacy envelope is deprecated but supported.** It is the success
envelope with an `errors` array, produced by `aohhttp.ErrResponse()` /
`aohhttp.ErrResponseWithData()`:

| Key | Type | Optional | Description |
|-----|------|----------|-------------|
| `data` | `object` or `array` | yes | Populated only by `ErrResponseWithData()` |
| `message` | `string` | yes | Short failure label, e.g. `"validation"` |
| `sent_at` | ISO 8601 string | yes | When the response was sent |
| `errors` | array of `{ message: string }` | yes | One entry per reported error |
| `trace_id` | 32-hex string | yes | Stamped from the active span; **omitted** when none is active |

```jsonc
// Legacy error body (HTTP 400) — deprecated; existing endpoints only
{
  "message": "validation",
  "sent_at": "2024-08-12T18:32:42Z",
  "errors": [
    { "message": "name: name must not be empty" }
  ],
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736"
}
```

Every key is `omitempty`, so a field the call site left empty is absent
rather than `null` — which is why `data` does not appear above.

Its existing fields are frozen: five published SDKs parse them, so
renaming, removing or retyping one is a breaking change. The single change
it has taken is additive — the optional `trace_id` above, under the same
stamping rules as the contract body — which buys correlation for the
unmigrated call sites without waiting for their migration.

**Do not emit the legacy envelope from new code.** Migration is
per-service, so until a service has migrated a client may see either shape
from it, and must tolerate both.

## Pagination contract

Paginated endpoints MUST follow this contract:

### Request query params

| Name | Type | Default | Notes |
|------|------|---------|-------|
| `page` | int | `1` | 1-indexed. Invalid values fall back to default. |
| `size` | int | `10` | Page size. Invalid values fall back to default. |
| `sort` | `field,direction` | `created_at,desc` | Repeatable. `direction` is `asc` or `desc` (case-insensitive on input; default `desc`). Responses always echo it lowercase. |

### Response shape

```jsonc
{
  "data": [ /* page rows */ ],
  "page": {
    "number":        4,                          // current page (1-indexed)
    "size":          3,                          // page size
    "total_records": 10,                         // total rows after filters
    "count":         1,                          // optional: rows in this page
    "sort":          ["username,desc", "email,asc"] // optional: sorts applied, in order
  },
  "message": "...",
  "sent_at": "..."
}
```

`number`, `size`, and `total_records` are required so the caller can
compute total pages and render UI controls. `count` and `sort` are
optional.

**`sort` is an array of strings** — one entry per sort clause, in the order
they were applied. Each entry is `"<field>,<direction>"` with direction
lowercased. Echoing the array (rather than a joined string) lets the caller
parse it back into structured form symmetrically with how it was sent on the
request side (the request also accepts repeated `sort=` query params, one per
clause).

In Go services, this is produced by
`aohhttp.PaginationResponse(code, msg, aohhttp.PageResponse{...}, data)`.

### Examples

```bash
# page 1, default size, default sort
GET /v1/users

# page 3, 2 per page, default sort
GET /v1/users?page=3&size=2

# page 3, 2 per page, sort by username desc (default direction)
GET /v1/users?page=3&size=2&sort=username

# multi-sort: email desc then username asc
GET /v1/users?page=3&size=2&sort=email,desc&sort=username,asc
```

## Optimistic concurrency on mutating requests

Endpoints that bump `occ_lock` (any UPDATE, including soft-delete) require the
caller to pass the **current** `occ_lock` value so the server can reject stale
writes with HTTP 409.

**The caller always sends `occ_lock` in the request body**, regardless of HTTP
method. This includes `DELETE` (soft-delete) — DELETE is allowed to carry a
body, and we use that uniformly so clients don't have to switch between body
fields and headers depending on the verb.

```jsonc
// PUT / PATCH / DELETE request body (alongside any other fields)
{
  "occ_lock": 3,
  // ... other update fields ...
}
```

Server contract:

1. Read the row by `(tenant_id, id)`.
2. If the row's `occ_lock` differs from the body's `occ_lock`, return
   **`409 Conflict`** (`ErrConflict` in Go) and do not write.
3. Otherwise UPDATE with `WHERE occ_lock = <provided>`, bumping `occ_lock` and
   `updated_at` in the same statement. Return the updated resource so the
   caller can refresh its state.

Errors:

- `409 Conflict` — supplied `occ_lock` did not match (concurrent modification).
  Caller should re-read and retry.
- `400 Bad Request` — `occ_lock` field missing or not an integer.

## Auth propagation on downstream service-to-service calls

When an AOH service calls another AOH service as a side effect of a user
request (e.g. `incident-svc` calling UNH after `POST /v1/incidents`), the
bearer token used for the downstream call MUST preserve the inbound
request's tenant context.

**The rule.** Forward the inbound JWT's `active_tenant` context on
downstream service-to-service calls. Do not call a downstream that needs
tenant context with a raw `client_credentials` token.

**Why.** A `client_credentials` token authenticates the *service*, not a
user, and therefore carries no `active_tenant` claim. Endpoints that
require tenant context (e.g. UNH `/notification/send/*`, anything mounted
behind AAS tenant-role gates) reject these tokens with `"invalid jwt"`.
The trap is to grab the service's own client-credentials token and reuse
it for downstream calls — it works against `/livez` and any
service-account-scoped admin endpoint, then silently 401s on every
tenant-scoped endpoint.

**Three valid posture options**, in order of preference:

1. **Forward the inbound bearer.** Take the `Authorization` header from
   the inbound request, extract or carry it through (`aoh-golib`'s
   request-context-aware HTTP client handles this), and pass it on the
   downstream call. The downstream sees the same `active_tenant` and the
   same user identity. Requires the user's role to grant the downstream
   permission (e.g. the Field Reporter must be allowed to trigger that
   downstream send).

2. **AAS token-exchange.** When the user's role should NOT grant the
   downstream permission directly (the user did X, but the *service*
   should be the one calling Y on their behalf), exchange the inbound
   token via AAS for a service token that preserves `active_tenant` from
   the inbound. Wire this once at service startup, not per-request.
   This is the production pattern referenced in `aoh-knowledge/services/unh.md`.

3. **Password grant (dev only).** Authenticate as a fixed admin user via
   password grant against the `iams` Keycloak client. Matches the pattern
   used by `unh-init`, `project-aas-init`, and other bootstrap scripts.
   Acceptable in dev compose; never in prod.

**Background goroutine gotcha.** When dispatching the downstream call to
a goroutine after the response has been written (best-effort
notification, audit log, etc.), extract the bearer **before** the
goroutine starts and pass it explicitly. The original request's
context is canceled the moment the handler returns — reading the
bearer from `ctx` inside the goroutine yields the empty string.

```go
// Wrong — context is canceled by the time the goroutine runs.
go h.notifier.Notify(ctx, incident)

// Right — capture the bearer for the downstream call before spawning.
bearer := getBearerToken(ctx) // scaffold middleware helper (aoh-go-init): reads
                              // the bearer the auth middleware stashed in ctx
go h.notifier.Notify(context.Background(), bearer, incident)
```

## What this contract DOESN'T cover

- **OIDC login flow itself** — see `web.md` for browser-side OIDC PKCE,
  `go.md` for backend bearer validation middleware.
- **Specific endpoint paths and methods** — those are per-service.
  Consult the relevant service's OpenAPI spec via `aoh-knowledge`.
- **Real-time / streaming events** — RTUS and SSE patterns are covered
  by `aoh-knowledge` → `integration-patterns.md`.
