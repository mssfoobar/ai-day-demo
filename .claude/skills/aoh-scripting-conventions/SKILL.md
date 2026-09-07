---
name: aoh-scripting-conventions
description: >
  Choice-of-language convention for executable scripts in AOH meta-platform
  repos (`agent-skills`, `aa-cli`, etc.). Three categories: (1) Python via
  `uv run python scripts/<name>.py` for cross-platform dev/setup tooling
  every contributor must run; (2) bash with `set -euo pipefail`, POSIX,
  shellcheck-clean for Linux-only / container / server-side scripts;
  (3) Makefiles forbidden as repo tooling, allowed only as template content
  emitted into downstream Linux/macOS-targeted projects. The rule exists
  because AOH developers run native Windows PowerShell, where `make`,
  `awk`, and many shell utilities aren't available — repo tooling that
  depends on them strands Windows contributors at first clone.
  **Always consult before adding or modifying any executable script in
  these repos (under `scripts/`, a skill's `scripts/`, or the repo
  root).** Trigger on "add a setup script", "Makefile for X", "is
  Make ok here", "replace this Makefile", "bootstrap script",
  "scaffold helper", "make setup", "needs to run on Windows".
---

# Scripting language conventions

The AOH meta-platform repos (`agent-skills`, `aa-cli`, etc.) have a strict
choice-of-language rule for executable scripts. The rule exists because
**AOH platform developers run native Windows PowerShell** alongside
macOS/Linux. `make`, `awk`, and many shell utilities aren't available on
PowerShell by default, so repo tooling that depends on them strands Windows
contributors at the first `make setup`.

## The rule

| Audience / context | Language | Invocation |
|---|---|---|
| Cross-platform dev/setup tooling — every contributor must run it, regardless of OS | **Python**, via `uv run python scripts/<name>.py` | `uv` is a hard dependency for these repos; it brings the project's pinned Python and dependencies without polluting the system. |
| Linux-only / container / server-side scripts — runs inside containers, on servers we provision, or in environments where bash is guaranteed | **bash**, POSIX-compliant, `set -euo pipefail`, shellcheck-clean | Direct invocation. |
| Makefile | **Avoid** for repo-owned tooling. **Allowed only** as template content emitted into downstream projects whose audience is guaranteed Linux / macOS / WSL. | — |

## Decision rubric

When adding a script, ask two questions:

1. **Audience OS** — does the script need to run on native Windows PowerShell?
   - YES → Python.
   - NO (Linux-only by environment, e.g. inside a Docker container) → bash is fine.
2. **Execution context** — where does the script run?
   - On a contributor's machine, as part of first-clone bootstrap or daily dev workflow → cross-platform → Python.
   - Inside a Docker container or on a server we provision → Linux guaranteed → bash is fine.

Both dimensions point to bash only when the script is *Linux-only AND server-side*. Otherwise, default to Python.

If your reflex is "this is a few one-liners, surely Make is fine" — it isn't. The few-one-liner case is exactly where Python wins easily and Make breaks on Windows.

## In-tree examples

### Python (cross-platform tooling)

- `scripts/setup.py` — repo-level dev setup. Replaced an earlier Makefile precisely because of this convention.
- `skills/aoh-compose/scripts/bootstrap.py` — generates Docker Compose layouts; runs on every contributor's machine.
- `skills/aoh-go-init/scripts/scaffold.py` — Go microservice scaffolder.

All invoked via `uv run python <path> ...` so they pick up the project's pinned Python and dependencies.

### Bash (Linux-only / server-side)

- `deploy/spoke-services/*/bootstrap.sh` (`act-runner`, `backstage-spoke`, `forgejo`, `gitlab-runner`, `gitlab`) — runs inside containers during spoke startup. Linux guaranteed by the container image.
- `skills/aoh-hub-nexus/scripts/*.sh` (`bootstrap.sh`, `up.sh`, `down.sh`, `check.sh`) — operates on a containerised Nexus instance.

All POSIX, `set -euo pipefail`, shellcheck-clean.

### Makefile (allowed: template content only)

- `skills/aoh-go-init/assets/Makefile` — copied into scaffolded Go services. The downstream audience is guaranteed Linux / macOS / WSL (Go toolchain users). This is template content emitted into another repo, not tooling for *this* repo.

## Explicit exception: the installer pair

`install.sh` + `install.ps1` — first-install bootstrap for fresh contributor machines. This is the one place we ship two native-shell scripts side-by-side instead of one Python script, because *installing* `uv` / Python is exactly the prerequisite the rest of the convention assumes. **Don't extend this pattern to other tooling** — it is a one-off compromise for the bootstrap edge.

## Anti-pattern

**Don't reach for Make.** Even simple `make setup` / `make hooks` / `make help` patterns break native Windows PowerShell. The `agent-skills` repo had a Makefile for dev setup briefly; it didn't work for Windows contributors and was replaced with `scripts/setup.py` for exactly this reason.
