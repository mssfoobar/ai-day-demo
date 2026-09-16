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

Download both files from the workshop link:

| File | |
| --- | --- |
| `ai-day-demo.zip` | the project |
| `ai-day-workshop-images.tar.gz` | the container images, 451MB |

Keep both in the same folder and unzip the project there, so you end up with
this. Everything below runs from inside `ai-day-demo`.

```
somewhere-you-can-find-again/
├── ai-day-demo/
└── ai-day-workshop-images.tar.gz
```

If you put the tarball elsewhere, use its full path in the load step below
rather than `../`.

## Stage 1: on your own internet

Load the images. This takes a minute and needs no network.

```sh
podman load -i ../ai-day-workshop-images.tar.gz
```

Start the container and the database:

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

Then open **<http://localhost:5173>**.

**This is the checkpoint.** If you see the dispatch console with a list of
units, you are done and everything you need is now on your machine. If you do
not, fix it now rather than on the day: there is no second chance to download
anything.

Leave `pnpm start` running, or stop it with `Ctrl+C`. The database keeps
running either way.

## Stage 2: on the workshop network

Join the workshop wifi, then:

```sh
podman compose -f compose/compose.yml -f compose/compose.devcontainer.yml up -d
podman compose -f compose/compose.yml -f compose/compose.devcontainer.yml exec workshop bash
pnpm start
```

In a second terminal, inside the container, confirm the model is reachable:

```sh
pnpm doctor
```

All four checks should pass. Then start the agent:

```sh
claude
```

No login and no API key: it is already pointed at the workshop's model server.
