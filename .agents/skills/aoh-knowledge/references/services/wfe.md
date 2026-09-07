# WFE — Workflow Engine

## What It Does

WFE provides BPMN 2.0 workflow orchestration for the AOH platform. It lets you design
business processes visually, execute them as durable workflows, and monitor their progress.
Built on [Temporal](https://temporal.io) for reliability and scalability.

Use WFE when you need to automate multi-step business processes, coordinate activities
across services, or implement eSOP (electronic Standard Operating Procedures).

## Architecture

WFE ships as two Go binaries (one Go module, separate entrypoints) plus an
activity worker. The designer/monitor UI is an embeddable SDK a host app mounts,
not a standalone service.

```
┌──────────────────┐  REST /v1   ┌──────────────┐  Temporal   ┌──────────────┐
│ Host app +       │────────────►│ wfe-manager  │────────────►│ wfe-engine   │
│ wfe-web-sdk UI       │             │ (REST API)   │             │ (DSL worker) │
│ designer/monitor │             └──────────────┘             └──────┬───────┘
└──────────────────┘                                                 │ schedules
                                                              ┌──────▼───────┐
                                                              │ wfe-worker   │
                                                              │ (activities  │
                                                              │  — yours/    │
                                                              │   sample)    │
                                                              └──────────────┘
```

## Components

| Component | Image / package | Purpose |
|-----------|-----------------|---------|
| **workflow-manager** (WFM) | `ghcr.io/mssfoobar/ops-hub/wfe-manager` | REST API (`/v1`) for templates + execution; owns the Postgres schema (golang-migrate on boot); Temporal client |
| **workflow-engine** (WFE) | `ghcr.io/mssfoobar/ops-hub/wfe-engine` | Temporal worker running the DSL interpreter (the four contract workflow types) |
| **activity worker** (WFW) | `ghcr.io/mssfoobar/ops-hub/wfe-worker` (sample) | Executes the activities a workflow schedules. The published image is a DEMO worker built from `modules/wfe/worker-template`; real deployments run a consumer-owned worker from a copy of that template |
| **designer + monitor** (WFD) | `@mssfoobar/wfe-web-sdk` | Embeddable SvelteKit SDK (BPMN-style designer + run monitor) a host app mounts — not a standalone service. See `modules/wfe/web` + the reference-host showcase |

## Key Concepts

### Domain Model

- **Workflow Template** — A BPMN process definition. Has two representations:
  - `workflow_json` — Execution schema (DSL used by WFE to run the workflow)
  - `designer_json` — JointJS+ BPMN JSON for rendering the visual diagram
- **Workflow** — A running instance of a workflow template, identified by `workflow_id`
- **Activity** — A single unit of work executed by a worker (Go function registered in Temporal)

### Template Lifecycle

Templates have two states controlled by the `editable` field:
- `editable: true` — Draft (saved but not published, can still be edited)
- `editable: false` — Published (immutable, safe to execute)

Use `PUT /v1/workflow_template/save` to save a draft, then
`PUT /v1/workflow_template/publish` to publish it for execution.

### Activities

Activities are Go functions registered in the Temporal server via a worker service (WFW or
your own custom worker). Workflow Designer only shows activities that are registered in
the `service_activity` database table — you must insert rows there for each activity type.

Activity registration fields:
- `service_name` — container name of the worker service
- `activity_type` — Go function name
- `activity_icon` — icon identifier for designer UI
- `activity_param` — ordered list of parameter name→type pairs (must match function args)
- `activity_result` — return type: `{"object": {}}`, `"string"`, `true/false`, `0`, or `null`
- `timeout_in_second` — start-to-close timeout (default 300)
- `heartbeat_timeout_in_second` — heartbeat timeout (default 60)
- `maximum_retry` — max attempts incl. the first (default 10; Temporal RetryPolicy MaximumAttempts)

These three lifecycle fields are developer-owned: the designer bakes them into the
workflow at save and the engine applies them (events take none — they run a fixed
single-attempt policy).

### Workflow Variables and Expressions

Workflow templates carry `variables` (key-value pairs). Values support expr-lang
expressions (the [`expr-lang/expr`](https://expr-lang.org) language, prefixed with
`=`) that reference results from previous activities:

```
Activity_String: '=Result_PreviousActivity.someKey'
```

Use `POST /v1/validate/expression` and `POST /v1/validate/condition` to validate
expressions before embedding them in templates.

### Workflow Metadata

Metadata is a key-value map passed when starting a workflow (`POST /v1/workflow`).
Inside activities, retrieve it via `temporal.ContextValue(ctx)` from `aoh-golib/temporal`:

```go
import "github.com/mssfoobar/ops-hub/packages/aoh-golib/temporal"
metadata := temporal.ContextValue(ctx)
value := metadata["trigger_rule_id"]
```

### Forms (human-in-the-loop steps)

Forms are **delegated to the AOH Form module** (SurveyJS) — WFE stores no form
schema and exposes no `form_template` endpoints. A `FormTask` step references a
Form-module form and
carries a pre-fill `{key: value}` map; the host frontend renders the SurveyJS
form and, on submit, signals the workflow to resume via
`POST /v1/workflow/{workflow_id}/activity_name/{activity_name}`. WFE's backend
has no runtime Form coupling.

### Signalling Workflows

The signal endpoint sends data to a paused workflow (e.g., waiting for form input or a
recover task). The `activity_name` is the name of the activity node in the workflow graph
that is waiting.

### Custom Activity Development

1. Copy `modules/wfe/worker-template` out of the monorepo (or scaffold one with
   the `aoh-wfe-worker` skill) — the canonical activity-worker scaffold. When
   copied out, drop the local `aoh-golib` `replace` from its `go.mod` and depend
   on `aoh-golib` normally
2. Implement your activities and register them with the worker
3. For long-running activities (>1 minute), call `activity.RecordHeartbeat(ctx, ...)`
   at regular intervals and check `ctx.Done()` for cancellation
4. Build a Docker image (`worker-template/worker.Dockerfile`) and deploy it as
   an activity worker on the same `TEMPORAL_TASK_QUEUE` as the engine
5. Register each activity in the manager's `service_activity` catalogue (the
   read-only `GET /v1/service_activity` the designer reads) — see
   `worker-template/sql/seed_service_activities.sql` for the row shape

### Temporal Admin UI

The Temporal web UI is accessible at `http://wf-admin.${DEV_DOMAIN}` for monitoring
workflow executions, debugging activity failures, and viewing event histories.

## API Endpoints (WFM)

All **business** endpoints are under `/v1/` and require bearer-token auth. The
health probes (`/livez`, `/readyz`) sit at the **root** and are unauthenticated.

### Workflow Templates

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/v1/workflow_template` | List templates (paginated) |
| `GET` | `/v1/workflow_template/{template_id}` | Get template |
| `PUT` | `/v1/workflow_template/save` | Save draft template (editable: true) |
| `PUT` | `/v1/workflow_template/publish` | Publish template (editable: false, immutable) |
| `DELETE` | `/v1/workflow_template/{template_id}` | Delete template |

### Activity & Event Registry (read-only)

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/v1/service_activity` | List registered activity types (for designer) |
| `GET` | `/v1/service_event` | List registered event types (for designer) |

### Expression Validation

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `POST` | `/v1/validate/expression` | Validate an expr-lang expression against variables |
| `POST` | `/v1/validate/condition` | Validate an expr-lang condition against variables |

### Workflow Execution

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `POST` | `/v1/workflow` | Start a workflow from a template |
| `GET` | `/v1/workflow` | List the caller's tenant's executions (paginated); `?status=` filters (`running` default, `closed`, `completed`, `failed`, `canceled`, `terminated`, `continued_as_new`, `timed_out`, `all`) |
| `GET` | `/v1/workflow/{workflow_id}` | Get workflow event history (paginated) |
| `DELETE` | `/v1/workflow/{workflow_id}` | Terminate a workflow |
| `POST` | `/v1/workflow/{workflow_id}/activity_name/{activity_name}` | Signal a paused workflow activity |

Execution endpoints are tenant-scoped (the caller's `active_tenant`): list returns
only the tenant's runs, and history/terminate/signal 404 on another tenant's
workflow. This relies on Temporal **Advanced Visibility** (Postgres 12+/MySQL
8.0.17+ or Elasticsearch) — see `modules/wfe/deploy/README.md`.

### Health

| Method | Endpoint | Purpose |
|--------|----------|---------|
| `GET` | `/livez` | Liveness check (root, unauthenticated) |
| `GET` | `/readyz` | Readiness check (root, unauthenticated) |

## Data Models

```
WorkflowTemplate:
  id: UUID
  name: string
  workflow_json:            # Execution DSL
    specVersion: "2.0"
    start: string           # Name of first state
    states: array           # Activity nodes
    variables: object       # Key-value input variables
  designer_json:            # JointJS+ BPMN for visual rendering
    type: "bpmn"
    cells: array            # BPMN nodes and edges
    name: string
  editable: boolean         # true = draft, false = published
  occ_lock: integer
  created_at, updated_at: timestamp

WorkflowExecution:
  workflow_id: UUID
  start_time: timestamp

WorkflowHistoryEvent:
  event_type: string        # WorkflowExecutionStarted, ActivityTaskCompleted, etc.
  timestamp: timestamp
  task_name: string         # activity node name
  task_type: string         # activity function type
  attributes: object        # input/result/failure depending on event_type
```

### Workflow Event Types

| Event | Description |
|-------|-------------|
| `WorkflowExecutionStarted` | Workflow instance created |
| `WorkflowExecutionCompleted` | All activities finished successfully |
| `WorkflowExecutionFailed` | Workflow failed |
| `WorkflowExecutionTerminated` | Workflow manually terminated |
| `ActivityTaskScheduled` | Activity queued for execution |
| `ActivityTaskStarted` | Activity began executing |
| `ActivityTaskCompleted` | Activity finished successfully |
| `ActivityTaskFailed` | Activity failed |
| `ActivityTaskCanceled` | Activity was cancelled |
| `FormTaskScheduled/Started/Completed/Failed` | Form activity lifecycle |
| `RecoverTaskScheduled/Started/Completed/Failed` | Recovery task lifecycle |

## Access

| URL | What |
|-----|------|
| `http://wfm.${DEV_DOMAIN}` | workflow-manager REST API |
| `http://wfm.${DEV_DOMAIN}/swagger-ui/index.html` | Swagger UI |
| `http://wf-admin.${DEV_DOMAIN}` | Temporal admin UI |

The designer + monitor are an embeddable SDK (`@mssfoobar/wfe-web-sdk`) a host app
mounts (e.g. the reference-host showcase), which calls the manager above.

## Dependencies

- **iams** (Keycloak for auth)
- **form** (the AOH Form module) — for human-in-the-loop **Form-task** steps: the
  designer's Form-step picker and a running workflow's form rendering delegate to
  it (the host app injects the Form clients). The WFE backend has no runtime Form
  coupling, so plain workflows don't need it — but any stack using Form tasks does.
