# fleet-dispatch-console

A workshop dispatch console on the AOH platform: a SvelteKit frontend (`apps/dispatch-web`)
and a Go + PostgreSQL field-unit service (`apps/dispatch-svc`). **No authentication** —
by design, for the workshop.

## Prerequisites

- **Node 24+**
- **pnpm 10** — `npm i -g pnpm`, or `corepack enable` on Node 24 (Node 25+ dropped corepack)
- **Go 1.25+**
- **Podman** — only PostgreSQL runs in a container
- **A GitHub token with `read:packages`** in `~/.npmrc`, because the design system
  `@mssfoobar/ui` is published to GitHub Packages:

  ```
  //npm.pkg.github.com/:_authToken=<token>
  ```

  Without it `pnpm install` fails with `401 Unauthorized`. This is the only credential
  the repo needs — the Go shared library `aoh-golib` is checked in under
  `packages/aoh-golib`, so no access to the private `ops-hub` repo is required.

## Run it in a container

The devcontainer carries the whole toolchain, so the only thing you install is
Podman. Node, pnpm, Go, Claude Code and an already warm Go build cache are all
in the image, and none of them appear in the prerequisite list above.

The GitHub token is still required. Those six packages cannot be baked into a
public image without republishing them. Create `~/.npmrc` **before** you start
the container: a missing host file is mounted as an empty directory, and the
install then fails confusingly.

```sh
podman compose -f compose/compose.yml -f compose/compose.devcontainer.yml up -d
podman compose exec workshop bash
pnpm start
```

Then open <http://localhost:5173>. `pnpm start` installs dependencies, notices
PostgreSQL is already up as a sibling container, and starts the service and the
console. It is idempotent, so running it again is a sub-second no-op.

The image is published public at `ghcr.io/mssfoobar/ai-day-workshop`, so the
pull needs no credential. Compose builds it locally when the tag is
unavailable, which takes a few minutes.

VS Code's Dev Containers extension does the same thing with **Reopen in
Container**. It assumes Docker, so point it at Podman first:

```json
"dev.containers.dockerPath": "podman",
"dev.containers.dockerComposePath": "podman-compose"
```

Use `pnpm` on the host or in the container, not both. `node_modules` lands on
the bind mount, and the two platforms need different native binaries.

The deck under `slides/` is its own pnpm workspace, so the install above does
not reach it. `pnpm slides` installs it on demand.

## Run it natively

```sh
pnpm start
```

Then open **http://localhost:5173**. `Ctrl+C` stops the two apps; the database keeps
running (`pnpm stop` removes it, `pnpm reset-db` also wipes its data).

`pnpm start` is a thin runner (`scripts/dev.mjs`) that does exactly these three things, in
order, and nothing else — run them yourself if you prefer to see the parts:

```sh
podman compose -f compose/compose.yml up -d postgres
(cd apps/dispatch-svc && go run ./cmd/server)            # http://localhost:8081
(cd apps/dispatch-web && pnpm dev)                       # http://localhost:5173
```

The two apps run natively (`go run`, `vite dev`) so edits reload instantly — that is the
AOH convention for local development. Migrations and seed data apply themselves when the
service starts.

| Command | Does |
|---|---|
| `pnpm start` | install → db → service → console, with a port preflight |
| `pnpm start --no-db` | same, assuming Postgres is already up |
| `POSTGRES_PORT=5441 pnpm start` | same, when something else already holds 5432 |
| `pnpm stop` / `pnpm reset-db` | stop the database / stop it **and delete its data** |
| `pnpm verify` | lint, type-check and build across both apps |

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
compose/               Postgres, plus the devcontainer overlay
.devcontainer/         the preloaded toolchain image
scripts/dev.mjs        `pnpm start`
WORKSHOP.md            the three stubbed features, as user stories
openspec/              planning artifacts for each change
```

Conventions for contributors and agents live in `AGENTS.md`; the project's own vocabulary
in `UBIQUITOUS_LANGUAGE.md`.
