---
name: aoh-module-init
description: >
  Scaffold a new AOH feature module inside the ops-hub platform-monorepo —
  creates modules/<name>/{web,client,types,service,deploy}/ with the full
  vertically-sliced package set (Svelte SDK, typed REST client, TS types,
  Go service, deploy snippet). Use this skill whenever the user wants to
  scaffold a new ops-hub module, add an AOH module, start a new
  vertically-sliced feature in ops-hub, or bootstrap a module under
  modules/<name>/. Also trigger when the user says "new ops-hub module",
  "scaffold a module", "add an AOH module", or mentions modules/<name>/
  in a scaffolding context.
---

# AOH Module Init

Scaffold a complete AOH feature module inside the **ops-hub
platform-monorepo** (`mssfoobar/ops-hub`). A Python script generates the
sibling packages (`web`, `client`, `types`, `deploy`) from templates,
then delegates the Go service scaffold to `aoh-go-init` via
`--target-dir`.

The result is a working `modules/<name>/{web,client,types,service,deploy}/`
tree that passes `pnpm install`, `turbo check`, and (for the service)
`make build`, `go test ./...` out of the box.

For the architectural context — what each sub-package does, the
one-way dependency chain (`web → client → types → service`), and how the
module fits into the ops-hub shape — see
`aoh-conventions/references/project.md → "Ops-hub platform-monorepo shape"`.

## When this skill applies

* You're working in `mssfoobar/ops-hub` (verify: `modules/` directory
  exists, `pnpm-workspace.yaml` globs `modules/*/*`).
* You want to add a new feature module — full vertical slice, not just
  a backend service.

If the user only needs a Go service (no SDK / client / types), use
`aoh-go-init` directly with `--target-dir modules/<name>/service`. The
sibling packages are the value-add of this skill.

If the user is in a **consumer-app** monorepo (no `modules/` directory),
this skill does not apply — fall back to `aoh-go-init` for the backend
and treat the frontend as a regular SvelteKit app.

## Gather inputs

Ask the user for (if not already provided in the prompt):

1. **Module name** (required) — kebab-case, e.g. `inventory`,
   `asset-tracking`. Used as both the directory name (`modules/<name>/`)
   and the published-package suffix (`@mssfoobar/<name>-web-sdk`,
   `@mssfoobar/<name>-client`, etc.).
2. **Description** (required) — one-liner used in every sub-package's
   `package.json` description + README header. Keep it concise (under
   100 chars).
3. **Go module path** (optional) — defaults to
   `github.com/mssfoobar/ops-hub/modules/<name>/service`. Only override
   if you have a non-standard repo URL.

## Steps

### Step 1. Run the scaffold script

```bash
python3 .claude/skills/aoh-module-init/scripts/scaffold.py \
    --name <module-name> \
    --description "<one-line module description>" \
    --repo-root <path-to-ops-hub>
```

Example:

```bash
python3 .claude/skills/aoh-module-init/scripts/scaffold.py \
    --name inventory \
    --description "Inventory tracking for AOH assets" \
    --repo-root /Users/me/work/ops-hub
```

The script:

1. **Preflight-validates** the repo (must look like ops-hub: has
   `modules/`, `pnpm-workspace.yaml` with `modules/*/*` globs,
   `go.work`). Refuses to overwrite an existing `modules/<name>/`.
2. **Copies the sibling-package templates** from `assets/` into
   `modules/<name>/`, substituting `__AOH_MODULE_NAME__`,
   `__AOH_MODULE_DESCRIPTION__`, and `__AOH_PASCAL_NAME__` placeholders.
3. **Delegates the Go service scaffold** to `aoh-go-init`'s
   `scripts/scaffold.py` with
   `--target-dir modules/<name>/service`.

   Note: the delegated scaffold emits a **standalone, network-fetching**
   Dockerfile — aoh-golib is fetched from the private ops-hub repo at its
   `packages/aoh-golib/v*` tag (build context = the service dir, token or
   spoke-mirror auth). The existing ops-hub services instead use a repo-root
   build context + a go.mod `replace` (token-free in CI). If the new module
   should match its siblings' deploy workflows, adapt the Dockerfile and
   go.mod from any `modules/*/service` after scaffolding.

If the user only needs the sibling packages (rare — e.g., the service
already exists), pass `--skip-service`.

### Step 2. Install + check

```bash
cd <repo-root>
pnpm install                                       # picks up new packages via existing globs
pnpm exec turbo run check --filter=./modules/<name>/*
```

All three TS packages (`types`, `client`, `web`) should typecheck clean.
The SDK runs `svelte-kit sync` first (via its `prepare` script during
install).

### Step 3. Verify the service

```bash
cd modules/<name>/service
make build
make lint-code
go test ./...
```

This is the same verification `aoh-go-init` performs on a fresh
scaffold — the example entity files (`TODELETE(scaffold)`-annotated)
are kept until the user adds real entities.

### Step 4. Hand off to the user

The scaffold script prints a "Next steps" block. The headline:

* Fill in DTOs in `modules/<name>/service/internal/dto/*.go`.
* Mirror them in `modules/<name>/types/src/index.ts`.
* Add resource APIs to `modules/<name>/client/src/index.ts`.
* Add Svelte components under `modules/<name>/web/sdk/components/`.

The AGENTS.md files (`modules/<name>/AGENTS.md` plus the `web/` and
`service/` sub-package AGENTS.md files, each with a one-line `@AGENTS.md`
CLAUDE.md import) document the invariants — wire-contract lockstep,
one-way dependency chain, peer-dep declarations.

## Output

After all steps complete, display a summary:

```
## Module Scaffolded: <module-name>

**Path:** modules/<module-name>/
**Packages:**
- @mssfoobar/<name>-types
- @mssfoobar/<name>-client
- @mssfoobar/<name>-web-sdk
- <name>-service (Go)
**Deploy:** modules/<name>/deploy/compose.snippet.yml

### What's working
- pnpm install picks up all four packages via existing workspace globs.
- turbo check passes (tsc clean for TS packages, svelte-check clean for the SDK).
- The Go service builds, lints, and tests pass (via aoh-go-init).

### Next steps
1. Fill in wire DTOs (service/internal/dto/*.go) and TS mirrors (types/src/index.ts) in lockstep.
2. Add resource APIs to client/src/index.ts; test via vitest mock fetch.
3. Build SDK components in web/sdk/components/, exporting them via package.json `exports`.
4. Wire deploy/compose.snippet.yml into the hub stack as needed.
```

## Design notes

**Why a Python script and not instruction-driven?** Same rationale as
`aoh-go-init`: deterministic re-runnability, refuse-to-overwrite
preflight checks, and ~22 file copies + placeholder substitutions are
mechanical work that's easier to verify in one place than across many
agent tool calls. The script is ~300 LOC of straightforward `pathlib` +
`shutil` — no clever generation logic, no JSON surgery, no toolchain
detection.

**Why inline the SDK scaffolding instead of a separate
`aoh-svelte-sdk-init` skill?** Several modules now ship a `web/` Svelte
SDK on this shape (`gis`, `msr`, `unh`, `iams`, `wfe`, published as
`@mssfoobar/<name>-web-sdk`), so the layout is settled and the templates
track it. Extracting a standalone `aoh-svelte-sdk-init` skill is a
deferred call — the inlined templates stay until that extraction is
taken on as its own change.

**Why a `0.0.0` version on the sibling packages?** They start as
placeholder shells — `description`, `exports`, etc. are wired but
there's nothing to consume yet. The first real release per package
bumps to `0.1.0` via a changeset; see
`aoh-monorepo-release/references/example-modules-gis-sdk.md` for the
worked release flow on this monorepo shape.
