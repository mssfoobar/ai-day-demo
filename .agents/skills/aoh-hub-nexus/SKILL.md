---
name: aoh-hub-nexus
description: >
  Stand up an AOH hub Nexus OSS — the platform-team-operated package registry that
  proxies the public registries (Maven Central, npm, PyPI, proxy.golang.org, APT,
  YUM, Docker Hub) and hosts internal cross-project packages. Supports two profiles:
  `laptop` for a platform developer dress-rehearsing Nexus config on their own
  machine (compose + plaintext HTTP + local admin), and `prod` for the eventual
  Kubernetes deployment via Helm (stub — not yet implemented). Use this whenever the
  user wants to bring up hub Nexus, prototype proxy-repo layouts, test pull-through
  caching, validate the public-registry proxy set for a project stack, or rehearse
  internal package publishing. Trigger keywords "nexus", "hub nexus", "package
  registry", "proxy registry", "maven proxy", "npm proxy", "pypi proxy", "docker
  pull-through", "air-gap packages".
license: Proprietary
metadata:
  owner: AOH Platform Team
  status: v1-laptop-only
  not-for-production: "true"
---

# AOH Hub Nexus

Brings up hub Nexus OSS using compose. Nexus is the platform-team-operated registry
that does two jobs:

1. **Proxy** the public registries our stacks consume (Maven Central, npmjs, PyPI,
   `proxy.golang.org`, APT, YUM) and Docker Hub (pull-through — **no hosted Docker
   repos**; internal images belong in Harbor, not Nexus).
2. **Host** internal cross-project packages the platform team publishes for all
   projects to consume (shared Maven BOMs, common npm utilities, etc.).

Per-project internal packages live in that project's Forgejo built-in package
registry, not here. That split is deliberate: Forgejo has per-project RBAC and sits
next to the code; Nexus-hosted is for the small set of things that cross project
boundaries.

## Profiles

| Profile | Audience | Use case | State |
|---|---|---|---|
| `laptop` | Platform dev | Dress-rehearse Nexus config, test proxy resolution, prototype the repo set | Implemented |
| `prod`   | Platform-team SRE | Kubernetes deployment via Helm, real TLS via Traefik, IAMS OIDC, PV-backed storage, HA-ish | Not yet implemented — tracked in [AOH-6740](https://linear.app/ptd/issue/AOH-6740) |

**Laptop profile uses compose; prod profile will use Helm on K8s.** This parallels
`aoh-spoke`, but where the spoke runs on compose in both profiles (per AOH-6759),
hub Nexus uses Kubernetes for the production deployment (per AOH-6735 hard stack).
The laptop profile is a *development iteration surface* for the config that
eventually ships as a Helm chart, not a deployment target.

## Usage (laptop profile)

From the skill directory:

```bash
cd assets
cp .env.laptop.example .env    # or let ./scripts/up.sh do this on first run
../scripts/up.sh laptop        # pulls image, waits healthy, bootstraps admin + repos
../scripts/check.sh            # smoke test: api healthy, repos exist, proxy resolves
```

After `up.sh` finishes:
- Nexus web UI: http://localhost:8181
- Default admin: `admin` / `aohadmin` (change for anything non-laptop)
- Docker Hub pull-through endpoint: http://localhost:8182 (host-side port only;
  configurable via `NEXUS_DOCKER_PROXY_HOST_PORT`)

To tear down:

```bash
../scripts/down.sh             # stop container, keep volume
../scripts/down.sh --volumes   # stop + wipe all data (fresh start)
```

## What `up.sh` does on first run

1. Creates `.env` from `.env.laptop.example` if missing
2. `podman compose -f compose.yml -f compose.laptop.yml up -d nexus`
3. Waits for Nexus to be writable (typically ~1-3 min on first boot; hard timeout 5 min)
4. Runs `bootstrap.sh`:
   - Reads the auto-generated admin password from `/nexus-data/admin.password`
     inside the container
   - Resets admin password to `${NEXUS_ADMIN_PASSWORD}` from `.env`
   - Disables anonymous access
   - Creates the proxy repositories for the ecosystems in `${NEXUS_PROXY_FORMATS}`
     (default: maven, npm, pypi, go, apt, yum, docker)
   - Creates hosted repositories for internal publishing (maven, npm, pypi, raw)
   - Creates group repositories aggregating proxy + hosted
   - Prints a pointer to the UI for manual cleanup-policy setup (that REST
     endpoint is Nexus Pro only — OSS forces a UI step)

`up.sh` is idempotent — safe to re-run. Bootstrap steps skip if the target already
exists (admin password already reset, repo already present).

## Testing proxy resolution end-to-end

Anonymous access is disabled by bootstrap, so curl calls need `-u`. For Docker, the client's own token dance authenticates separately — use `docker login localhost:8182` first, or the insecure-registry-mirror config in `references/BUILD_TOOL_CONFIG.md`.

```bash
# npm — resolves the `is-number` package through the proxy
curl -fsS -u admin:aohadmin http://localhost:8181/repository/npm-proxy/is-number | head -c 200

# Maven Central — resolves commons-lang3 metadata
curl -fsS -u admin:aohadmin http://localhost:8181/repository/maven-central-proxy/org/apache/commons/commons-lang3/maven-metadata.xml

# Docker Hub pull-through (after `docker login localhost:8182`)
docker pull localhost:8182/library/alpine:3.19
```

The group repositories are what developers actually configure in their build
tools — they aggregate the proxy and the hosted repos for a given format:

```
# .npmrc
registry=http://localhost:8181/repository/npm-group/

# ~/.m2/settings.xml
<mirror>
  <id>nexus</id>
  <url>http://localhost:8181/repository/maven-group/</url>
  <mirrorOf>*</mirrorOf>
</mirror>

# Go
export GOPROXY=http://localhost:8181/repository/go-proxy/,direct

# pip
pip install --index-url http://localhost:8181/repository/pypi-group/simple/ <pkg>
```

See `references/BUILD_TOOL_CONFIG.md` for the full matrix.

## What this skill explicitly does NOT cover

- TLS / proper auth (laptop profile is plaintext HTTP on localhost; OIDC deferred
  until dev-side SSO story lands — see DEFERRED below)
- Kubernetes deployment, Helm chart, Traefik ingress, persistent volume claims
- IAMS / Keycloak OIDC integration
- OpenBao-backed secrets (laptop profile keeps admin password in `.env`)
- Backup + restore
- Replication / air-gap bundle seeding (tracked in AOH-6753)
- Disk-quota enforcement beyond the single default cleanup policy

## Deferred decisions

- **Dev-side SSO.** The application IAMS/Keycloak is scoped to the apps projects
  ship, not to the dev-tooling plane. Whether hub Nexus (+ hub Forgejo, Harbor,
  Backstage) federate against a *separate* dev Keycloak or stay with local admin
  accounts is an open question. The `prod` profile will wire OIDC once this is
  decided.
- **Docker format as pull-through only.** Nexus exposes a Docker proxy connector
  (`NEXUS_DOCKER_PROXY_HOST_PORT`, default 8182) for caching Docker Hub; internal
  container images live in Harbor (AOH-6739). Do not enable Nexus-hosted Docker
  repos without revisiting the Harbor/Nexus split.
- **Forgejo built-in vs Nexus-hosted for internal publishing.** Per AOH-6735
  "Committed via this analysis", both are accepted: Forgejo built-in for
  per-project internal packages (close to code, per-project RBAC free);
  Nexus-hosted only for platform-team-published cross-project shared artifacts.

## Common operations

See `references/TROUBLESHOOTING.md` for known issues and fixes.
See `references/BUILD_TOOL_CONFIG.md` for the per-ecosystem client configuration.
Production build-out (Helm chart, OIDC, TLS, K8s) is the `prod` profile — tracked
in [AOH-6740](https://linear.app/ptd/issue/AOH-6740); `references/OPERATING.md`
will land with that work.
