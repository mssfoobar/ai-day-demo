# AGENTS.md

Monorepo-wide conventions. App- and package-specific rules live in the
`AGENTS.md` inside each `apps/<name>/` or `packages/<name>/` directory.

## Branching

`develop` is the canonical branch. Cut feature branches from `develop`;
PRs target `develop`. Use `type/short-description` branch names
(`feat/…`, `fix/…`, etc.).

## Pull requests

All changes land via PR — never commit directly to `develop`. Keep PRs
scoped to one logical change.

## Commits

Conventional Commits: `type(scope): description`.

## Verify before commit

Run every check the repo configures — lint, type-check, build, and tests
across all workspaces — and make them all pass. Do not skip checks or
bypass hooks (`--no-verify`, etc.) to push past failures; fix the
underlying issue.

## Agent autonomy & pre-merge review

When working at high autonomy, every PR you cut MUST pass an
agent-driven pre-merge review loop before it's presented to a developer:

1. Author the change on a feature branch; run all checks green.
2. Spawn separate agent code reviews — an indeterminate number, as many
   as the change warrants (parallel specialists across code, tests,
   docs, behavior).
3. Analyze every review's findings; fix all important issues on the
   same branch.
4. Re-run all tests and checks after the fixes.
5. Repeat steps 2–4 iteratively until a review pass surfaces no
   important issues.
6. Only then present the work back to the developer for review.

CI green is necessary but not sufficient — the review loop catches what
CI and self-review miss. Do not surface a PR for human review mid-loop.

## Tracked work

Track non-trivial work in your issue tracker before writing code; keep
the issue updated as work progresses. Trivial changes (typos, automated
bumps, single-line fixes) are exempt.

## Agent context

This repo follows the AOH layered agent-context model: `AGENTS.md` is the
cross-tool source of truth; `CLAUDE.md` is a one-line `@AGENTS.md` import.
Edit `AGENTS.md` — never `CLAUDE.md` directly. Each app/package may carry
its own `AGENTS.md` for architecture an agent can't infer from the code.

## Domain language

This project's own domain vocabulary lives in `UBIQUITOUS_LANGUAGE.md` at the
repo root — author it from your domain's ubiquitous language and use those terms in
specs, tests, and code. **Platform** terms (modules, `Provider`/`BFF`,
`geo-entity`, `active_tenant`, response envelope, …) are defined by the AOH
glossary in the `aoh-knowledge` skill — **reference them, never redefine them**.
In `UBIQUITOUS_LANGUAGE.md` capture only your own terms and how they map to
platform ones (e.g. "our *Vessel* ↔ a GIS `geo-entity`"), and note the
`aoh-knowledge` version you built against so a skill bump surfaces vocabulary
changes for review.

## Defaults

- Stay in scope. No files created unless asked, no refactors, and no
  speculative error handling beyond the task.
