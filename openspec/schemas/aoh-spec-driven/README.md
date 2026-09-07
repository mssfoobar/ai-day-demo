# aoh-spec-driven

OpenSpec schema for proposing, designing, and implementing changes on the **AOH
(AGIL Ops Hub) platform**. It enforces the AOH conventions that ad-hoc
spec-driven workflows tend to miss — IAMS-AAS authorization, the canonical
response envelope, `@mssfoobar/ui` primitives, the compose stack, SDS-backed
sessions, and the scaffolding skills.

The schema is referenced by `openspec/config.yaml`:

```yaml
schema: aoh-spec-driven
```

## Workflow

```
proposal ──► specs ──┐
        └──► design ─┤
                     ├──► tasks ──► lint ──► apply
                     │            (gate)
                     │
                     └─ tasks pulls from specs + design
                        lint reads ALL prior artifacts
```

Six artifacts, four of which describe the change and one (`lint`) that gates
implementation:

| Stage | Generates | Purpose |
|---|---|---|
| `proposal` | `proposal.md` | WHY this change is needed — Why, What Changes, Capabilities, Impact. 1–2 pages. |
| `specs` | `specs/<capability>/spec.md` | WHAT the system SHALL do — requirements + scenarios in WHEN/THEN form. One file per capability. |
| `design` | `design.md` | HOW to build it — Runtime dependencies, API surface, UI design system, Decisions, Risks, Open Questions. |
| `tasks` | `tasks.md` | Implementation checklist — one section per app touched, plus Scaffold / Compose / Seed / E2E + reproducibility gate. |
| `lint` | `lint.md` | Mechanical pre-apply check over every prior artifact. 30+ checks, 10 failure families. |
| `apply` | _(operates on `tasks.md`)_ | Walks the task list, marking complete as it goes. Requires `lint`. |

The lint is the **apply-readiness gate**. `apply.requires: lint`, so the
implementing agent never starts against broken assumptions.

## Commands

User-invoked slash commands that drive the workflow:

| Command | What it does |
|---|---|
| `/opsx:propose <name>` | Walks the user through proposal → specs → design → tasks for a brand-new change. |
| `/opsx:explore` | Thinking-partner mode before committing to a proposal. Use when the change is fuzzy. |
| `/opsx:apply [name]` | Implements a change. Requires `lint.md` present and all checks PASS or N/A. Reads `tasks.md` and walks through the checklist. |
| `/opsx:archive [name]` | Archives a completed change once all tasks are done. |

`/opsx:apply` is the only command that produces code. The other three produce
artifacts under `openspec/changes/<change>/`.

## Files in this directory

```
.
├── schema.yaml        # Artifact graph + per-artifact instruction blocks.
├── templates/
│   ├── proposal.md    # What sections a proposal must have.
│   ├── spec.md        # Requirement + Scenario format.
│   ├── design.md      # Sections: Context, Runtime deps, API surface, UI, Decisions, …
│   ├── tasks.md       # Section archetypes: Scaffold / Compose / Seed / Implement / E2E + gate.
│   └── lint.md        # 10 sections of mechanical checks (this is the lint output template).
└── README.md          # This file.
```

`schema.yaml` is the source of truth. Templates are the structural shells the
implementing agent fills in; the per-artifact `instruction` blocks in
`schema.yaml` are the prose that tells the agent HOW to fill them.

## Artifact details

Each artifact's per-section structure + forced defaults are documented in
`schema.yaml` under the matching `instruction:` block. Mechanical
cross-checks live in the `lint` artifact (`templates/lint.md`) — when a rule
matters enough to enforce, it appears as a lint check, not just prose. The
short version of each artifact:

| Artifact | What it is | Where the agent reads the rules |
|---|---|---|
| `proposal.md` | WHY this change is needed — Why, What Changes, Capabilities, Impact. 1–2 pages. | `schema.yaml` → `proposal.instruction` |
| `specs/<capability>/spec.md` | WHAT the system SHALL do — `### Requirement:` + `#### Scenario:` (exactly four hashes). One file per capability. Delta ops `## ADDED` / `## MODIFIED` / `## REMOVED` / `## RENAMED`. | `schema.yaml` → `specs.instruction`; example shape in `templates/spec.md` |
| `design.md` | HOW to build it — Context, Runtime dependencies, API surface, UI / Design System (conditional), Decisions, Risks, Migration, Open Questions. | `schema.yaml` → `design.instruction` |
| `tasks.md` | Implementation checklist — section menu (Scaffold, Compose, Seed, Implement-per-app, E2E + gate). | `schema.yaml` → `tasks.instruction` |

### `lint.md`

The **apply-readiness gate**. Ten sections of mechanical checks against every
prior artifact, organised by failure family:

1. **Authorization model** — Keycloak vs AAS, `active_tenant.permissions` claim
2. **Session storage** — SDS mandatory for web apps
3. **API contract** — `/v{N}/`, gateway path shape, envelope shape, `/livez` + `/readyz`
4. **UI surfaces** — mockup file exists, `@mssfoobar/ui` primitives, role gating
5. **Cross-artifact consistency** — capabilities↔specs, requirements↔scenarios, API↔scenarios, runtime-deps↔owners
6. **Verification feasibility** — observable THEN clauses, named token acquisition
7. **Tasks structure** — verification commands, dual-compose form, no `compose up <app>`, section ordering
8. **Skill triggering** — backtick'd skill names per task type
9. **Native dev env documentation** — env blocks per Implement section, `compose.override.yml` ports
10. **Reproducibility gate** — last task of E2E section, `down -v` + `up -d`, restarts native processes, re-runs SAME spec

Each check is **PASS** / **FAIL** / **N/A**:

- **PASS** — the check holds.
- **FAIL** — quote the offending line AND name the file path
  (e.g. `design.md:32`, `tasks.md:47`). Vague FAILs don't help fix the
  artifact.
- **N/A** — with a one-line reason ("no UI surface", "no auth surface", etc.).

**If any item is FAIL, STOP.** Fix the cited artifact, then re-run the lint.
If the fix is to an upstream artifact (proposal/design/specs), regenerate
`tasks.md` before re-linting — tasks may have inherited the broken
assumption.

### `apply`

Operates on `tasks.md`. Walks through pending checkboxes, implementing each
and marking complete. Requires `lint`; refuses to start while any lint FAIL
remains. See `/opsx:apply` for invocation.

## Skill integration

The schema delegates the actual "how to write X" knowledge to skills. The
artifact instructions reference them by name (in backticks, so they fire when
the implementing agent reads tasks.md):

| Skill | Owns |
|---|---|
| `aoh-conventions` | Code-level conventions — Go layering, swag annotations, response envelope, Svelte 5 runes, `@mssfoobar/ui` imports, DB schema rules. |
| `aoh-knowledge` | Service catalogue — what each AOH service does (IAMS, SDS, RTUS, GIS, UNH, …) and how to integrate. Includes the IAMS authz model and the Keycloak realm guide. |
| `aoh-design` | Page archetypes, `@mssfoobar/ui` subpath catalogue, design tokens, copy & tone, mockup pattern. |
| `aoh-dashboard` | **UI-builder skill** — dashboard surfaces on `@mssfoobar/dash-web-sdk` (the `<Grid>`, `WidgetPicker`, widgets via `defineWidget`, dashboard seeding). |
| `aoh-go-init` | Scaffolds new Go services with chi router, sqlx, Viper, mockery, golangci-lint, swag. |
| `aoh-web-init` | Scaffolds new SvelteKit apps with OIDC + SDS, a default sidebar/header layout, `@mssfoobar/ui`, Tailwind v4, gateway proxy. |
| `aoh-compose` | Generates the compose stack — Traefik + IAMS + SDS + project-aas-init + custom services. |

**UI-builder skills** are a category: each pairs an AOH skill with a platform
SDK that builds a specific UI surface type — today `aoh-dashboard` →
`@mssfoobar/dash-web-sdk`; future surface SDKs (e.g. a form builder) join the same
group. They sit alongside `aoh-design` (visual contract / tokens / archetype)
and `aoh-conventions/web.md` (general web rules), not in place of them. When a
change's surface matches one, the implementation builds on the SDK rather than
hand-rolling from `@mssfoobar/ui`. Registering a new UI-builder skill is a
three-place edit: a row here, the list in `templates/tasks.md` §5.2, and the
checks in `templates/lint.md` §4 / §8.

When a wrong-by-default pattern is discovered (e.g. the JWT lacks a
`permissions` claim, SDS is mandatory not optional), the canonical fix is
twofold: **patch the skill assets** so future scaffolds get it right, and
**add a lint check** so the schema catches it if the author misses the doc.

## Extending the schema

When a new wrong-by-default pattern surfaces during `/opsx:apply` and you want
to make sure it doesn't bite the next change:

1. **Patch the responsible skill** (`aoh-conventions`, `aoh-knowledge`,
   `aoh-go-init`, etc.) so the scaffolded code or doc is correct by default.
   This is the highest-leverage fix — future authors never encounter the
   problem.
2. **Add a lint check** to `templates/lint.md` under the appropriate failure
   family. If a new family is needed, add a section. Cite the canonical doc
   (`aoh-knowledge/services/<svc>.md`, `aoh-conventions/<area>.md`) so a
   future author hitting the FAIL knows where to read.
3. **Reference the lint check** from the relevant artifact's `instruction`
   block in `schema.yaml` if the check is subtle. Most checks need no
   forward-reference — the lint catches them at the gate.

When adding a new artifact: add it under `artifacts:` in `schema.yaml`, give
it `requires:` for its dependencies, and create a template in `templates/`.
If it should gate apply, add it to `apply.requires`.

## Why lint runs after tasks (and re-generates them on upstream fixes)

An earlier draft had the lint between `design` and `tasks` so an upstream
failure couldn't propagate into tasks. We moved it to the end because
**`tasks.md` is the highest-risk artifact** — section ordering, dual compose
form, `compose up <app>` mistakes, missing skill backticks, malformed
reproducibility gates all live in tasks and only show up during apply. Linting
tasks beats catching upstream errors a step earlier, because:

- Upstream errors are caught by the same lint, just two checks above the
  tasks ones. They're not missed.
- Tasks regeneration after an upstream fix is a few minutes; apply-time
  debugging is hours.
- Having one gate (`lint`) before one implementing step (`apply`) is simpler
  to teach than "lint twice, at two different points."

The trade-off is explicit: an upstream fix forces `tasks.md` regeneration
before re-linting. The lint's Resolution section reminds the author.
