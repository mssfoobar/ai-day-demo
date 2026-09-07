# dispatch-svc

The dispatch field-unit service. Go + PostgreSQL, read-only, **no authentication**.

It owns the roster the console renders: `GET /v1/units` and `GET /v1/units/{unit_code}`.

## Prerequisites

- **Go 1.25+**
- **A container runtime** (Docker or Podman) for PostgreSQL — the only container in the
  workshop stack.
- **Access to the private `ops-hub` repo.** This service depends on
  `github.com/mssfoobar/ops-hub/packages/aoh-golib`, which is fetched over git. If
  `go mod download` fails, your git credentials cannot read that repo. Set:

  ```sh
  go env -w GOPRIVATE='github.com/mssfoobar/*'
  ```

  This is the second credential the workshop needs, alongside the `~/.npmrc` token the
  frontend requires for `@mssfoobar/ui`.

## Running

From the repo root, one command checks every prerequisite (including the two credentials
above), installs, and starts the database, this service, and the console:

```sh
pnpm launch        # or .\launch.ps1 / ./launch.sh from a fresh clone
```

`pnpm start` does only the last part, once you are set up.

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
| GET | `/livez` | Liveness. |
| GET | `/readyz` | Readiness — fails when the database is unreachable. |

Success bodies are the AOH envelope, never a bare array:

```jsonc
{ "data": [ /* units */ ], "sent_at": "2026-09-07T06:31:05Z" }
```

Failures use the AOH error contract — `{timestamp, errorCode, errorMessage, ...}` — so no
4xx/5xx has an empty body. Codes are namespaced: `DISPATCH_UNIT_NOT_FOUND`,
`DISPATCH_UNIT_CODE_REQUIRED`, `DISPATCH_UNIT_READ_FAILED`.

There are **no write endpoints**; `POST`/`PUT`/`PATCH`/`DELETE` return 405.

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

## Tests

```sh
go build ./... && go vet ./... && go test ./... -count=1
```

Service tests use a hand-written fake repo; handler tests assert the envelope shape, the
404 body, 405 on write verbs, and that `/readyz` fails when the database is unreachable.
There is deliberately no end-to-end test — see design.md D6.
