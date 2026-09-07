---
name: aoh-compose
description: >
  Create Docker Compose files for AOH services and external services in a compose/
  directory for local development. Use this skill when the developer already knows which
  services they need and wants to generate the compose configuration. For service discovery,
  what-each-service-does, and integration patterns, use the aoh-knowledge skill instead.
compatibility: Requires Docker or Podman with Compose plugin
allowed-tools: Read Write Edit Bash(docker:*) Bash(podman:*) Bash(python3:*) Glob Grep
---

# Compose File Generator

Generate Docker Compose configuration in a `compose/` directory at the project root.

The heavy lifting (copying assets, resolving transitive dependencies, auto-detecting
the container runtime, writing the top-level `compose.yml`) is done by
`scripts/bootstrap.py`. The conversational part of this skill is limited to gathering
the developer's intent and then invoking the script with structured args.

## Where knowledge lives

- **What each AOH service does, its architecture, its API endpoints, and which services
  to pick for a given use case** → `aoh-knowledge` skill. Read that skill's
  `references/services/<service>.md` and `references/service-catalogue.md` before
  recommending services.
- **Which assets get copied where, transitive dep resolution, runtime detection** →
  `scripts/bootstrap.py` (the `SERVICES` dict near the top is the authoritative
  dep graph).
- **How to customize the committed `aoh` Keycloak realm** (adding roles, seed users,
  project clients) → `.claude/skills/aoh-knowledge/references/keycloak-realm-guide.md`.

## Gather inputs

Ask the developer:

1. Which **AOH services** they want (iams + sds + the `otel` gateway
   collector are always included — they do not need to name those). Valid ids are
   the keys of `SERVICES` in `scripts/bootstrap.py` (`rtus`, `gis`, `unh`, `ian`,
   `dash`, `ptmgr`, `amm`, `form`, `wfe`, `signoz`). The `otel` gateway
   (the OTLP endpoint every service targets) is **core / default-on** — services export to it out of the box; `signoz` is the optional bundled
   backend (heavy ClickHouse/ZooKeeper) and the gateway logs harmless retries
   until it (or a BYO backend) is wired. Transitive deps are resolved by the
   script — the developer only needs to name the top-level ones.
2. Whether they have any **custom project services** to add (their own code under
   `apps/<name>/`). For each, ask the name and whether it's `go` (built with
   `aoh-go-init`) or `web` (built with `aoh-web-init`).

If the developer is unsure what services they need, direct them to `aoh-knowledge`
before continuing.

## Run the script

```bash
python3 .claude/skills/aoh-compose/scripts/bootstrap.py \
    --repo-root <repo-root> \
    [--services <comma,separated,list>] \
    [--custom-service name=<name>,type=<go|web>] ...
```

The script is idempotent. Re-running with a larger service list adds what's missing
without touching anything already in place.

### What the script does

1. Copies Traefik + IAMS (+ realm-import + iams-init postman + project-aas
   bootstrap template) + SDS to `compose/`. The project-aas template drops
   `compose/iams/init/project-aas/` (Dockerfile, bootstrap.py, roles.yaml,
   README.md) and the `project-aas-init` service is auto-wired into
   `compose/iams/compose.yml` so it runs on every `docker compose up`.
   Developers customize `roles.yaml` only.
   SDS is a companion service to IAMS, always bundled (sds-server + valkey),
   and **MANDATORY for every AOH web app to wire** — `SDS_URL` is required
   env, not a per-app decision. The web scaffold (`aoh-web-init`) sets it
   by default and the `auth.ts` cookie-only fallback exists only for
   emergency diagnostics. See `aoh-knowledge/services/sds.md` → "Why SDS
   is mandatory" before recommending any web auth posture that omits it.
2. Copies shared `.gitignore`, `.env.template`, `.env`, and `compose.override.sample.yml`.
   `.env` is auto-created from the template on first run; the dev fills in
   compose-consumed values like `DEV_DOMAIN`, `DEV_USER`, `DEV_PASSWORD`, and
   any AOH service image tags they want to pin.
3. Auto-detects the container runtime. If Docker is present and Podman is not, it
   writes `compose.override.yml` to remount the Docker socket on Traefik. Podman is
   the recommended runtime; all other cases leave the default in place.
4. Copies each requested AOH catalogue service's asset directory to `compose/<svc>/`,
   resolving transitive deps from the built-in catalogue.
5. For each `--custom-service`, copies the matching template
   (`assets/custom-service-template.yml` for `go`, `assets/web-service-template.yml`
   for `web`) to `compose/<name>/compose.yml` with `<name>` placeholders replaced.
   The developer then uncomments the blocks they need (db service, depends_on,
   env vars, healthcheck).
6. Regenerates the top-level `compose/compose.yml` with the union of includes.

## What the script does NOT do

These are intentionally left to the developer or a follow-up step — they are
project-specific and don't belong in a generic scaffold:

- **Project-specific Keycloak changes.** Adding new OIDC clients, claim mappers,
  or seed users goes into the existing `aoh` realm at
  `compose/iams/keycloak/realm-import.json`. See
  `.claude/skills/aoh-knowledge/references/keycloak-realm-guide.md`. Do not create
  a parallel realm. **Application roles (e.g. `field-reporter`, `operations-team`)
  do NOT belong here** — they are AAS tenant roles, not Keycloak realm roles
  (see next bullet).
  > ⚠️ Keycloak's `start --import-realm` is **skip-if-exists**. After editing
  > `realm-import.json`, you MUST `docker compose down -v iams-db && docker compose up -d`
  > to re-import — a plain restart or `--force-recreate iams-keycloak` will NOT pick
  > up the change. See the realm guide's "Re-importing after edits" section.
- **Customizing project-level AAS authz state (roles, groups, resources, scopes, permissions, assignments).**
  The bootstrap script lays down the `compose/iams/init/project-aas/`
  template (Dockerfile + `bootstrap.py` + `roles.yaml`) and auto-wires the
  `project-aas-init` compose service — that part is automated. The
  developer customizes `roles.yaml` (and only `roles.yaml`) to define
  their project's authz state. Application roles MUST NOT be added to
  Keycloak `realm-import.json`. See
  `.claude/skills/aoh-knowledge/references/services/iams.md` →
  "Project-level AAS bootstrap (reproducibility pattern)" for the schema,
  the dependency-ordered walk, and the agent contract.
- **Custom-service env vars.** The templates ship with commented samples. The dev
  uncomments what the service actually needs.
- **External services** not covered by AOH (PostgreSQL, Redis, RabbitMQ, MongoDB,
  Elasticsearch, Kafka, NATS, Mailpit, etc.). Generate a compose config from the
  official Docker image docs, use `name: aoh` so it joins the shared network, and
  add `./<name>/compose.yml` to the top-level `compose/compose.yml` include list.
- **Pre-building custom-service images.** Compose files for `apps/<name>/`
  services reference `image: <name>:${<NAME>_TAG:-local}` and do NOT have a
  `build:` block (podman's compose builder can't pass `--mount=type=secret`).
  Normal verification runs the app natively against compose-up'd infra and
  needs no image at all. If a change touches the Dockerfile or container
  config, the dev runs `podman build` themselves — see "Verify → When to
  verify the container itself" below.
- **Running `docker compose up`.** The dev inspects first.

## Verify

The compose stack is **infra only** for normal verification. Custom apps run
natively (`go run` / `pnpm dev`) against the composed-up infra — this is
the verification path mandated by `openspec/config.yaml` for all `apps/<name>/`
changes, including multi-app E2E. `podman build` + `compose up <app>` is
reserved for explicit container-packaging checks (Dockerfile changes,
container-only config) and is NOT the regular verification path.

```bash
cd compose
# Bring up infra (plus any transitive deps via the `include:` chain).
# Services under `apps/<name>/` are tagged with `profiles: [apps]`, so a
# bare `compose up` brings up infra only and does NOT try to pull the
# not-yet-built `<name>:local` image. Listing infra services explicitly
# still works and remains the safest way to be deterministic:
podman compose up -d                                    # infra only
# or — explicit form is fine too:
podman compose up -d iams-keycloak iams-aas <name>-db   # example
podman compose ps   # expect every infra container healthy
```

Then from each `apps/<name>/`, run the app natively:

- Go service: `go run ./cmd/server`
- Pnpm web: `pnpm dev`

The app picks up the same env vars the container would (e.g. point
`SQL_HOST`, `IAMS_KEYCLOAK_HOST` at `localhost` if the compose stack exposes
the relevant ports). For multi-app E2E, start each app natively in its own
shell and drive the cross-app scenario from there.

Commands must be run from `compose/` so Compose auto-discovers `.env`,
`compose.override.yml`, and resolves relative paths in included files correctly.

**If every Traefik-routed host 404s under Docker** (containers healthy, no
errors anywhere): the Docker socket override is missing — the runtime was
decided at scaffold time, not up time. Copy `compose.override.sample.yml` to
`compose.override.yml` and `docker compose up -d traefik`.

**On Apple Silicon** (`docker compose up` fails with `no matching manifest for
linux/arm64/v8`): the published `@mssfoobar/*` images are amd64-only. Run them
under Rosetta emulation by adding `platform: linux/amd64` to each mssfoobar
service in `compose.override.yml` (e.g. `iams-keycloak`, `iams-aas`, `iams-web`,
`sds-server`, `rtus-pms`, `rtus-seh`, and any module service like `gis-service`).
Postgres/Valkey/Traefik/OTEL are multi-arch and need no override. Ensure Docker
Desktop's "Use Rosetta for x86/amd64 emulation" is enabled.

### When to verify the container itself

If the change touches a Dockerfile, the container entrypoint, or any
container-only config (file permissions, non-root user, env var mapping),
verify the packaged image as a separate step. Pre-build the image and
`compose up` it:

```bash
# Export your GitHub PAT (shell rc / direnv / one-shot) — not stored in
# compose/.env since compose itself doesn't read it:
export ACCESS_TOKEN=ghp_xxxxxxxx

# Go service (Dockerfile copies only from its own dir):
podman build \
    --secret id=access_token,env=ACCESS_TOKEN \
    -t <name>:local \
    apps/<name>

# Pnpm web service (Dockerfile needs the full monorepo as context):
podman build \
    --secret id=access_token,env=ACCESS_TOKEN \
    -t <name>:local \
    -f apps/<name>/Dockerfile \
    .

# Then bring the service up via compose — the `apps` profile opts in
# to services under `apps/<name>/` that depend on a pre-built image:
cd compose && podman compose --profile apps up -d <name>
```

Why `podman build` instead of `podman compose up --build`: podman's compose
provider uses the classic Docker builder, which doesn't support BuildKit
`--mount=type=secret`. Our Dockerfiles need a GitHub PAT for private
`@mssfoobar/*` packages, so builds must happen out-of-band. `env=ACCESS_TOKEN`
reads the secret directly from the named env var — no temp file on disk.
Inside the Dockerfile the secret is mounted at `/run/secrets/access_token`.

Skip this section entirely for changes that don't touch the Dockerfile or
container config — native run against compose infra is sufficient.

### `<NAME>_TAG` env var convention

Custom-service compose files reference images via
`image: <name>:${<NAME>_TAG:-local}` where `<NAME>` is the app dir uppercased
with `-` → `_` (e.g. an app dir `foo-svc` → env var `FOO_SVC_TAG`). Default
is `local`. Override to test a CI-built image without editing files:

```bash
<NAME>_TAG=ci-1234 podman compose up -d <name>
```

**Optional repo-root convenience scripts** — if the repo has a top-level `package.json`
(Turborepo / pnpm workspaces), offer to add:

```json
{
  "scripts": {
    "compose:up": "cd compose && podman compose up -d",
    "compose:down": "cd compose && podman compose down",
    "compose:logs": "cd compose && podman compose logs -f",
    "compose:ps": "cd compose && podman compose ps"
  }
}
```

When the repo also needs the container-packaging path (Dockerfile changes,
CI smoke tests), add `compose:build:<name>` per custom service:

- **Go service:** `podman build --secret id=access_token,env=ACCESS_TOKEN -t <name>:local apps/<name>`
- **Pnpm web service:** `podman build --secret id=access_token,env=ACCESS_TOKEN -t <name>:local -f apps/<name>/Dockerfile .`

These are opt-in — normal verification runs the app natively against
compose-up'd infra, so most contributors won't invoke them.

Default IAMS admin credentials: `admin` / `P@ssw0rd`.
