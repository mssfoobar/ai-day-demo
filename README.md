# fleet-dispatch-console

A workshop dispatch console on the AOH platform: a SvelteKit frontend (`apps/dispatch-web`)
and a Go + PostgreSQL field-unit service (`apps/dispatch-svc`). **No authentication**,
by design, for the workshop.

**Attending the workshop? Start with [SETUP.md](SETUP.md).** The rest of this file
is for working on the repo itself.

## Prerequisites

The two apps run natively. Everything they depend on, sixteen containers from the
dispatch database to IAMS, SDS, RTUS, GIS and Traefik, runs in compose.
`./scripts/install-prereqs.sh` (or `.\scripts\install-prereqs.ps1` on Windows)
installs whatever is missing from this list; `--check` reports without
installing.

- **Node 24+**
- **pnpm 10**: `npm install -g pnpm@10`, or `corepack enable` on Node 24 (Node 25+
  dropped corepack). Take the `@10`: an unpinned install gives pnpm 12, and the
  lockfile and `pnpm-workspace.yaml` here were written for 10.
- **Go 1.25+**
- **Podman**, with a compose provider. `podman compose` delegates to
  `podman-compose` or `docker-compose`; check yours with `podman compose version`.
- **Claude Code**, for the workshop exercises:
  `curl -fsSL https://claude.ai/install.sh | bash`, or
  `irm https://claude.ai/install.ps1 | iex` in Windows PowerShell
- **Python 3.9+**, for the scripts the agent writes during the exercises. Not
  `uv`: `uv run` fetches an interpreter at invocation time, which fails on the
  offline workshop network.

Nothing in this repo is written in Python. Both of its scripts are Node
(`scripts/dev.mjs`, `scripts/doctor.mjs`), and `pnpm start`, `pnpm doctor` and
`pnpm verify` never call an interpreter other than Node and Go.

The one credential this needs is already here. A GitHub Packages token for the six
`@mssfoobar` dependencies is checked in at `.npmrc`, so there is nothing to create; it
expires shortly after the workshop. The Go shared library `aoh-golib` is checked in
under `packages/aoh-golib`, so no access to the private `ops-hub` repo is required.

## Run it

```sh
pnpm start
```

Then open **http://127.0.0.1.nip.io:5173/aoh/dispatch/units**. `Ctrl+C` stops the two
apps; the stack keeps running (`pnpm stop` removes it, `pnpm reset` also wipes its
volumes).

`pnpm start` is a thin runner (`scripts/dev.mjs`). It checks 8081 and 5173 are free,
installs dependencies, then starts these three, in order:

```sh
podman compose -f compose/compose.yml up -d
(cd apps/dispatch-svc && go run ./cmd/server)            # http://localhost:8081
(cd apps/dispatch-web && pnpm dev)                       # http://127.0.0.1.nip.io:5173
```

Run them yourself if you prefer to see the parts. On every run after the first,
the install and the stack are sub-second no-ops. Stop the stack with `Ctrl+C`
before starting it again: the port check refuses to start a second copy.

The console is served on `127.0.0.1.nip.io` rather than `localhost` so the session
cookie is issued on a parent of `rtus-seh.127.0.0.1.nip.io`, which is what lets the
map receive updates.

The deck under `slides/` is its own pnpm workspace, so the install above does
not reach it. `pnpm slides` installs it on demand.

The two apps run natively (`go run`, `vite dev`) so edits reload instantly. That is the
AOH convention for local development. Migrations and seed data apply themselves when the
service starts.

| Command | Does |
|---|---|
| `pnpm start` | install → stack → service → console, with a port preflight |
| `pnpm start --no-infra` | same, assuming the stack is already up |
| `POSTGRES_PORT=5441 pnpm start` | same, when something else already holds 5432 |
| `pnpm stop` / `pnpm reset` | stop the stack / stop it **and delete its volumes** |
| `pnpm verify` | lint, type-check and build across both apps |
| `pnpm doctor` | check the stack is up and the workshop model is reachable |

## Workshop exercises

Three features are deliberately **stubbed, not built**: dispatching a unit to an incident,
a per-unit activity timeline, and crew management. A floating button lists them and, one at a time,
sketches the missing control where it goes in the console; the service answers `501`
on their routes. Their user stories,
acceptance criteria and pointers to the code to copy are in **[WORKSHOP.md](WORKSHOP.md)**.

## Layout

```
apps/dispatch-web      SvelteKit console            → apps/dispatch-web/README.md
apps/dispatch-svc      Go field-unit service        → apps/dispatch-svc/README.md
packages/aoh-golib     local copy of the AOH Go library (see LOCAL_COPY.md there)
compose/               Postgres, the one container
scripts/install-prereqs.sh  the toolchain installer (.ps1 for Windows)
scripts/dev.mjs        `pnpm start`
scripts/doctor.mjs     `pnpm doctor`, the offline readiness check
SETUP.md               participant setup, the two stages
WORKSHOP.md            the three stubbed features, as user stories
openspec/              planning artifacts for each change
```

Conventions for contributors and agents live in `AGENTS.md`; the project's own vocabulary
in `UBIQUITOUS_LANGUAGE.md`.
