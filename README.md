# fleet-dispatch-console

A workshop dispatch console on the AOH platform: a SvelteKit frontend (`apps/dispatch-web`)
and a Go + PostgreSQL field-unit service (`apps/dispatch-svc`). **No authentication** —
by design, for the workshop.

## Prerequisites

- **Node 24+** (pnpm comes with it via corepack — `corepack enable` once if `pnpm` is not found)
- **Go 1.25+**
- **Podman or Docker** — only PostgreSQL runs in a container
- **A GitHub token with `read:packages`** in `~/.npmrc`, because the design system
  `@mssfoobar/ui` is published to GitHub Packages:

  ```
  //npm.pkg.github.com/:_authToken=<token>
  ```

  Without it `pnpm install` fails with `401 Unauthorized`. This is the only credential
  the repo needs — the Go shared library `aoh-golib` is checked in under
  `packages/aoh-golib`, so no access to the private `ops-hub` repo is required.

## Run it

```sh
pnpm install
pnpm start
```

Then open **http://localhost:5173**. `Ctrl+C` stops the two apps; the database keeps
running (`pnpm stop` removes it, `pnpm reset-db` also wipes its data).

`pnpm start` is a thin runner (`scripts/dev.mjs`) that does exactly these three things, in
order, and nothing else — run them yourself if you prefer to see the parts:

```sh
podman compose -f compose/compose.yml up -d postgres     # or: docker compose ...
(cd apps/dispatch-svc && go run ./cmd/server)            # http://localhost:8081
(cd apps/dispatch-web && pnpm dev)                       # http://localhost:5173
```

The two apps run natively (`go run`, `vite dev`) so edits reload instantly — that is the
AOH convention for local development. Migrations and seed data apply themselves when the
service starts.

| Command | Does |
|---|---|
| `pnpm start` | db → service → console, with a port preflight and a health wait |
| `pnpm start --no-db` | same, assuming Postgres is already up |
| `pnpm stop` / `pnpm reset-db` | stop the database / stop it **and delete its data** |
| `pnpm verify` | lint, type-check and build across both apps |

## Layout

```
apps/dispatch-web      SvelteKit console            → apps/dispatch-web/README.md
apps/dispatch-svc      Go field-unit service        → apps/dispatch-svc/README.md
packages/aoh-golib     local copy of the AOH Go library (see LOCAL_COPY.md there)
compose/               the one Postgres container
scripts/dev.mjs        `pnpm start`
openspec/              planning artifacts for each change
```

Conventions for contributors and agents live in `AGENTS.md`; the project's own vocabulary
in `UBIQUITOUS_LANGUAGE.md`.
