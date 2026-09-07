---
name: aoh-conventions
description: >
  Authoritative coding and design conventions for AOH (Agil Ops Hub) projects — Go backend
  code, SvelteKit frontend code, database schema design, HTTP API contracts, and
  OpenTelemetry observability.
  Use this skill when the developer is writing or editing Go code in an AOH service,
  writing or reviewing Go tests (unit, handler, or DB integration) for an AOH service,
  writing or editing SvelteKit code in an AOH frontend, designing or modifying database
  schemas / tables / columns / migrations, designing HTTP endpoints / response
  shapes / pagination behavior for an AOH service, or instrumenting a service with
  OpenTelemetry (spans, RED/runtime metrics, trace-correlated logs, service.name /
  resource conventions, the OTEL_* env contract).
allowed-tools: Read Grep Glob
---

# AOH Conventions

## When to read which reference

| Working on | Read |
|------------|------|
| **Project structure, monorepo layout, Docker Compose setup, linting/testing workflow, test-case design (EP/BVA), naming** | `references/project.md` |
| Go backend code: `apps/<svc>/internal/{handler,service,repo}/`, `cmd/server/`, `aohhttp`, `aohlog`, swag annotations, response envelopes, mockery | `references/go.md` |
| **Go tests** — writing or reviewing `*_test.go`: mockery vs hand-written fakes, handler tests without Keycloak, async publish assertions, `httptest` client fakes, seeder idempotency suites, build-tagged DB integration tests | `references/go-testing.md` |
| SvelteKit frontend code: `apps/<svc>/src/` — layout/sidebar navigation, routes, components, stores, Svelte 5 runes, gateway proxy, `@mssfoobar/ui` primitives, OIDC flow, Tailwind v4 | `references/web.md` |
| **Database schema** — naming, mandatory columns, association/view conventions (cross-runtime; currently Go services own DB schemas) | `references/database.md` |
| **HTTP wire contract** — success envelope shape, pagination params/response, liveness/readiness, web page routing namespaces (cross-runtime; backends produce, frontends consume) | `references/api.md` |
| **Error responses** — the error wire contract, `errorCode` vocabulary, class→status mapping, `trace_id`, log levels, BFF and frontend halves | the `aoh-error-handling` skill (not this one) |
| **Observability** — instrumenting a service with OpenTelemetry: spans, RED/runtime metrics, trace-correlated logs, `service.name`/resource conventions, the `OTEL_*` env contract, `team_id` baggage (cross-runtime; Go + SvelteKit) | `references/observability.md` |
| Cross-runtime feature (e.g. a backend endpoint + the frontend that consumes it) | Read the runtime files for both sides PLUS `api.md` |

## What this skill is NOT for

- **Service catalogue, OpenAPI specs, integration patterns** — use `aoh-knowledge` instead.
  That skill covers "what services exist on the AOH platform" and "how do they integrate."
  This skill covers "how do I write code in this repo."
- **Scaffolding new services** — use `aoh-go-init` (Go) or `aoh-web-init` (SvelteKit).
  This skill governs the conventions of code written *into* a scaffolded project.
