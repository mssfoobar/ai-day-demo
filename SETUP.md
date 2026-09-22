# Setup

Two stages. **Stage 1 needs internet and must be finished before you join the
workshop network.** There is no internet on that network, so nothing can be
downloaded or installed once you are on it.

Budget 20 minutes for stage 1, mostly waiting on a 1.1GB download.

## What you need

Podman Desktop, and nothing else. Node, pnpm, Go and Claude Code are all inside
the container.

- macOS and Windows: <https://podman-desktop.io>
- Linux: `sudo apt install podman podman-compose`, or your distribution's
  equivalent

Download the one zip that matches your machine. It is around 450MB, with the
container images inside it.

| Machine | File |
| --- | --- |
| Mac with Apple Silicon (M1 or later) | `ai-day-workshop-mac.zip` |
| Windows, Linux, or an Intel Mac | `ai-day-workshop-windows-linux.zip` |

Take the right one. The images are built per processor, and the wrong file
fails to start rather than running slowly. If you are on an Intel Mac, you want
the Windows and Linux file.

Unzip it. Everything below runs from inside the `ai-day-demo` folder it
creates.

## Stage 1: on your own internet

Open a terminal in the `ai-day-demo` folder. Every command below runs from
there; the compose paths do not resolve from anywhere else.

```sh
cd path/to/ai-day-demo
```

Load the images. This takes a minute and needs no network.

```sh
podman load -i ai-day-workshop-images.tar.gz
```

Start the container and the platform stack. This is sixteen services — the dispatch
PostgreSQL plus IAMS (Keycloak + AAS), SDS, RTUS, GIS and Traefik — so the first start
takes a few minutes while the images are pulled:

```sh
podman compose -f compose/compose.yml -f compose/compose.devcontainer.yml up -d
```

Step inside it:

```sh
podman compose -f compose/compose.yml -f compose/compose.devcontainer.yml exec workshop bash
```

Install and start everything:

```sh
pnpm start
```

Then open **<http://127.0.0.1.nip.io:5173>** and sign in:

| Account | Password | Can |
| --- | --- | --- |
| `admin` | `P@ssw0rd` | read and write — add, edit and delete units |
| `viewer` | `P@ssw0rd` | read only — the write controls are not shown |

Not `localhost`. The console is served on `127.0.0.1.nip.io`, which resolves to your own
machine exactly like `localhost` does but is a real domain name — which the session cookie
needs in order to reach the live-map feed. There is nothing to install or configure for
this.

**This is the checkpoint.** Sign in as `admin` and you should see the dispatch console with
five units, and a **Map** entry in the sidebar showing four of them. If you do not, fix it
now rather than on the day: there is no second chance to download anything.

The roster appears on the **first dispatcher sign-in**, not before. Signing in as `viewer`
on a brand-new stack shows an empty roster, and that is correct — seeding is a write, so
only `admin` triggers it. Sign in as `admin` once and the units are there for both.

Leave `pnpm start` running, or stop it with `Ctrl+C`. The stack keeps running either way
(`pnpm stop` removes it; `pnpm reset` also deletes its data, after which the realm, the
roles and the roster all rebuild themselves on the next start).

### If you set this up before the auth and map change

The compose project was renamed (`compose` → `aoh`) when the stack grew, and Podman names
volumes after the project. Your old `compose_dispatch-pgdata` is not the new
`aoh_dispatch-pgdata`, so the database starts empty — which is harmless, because the roster
reseeds itself on the first dispatcher sign-in. The devcontainer's three `node_modules`
volumes are renamed the same way, so the first `pnpm install` inside the container runs in
full again. **Do that on your own internet, in stage 1** — it needs the network.

## Stage 2: on the workshop network

Join the workshop wifi, then, from the `ai-day-demo` folder again. Everything is already
on your machine from stage 1, so nothing is downloaded here:

```sh
podman compose -f compose/compose.yml -f compose/compose.devcontainer.yml up -d
podman compose -f compose/compose.yml -f compose/compose.devcontainer.yml exec workshop bash
pnpm start
```

Give the stack a minute: `pnpm start` waits for every service to report healthy and for the
two one-shot init containers — which create the tenant and the application roles — to
finish. Starting the apps before those complete yields a sign-in that works and a console
that refuses everything.

In a second terminal, inside the container, confirm the model is reachable:

```sh
pnpm doctor
```

All four checks should pass. Then start the agent:

```sh
claude
```

No login and no API key: it is already pointed at the workshop's model server.
