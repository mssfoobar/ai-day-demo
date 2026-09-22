# dispatch-svc

The dispatch field-unit service. Go + PostgreSQL, behind **IAMS bearer authentication**.

It owns the roster the console renders and edits: read, create, replace and delete on
`/v1/units`, scoped to the caller's tenant, with optimistic concurrency via `occ_lock`.
Every positioned unit is mirrored into `gis-service` as a geo-entity through a
transactional outbox, which is what puts it on the map.

## Prerequisites

- **Go 1.25+**
- **A container runtime** (Docker or Podman) for the platform stack — PostgreSQL plus
  IAMS (Keycloak + AAS), SDS, RTUS and GIS. See `SETUP.md` at the repo root.
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

Or run this service on its own, against an already-running stack:

```sh
docker compose -f ../../compose/compose.yml up -d   # or: podman compose -f ... up -d

SQL_HOST=localhost SQL_PORT=5432 SQL_USER=dispatch SQL_PASSWORD=dispatch \
SQL_DATABASE_NAME=dispatch SQL_SCHEMA_NAME=dispatch SQL_SSL_MODE=disable \
IAMS_KEYCLOAK_HOST=http://iams-keycloak.127.0.0.1.nip.io IAMS_KEYCLOAK_PORT=80 \
IAMS_KEYCLOAK_REALM=aoh \
GIS_URL=http://gis.127.0.0.1.nip.io \
HTTP_PORT=8081 HTTP_ALLOWED_ORIGINS= \
go run ./cmd/server
```

`IAMS_KEYCLOAK_HOST` must carry the scheme; without it the composed issuer URL is
unparseable and every token validation fails. `HTTP_ALLOWED_ORIGINS` stays empty because
the browser never calls this service directly — the console reaches it from its SvelteKit
server.

Migrations are embedded and applied on start, so there is no separate migration step and
no external migration tool to install. The **roster is not** a migration: the service
seeds a tenant once, on its first request from a caller holding `dispatch-dispatcher`,
before answering it. There is no seed command to run
(`openspec/changes/dispatch-iams-and-unit-map/design.md` D12).

## Authentication and authorization

Every `/v1/units` route requires a bearer token issued by the bundled `aoh` realm,
validated by `aoh-golib`'s `aohhttp.BearerAuth` against Keycloak's userinfo endpoint.
`/livez` and `/readyz` stay unauthenticated — Kubernetes probes them without credentials.

Because validation goes through userinfo, a token issued **without `scope=openid`** is
rejected, and surfaces as an opaque 401:

```sh
TOKEN=$(curl -fsS -X POST \
  "http://iams-keycloak.127.0.0.1.nip.io/realms/aoh/protocol/openid-connect/token" \
  -d grant_type=password -d client_id=web -d scope=openid \
  -d username=admin -d password='P@ssw0rd' \
  | node -e "let d='';process.stdin.on('data',c=>d+=c).on('end',()=>console.log(JSON.parse(d).access_token))")

curl -s http://localhost:8081/v1/units -H "Authorization: Bearer $TOKEN"
```

Two **AAS tenant roles** gate the service — `dispatch-viewer` (read) and
`dispatch-dispatcher` (read + write). They are declared in
`compose/iams/init/project-aas/roles.yaml`, arrive in the JWT as `active_tenant.roles`,
and are projected onto read/write permissions in `internal/service/permissions.go`. They
are **not** Keycloak realm roles, and there is **no** `active_tenant.permissions` claim —
AAS does not emit one.

`tenant_id`, `created_by` and `updated_by` come from the token (`active_tenant.tenant_id`
and `sub`). If a request body carries any of the three they are ignored, never stored.

## Environment

The defaults target the in-container stack, so a **native** run must set at least
`IAMS_KEYCLOAK_HOST`, `IAMS_KEYCLOAK_PORT` and `GIS_URL` (see Running above) — their
defaults resolve only inside the compose network.

| Variable | Default | Meaning |
|---|---|---|
| `HTTP_PORT` | `8081` | Port the HTTP server binds. |
| `HTTP_ALLOWED_ORIGINS` | *(empty)* | Comma-separated CORS origins. Empty is correct: the browser never calls this service directly. |
| `SQL_HOST` | `localhost` | PostgreSQL host. |
| `SQL_PORT` | `5432` | PostgreSQL port. |
| `SQL_USER` | `dispatch` | PostgreSQL user. |
| `SQL_PASSWORD` | `dispatch` | PostgreSQL password. |
| `SQL_DATABASE_NAME` | `dispatch` | Database name. |
| `SQL_SCHEMA_NAME` | `dispatch` | Schema this service owns. Configurable so the service can share a database without clashing. |
| `SQL_SSL_MODE` | `disable` | `sslmode` for the connection. |
| `IAMS_KEYCLOAK_HOST` | `http://iams-keycloak` | Keycloak base URL. **Must include the scheme.** |
| `IAMS_KEYCLOAK_PORT` | `8080` | Keycloak port. `80` when reached through Traefik. |
| `IAMS_KEYCLOAK_REALM` | `aoh` | Realm every bearer token must come from. |
| `GIS_URL` | `http://gis-service:8080` | `gis-service` base URL — the only downstream for entity data. |
| `PROJECTION_ATTEMPTS` | `0` | Safety cap on delivery attempts per outbox row. **0 means "as many as fit before the operator's token expires"**, which is the real bound. |
| `PROJECTION_BACKOFF` | `500ms` | First retry delay; doubles per attempt. |
| `PROJECTION_MAX_BACKOFF` | `15s` | Ceiling for that doubling, so a long window is not spent asleep. |

The token's lifetime is the bound, not the attempt count. A `gis-service` outage of a few
tens of seconds sits comfortably inside a 300s token, and capping by attempts would strand
the row while the credential was still perfectly good — the opposite of "delivery resumes
after a brief outage".

There is deliberately **no** poll interval, client id or client secret. The projection is
triggered by the write that produced it and runs on that write's operator token, so a
timer waking with no credential could deliver nothing, and a client-credentials token
carries no `active_tenant` claim for `gis-service` to resolve a tenant from.

## API

Every `/v1/units` route below requires a bearer token; writes additionally require
`dispatch-dispatcher`. `/livez` and `/readyz` do not.

| Method | Path | Description |
|---|---|---|
| GET | `/v1/units` | The caller's tenant's field units, ordered by `unit_code`. Seeds an unseeded tenant first when the caller is a dispatcher. |
| GET | `/v1/units/{unit_code}` | One unit in the caller's tenant; 404 when the code is unknown **or belongs to another tenant**. |
| POST | `/v1/units` | Create. 201. Body: the writable fields below. |
| PUT | `/v1/units/{unit_code}` | Replace the writable fields. Body must carry the current `occ_lock`. 200. |
| DELETE | `/v1/units/{unit_code}?occ_lock=N` | Delete the unit and its crew. 204. |
| POST / DELETE | `/v1/units/{unit_code}/assignment` | **Workshop exercise 1** — 501 `DISPATCH_NOT_IMPLEMENTED` until built. |
| GET | `/v1/units/{unit_code}/events` | **Workshop exercise 2** — 501 until built. |
| PUT | `/v1/units/{unit_code}/crew` | **Workshop exercise 3** — 501 until built. |
| GET | `/livez` | Liveness. |
| GET | `/readyz` | Readiness — fails when the database is unreachable. |

Writable fields: `unit_code` (create only), `call_sign`, `status`, `unit_type`, `station`,
`sector`, `radio_channel`, `shift`, `capabilities`, `position`. Crew and assignment are
read-only for now. Every write bumps `last_contact` and `occ_lock`.

**Position.** `position` is `{lon, lat, at}`, all three or none — a unit without a fix
omits the key entirely, exactly as `assignment` does. `at` is optional on write and
defaults to the write time; it is the time the *fix* was taken and moves independently of
`last_contact`, so an operator editing a radio channel does not look like a fresh GPS fix.
A replace with **no** `position` in the body clears it. Coordinates are validated to
`[-180, 180]` and `[-90, 90]`.

**The GIS mirror.** Every write records a row in `gis_outbox` in the same transaction, and
a worker drains it into `gis-service` (`entity_id` = `unit_code`, `entity_type` = `track`,
`geojson.properties.kind` = `field-unit`). The intent is derived from the unit's position
*after* the write, not from the verb: no position — never had one, cleared, or deleted —
removes the entity, so the map never shows a stale marker. If `gis-service` is
unreachable the write still succeeds and the row stays pending; re-saving the unit
enqueues a fresh one.

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
/ `DISPATCH_UNIT_WRITE_FAILED` (500), `DISPATCH_UNIT_FORBIDDEN` (403, the caller's roles
do not permit the operation), `DISPATCH_TENANT_SEED_FAILED` (500),
`DISPATCH_TENANT_MISSING` / `DISPATCH_IDENTITY_MISSING` (401 / 500). A missing, malformed
or expired bearer answers 401. The four workshop stub route/method pairs answer 501
`DISPATCH_NOT_IMPLEMENTED` — but only to an authorised caller (see `WORKSHOP.md` at the
repo root).

`PATCH`, and `PUT`/`DELETE` on the collection, return 405.

## Layout

```
cmd/server            wiring and graceful shutdown
internal/config       Viper config with local defaults
internal/domain       the Unit / Crew / Assignment / Position model
internal/auth         the caller's identity, from the validated bearer onto the context
internal/db           connection + embedded migration runner
internal/repo         sqlx queries          (persistence, incl. the outbox and the seed)
internal/service      classification        (domain rules, aoherr, role gating, seeding)
internal/projection   the GIS geo-entity payload and the outbox worker
internal/handler      chi routes + envelope (HTTP)
migrations            schema, embedded via go:embed
```

Layered handler → service → repo, so the persistence layer is swappable and the service
can be tested with a fake repo and no database.

## Tests

```sh
pnpm test              # or: go test ./... -count=1 -race
pnpm test:integration  # needs a real PostgreSQL — see below
```

The unit tests use hand-written fakes against the consumer-declared interfaces
(`service.UnitStore`, `projection.OutboxStore`) and need nothing running. The repo tests
are gated behind the `integration` build tag and a DSN, because what they prove —
outbox-and-unit atomicity, and the `ON CONFLICT DO NOTHING` seed guard under two
concurrent dispatchers — are properties of PostgreSQL rather than of Go:

```sh
DISPATCH_SVC_TEST_DSN='host=localhost port=5432 user=dispatch password=dispatch dbname=dispatch sslmode=disable' \
  pnpm test:integration
```

They migrate themselves into a `dispatch_it` schema and use a unique tenant per run, so
they never touch the schema a running service is using.

## Deviation from `aoh-go-init`

This service was **hand-written** to the `aoh-go-init` architecture rather than generated
by it. That skill's scaffold is a Python script, and Python is not installed on the
machine this was built on. The layering, the error contract, the envelope, the health
probes and the config approach all follow the skill; what is *not* reproduced is its
extras — mockery config, swag/OpenAPI annotations, its Makefile and Dockerfile. Running
the real scaffold later and diffing against this is a reasonable follow-up.

See `openspec/changes/dispatch-units-service/design.md` (D1) for the full rationale.

