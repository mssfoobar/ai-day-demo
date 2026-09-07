# fleet-dispatch-console

A workshop dispatch console on the AOH platform: a SvelteKit frontend (`apps/dispatch-web`)
and a Go + PostgreSQL field-unit service (`apps/dispatch-svc`). **No authentication** —
by design, for the workshop.

## Run it

From a fresh clone, one command checks every prerequisite, installs, and starts the
database, the service and the console:

```powershell
.\launch.ps1          # Windows
```

```sh
./launch.sh           # macOS / Linux
```

Then open **http://localhost:5173**. Stop with `Ctrl+C`.

If you already have pnpm on your PATH the same thing is:

| Command | Does |
|---|---|
| `pnpm launch` | check prerequisites → install → start everything |
| `pnpm bootstrap` | check prerequisites → install (no start) |
| `pnpm start` | start db + service + console (assumes setup is done) |
| `pnpm stop` | stop the database container |
| `pnpm reset-db` | stop it **and delete its data** — the seed re-applies on next start |
| `pnpm verify` | lint, type-check, build and unit tests across both apps |

### What `bootstrap` checks

| Prerequisite | Fixed for you? |
|---|---|
| Node ≥ 24, corepack, pnpm (pinned) | pnpm yes, via corepack |
| Go ≥ 1.25 | no — install from go.dev |
| Docker or Podman, **daemon running** | tries to start Docker Desktop (Windows/macOS) |
| `GOPRIVATE` for the private `aoh-golib` module | yes |
| Git access to `mssfoobar/ops-hub` | no — sign in with `gh auth login` |
| GitHub Packages token in `~/.npmrc` for `@mssfoobar/ui` | yes **if** `GITHUB_TOKEN` is set; otherwise it prints the line to add |

The two credentials are the only things it cannot conjure. You need a GitHub token with
`read:packages` (for the design system) and git access to the private `ops-hub` repo (for
the Go shared library). Everything else is installable software.

Only PostgreSQL runs in a container. The two apps run natively (`go run`, `vite dev`) so
edits reload instantly — that is the AOH convention for local development.

## Layout

```
apps/dispatch-web      SvelteKit console            → apps/dispatch-web/README.md
apps/dispatch-svc      Go field-unit service        → apps/dispatch-svc/README.md
compose/               the one Postgres container
scripts/setup.mjs      prerequisite check + install
scripts/dev.mjs        start everything
openspec/              planning artifacts for each change
```

Conventions for contributors and agents live in `AGENTS.md`; the project's own vocabulary
in `UBIQUITOUS_LANGUAGE.md`.
