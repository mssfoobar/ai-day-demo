# Go Service Conventions

Deep conventions for Go microservices scaffolded via `aoh-go-init`. The
scaffolded service ships with a thin `AGENTS.md` containing only project
description and the load-bearing architecture rule; this file holds the
detailed conventions an agent needs while writing code.

For conventions that aren't Go-specific:

- **Database schema, naming, mandatory columns, association/view conventions** — see
  `database.md`. Go services own DB schemas, but the rules are
  language-agnostic.
- **HTTP wire contract (response envelope, pagination, health endpoints,
  routing namespaces)** — see `api.md`. The Go scaffold's `aohhttp` helpers
  *implement* this contract; the contract itself is cross-runtime.

## Architecture

Layered Go microservice. Dependencies flow one way: **handler → service → repo**.
Never call repo from handler, never inject service into another service. This
is enforced by code review, not by tooling — but it's the load-bearing rule.

## Adding a new entity

1. Add row/filter types in `internal/repo/types.go`
2. Add repository interface in `internal/repo/interfaces.go`
3. Add repository implementation with SQL queries in `internal/repo/{entity}.go`
4. Add service interface in `internal/service/interfaces.go` (with request/response types) and implementation in `internal/service/{entity}.go`
5. Add handler in `internal/handler/{entity}.go` with chi sub-router and swagger annotations
6. Mount the handler in `internal/handler/router.go`
7. Wire the repository and service in `cmd/server/init.go`
8. Add interfaces to `.mockery.yaml` and run `make mock`
9. Write tests per `go-testing.md` — service units, handler tests, and (for SQL-heavy entities) build-tagged integration tests
10. Generate migration files with `make migrate-create NAME={entity}`, fill in the SQL, and verify with `make run` (migration runs on startup)

## Required libraries

Tooling choices, not derivable from imports alone — use these specifically:

- **HTTP router**: `go-chi/chi/v5` — always use chi, never `net/http` mux
- **HTTP responses**: `aohhttp "github.com/mssfoobar/ops-hub/packages/aoh-golib/http"` — use `aohhttp.Response()` for single-object success, `aohhttp.PaginationResponse()` for paginated lists, `render.Render()` to send. See "Response envelopes" below for the swag annotation rules.
- **HTTP errors**: `"github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"` — `aoherr.Render(w, r, err)`, which also emits the one log record for the failure. `aohhttp.ErrResponse()` is the deprecated legacy envelope; new code does not use it. The contract, the `errorCode` vocabulary and the log semantics are owned by the `aoh-error-handling` skill.
- **Logging**: `aohlog "github.com/mssfoobar/ops-hub/packages/aoh-golib/logger"` — never use standard `log` package. Use structured fields: `aohlog.Info("msg", zap.String("key", val))`. Inside a request, prefer the context-aware emitters (`aohlog.Ctx(ctx)`, `aohlog.ErrorCtx(ctx, ...)`) so the record carries the active span's `trace_id`/`span_id` without splatting fields.
- **Database**: `jmoiron/sqlx` with `lib/pq` driver — use positional params ($1, $2), not named
- **Migrations**: `golang-migrate/migrate/v4` — CLI managed via Makefile, runtime via `repo.RunMigrations(db)`. Migration files live in `schema/` as `NNNN_{title}.up.sql` / `NNNN_{title}.down.sql`. Generate with `make migrate-create NAME={title}`. Service runs migration up on startup.
- **Config**: `spf13/viper` + `spf13/pflag` — all config structs use `mapstructure` tags
- **UUIDs**: `google/uuid` — use `uuid.NewV7()` for new IDs (time-sortable)

## Running & verifying locally

- **Config via `config.yaml`, not an inline env wall.** Copy
  `config.yaml.example` → `config.yaml` (gitignored) and edit it once; viper
  reads `./config.yaml`, so `go run ./cmd/server` (or `make run`) needs zero
  environment. Env vars still override (viper `AutomaticEnv`), but the file is
  the primary path — don't prefix every run with `SQL_*=… IAMS_*=… go run`. Set
  `http.port` to match the web app's gateway upstream (the module `host` in its
  `gateway.config.ts`).
- **Verify through the Makefile targets**, never ad-hoc `go`/tool invocations:
  `make fmt` (gci import formatter — the generators don't pre-format), then
  `make lint-code`, `make test` (regenerates mocks via mockery, then `go test`),
  `make build`, `make migrate-create NAME=<x>`. They pin tool versions in
  `.bin/`; invoking `swag`/`golangci-lint`/`mockery`/`migrate` directly (or with
  `@latest`) drifts from the pinned toolchain and yields artifacts (swagger,
  mocks, migration filenames) that differ from CI/teammates.

## Conventions

- Error wrapping: always use `fmt.Errorf("context: %w", err)`
- Context: pass `context.Context` as first argument through all layers
- Constructor injection: all services/repos accept dependencies via constructor, never globals
- Transactions: multi-write service operations run in a single transaction via `repo.DB.TxExecute(ctx, fn)` — handlers never start transactions themselves
- Typed context keys: use `iota` constants, never string keys
- No `panic` in application code — the linter will reject it
- No `time.Sleep` in application code or tests — use `require.Eventually` / channel-`select` patterns (see `go-testing.md`)
- Request validation: implement `Bind(*http.Request) error` on request structs for `render.Bind()`
- Swagger: add annotations above every handler method; UI is served at `/swagger-ui/index.html`. Response annotations MUST reflect the actual envelope (see "Response envelopes")
- The service layer classifies failures as `*aoherr.Error` (see `internal/service/errors.go` in a scaffolded service); the handler renders them with `aoherr.Render`, which derives the status from the class. Handlers never pick a status per call site
- **Repo never imports `internal/service`.** To signal common failures, the repo returns its own sentinel errors (`repo.ErrNotFound` for `sql.ErrNoRows` lookups, `repo.ErrConflict` for OCC-mismatched UPDATEs that affect 0 rows). The service translates these into classified `*aoherr.Error` values for the handler. This keeps the dependency direction one-way (service → repo) and avoids circular imports.

## Response envelopes

Every JSON **success** response goes through one of two `aoh-golib` helpers.
Match the swag annotation to the helper used — otherwise the OpenAPI spec lies
about the wire shape.

| Helper | Wire shape | swag annotation |
|--------|-----------|-----------------|
| `aohhttp.Response(code, msg, data)` | `{data: <T>, message, sent_at}` | `{object} aohhttp.ResponsePayload{data=<T>}` |
| `aohhttp.PaginationResponse(code, msg, page, data)` | `{data: [<T>], message, sent_at, page: {number, size, total_records, count, sort}}` | `{object} aohhttp.PaginationResponsePayload{data=[]<T>}` |

Failures go through `aoherr.Render(w, r, err)` instead — a different shape with
a different owner (`aoh-error-handling`), not a third envelope helper.
`aohhttp.ErrResponse()` still exists for the unmigrated call sites and is
deprecated; see `api.md` → "Error response body shape".

> Note on error annotations: failure responses are annotated with `@Failure <code>` and typically left without a body type (e.g. `@Failure 404`). The renderer's payload type is unexported by design, so there is no struct to point swag at — document the fields in the description if you need them in the spec.

**For lists, always use `aohhttp.PaginationResponse()`** — it is the dedicated
helper for paginated data and produces the canonical `{data, page, message,
sent_at}` envelope with proper page metadata (`number`, `size`, `total_records`,
`count`, `sort`). Do not hand-roll a paginated payload through `aohhttp.Response()`.

**Anti-pattern — don't do this:**

```go
// BAD: aohhttp.Response is for single-object success only. Paginated data
// belongs in aohhttp.PaginationResponse, which carries the page metadata in
// its own `page` field instead of stuffing it into an ad-hoc map.
// (As a side effect, this also produces a nested `data.data` because
// aohhttp.Response wraps its third arg in `data`.)
_ = render.Render(w, r, aohhttp.Response(http.StatusOK, "", map[string]any{
    "data":  results,
    "page":  page,
    "total": total,
}))
```

The fix is `aohhttp.PaginationResponse(...)` with `aohhttp.PageResponse{...}`,
annotated as `aohhttp.PaginationResponsePayload{data=[]<T>}`. The `Makefile`'s
`make swag` target runs with `--parseDependency`, so swag resolves these
`aoh-golib` types into the generated `swagger.json`.

### Minimal canonical paginated handler

If you're scaffolded via `aoh-go-init`, `internal/handler/example.go` shows the
full pattern. If you're not, this is the smallest version that gets the
helper / swag annotation / wire shape correct together:

```go
// @Param        page query int    false "Page number" default(1)
// @Param        size query int    false "Page size"   default(10)
// @Param        sort query string false "Sort order"  example:"created_at,desc"
// @Success      200 {object} aohhttp.PaginationResponsePayload{data=[]service.Foo}
// @Router       /foos [get]
func (h *fooHandler) list(w http.ResponseWriter, r *http.Request) {
    pageReq, err := aohhttp.GetQueryPagination(r)
    if err != nil {
        aoherr.Render(w, r, aoherr.Wrap(aoherr.ClassValidation,
            aoherr.CodeMalformedRequest, "invalid pagination parameters", err))
        return
    }
    items, total, err := h.svc.List(r.Context(), service.ListFoosInput{
        TenantID: getTenantID(r.Context()),
        Page:     *pageReq,
    })
    if err != nil { aoherr.Render(w, r, err); return } // status from the class
    _ = render.Render(w, r, aohhttp.PaginationResponse(http.StatusOK, "", aohhttp.PageResponse{
        Number:       pageReq.Number,
        Size:         pageReq.Size,
        TotalRecords: int(total),
        Count:        len(items),
        Sort:         echoSort(pageReq.Sorts), // []string of "field,direction"
    }, items))
}
```

Matching test assertion that pins the envelope so future regressions fail loudly:

```go
var resp map[string]any
require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

// PaginationResponsePayload puts items at top-level `data` (array, not object).
data, ok := resp["data"].([]any)
require.True(t, ok, "data must be an array directly — not nested under data.data")

// `page` is a sibling object carrying {number, size, total_records, count, sort}.
page, ok := resp["page"].(map[string]any)
require.True(t, ok, "response must have a sibling `page` object")
assert.EqualValues(t, 11, page["total_records"])
assert.IsType(t, []any{}, page["sort"]) // sort echoes as []string, not a single comma-joined string
```

## Authentication

JWT extraction in handlers — `withUserInfo` puts these in `context.Context`:
`getUserID(ctx)`, `getUserName(ctx)`, `getTenantID(ctx)`, `getTenantRoles(ctx)`,
`getBearerToken(ctx)`. Health endpoints (`/livez`, `/readyz`) and Swagger UI
(`/swagger-ui/`) are unauthenticated by design — everything else requires a
Bearer token via the scaffold-provided `aohhttp.BearerAuth` middleware.

### Where roles come from (don't conflate Keycloak realm roles with AAS tenant roles)

The values `getTenantRoles(ctx)` returns are the **AAS-issued
`active_tenant.roles` JWT claim** — NOT Keycloak realm roles.

- **Realm roles** (`system-admin`, `tenant-admin`, `default-roles-aoh`) are
  platform-bootstrap, defined in upstream `realm-import.json`, and applications
  MUST NOT add to them.
- **Application roles** (your service's domain roles — e.g. `field-reporter`,
  `operations-team`, `incidents-reader`) are **AAS tenant roles** created in
  the existing `development` tenant via
  `POST /admin/tenants/{tenantId}/roles`, assigned to users via
  `POST /admin/tenants/{tenantId}/users/{userId}/roles`, and surfaced into the
  JWT by AAS as `active_tenant.roles`.

Defining a new application role is a bootstrap-time concern. Do NOT edit
`realm-import.json`. Instead, add it to the project-owned YAML at
`compose/iams/init/project-aas/roles.yaml` (laid down + auto-wired by the
`aoh-compose` skill). An idempotent Python reconciler in the same directory
applies the YAML on every `docker compose up`, so a fresh
`docker compose down -v && docker compose up -d` recreates the role. For the
full pattern (file layout, schema, dependency-ordered walk, compose service
wiring), see the `aoh-knowledge` skill → `references/services/iams.md` →
"Authorization is via AAS, not Keycloak" and "Project-level AAS bootstrap
(reproducibility pattern)".

For a request-time role check, read `getTenantRoles(ctx)` and compare against
the role name string. Do NOT make a live AAS `/evaluate` call unless your
endpoint has a per-resource permission model (e.g. per-incident access
control) — the JWT claim IS the AAS authorization decision for coarse role
checks.

> ⚠️ **There is no `active_tenant.permissions` claim in the JWT.** AAS surfaces
> roles only; resolved permissions are NOT projected into the token. A middleware
> that 403s on missing `active_tenant.permissions` will reject every legitimate
> user — the typed `aohhttp.JwtClaim` struct doesn't even declare the field. If
> you need permission-level gating, write a `RequirePermission(perm)` helper that
> reads `getTenantRoles(ctx)` and applies a static role→permission projection
> mirroring `compose/iams/init/project-aas/roles.yaml`. The map duplicates the
> YAML but stays close to the code, and `roles.yaml` remains the single source
> of truth — the projection is just code-readable form of the same data.
> See `aoh-knowledge/services/iams.md` → "Authorization Models" for the three
> approaches and when to pick each.

### Testing with cURL — request `scope=openid`

`aohhttp.BearerAuth` validates every JWT by calling Keycloak's userinfo endpoint.
Userinfo returns **403** for tokens issued without the `openid` scope, which the
middleware translates into `"invalid jwt"` and a **401** to the caller. Request
tokens with `scope=openid` explicitly when hitting the service with cURL or
Postman:

```bash
TOKEN=$(curl -s -X POST "http://iams-keycloak.127.0.0.1.nip.io/realms/aoh/protocol/openid-connect/token" \
  -d grant_type=password -d client_id=<public-client> \
  -d username=<user> -d password=<pw> -d scope=openid | jq -r .access_token)

curl -s http://<service>.127.0.0.1.nip.io/<endpoint> \
  -H "Authorization: Bearer $TOKEN"
```

Web-base and other SvelteKit frontends already include `openid` in their
Authorization Code Flow, so browser traffic is unaffected.
