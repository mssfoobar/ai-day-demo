# The activity ↔ catalog contract

A WFE activity exists in three places that must agree. This is the mapping.

1. **The Go method** on a registered `Activities` struct (what actually runs).
2. **The `service_activity` (or `service_event`) row** in the manager's Postgres
   (what the designer offers — `GET /v1/service_activity` / `GET /v1/service_event`,
   read-only via the API; populated by the seed SQL).
3. **The DSL `Activity` step** the designer emits (`Type` = the method name;
   arguments map positionally to the parameters).

This worker owns (1) and (3)'s contract; you seed (2). The engine dispatches on
the **method name**, so it is the join key across all three.

## `service_activity` columns

| Column | Meaning | Derived from the Go method |
|---|---|---|
| `service_name` | The worker that owns the activity (free-form label). | Your worker's name. |
| `activity_type` | **The join key.** | EXACTLY the exported Go method name (`HelloWorld`, …). Rename the method → change this. |
| `activity_icon` | Designer node icon. | Cosmetic — one of the `enum_activity_icon` ids (`service`, `script`, `none`, …). |
| `activity_param` | Ordered input descriptor (drives the designer's input fields). | One entry per `func` parameter **after `ctx`**, in signature order (see below). |
| `activity_result` | Example return value (types downstream variables). | An example of the method's return type (see below). |
| `timeout_in_second` | Start-to-close timeout (default 300). | Raise for long-running activities (waits/polls/large transfers can exceed 300s). |
| `heartbeat_timeout_in_second` | Heartbeat timeout (default 60). | The max gap between your `activity.RecordHeartbeat` calls before Temporal fails the activity — raise it if your activity can only heartbeat on a slower cadence. |
| `maximum_retry` | Max attempts **incl. the first** (default 10; → Temporal `RetryPolicy.MaximumAttempts`). | Lower to fail fast, raise for flaky work. `0` falls back to the default, not "unlimited". |

These three are the activity's **lifecycle policy** — set them per activity type
in the catalogue. The designer bakes them into the workflow at save time (the
workflow author can't override them); the engine applies them when scheduling the
activity. They are developer-owned, not per-workflow.

## `activity_param` — ordered, one entry per parameter

A JSON **array** of single-key `{name: sample}` objects, in the method's
parameter order (excluding `ctx context.Context`). The sample value's **type**
picks the designer's input widget; the value itself is ignored (fields default to
the type's zero value). The key is the field label.

| Go parameter type | `activity_param` entry | Designer widget |
|---|---|---|
| `string` | `{"endpoint": ""}` | text |
| `float64` (number) | `{"seconds": 0}` | number |
| `bool` | `{"enabled": false}` | true/false |
| `any` / object / array / `map[string]…` | `{"payload": {}}` | JSON editor |

Example — a four-parameter activity `Call(ctx, endpoint string, method string, payload any, headers map[string]string)`:

```json
[{"endpoint": ""}, {"method": ""}, {"payload": {}}, {"headers": {}}]
```

A no-argument activity (`func (a *Activities) Ping(ctx) error`) uses an empty array: `[]`.

## `activity_result` — a type-accurate example

A single JSON value whose **type** mirrors the method's first return value
(the second is always `error`). It types the workflow variable that captures the
step's result.

| Go return | `activity_result` |
|---|---|
| `(string, error)` | `""` |
| `(float64, error)` | `0` |
| `(bool, error)` | `false` |
| `(SomeStruct, error)` / `(any, error)` object | `{...}` (an example object, e.g. `{"status": 200, "headers": {}, "body": {}}`) |
| `error` only (no value) | `null` |

## Events use the same contract

A BPMN **event** is the same thing one layer over: a worker method dispatched by
name (the engine runs it as an activity), but listed in **`service_event`**
instead of `service_activity`. The columns mirror the activity ones —
`event_type` (= the Go method name), `event_param`, `event_result` — with one
swap: `event_icon` is an `enum_event_icon` id (`message1`, `timer1`, `signal1`,
`error1`, …) and the designer draws it as a BPMN event node, not an activity box.
Events have **no lifecycle columns** (no `timeout_in_second` /
`heartbeat_timeout_in_second` / `maximum_retry`): the engine runs them with a
fixed policy — a 10-year start-to-close timeout, a 1-minute heartbeat timeout,
`WaitForCancellation`, and a **single attempt (no retry)**. An event is one
long-running wait, so on failure it propagates to the workflow rather than
retrying. A long event (like the shipped `Timer`) MUST heartbeat (≤ 1 min) and
return on `ctx.Done()` to stay alive and to be interruptible.

## The shipped rows

The scaffold seeds exactly two — one activity and one event:

| Catalog | Method | `*_param` | `*_result` | icon |
|---|---|---|---|---|
| `service_activity` | `HelloWorld(ctx, name string) (string, error)` | `[{"name": ""}]` | `""` | `script` |
| `service_event` | `Timer(ctx, seconds float64) error` | `[{"seconds": 0}]` | `null` | `timer1` |

Those are the rows in `assets/.../sql/seed_catalog.sql`. For other
parameter/return types, combine the rules above — e.g.:

| Method | `*_param` | `*_result` |
|---|---|---|
| `Echo(ctx, n float64) (float64, error)` | `[{"n": 0}]` | `0` |
| `Toggle(ctx, on bool) (bool, error)` | `[{"on": false}]` | `false` |
| `Transform(ctx, doc any) (any, error)` | `[{"doc": {}}]` | `{}` |
| `Notify(ctx) error` | `[]` | `null` |

When you add or change a method, add or edit its catalog row to match — then
re-run the seed (idempotent on the unique type).
