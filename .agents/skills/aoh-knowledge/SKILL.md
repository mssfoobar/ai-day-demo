---
name: aoh-knowledge
description: >
  Knowledge discovery for the AOH (Agil Ops Hub) platform — a modular framework for building
  Command and Control (C2) applications. Explains what services exist, how they work, their
  architecture and data models, API integration patterns, the canonical platform vocabulary
  (the published-language glossary), and recommends which services to use for a given use case.
  Use this skill when someone asks about AOH services, wants to understand the platform, needs
  integration guidance, asks what an AOH term means ("what is a geo-entity / BFF / active_tenant
  / replay session"), or wants recommendations for their use case.
allowed-tools: Read Grep Glob
---

# AOH Platform Knowledge Discovery

Answer questions about the AOH platform — what services exist, how they work, how to
integrate with them, and which ones to use for a given use case.

This skill is read-only and informational. It does not generate code or infrastructure
files. Point developers to the `aoh-compose` skill when they're ready to set up
services locally, `aoh-web-init` for SvelteKit frontends, or `aoh-go-init`
for Go backends.

## References

### Platform-level

| Reference | What it contains |
|-----------|-----------------|
| `references/glossary.md` | **AOH platform glossary (published language)** — the canonical term-for-term vocabulary consumers inherit (modules, `Provider`/`BFF`, `geo-entity`, `active_tenant`, response envelope, GIS/MSR/Dashboard/Form). Consumers **reference** it and define only their own domain terms + boundary mappings in their project's `UBIQUITOUS_LANGUAGE.md`. Use it to answer "what does `<AOH term>` mean". |
| `references/platform-overview.md` | Platform architecture, tech stack, service map, routing model |
| `references/service-catalogue.md` | All services at a glance, dependency graph, recommendation bundles |
| `references/service-selection-criteria.md` | **Decision keypoints for when an AOH module IS — and is NOT — the right call.** Default is in-app primitives; AOH modules earn their cost only when the use case requires a capability the service uniquely provides. Apply this BEFORE picking a service from the catalogue. |
| `references/integration-patterns.md` | Common patterns: auth flow, real-time events, file uploads, notifications, API calls |
| `references/keycloak-realm-guide.md` | How to extend the bundled `aoh` Keycloak realm: adding roles, seed users, project-specific OIDC clients |

> Code-writing conventions, swag annotation patterns, response envelopes,
> Svelte runes, AND what each scaffolding skill provides as primitives
> all live in the `aoh-conventions` skill — not here. Consult
> `aoh-conventions` for "how to write code" questions and for what to NOT
> re-implement (the scaffold delivers it). This skill stays focused on
> "what services exist on the platform and how to integrate with them."

### Per-service knowledge

Each service has a conceptual overview (`<service>.md`) and most have a formal API
spec (`<service>-openapi.json`, OpenAPI 2.0/3.0). Use the overview to understand
*what* the service does; use the OpenAPI spec to look up *exact* endpoints,
request/response schemas, and parameter names.

| Service | Overview | OpenAPI spec |
|---------|----------|--------------|
| Traefik (routing) | `references/services/infra.md` | — (no business API) |
| Identity & Access Management | `references/services/iams.md` | `references/services/iams-aas-openapi.json` (AAS), `references/services/iams-keycloak-openapi.json` (Keycloak admin) |
| Session Data Store | `references/services/sds.md` | `references/services/sds-openapi.json` |
| Real-time Update Service | `references/services/rtus.md` | `references/services/rtus-pms-openapi.json` (PMS control plane) |
| Geospatial Information System | `references/services/gis.md` | `references/services/gis-openapi.json` |
| Unified Notification Hub | `references/services/unh.md` | `references/services/unh-openapi.json` |
| In-App Notification | `references/services/ian.md` | `references/services/ian-openapi.json` |
| Attachment Management | `references/services/amm.md` | `references/services/amm-openapi.json` |
| Dashboard Service | `references/services/dash.md` | `references/services/dash-openapi.json` |
| Push Token Manager | `references/services/ptmgr.md` | `references/services/ptmgr-openapi.json` |
| Form | `references/services/form.md` | `references/services/form-openapi.json` |
| Workflow Engine | `references/services/wfe.md` | `references/services/wfe-openapi.json` |

Read only the references relevant to the developer's question. Start with the platform-level
references for broad questions, and drill into per-service files for specific questions.

### Reading OpenAPI specs efficiently

Some specs are large (unh ~110 KB, form ~150 KB). Don't read them
top-to-bottom. Use targeted lookup:

- **Find an endpoint by path or keyword** — `Grep` for the path or operation summary
  inside the `*-openapi.json` file (e.g. `grep -n '"/forms"' form-openapi.json`,
  `grep -n '"summary"' wfe-openapi.json`).
- **List all endpoints** — `grep -nE '"/[a-zA-Z]' <file>-openapi.json` gives a quick
  index of all paths.
- **Inspect a request/response schema** — once you have a `$ref` like
  `#/definitions/TaskRequestBody` or `#/components/schemas/Task`, `Grep` for that
  definition name.
- **Read a focused slice** — use `Read` with `offset`/`limit` once Grep tells you
  the line range. Avoid reading the whole file.

Only consult the OpenAPI spec when the developer needs precise API details
(endpoint path, HTTP method, exact request/response shape, status codes, query
params). For "what does this service do" questions, the `.md` overview is enough.

## How to respond

### When the developer asks "what is X?" or "how does X work?"

1. Read the relevant service `.md` reference(s)
2. Explain the service's purpose, architecture, and key concepts
3. Mention dependencies and related services
4. Provide API endpoint patterns if relevant — pull exact paths from the
   `*-openapi.json` spec only if precision matters

### When the developer asks "which service should I use for...?"

1. **Read `references/service-selection-criteria.md` FIRST** and apply the five decision questions. Often the right answer is "no AOH module — use Svelte form + Zod + an existing service endpoint" or "compose `@mssfoobar/ui` primitives in a fixed grid." Don't reach for an AOH module by reflex.
2. If an AOH module does earn its cost, read `references/service-catalogue.md` for the recommendation bundles
3. Map their use case to the relevant services
4. Explain why each recommended service fits and what *unique* capability it provides (per the criteria in step 1) — not a feature aggregation list
5. Show the dependency chain they'll need
6. Mention related skills for next steps (`aoh-compose`, `aoh-web-init`, `aoh-go-init`, plus per-service builder skills like `aoh-dashboard` if applicable)

### When the developer asks about integration

1. Read `references/integration-patterns.md` for the general pattern
2. Read the specific service `.md` reference for context (auth, dependencies, gotchas)
3. For exact API details (paths, methods, request/response schemas), `Grep` the
   service's `*-openapi.json` instead of guessing — never invent endpoint shapes
4. Explain the integration approach with endpoint patterns and auth requirements
5. Mention any gotchas or common pitfalls

### When the question touches application roles, role checks, or who-can-do-what

This is load-bearing and easy to get wrong by defaulting to OIDC priors.

1. **MUST read `references/services/iams.md`** — specifically the sections
   "Authorization is via AAS, not Keycloak", "Domain Model", "Authorization
   Models", and "Project-level AAS bootstrap (reproducibility pattern)".
   These rules are NOT duplicated at the platform-overview / service-catalogue
   / integration-patterns layer — `integration-patterns.md` only carries a
   pointer to them.
2. Distinguish realm-level (platform-bootstrap) roles from AAS tenant
   (application) roles before recommending where a new role is defined.
   Default to AAS tenant roles for anything application-specific.
3. Recommend the project-owned `compose/iams/init/project-aas/roles.yaml`
   (laid down + auto-wired by `aoh-compose`) for seeding new roles /
   groups / resources / scopes / assignments / permissions — not edits to
   `realm-import.json` and not ad-hoc cURL. Developers edit `roles.yaml`
   only; the surrounding `bootstrap.py`, `Dockerfile`, and compose service
   are generic and managed by the skill.
4. For request-time role checks, point the developer at the JWT
   `active_tenant.roles` claim (exposed in Go as `getTenantRoles(ctx)`), not
   a live AAS `/evaluate` call, unless they actually have a per-resource
   permission model.

### When the developer asks for an exact API shape

(e.g. "what fields does the create-task request take?", "what's the response of
GET /forms/{id}?", "which endpoints does WFE expose?")

1. Identify the service from the question — map to `<service>-openapi.json`
2. `Grep` the spec for the path, operation summary, or schema name
3. `Read` the matched line range with `offset`/`limit` — do not read the whole file
4. Quote the relevant path, method, and schema fields back to the developer
5. Note the auth requirement (almost always Bearer token via IAMS)

### General guidelines

- Be concise. Lead with the answer, add detail as needed.
- Use the dependency graph to explain what else comes along with a service.
- When multiple services could solve a problem, compare them briefly.
- Always mention auth requirements — most services need IAMS.
- Reference Traefik routing conventions (`<service>.${DEV_DOMAIN}`) when discussing access.
