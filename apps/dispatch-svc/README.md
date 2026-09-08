# dispatch-svc

The dispatch field-unit service. Go + PostgreSQL, **no authentication**.

It owns the roster the console renders and edits: read, create, replace and delete on
`/v1/units`, with optimistic concurrency via `occ_lock`.

## Prerequisites

- **Go 1.25+**
- **A container runtime** (Docker or Podman) for PostgreSQL — the only container in the
  workshop stack.
That is all. The AOH shared library this service uses,
`github.com/mssfoobar/ops-hub/packages/aoh-golib`, is checked in as a local copy at
`packages/aoh-golib` and wired in with a `replace` directive, so **no access to the
private `ops-hub` repo and no `GOPRIVATE` setting is needed**. See
`packages/aoh-golib/LOCAL_COPY.md` for the version and how to refresh it.

## Running

From the repo root, one command starts the database, this service, and the console:

```sh
pnpm install       # once
pnpm start
```

Or run this service on its own, against an already-running database:

```sh
docker compose -f ../../compose/compose.yml up -d postgres   # or podman compose ...
go run ./cmd/server
```

Migrations and the seed are embedded and applied on start, so there is no separate
migration step and no external migration tool to install.

## Environment

All optional — the defaults target the compose Postgres on localhost.

| Variable | Default | Meaning |
|---|---|---|
| `HTTP_PORT` | `8081` | Port the HTTP server binds. |
| `SQL_HOST` | `localhost` | PostgreSQL host. |
| `SQL_PORT` | `5432` | PostgreSQL port. |
| `SQL_USER` | `dispatch` | PostgreSQL user. |
| `SQL_PASSWORD` | `dispatch` | PostgreSQL password. |
| `SQL_DATABASE_NAME` | `dispatch` | Database name. |
| `SQL_SCHEMA_NAME` | `dispatch` | Schema this service owns. Configurable so the service can share a database without clashing. |
| `SQL_SSL_MODE` | `disable` | `sslmode` for the connection. |

## API

| Method | Path | Description |
|---|---|---|
| GET | `/v1/units` | Every field unit, ordered by `unit_code`. |
| GET | `/v1/units/{unit_code}` | One unit; 404 when the code is unknown. |
| POST | `/v1/units` | Create. 201. Body: the writable fields below. |
| PUT | `/v1/units/{unit_code}` | Replace the writable fields. Body must carry the current `occ_lock`. 200. |
| DELETE | `/v1/units/{unit_code}?occ_lock=N` | Delete the unit and its crew. 204. |
| POST / DELETE | `/v1/units/{unit_code}/assignment` | **Workshop exercise 1** — 501 `DISPATCH_NOT_IMPLEMENTED` until built. |
| GET | `/v1/units/{unit_code}/events` | **Workshop exercise 2** — 501 until built. |
| PUT | `/v1/units/{unit_code}/crew` | **Workshop exercise 3** — 501 until built. |
| GET | `/livez` | Liveness. |
| GET | `/readyz` | Readiness — fails when the database is unreachable. |

Writable fields: `unit_code` (create only), `call_sign`, `status`, `unit_type`, `station`,
`sector`, `radio_channel`, `shift`, `capabilities`. Crew and assignment are read-only for
now. Every write bumps `last_contact` and `occ_lock`.

**Optimistic concurrency.** Every unit carries an integer `occ_lock`. PUT and DELETE must
echo the value the client last read; if someone else wrote first the service answers
**409 `DISPATCH_UNIT_STALE`** and changes nothing. That is the AOH convention — a stale
edit is refused, never silently applied over a newer one.

Success bodies are the AOH envelope, never a bare array:

```jsonc
{ "data": [ /* units */ ], "sent_at": "2026-09-07T06:31:05Z" }
```

Failures use the AOH error contract — `{timestamp, errorCode, errorMessage, details?}` — so
no 4xx/5xx has an empty body. Validation failures (400, `DISPATCH_UNIT_INVALID`) list every
offending field in `details`. Other codes: `DISPATCH_UNIT_NOT_FOUND` (404),
`DISPATCH_UNIT_CODE_TAKEN` (409), `DISPATCH_UNIT_STALE` (409), `DISPATCH_UNIT_READ_FAILED`
/ `DISPATCH_UNIT_WRITE_FAILED` (500). The three workshop stubs answer 501
`DISPATCH_NOT_IMPLEMENTED` (see `WORKSHOP.md` at the repo root).

`PATCH`, and `PUT`/`DELETE` on the collection, return 405.

## Layout

```
cmd/server            wiring and graceful shutdown
internal/config       Viper config with local defaults
internal/domain       the Unit / Crew / Assignment model
internal/db           connection + embedded migration runner
internal/repo         sqlx queries          (persistence)
internal/service      classification        (domain rules, aoherr)
internal/handler      chi routes + envelope (HTTP)
migrations            schema + idempotent seed, embedded via go:embed
```

Layered handler → service → repo, so the persistence layer is swappable and the service
can be tested with a fake repo and no database.

## Deviation from `aoh-go-init`

This service was **hand-written** to the `aoh-go-init` architecture rather than generated
by it. That skill's scaffold is a Python script, and Python is not installed on the
machine this was built on. The layering, the error contract, the envelope, the health
probes and the config approach all follow the skill; what is *not* reproduced is its
extras — mockery config, swag/OpenAPI annotations, its Makefile and Dockerfile. Running
the real scaffold later and diffing against this is a reasonable follow-up.

See `openspec/changes/dispatch-units-service/design.md` (D1) for the full rationale.

