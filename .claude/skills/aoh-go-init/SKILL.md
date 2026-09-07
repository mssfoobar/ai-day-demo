---
name: aoh-go-init
description: >
  Scaffold a new Go microservice project with chi router, golangci-lint, mockery, Viper config,
  PostgreSQL with sqlx, and a layered architecture (handler → service → repo). Use this skill whenever the user wants to
  create a new Go microservice, scaffold a Go service, initialize a Go backend project, set up a Go API server, or
  bootstrap a Go REST service. Also trigger when the user mentions "new Go project", "go service template", or
  "microservice boilerplate" in a Go context.
---

# Go Microservice Init

Scaffold a production-ready Go microservice inside a Turborepo monorepo. A Python script
generates all infrastructure and example entity code. The scaffold compiles, passes lint,
and passes tests out of the box.

Entity-specific code is NOT part of this skill — add entities later through the task/change
workflow. The scaffold includes example entity files annotated with `TODELETE(scaffold)`
as a reference for the pattern.

For platform context — which AOH services to integrate with (IAMS JWT validation,
RTUS SSE, SDS session store, etc.), their API patterns, and which data models to
expect — consult the `aoh-knowledge` skill before writing entity code.

The scaffold is **born conformant to the AOH error contract**: the service layer
classifies failures as `*aoherr.Error` with module-namespaced codes, handlers
render them with `aoherr.Render`, and panics, unmatched routes, disallowed methods
and rejected tokens are covered by the `aohhttp` middlewares — so no 4xx/5xx has a
zero-length or plain-text body. Consult `aoh-error-handling` before adding error
paths of your own.

## Gather inputs

Ask the user for:
1. **Service name** (e.g., `inventory`) — the workspace directory name.
   If the workspace already exists, use its name.
2. **Go module path** (e.g., `github.com/org/my-turborepo/apps/inventory`) — ask if not provided
3. **Service description** — brief one-liner for AGENTS.md
4. **Monorepo shape** — consumer-app (default) or ops-hub platform-monorepo. This selects the
   target directory:
   - Consumer-app shape: `apps/<service-name>/` (default — no flag needed)
   - Platform-monorepo shape (ops-hub): pass `--target-dir modules/<service-name>/service`

   See `aoh-conventions/references/project.md` for which shape applies. If you're scaffolding
   an entire ops-hub module (not just the service), prefer the `aoh-module-init` skill — it
   creates the sibling `sdk/`, `client/`, `types/`, `deploy/` packages too and delegates
   the service scaffold to this skill via `--target-dir`.

## Steps

### Step 1. Run the scaffold script

```bash
python3 .claude/skills/aoh-go-init/scripts/scaffold.py \
    --name {service-name} \
    --module {module-path} \
    --repo-root {repo-root} \
    --description "{service-description}"
```

For the ops-hub platform-monorepo shape, add `--target-dir modules/{service-name}/service`:

```bash
python3 .claude/skills/aoh-go-init/scripts/scaffold.py \
    --name gis \
    --module github.com/mssfoobar/ops-hub/modules/gis/service \
    --repo-root /path/to/ops-hub \
    --target-dir modules/gis/service \
    --description "GIS geospatial entity + bookmark service"
```

The script creates the service directory (defaults to `apps/{service-name}/`, or whatever
`--target-dir` resolves to), registers it in `go.work` + `turbo.json`, then generates:

- All infrastructure Go files (config, router, health, middleware, db, errors, version),
  **born instrumented** with OpenTelemetry via `aoh-golib/otel` (server spans named by
  route, RED/HTTP + Go-runtime metrics, trace-correlated access logs; no-op without
  `OTEL_EXPORTER_OTLP_ENDPOINT`) and **born conformant to the AOH error contract**
  (`internal/service/errors.go` declares the service's `errorCode` namespace, derived
  from `--name`; the router mounts `aohhttp.Recoverer` / `NotFoundHandler` /
  `MethodNotAllowedHandler`)
- Example entity files annotated with `TODELETE(scaffold)` comments
- Asset files (Makefile, Dockerfile, .mockery.yaml, .gitignore, AGENTS.md, CLAUDE.md, README.md)
- package.json, config.yaml.example, go.work, turbo.json updates
- A go.mod `require github.com/mssfoobar/ops-hub/packages/aoh-golib vX.Y.Z` pin — the shared
  library (incl. its `otel` subpackage) is a normal registry dependency fetched from the
  private ops-hub repo at a released `packages/aoh-golib/v*` tag. No `replace`, no vendored
  copy; the `Dockerfile` is a standalone single-module build (context = the service dir) with
  a podman/docker-compatible file-sourced secret mount for the private fetch.
- Runs `go mod tidy` to install dependencies. **This fetches the private aoh-golib module** —
  the host needs GitHub access to `github.com/mssfoobar/ops-hub` (normal git credentials; the
  script sets `GONOSUMDB=github.com/mssfoobar/*` unless the host already configures
  GOPRIVATE/GONOSUMDB) or a spoke registry mirror (`GOPROXY=<mirror> GOSUMDB=off`).

Verify the output shows "Scaffold complete" with no errors.

### Step 2. Generate swagger docs

```bash
cd {service-dir}    # apps/{service-name} or modules/{service-name}/service
make swag
```

### Step 3. Generate mocks, lint, and test

```bash
cd {service-dir}
make mock          # generate mocks (overwrites placeholders)
go mod tidy        # finalize deps after generation
make fmt           # apply the gci import formatter (generators don't pre-format)
make lint-code     # verify linting passes
make test          # regenerates mocks then runs go test
```

Fix any failures before completing.

## Output

After all steps complete, display a summary:

```
## Scaffold Complete: {service-name}

**Service:** {service-dir}/    (apps/{service-name} by default, or the --target-dir value)
**Module:** {module-path}

### What was created
- Go microservice with chi router, IAMS auth, sqlx, Viper config
- OpenTelemetry born-instrumented (traces + RED/HTTP + runtime metrics + trace-correlated logs)
- Health endpoints: /livez, /readyz
- AOH error contract wired end to end — namespaced `errorCode`s in `internal/service/errors.go`, `aoherr.Render` in the handlers, `aohhttp.Recoverer` / `NotFoundHandler` / `MethodNotAllowedHandler` on the router
- Example entity (POST /examples + GET /examples) — annotated with TODELETE(scaffold), demonstrates `aohhttp.Response` (single object) and `aohhttp.PaginationResponse` (list) envelope patterns with matching swag annotations, plus a test that pins the error-contract body
- Swagger UI at /swagger-ui/index.html
- Makefile with build, lint, mock, swag targets
- Dockerfile for production builds

### Next steps
1. Consult `aoh-knowledge` for integration patterns (IAMS JWT claims, RTUS topics, SDS session keys) before writing entity code
2. Add real entity code (replace example entity files marked with `TODELETE(scaffold)`)
3. Replace the placeholder codes in `internal/service/errors.go` with codes that name real failures — keep the namespace prefix; consult `aoh-error-handling`
4. Run `grep -r "TODELETE(scaffold)" {service-dir}/` to find all cleanup points
5. Set up Docker Compose with `/aoh-compose` if not already done
```
