# AGENTS.md

Agent context for this Go microservice. App-specific architecture lives
here; monorepo-wide conventions live in the repo-root `AGENTS.md`.

- For project overview, layout, and how to run things locally, see `README.md`.
- For coding conventions AND what this scaffold provides as primitives (so you
  don't re-implement them), the `aoh-conventions` skill auto-activates on AOH Go
  code work; consult its `references/go.md` if not loaded.
- For error responses and logging — the wire contract, the `errorCode`
  vocabulary, status mapping, `trace_id`, log levels — consult the
  `aoh-error-handling` skill.

## Architecture (load-bearing)

Layered Go microservice. Dependencies flow one way: **handler → service → repo**.
Never call repo from handler, never inject service into another service. Enforced
by code review, not by tooling — but it's the rule everything else is built on.

Failures follow the same one-way flow. The service layer classifies them
(`internal/service/errors.go` — an `aoherr` class plus a namespaced code) and
returns them; the handler renders them with `aoherr.Render`, which derives the
HTTP status, stamps the `trace_id`, suppresses internal detail on 5xx, and emits
the single log record. **No layer below the handler logs a failure it returns**,
and no handler picks a status or sanitizes by hand.
