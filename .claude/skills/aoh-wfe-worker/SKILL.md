---
name: aoh-wfe-worker
description: >
  Scaffold and implement a WFE Temporal **activity worker** — the user-owned Go
  service that runs the activities a WFE workflow schedules. Generates a
  standalone worker (a HelloWorld example activity, health endpoints, Dockerfile,
  lint/test) and guides implementing your own activities,
  the two engine contracts (task queue + activity method names), and registering
  them in the manager's `service_activity` catalog. Use this skill whenever a user
  wants to build a WFE worker, add or implement a workflow activity, scaffold a
  worker, or run designer-authored workflows end-to-end. Trigger phrases: "WFE
  worker", "activity worker", "implement a workflow activity", "register an
  activity", "my activities won't run", "seed service_activity". For the
  designer/monitor UI use `aoh-wfe-designer`; for what WFE is, use
  `aoh-knowledge`.
license: Proprietary
metadata:
  owner: AOH Platform Team
  status: v1
---

# AOH WFE Worker

Scaffold and implement a **WFE activity worker**: the Temporal *activity* worker
that executes the activities a WFE workflow schedules. It is the user-owned
counterpart to the platform's two backend binaries — the **workflow-engine** runs
the DSL and schedules activities onto a Temporal task queue; **this worker**
consumes that queue and runs them. The engine and the worker are decoupled purely
by the task queue, so you own and deploy your activities independently.

This skill is a **scaffolder**: it ships a complete worker template as its own
`assets/`, and generates a standalone worker into your project from them. The
output is a self-contained Go module — `github.com/mssfoobar/ops-hub/packages/aoh-golib` is a
normal registry dependency (no local `replace`), and the Dockerfile is a
single-module build. It ships with a single `HelloWorld` example activity (plus
health endpoints, lint, and a passing test) — so it builds and runs end-to-end
before you write your own.

The engine and manager the worker talks to are the deployed WFE backend
(published as `ghcr.io/mssfoobar/ops-hub/wfe-engine` and
`ghcr.io/mssfoobar/ops-hub/wfe-manager`) — you do **not** build or vendor them
here; your worker only needs a reachable Temporal cluster plus the manager's
Postgres to seed the catalog.

## Where knowledge lives

- **Generic AOH Go conventions** (Viper config, `aoh-golib` logger/health/temporal
  client, the shipped `.golangci.yml`, layout) → the **`aoh-conventions`** skill's
  `references/go.md`. This skill does not restate them.
- **The WFE engine + manager backend** (the DSL, the REST API, the
  `service_activity` table that this worker seeds) → the deployed WFE backend
  (`ghcr.io/mssfoobar/ops-hub/wfe-{engine,manager}`); concepts in
  **`aoh-knowledge`** (`references/services/wfe.md`).
- **The designer / monitor UI** that authors the workflows this worker runs →
  the **`aoh-wfe-designer`** skill.
- **Bringing up Temporal + the WFE backend locally** → the **`aoh-compose`** skill.
- **The activity ↔ catalog contract** (param/result shapes) →
  `references/activity-contract.md`.

## Scaffold the worker

```bash
python3 .claude/skills/aoh-wfe-worker/scripts/scaffold.py \
    --name <worker-name> \
    --module <go-module-path> \
    --repo-root <repo-root> \
    [--target-dir <relative-path>] \
    [--description "<one-line description>"]
```

Lands at `apps/<name>/` by default (pass `--target-dir` for another path, e.g.
`services/<name>`). Idempotent — re-running skips files already in place. It
only touches an existing `go.work` / `turbo.json`; it never forces a workspace on
a standalone repo. Then:

```bash
cd <target>
go mod tidy                          # resolve deps + write go.sum
cp worker.config.yaml config.yaml    # set temporal.task_queue to the engine's
make test                            # the activity tests pass out of the box
make run                             # build + run with debug logging
```

`go mod tidy` fetches `github.com/mssfoobar/ops-hub/packages/aoh-golib` from the
private ops-hub repo — set `GONOSUMDB=github.com/mssfoobar/*` with normal GitHub
git credentials, or use a spoke registry mirror (`GOPROXY=<mirror> GOSUMDB=off`).
(Avoid `GOPRIVATE` if you rely on a mirror — it implies `GONOPROXY` and bypasses it.)

## The two contracts

A worker is correct only when it agrees with the engine on two things. Both are
load-bearing — get either wrong and workflows silently stall.

1. **Task queue.** `temporal.task_queue` (config / `TEMPORAL_TASK_QUEUE` env) MUST
   equal the WFE engine's, default **`wfe`**. A mismatch means the engine
   schedules activities onto a queue this worker never polls — they sit pending
   forever, with no error.
2. **Activity name = Go method name.** Every exported method on a registered
   `Activities` struct becomes a Temporal activity addressable by its **method
   name**. That exact string is what a DSL `Activity` step puts in its `Type`, and
   what the manager's `service_activity` (or `service_event`) registry advertises
   to the designer. **Renaming a method is a contract change** — update the seed
   (`sql/seed_catalog.sql`) and any DSL referencing it.

## Implement activities

Add methods to an `Activities` struct (or a new package) and register the struct
in `cmd/worker/main.go` (`w.RegisterActivity(&yourpkg.Activities{})`). Rules:

- **Activities only.** The worker runs with `DisableWorkflowWorker` — the mirror
  of the engine's `LocalActivityWorkerOnly`. Never register workflows here.
- **Signature.** `func (a *Activities) Name(ctx context.Context, ...) (T, error)`.
  Parameters map positionally from the DSL step's arguments; Temporal decodes JSON
  objects/arrays into `any`/structs (no manual string parsing). The return type
  informs how downstream workflow variables are typed.
- **Keep long activities cancellable.** Anything that can run > ~1 minute must
  `activity.RecordHeartbeat(ctx, ...)` periodically and return on `ctx.Done()`, so
  a workflow can cancel it mid-flight. The shipped `Timer` event is the reference.
- **Failing without retries:** return `temporal.NewNonRetryableApplicationError(...)`.
- **Events are the same Go.** A BPMN *event* (the shipped `Timer`) is an ordinary
  worker method too — the engine dispatches it as an activity by name. The only
  difference is the catalog: list it in `service_event` (not `service_activity`)
  and the designer renders it as an event node. `Timer` also demonstrates the
  long-running path: the engine runs events with a 1-minute heartbeat timeout,
  `WaitForCancellation`, and a single attempt (no retry), so it heartbeats each
  tick and returns on `ctx.Done()` — which is what lets an *interrupting* event
  actually interrupt. Events have no lifecycle catalogue columns; only activities
  carry `timeout_in_second` / `heartbeat_timeout_in_second` / `maximum_retry`.

## Register activities + events in the designer

The designer only offers what the **manager's** catalog advertises — activity
types from `service_activity` (`GET /v1/service_activity`) and event types from
`service_event` (`GET /v1/service_event`), both read-only. This worker has no
database, so seed the catalog against the **workflow-manager's** Postgres:

```bash
psql "$WFE_MANAGER_DSN" -f sql/seed_catalog.sql
```

Idempotent (keyed on the unique `activity_type` / `event_type`; re-run to
re-sync). Keep each row's type equal to your Go method name, and update `*_param`
(ordered, one entry per parameter) / `*_result` (a type-accurate example)
whenever you add, rename, or change a method. The exact shapes are in
`references/activity-contract.md`.

## Verify

```bash
make test          # go test ./... (Temporal's activity test env; HelloWorld test included)
make lint-code     # golangci-lint with the AOH conventions
make build         # compiles the worker binary
```

Container check (optional): `go mod tidy` first so `go.sum` exists, then
`podman build --secret id=access_token,src=.secrets/access_token -f worker.Dockerfile .`
(file-sourced secret authenticates the private aoh-golib fetch — works in podman
and docker; spoke users pass `--build-arg GOPROXY=<mirror> --build-arg GOSUMDB=off`
instead, no token) — a multi-arch, non-root (`wfe` uid 1000) image. End-to-end: with the engine + a reachable Temporal up and the catalog
seeded, a designer workflow that schedules one of your activities runs to
completion.

## What this skill does NOT do

- **Author workflows or run the DSL.** That's the WFE **workflow-engine** (the
  deployed `wfe-engine`) — never register workflows in this worker.
- **Provide the REST API / manage templates.** That's the **workflow-manager**.
- **Build the designer/monitor UI.** Use **`aoh-wfe-designer`**.
- **Stand up Temporal / the WFE backend / Postgres.** Use **`aoh-compose`**.
- **Restate generic Go service conventions.** Use **`aoh-conventions`**.

## References

- `references/activity-contract.md` — the `service_activity` row ↔ Go method
  signature mapping (`activity_param` / `activity_result` shapes, icons, timeout).
- The scaffolded worker itself — its generated `README.md` and `AGENTS.md`
  (both produced from this skill's `assets/`) document the day-to-day workflow
  once the worker is in your project.
- Concepts + API for the WFE engine/manager backend: the `aoh-knowledge` skill →
  `references/services/wfe.md`.
