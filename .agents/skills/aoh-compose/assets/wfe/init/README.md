# OPTIONAL WFE catalog seed (`wfe-init`)

Registers a worker's **activities** (`service_activity`) and **events**
(`service_event`) with the WFE workflow-manager so the designer offers them,
from a declarative `catalog.yaml`.

## When you need this

Only when the catalog must be populated **before first use**, reproducibly on a
fresh `down -v && up`. If you seed by hand or your worker self-seeds on deploy,
you do **not** need this — leave the `wfe-init` block in `../compose.yml`
commented out (and you may delete this `./init` dir).

## Editing

`catalog.yaml` ships with an **example** — replace it with **your** worker's
activities/events. Each `type` MUST equal an exported Go method name your worker
registers; `param` / `result` shapes are documented in the `aoh-wfe-worker` skill.
Re-running is idempotent (keyed on the unique type), so re-runs and fresh stacks
converge.

## How it runs

The `wfe-init` one-shot ships commented out in `../compose.yml`. Uncomment it to
enable; it runs on every `compose up`. It waits for the manager to migrate the
schema, then upserts each row in `catalog.yaml` into the manager's database. It
seeds the **manager's** database (the worker has none) and does **not** require
the worker to be running.
