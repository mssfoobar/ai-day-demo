# AGENTS.md

Agent context for `dispatch-svc`. Monorepo-wide conventions live in the repo-root
`AGENTS.md`; how to run and configure this service lives in `README.md`.

- Coding conventions: the `aoh-conventions` skill — `references/go.md` (layering,
  `aohhttp`, `aohlog`), `references/api.md` (envelope, `/v{N}` paths, health probes),
  `references/database.md` (naming, mandatory columns).
- Error contract: the `aoh-error-handling` skill.

## Architecture (load-bearing)

- **Layered handler → service → repo.** The handler speaks HTTP and nothing else; the
  service classifies failures; the repo speaks SQL. `service.UnitReader` is declared at
  the consumer, so tests substitute a fake repo with no database and no mock framework.
- **This service was hand-written, not scaffolded.** `aoh-go-init`'s generator is a Python
  script and Python was unavailable. It follows the same architecture; it does not have
  the scaffold's mockery config, swag annotations, Makefile or Dockerfile. Don't assume
  those files exist. See `openspec/changes/dispatch-units-service/design.md` D1.
- **`aoh-golib` is a local copy, not a fetched module.** `go.mod` has a `replace` pointing
  at `packages/aoh-golib` (also in `go.work`), so builds need no ops-hub access. The
  import path is unchanged — code still says `github.com/mssfoobar/ops-hub/packages/
  aoh-golib/...`. Never edit the copy; fixes go upstream, and `LOCAL_COPY.md` there says
  how to refresh. `go mod tidy` will not add a `go.sum` line for it — that is expected.
- **Bearer auth, tenant scoping and role gating.** `/v1/units` is behind
  `aohhttp.BearerAuth` (validates against Keycloak's **userinfo** endpoint, so a token
  issued without `scope=openid` is rejected as an opaque 401); `/livez` and `/readyz` are
  mounted outside it. `internal/auth` then puts `sub`, `active_tenant.tenant_id` and
  `active_tenant.roles` on the request context. Every query carries
  `WHERE tenant_id = $n`, and `created_by` / `updated_by` / `tenant_id` come from the
  token — a request body carrying them is ignored, never trusted.
- **Roles project onto permissions in-process** (`internal/service/permissions.go`), from
  `active_tenant.roles`. **There is no `active_tenant.permissions` claim** — AAS does not
  emit one, and code that gates on it 403s every legitimate user. The map's source of
  truth is `compose/iams/init/project-aas/roles.yaml`, and its TypeScript twin is
  `apps/dispatch-web/src/lib/aoh/dispatch/permissions.ts`; all three move together.
- **The GIS projection is a transactional outbox, never an inline call.** A unit write
  records a `gis_outbox` row in the *same transaction* (`repo.enqueue`), and
  `internal/projection`'s worker drains it. Two rules are load-bearing and easy to
  "simplify" wrongly:
  - **The intent comes from the unit's position after the write, not from the verb.** No
    position — never had one, cleared by a replace, or deleted — enqueues a `delete`.
    Drop that and a cleared position leaves a stale marker on the map at the unit's last
    known location, which looks like data rather than an absence.
  - **Delivery carries the writing operator's own bearer**, captured by value before the
    post-commit goroutine detaches. A client-credentials token carries no `active_tenant`
    claim and `gis-service` resolves the tenant from it, so a service account cannot do
    this. A row that outlives its token stays **pending and visible**; do not add a
    sweeper or drain it on a later request — that would attribute one operator's write to
    another and could make a `dispatch-viewer`'s read perform a GIS write. See
    `openspec/changes/dispatch-iams-and-unit-map/design.md` D2a and D6a.
- **Writes are guarded by `occ_lock`.** PUT and DELETE take the client's last-read version
  and the SQL `WHERE` includes it; zero rows affected means stale → `repo.ErrStale` →
  409 `DISPATCH_UNIT_STALE`. Never drop the guard "to make a test pass" — silently
  overwriting a newer row is the failure it exists to prevent. `staleOrMissing` tells stale
  apart from not-found so the client gets the right status.
- **Postgres errors are mapped at the repo boundary** (`mapWriteError`): `23505` →
  `ErrConflict`, `23514` → `ErrInvalid`. The service classifies those into `aoherr`; the
  handler never sees a driver error. Add new constraint codes there, not in handlers.
- **Every write bumps `last_contact`** (design decision, `dispatch-units-crud` D4). If that
  stops being wanted, change the UPDATE, not the frontend.
- **Assignment is writable through its own resource, not through `UnitInput`.**
  `POST` / `DELETE /v1/units/{unit_code}/assignment` own it, and the status rule lives in
  the service: dispatch means `En route`, stand down means `Available`. `UnitInput` still
  omits assignment and crew, so a replace can never clear an incident by accident. Crew is
  not writable yet. `PATCH` and collection-level `PUT`/`DELETE` are unmounted and return
  405. Writes exist now, so live fan-out to other sessions is the next gap — that is RTUS,
  not polling.
- **Migrations are embedded and run on start** (`internal/db`, `migrations/`), but the
  **roster is not a migration**. The service seeds a tenant once, on its first request
  from a caller holding `dispatch-dispatcher`, before answering it — a committed
  migration cannot know the tenant id (AAS assigns it at stack-up) and SQL rows bypass the
  outbox, so they would exist in the console and be permanently absent from the map.
  `0004_drop_preauth_seed.up.sql` deletes what the old SQL seed left behind. The roster
  data is `internal/service/roster.go`; there is no seed endpoint, binary or script.
- **The seed's guard is a marker row, not an emptiness check.** `tenant_seed` makes it
  *seed once per tenant, ever*. "This tenant has no units" is the tempting trigger and is
  a trap: the console ships a working delete, so a dispatcher who deleted the last unit
  would have the whole roster silently resurrected on their next click. The marker insert
  MUST stay the seed transaction's first statement and MUST report whether it inserted
  (`ON CONFLICT DO NOTHING RETURNING`) — it raises nothing, so a guard that runs it and
  carries on lets two racers write the roster twice with row counts that still look right.
- **`{{SCHEMA}}` is substituted at runtime** from `SQL_SCHEMA_NAME`. Write migrations
  against that token, never a hardcoded `dispatch.`.
- **Tests are split by what they can honestly prove.** `go test ./...` runs against
  hand-written fakes on the consumer-declared interfaces (`service.UnitStore`,
  `projection.OutboxStore`) and needs nothing running. The repo tests are behind
  `//go:build integration` + `DISPATCH_SVC_TEST_DSN` (`pnpm test:integration`) because
  outbox-and-unit atomicity and the seed's `ON CONFLICT` race are properties of
  PostgreSQL, not of Go — a fake would prove only that it returned what it was told.
  Never let a default `go test ./...` need a database.

## Conventions that bite

- **Errors must be classified.** Return `*aoherr.Error` from the service layer, never a
  bare `sql.ErrNoRows` — an unclassified error renders as a suppressed 500, so a missing
  unit would look like an outage. `repo.ErrNotFound` → `aoherr.ClassNotFound`.
- **Error codes are `UPPER_SNAKE_CASE` and module-prefixed**, declared in
  `internal/service/errors.go` with `aoherr.MustCode` so a malformed code panics at init
  rather than reaching a client.
- **Every 2xx goes through `aohhttp.Response`** — the envelope is the contract; never
  write a bare array or object.
- **`aoherr.Error`'s fields are unexported**: use the `Class()` / `Code()` accessors, not
  `.Class` / `.Code`.
- **An unassigned unit omits `assignment` entirely**, and an un-positioned one omits
  `position` (pointer + `omitempty`). The frontend branches on absence; an empty object
  would render as a phantom assignment, and a zero coordinate as a marker off West Africa.
- **`last_contact` and `position_at` are different events.** Every write still bumps
  `last_contact`, and `position_at` must NOT follow it — it is the time the *fix* was
  taken, and it moves only when a write supplies a position. Tie them together and every
  edit looks like a fresh GPS fix, which makes the map's fix age a lie.
- **The status vocabulary is enforced by a DB `CHECK`**, not by Go. Adding a status means
  a migration, and the frontend's runtime guard has to learn it too.
