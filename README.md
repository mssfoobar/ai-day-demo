# fleet-dispatch-console

A workshop dispatch console on the AOH platform: a SvelteKit frontend (`apps/dispatch-web`)
and a Go + PostgreSQL field-unit service (`apps/dispatch-svc`), with a live operator map.
Operators sign in through **IAMS** (Keycloak + AAS), and positioned units are mirrored into
**GIS** and streamed to the browser over **RTUS**.

**Attending the workshop? Start with [SETUP.md](SETUP.md).** The rest of this file
is for working on the repo itself.

## Prerequisites

- **Node 24+**
- **pnpm 10** — `npm i -g pnpm`, or `corepack enable` on Node 24 (Node 25+ dropped corepack)
- **Go 1.25+**
- **Podman** (or Docker) — the platform stack runs in containers: the dispatch
  PostgreSQL plus IAMS, SDS, RTUS, GIS and Traefik. The two apps themselves run natively.

The one credential this needs is already here. A GitHub Packages token for the six
`@mssfoobar` dependencies is checked in at `.npmrc`, so there is nothing to create; it
expires shortly after the workshop. The Go shared library `aoh-golib` is checked in
under `packages/aoh-golib`, so no access to the private `ops-hub` repo is required.

## Run it in a container

The devcontainer carries the whole toolchain, so the only thing you install is
Podman. Node, pnpm, Go, Claude Code and an already warm Go build cache are all
in the image, and none of them appear in the prerequisite list above.

The six `@mssfoobar` packages stay out of the public image, so `pnpm install`
runs on first open against the checked-in `.npmrc`.

```sh
podman compose -f compose/compose.yml -f compose/compose.devcontainer.yml up -d
podman compose exec workshop bash
pnpm start
```

Then open <http://127.0.0.1.nip.io:5173> and sign in — the seeded accounts are in
[SETUP.md](SETUP.md). `pnpm start` installs dependencies, brings the stack up, waits for it
to converge, and starts the service and the console. It is idempotent, so running it again
is a sub-second no-op.

Not `localhost`: the console is served on the platform's dev domain because the session
cookie has to reach `rtus-seh.127.0.0.1.nip.io` for the map's live feed. `nip.io`
wildcard-resolves to the loopback on every OS, so there is no hosts file to edit.

The image is not published to a registry. Load it from
`ai-day-workshop-images.tar.gz`, which ships alongside the project download and
also carries `postgres:16-alpine`. Failing that, compose builds it from
`.devcontainer/Dockerfile`, which takes a few minutes and needs the network.

The rest of the stack pulls from `ghcr.io/mssfoobar` on first start, which needs network
access to that registry.

VS Code's Dev Containers extension does the same thing with **Reopen in
Container**. It assumes Docker, so point it at Podman first:

```json
"dev.containers.dockerPath": "podman",
"dev.containers.dockerComposePath": "podman-compose"
```

The container keeps `node_modules` in a named volume rather than on the bind
mount. Windows bind mounts cannot set file times, which fails pnpm with `EPERM
futime`, and the host and the container need different native binaries anyway.
A native run installs its own copy on the host; the two do not collide.

The deck under `slides/` is its own pnpm workspace, so the install above does
not reach it. `pnpm slides` installs it on demand.

## Run it natively

```sh
pnpm start
```

Then open **<http://127.0.0.1.nip.io:5173>** and sign in. `Ctrl+C` stops the two apps; the
stack keeps running (`pnpm stop` removes it, `pnpm reset` also wipes its data — the realm,
the AAS roles, the units and their geo-entities all reconverge from checked-in artifacts on
the next start).

`pnpm start` is a thin runner (`scripts/dev.mjs`) that does exactly these three things, in
order, and nothing else — run them yourself if you prefer to see the parts:

```sh
podman compose -f compose/compose.yml up -d              # or: docker compose -f ... up -d
(cd apps/dispatch-svc && go run ./cmd/server)            # http://localhost:8081
(cd apps/dispatch-web && pnpm dev)                       # http://127.0.0.1.nip.io:5173
```

Run natively, both apps need an environment the compose defaults do not supply — see the
env blocks in each app's README, which is what `pnpm start` sets for you.

The two apps run natively (`go run`, `vite dev`) so edits reload instantly — that is the
AOH convention for local development. Migrations apply themselves when the service starts.
The **roster does not**: the service seeds a tenant once, on its first request from a
caller holding `dispatch-dispatcher`. There is no seed command to run — signing in as the
dispatcher is what populates the console.

| Command | Does |
|---|---|
| `pnpm start` | install → stack → service → console, with a port preflight |
| `pnpm start --no-infra` | same, assuming the compose stack is already up |
| `POSTGRES_PORT=5441 pnpm start` | same, when something else already holds 5432 |
| `pnpm stop` / `pnpm reset` | stop the stack / stop it **and delete all its data** |
| `pnpm e2e` | the end-to-end smoke test, against a running stack |
| `pnpm verify` | lint, type-check and build across both apps |

## Workshop exercises

Three features are deliberately **stubbed, not built**: dispatching a unit to an incident,
a per-unit activity timeline, and crew management. A floating button lists them and, one at a time,
sketches the missing control where it goes in the console; the service answers `501`
on their routes — to an authorised caller. Unauthenticated, they answer `401` first. Their user stories,
acceptance criteria and pointers to the code to copy are in **[WORKSHOP.md](WORKSHOP.md)**.

## Layout

```
apps/dispatch-web      SvelteKit console            → apps/dispatch-web/README.md
apps/dispatch-svc      Go field-unit service        → apps/dispatch-svc/README.md
packages/aoh-golib     local copy of the AOH Go library (see LOCAL_COPY.md there)
compose/               the platform stack (IAMS, SDS, RTUS, GIS, Traefik) plus
                       the dispatch Postgres and the devcontainer overlay
.devcontainer/         the preloaded toolchain image
scripts/dev.mjs        `pnpm start`
scripts/e2e-smoke.mjs  `pnpm e2e` — the end-to-end smoke test
SETUP.md               participant setup, the two stages
WORKSHOP.md            the three stubbed features, as user stories
openspec/              planning artifacts for each change
```

Conventions for contributors and agents live in `AGENTS.md`; the project's own vocabulary
in `UBIQUITOUS_LANGUAGE.md`.
