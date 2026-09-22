# Setup

Two stages. **Stage 1 needs internet and must be finished before you join the
workshop network.** There is no internet on that network, so nothing can be
downloaded or installed once you are on it.

Budget 30 minutes for stage 1, mostly waiting on installers and a first
`pnpm install`.

The two apps run natively on your machine. Everything they depend on runs in
containers: the dispatch database, plus IAMS (Keycloak and AAS), SDS, RTUS, GIS
and a Traefik router, sixteen in all.

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

## The offline bundle

The installer above downloads as it goes, so it needs internet. If you are
already on the workshop network, or the machine never had internet, use the
bundle instead: a folder carrying Node, Go, pnpm, Claude Code and, on Windows,
Python, with nothing left to fetch.

Unzip `prereq-bundle-<your platform>.zip`, then run the installer inside it:

```sh
./install-prereqs-offline.sh            # macOS and Linux
```

```powershell
.\install-prereqs-offline.ps1           # Windows, from PowerShell
```

`--check` (`-Check` on Windows) reports what is missing without installing
anything, same as above. Windows needs no Administrator: Node and Go go under
`%LOCALAPPDATA%\ai-day-demo`. macOS and Linux put them in `/usr/local` and ask
for `sudo` once, falling back to `~/.local` where there is no `sudo`.

Podman is not in the bundle. Install Podman Desktop from
<https://podman-desktop.io> while you still have internet.

Whoever hands the bundle out builds it on a connected machine, once:

```sh
pnpm bundle:prereqs --zip
```

That writes `prereq-bundle/` and one zip per platform beside it, each carrying
only what that machine needs:

| Zip | Size |
| --- | --- |
| `prereq-bundle-darwin-arm64.zip` | 215 MB |
| `prereq-bundle-linux-x64.zip` | 233 MB |
| `prereq-bundle-win32-x64.zip` | 253 MB |

Add `--platform darwin-arm64` to build one of the three. Every file is checked
against the sha256 its publisher ships, and written to a `SHA256SUMS` the
offline installers re-check after the copy across, so a truncated USB transfer
is caught before anything is installed.

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
on your PATH as well. Podman Desktop offers to install one during onboarding.
Check what you have with:

```sh
podman compose version
```

If that prints `podman-compose`, you are set. If it reports no provider,
install it (`brew install podman-compose`, or `pip install podman-compose`).

Take `podman-compose`, not `docker-compose`. The two resolve relative paths
inside an included compose file differently, and `compose/` is written for
`podman-compose`. Podman prefers `docker-compose` when both are on your PATH,
and the stack then starts with empty config mounts.

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
2. `podman compose up -d` pulls the sixteen stack images and starts them.
3. `go run ./cmd/server` downloads the Go modules and compiles the service.
4. `vite dev` starts the console.

Steps 1 to 3 are the ones that need the network. Each writes into a cache on
your disk that survives going offline.

Step 2 takes the longest on a first run: Keycloak imports the realm before
anything that depends on it can start.

Then open **<http://127.0.0.1.nip.io:5173/aoh/dispatch/units>**. The console is
served on that domain rather than `localhost`, because the session cookie has
to be issued on a parent of `rtus-seh.127.0.0.1.nip.io` for the map to receive
updates.

**This is the checkpoint.** If you see the dispatch console with a list of
units, you are done and everything you need is now on your machine. If you do
not, fix it now rather than on the day: there is no second chance to download
anything.

Leave `pnpm start` running, or stop it with `Ctrl+C`. The database keeps
running either way.

### If the container images came as a tarball

Some downloads ship `ai-day-workshop-images.tar.gz` alongside the project. It
carries the stack images, so loading it skips the pulls in step 2:

```sh
podman load -i ai-day-workshop-images.tar.gz
```

Run that before `pnpm start`. It needs no network.

### If an init container times out on a proxy

`podman machine init` copies whatever proxy your machine was using at the time
into the VM, and Podman then sets it inside every container it starts. The
containers address each other by compose name, which no proxy can route, so the
run stops with something like:

```text
requests.exceptions.ProxyError: HTTPConnectionPool(host='192.168.88.2', port=3128):
Max retries exceeded with url: http://iams-keycloak:8080/realms/aoh/...
```

`pnpm start` names this before it starts the stack. Turn the inheritance off
once, then start again:

```sh
podman machine ssh 'sudo mkdir -p /etc/containers/containers.conf.d && printf "[containers]\nhttp_proxy = false\n" | sudo tee /etc/containers/containers.conf.d/99-no-container-proxy.conf'
```

This affects only what containers inherit. Podman still uses your proxy for its
own image pulls. `./scripts/install-prereqs.sh` applies it for you, so you hit
this only on a machine created before you ran the installer.

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
| `Ctrl+C` | stops the console and the service; the stack keeps running |
| `pnpm stop` | stops and removes the stack containers |
| `pnpm reset` | the same, and deletes the seeded volumes with it |

After `pnpm reset`, the next `pnpm start` re-imports the Keycloak realm and
re-seeds each database, so it takes as long as a first run.
