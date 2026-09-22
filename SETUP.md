# Setup

Two stages. **Stage 1 needs internet and must be finished before you join the
workshop network.** There is no internet on that network, so nothing can be
downloaded or installed once you are on it.

Budget 30 minutes for stage 1, mostly waiting on installers and a first
`pnpm install`.

Everything runs natively on your machine. Only PostgreSQL runs in a container.

## The short way

Unzip the project first, then run the installer for your platform from inside
the `ai-day-demo` folder. It installs only what is missing, and is safe to run
again.

```sh
./scripts/install-prereqs.sh            # macOS and Linux
```

```powershell
.\scripts\install-prereqs.ps1           # Windows, from PowerShell
```

Add `--check` (`-Check` on Windows) to see what you are missing without
installing anything. Both print a summary at the end, and neither needs
Administrator or `sudo` for the tools themselves.

Open a new terminal afterwards, so every PATH change takes effect. Then skip to
[Get the project](#get-the-project).

If the installer cannot do something on your machine, it says which tool and
where to get it, and the rest of this section covers each one by hand.

## What you need

Six things. Install them in this order.

| Tool | Version | Where |
| --- | --- | --- |
| Node | 24 or newer | <https://nodejs.org> |
| pnpm | 10 | see below |
| Go | 1.25 or newer | <https://go.dev/dl/> |
| Podman | any recent | see below |
| Claude Code | latest | see below |
| Python | 3.9 or newer | see below |

Node, Go and Podman have installers you download and double-click. pnpm and
Claude Code are installed from the terminal, and both have a section below.

Podman:

- macOS and Windows: Podman Desktop, <https://podman-desktop.io>
- Linux: `sudo apt install podman podman-compose`, or your distribution's
  equivalent

On macOS and Windows, Podman Desktop needs a machine running before any
container starts. Open it once and start the machine it offers.

`podman compose` delegates to a compose provider, so you need `podman-compose`
or `docker-compose` on your PATH as well. Podman Desktop offers to install one
during onboarding. Check what you have with:

```sh
podman compose version
```

If that prints a version, you are set. If it reports no provider, install
`podman-compose` (`brew install podman-compose`, or
`pip install podman-compose`).

### Python

Nothing in this project is written in Python. It is here for the helper scripts
Claude Code writes while you work through the exercises, so that a script it
reaches for runs instead of failing and being rewritten.

```sh
python3 --version
```

- macOS: already installed. If that command offers to install the Xcode
  Command Line Tools, accept it now, while you still have internet.
- Linux: already installed.
- Windows: not installed. Take the installer from <https://www.python.org>,
  and tick **Add python.exe to PATH** on the first screen.

Do not install `uv`. `uv run` downloads an interpreter and its dependencies
when it is invoked, so it fails on the workshop network even on a machine that
already has Python.

### pnpm

Install Node first. `npm` comes with it, and this is the only thing pnpm needs.
Then, in any terminal:

```sh
npm install -g pnpm@10
```

Take the `@10`. A bare `npm install -g pnpm` gives you pnpm 12, and this repo's
lockfile and `pnpm-workspace.yaml` were written for pnpm 10. Check what you got:

```sh
pnpm --version
```

Any `10.x` is fine.

You should not need `sudo`. If the install fails with `EACCES`, your Node was
installed somewhere only root can write to. Reinstall Node from
<https://nodejs.org> rather than rerunning with `sudo`, which leaves
root-owned files in your npm directory and breaks later installs.

`corepack enable` is the other usual way to get pnpm, but Node 25 dropped
corepack. On Node 25 or newer, use the command above.

### Claude Code

Claude Code has its own installer, which does not go through npm. Run the line
for your terminal:

| Terminal | Command |
| --- | --- |
| macOS, Linux, WSL | `curl -fsSL https://claude.ai/install.sh \| bash` |
| Windows PowerShell | `irm https://claude.ai/install.ps1 \| iex` |
| Windows CMD | `curl -fsSL https://claude.ai/install.cmd -o install.cmd && install.cmd && del install.cmd` |

On Windows, your prompt tells you which one you are in: PowerShell shows
`PS C:\`, CMD shows `C:\` with no `PS`. You do not need to run as
Administrator.

Close the terminal and open a new one, so the installer's changes to your PATH
take effect. Then:

```sh
claude --version
```

That should print a version such as `2.1.211 (Claude Code)`. If it says
`command not found`, the new terminal is the usual fix.

Do not run `claude` to log in yet. This workshop does not use an Anthropic
account, and stage 2 covers starting it against the workshop's model server.

If you would rather install through npm, then
`npm install -g @anthropic-ai/claude-code` works and fetches the same binary.
Never prefix it with `sudo`.

## Get the project

Unzip the download, or clone the repo. Everything below runs from inside the
`ai-day-demo` folder.

```sh
cd path/to/ai-day-demo
```

The one credential this needs is already here. A GitHub Packages token for the
six `@mssfoobar` dependencies is checked in at `.npmrc`, so there is nothing to
create. The Go shared library is checked in under `packages/aoh-golib`, so no
access to the private `ops-hub` repo is required.

## Stage 1: on your own internet

Open a terminal in the `ai-day-demo` folder. Every command below runs from
there; the compose paths do not resolve from anywhere else.

```sh
pnpm start
```

That single command does the whole of stage 1:

1. `pnpm install` pulls the Node dependencies into `node_modules`.
2. `podman compose up -d postgres` pulls `postgres:16-alpine` and starts it.
3. `go run ./cmd/server` downloads the Go modules and compiles the service.
4. `vite dev` starts the console.

Steps 1 to 3 are the ones that need the network. Each writes into a cache on
your disk that survives going offline.

Then open **<http://localhost:5173>**.

**This is the checkpoint.** If you see the dispatch console with a list of
units, you are done and everything you need is now on your machine. If you do
not, fix it now rather than on the day: there is no second chance to download
anything.

Leave `pnpm start` running, or stop it with `Ctrl+C`. The database keeps
running either way.

### If the container images came as a tarball

Some downloads ship `ai-day-workshop-images.tar.gz` alongside the project. It
carries `postgres:16-alpine`, so loading it skips the pull in step 2:

```sh
podman load -i ai-day-workshop-images.tar.gz
```

Run that before `pnpm start`. It needs no network.

### If something already holds port 5432

`pnpm start` names the port and stops rather than half-starting. Give the
database another one:

```sh
POSTGRES_PORT=5441 pnpm start
```

## Stage 2: on the workshop network

Join the workshop wifi, then, from the `ai-day-demo` folder again:

```sh
pnpm start
```

In a second terminal, in the same folder, confirm the model is reachable:

```sh
pnpm doctor
```

All four checks should pass. Then start the agent:

```sh
claude
```

No login and no API key: `.claude/settings.json` in this repo already points
Claude Code at the workshop's model server. That only works when you run
`claude` from inside the `ai-day-demo` folder.

## Stopping

| Command | Does |
| --- | --- |
| `Ctrl+C` | stops the console and the service; the database keeps running |
| `pnpm stop` | stops and removes the database container |
| `pnpm reset-db` | the same, and deletes the seeded data with it |
