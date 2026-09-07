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
- **No authentication.** The workshop is unauthenticated by decision — there is no
  `BearerAuth` middleware, no JWT parsing, no `active_tenant`. Rows still carry
  `tenant_id` so multi-tenancy is not designed out, but nothing populates it from a token.
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
- **Crew and assignment are not writable yet.** `UnitInput` deliberately omits them.
  `PATCH` and collection-level `PUT`/`DELETE` are unmounted and return 405. Writes exist
  now, so live fan-out to other sessions is the next gap — that is RTUS, not polling.
- **Migrations are embedded and run on start** (`internal/db`, `migrations/`). The seed is
  idempotent (`ON CONFLICT … DO UPDATE`); keep it that way or the reproducibility gate
  stops meaning anything.
- **`{{SCHEMA}}` is substituted at runtime** from `SQL_SCHEMA_NAME`. Write migrations
  against that token, never a hardcoded `dispatch.`.

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
- **An unassigned unit omits `assignment` entirely** (pointer + `omitempty`). The frontend
  branches on absence; an empty object would render as a phantom assignment.
- **The status vocabulary is enforced by a DB `CHECK`**, not by Go. Adding a status means
  a migration, and the frontend's runtime guard has to learn it too.
