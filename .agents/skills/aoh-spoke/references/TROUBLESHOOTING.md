# Troubleshooting a running spoke

Diagnostics for spokes stood up via `aia spoke up`. Commands below assume the
CM's spoke lives at `~/.aoh/spokes/<name>/` — adjust paths if a different
spoke dir was picked at `aia spoke init` time.

For finding the right container: every fragment labels its services
`aoh.service=<name>`, so `podman ps --filter label=aoh.service=forgejo` works
independent of compose project prefixes.

## Bring-up failures (spoke never becomes ready)

### `aia spoke up` exits non-zero, "Nexus bootstrap failed"

Nexus's bootstrap exits 78 when any repo create hit a fatal-class HTTP code
(401/5xx/000), not on 400 validation warnings. If you see it:

- `podman logs aoh-spoke-nexus` — Nexus may be crash-looping (OOM during
  first-boot is common on memory-tight dev laptops; raise
  `INSTALL4J_ADD_VM_PARAMS` in `services/nexus/compose.laptop.yml` or
  `.env`).
- Check `aia spoke status` — if Nexus shows as unhealthy even though bootstrap
  ran, the first-run JVM boot may not have finished. Default timeout is 5 min;
  retry `aia spoke up` if you hit that.

### `aia spoke up` exits "cannot authenticate as admin and /nexus-data/admin.password is absent"

Someone changed the Nexus admin password via the UI but `.env` still has the
old one. Either (a) update `NEXUS_ADMIN_PASSWORD` in `~/.aoh/spokes/<name>/.env`
to match what was set, or (b) `aia spoke down --volumes && aia spoke up` to
wipe Nexus and re-bootstrap.

### Forgejo container exits before becoming healthy

`podman logs aoh-spoke-forgejo`. Most common causes:

- **Postgres not reachable**: Forgejo's `depends_on: postgres (service_healthy)`
  should prevent this, but if the healthcheck lies (e.g. Postgres healthy
  but DB doesn't yet exist), Forgejo fails to connect. Check
  `podman logs aoh-spoke-postgres` — the `10-create-databases.sh` init
  should have logged `Creating database 'forgejo'`. If missing,
  `POSTGRES_DATABASES` was blank at postgres first-boot. Recreate postgres
  via `aia spoke down postgres --volumes && aia spoke up`.
- **Port already in use**: `FORGEJO_HTTP_PORT` or `FORGEJO_SSH_PORT`
  collision with another process on the host. Change in `.env` and
  `aia spoke up`.
- **Stale volume from an older spoke**: `aia spoke down --volumes` then
  `aia spoke up`. Cleans the Forgejo database, runner registration, and
  any mirrored repos.
- **Container runtime not running**: on macOS, `podman machine start`.

## Runner issues

### Runner container is up but no jobs execute

Check `podman logs aoh-spoke-runner` for "declared successfully" — if missing,
the runner didn't complete registration with Forgejo. Re-run bootstrap:

```bash
aia spoke bootstrap act-runner
```

If that doesn't help: the `/data` volume and Forgejo's admin-user state may
have diverged (volume survived but Forgejo DB got wiped). The act-runner
bootstrap detects this and re-registers on the next `aia spoke up`; if
still stuck, `aia spoke down --volumes` for a clean restart.

### Workflows stuck "running" with no step logs (laptop profile)

Laptop profile uses the `host` executor with the default
`data.forgejo.org/forgejo/runner:6` image, which has `git` but NOT `node`,
`curl`, or most CI tooling. Any JS action (`actions/checkout@v4`,
`astral-sh/setup-uv@v5`, `actions/setup-python@v5`) will dispatch but silently
fail to execute.

Three fixes in ascending effort:

1. Write workflows that use only shell steps (shell-only smoke tests are
   fine; production CI is not).
2. Switch to the `docker` executor — isolated per-job containers with a
   pre-baked tooling image (`catthehacker/ubuntu:act-latest`). Standard
   act_runner pattern.
3. Build a custom runner image: `FROM data.forgejo.org/forgejo/runner:6`
   and `RUN apk add nodejs npm curl`. Lower effort than (2) on
   macOS+podman.

Team profile (AOH-6854) will ship the docker executor + a tooling-rich
image by default — laptop's `host` executor is explicitly for smoke-testing
the wiring, not for running real CI.

### Runner "no space left on device" / permission denied

`host` executor reuses the runner container filesystem across jobs;
tooling accumulates. Easy fix: `aia spoke down --volumes && aia spoke up`.
Permanent fix: switch to docker executor (see above).

### Git clone in a workflow fails with 401 or hangs

Forgejo's default config rejects anonymous HTTP git clones even for repos
marked `private=false` (the `private` flag controls visibility, not
anonymous-access). Options:

- Use `actions/checkout@v4` if you've solved the runner-tooling issue
  above (auto-injects `GITHUB_TOKEN`).
- Pass the token explicitly:
  ```yaml
  env:
    GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  run: |
    git -c "http.extraheader=Authorization: Basic $(printf 'x-access-token:%s' "$GH_TOKEN" | base64)" \
        fetch --depth=1 origin "$GITHUB_SHA"
  ```

## Nexus issues

### npm/pip/go install returns 401 from Nexus

Nexus has anonymous access disabled by bootstrap (by design). Clients
need auth. For laptop, embed credentials inline:

```
# .npmrc
registry=http://admin:aohadmin@localhost:8181/repository/npm-group/
```

See `deploy/spoke-services/nexus/` for per-ecosystem config examples (the
existing BUILD_TOOL_CONFIG.md in `skills/aoh-hub-nexus/` applies to
spoke Nexus too; the skill's URL examples use port 8181/8182 by default).

### Docker pull-through returns 500 or 401

- Clients need `docker login localhost:8182 -u admin` once before pulling.
  Anon pulls are blocked.
- Rate-limited pulls show as 429 toomanyrequests — Nexus proxy needs a
  Docker Hub account configured in the UI (Repositories →
  docker-hub-proxy → HTTP → Authentication).
- Windows base images fail with `blob unknown` — the proxy has
  `cacheForeignLayers: false` by default. Enable in the UI.

## Postgres / shared-DB issues

### A service reports "database does not exist"

`POSTGRES_DATABASES` was empty when Postgres first started, so the init
script skipped creation. This usually means a service was added to the
spoke AFTER Postgres was already running, so Postgres's `first-boot`
hook never re-fires.

Fix: stop-and-recreate Postgres:
```bash
aia spoke down postgres --volumes
aia spoke up
```

Other services will reconnect once Postgres comes back up with the
correct `POSTGRES_DATABASES` list.

(This is a sharp edge of the single-shared-Postgres design. Future
improvement: drive database creation from a post-start hook that runs
against a running Postgres, so adding a service doesn't require a full
Postgres wipe.)

### Postgres healthy but authentication fails

`POSTGRES_PASSWORD` in `.env` doesn't match what Postgres was initialised
with. The password baked in on first boot is permanent until the volume
is wiped. Either revert `.env` to match, or `aia spoke down postgres
--volumes && aia spoke up`.

## Backstage issues

### Backstage container won't start

The default fragment uses a *demonstration* Backstage image. That image
exists, but community-maintained Backstage Docker images are unstable
targets — they break across versions. If yours is broken:

- Set `BACKSTAGE_IMAGE` in `.env` to a known-good tag, or
- Build your own with `yarn create app` + `yarn build-image` (the real
  deployment path; AOH-6764 will ship a project-specific build pipeline).

### Backstage up but catalog is empty

Expected for a fresh boot — the default Backstage catalog is empty until
entities are registered. The bootstrap script is a no-op for the stub
image; a real build would seed an initial catalog.

## "Starting over from scratch"

```bash
aia spoke down --volumes
rm -f ~/.aoh/spokes/<name>/.env ~/.aoh/spokes/<name>/.env.shared
aia spoke up
```

This wipes all service databases, runner registrations, Nexus blob
storage, Backstage data, and the CM's env selections. `aia spoke init`
is re-prompted on the next `up`.
