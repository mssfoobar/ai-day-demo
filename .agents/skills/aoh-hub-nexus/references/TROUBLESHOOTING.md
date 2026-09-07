# Troubleshooting

## Nexus container won't become writable

First boot is slow (60–180s) and memory-sensitive. Check logs: `podman logs aoh-hub-nexus`. Common causes:

- **OOM kill during startup**: the laptop profile sets `-Xms512m -Xmx1g` which is tight. If the container exits during bootstrap or shows OOM, raise `INSTALL4J_ADD_VM_PARAMS` in `compose.laptop.yml` (or move to the prod profile once it lands).
- **Port already in use**: another service is on `NEXUS_HTTP_PORT` (default 8181) or `NEXUS_DOCKER_PROXY_HOST_PORT` (default 8182). Change them in `.env` and re-run `./scripts/up.sh laptop`.
- **Stale volume from an older run**: `./scripts/down.sh --volumes` then `./scripts/up.sh laptop` to wipe and rebootstrap. Usually the fastest fix for mysterious state.
- **Container runtime not running**: on macOS, `podman machine start` before running `./scripts/up.sh`.

## Bootstrap fails with "cannot authenticate as admin"

Happens when `/nexus-data/admin.password` no longer exists (past first run) AND `NEXUS_ADMIN_PASSWORD` in `.env` doesn't match what admin was actually changed to. Two ways out:

- **Know the current admin password**: set `NEXUS_ADMIN_PASSWORD` in `.env` to the real value, re-run `./scripts/up.sh laptop`. bootstrap.sh will no-op and move on.
- **Don't know / don't care**: `./scripts/down.sh --volumes && ./scripts/up.sh laptop` — wipes Nexus entirely, starts fresh.

## A proxy repo resolves with 404 even though bootstrap said "created"

The proxy repo shell exists but the upstream isn't reachable, or the URL is wrong for this format.

- Check the proxy's remote URL in the UI (Administration → Repositories) — compare against the expected value for the format. Nexus format conventions are picky: Maven wants a `/maven2/` suffix, npm wants the bare registry URL.
- For `apt-proxy`: confirm `NEXUS_APT_DISTRIBUTION` matches a distribution actually served by `NEXUS_APT_REMOTE_URL`. Ubuntu archive doesn't serve every codename.
- Hit the upstream directly from inside the container to rule out network:
  `podman exec aoh-hub-nexus curl -fsI https://registry.npmjs.org`
  If that fails, your machine can't reach the registry — fix networking, not Nexus.

## Docker pull-through proxy returns 500 / can't pull

Docker clients need the proxy reachable on its dedicated port — 8182 by default, separate from the main 8181 UI port. Anonymous access is disabled by bootstrap, so you need `docker login localhost:8182` once before pulling; an un-authed pull returns 401 with a bearer-realm challenge that looks cryptic if you're not expecting it.

```bash
docker login localhost:8182 -u admin    # password = NEXUS_ADMIN_PASSWORD from .env
docker pull localhost:8182/library/alpine:3.19
```

If that still fails:
- Confirm `compose.laptop.yml` exposes 8182 and `bootstrap.sh` created `docker-hub-proxy` with `httpPort: 8182`. Nexus listens on that connector only if both agree.
- Check logs for the Docker proxy: look in the Nexus UI at Logging → Main log, filter for "docker".

## Docker pulls fail with 429 toomanyrequests

Docker Hub rate-limits anonymous pulls to 100 per 6-hour window per source IP. When the Nexus docker-hub-proxy runs without Docker Hub credentials (the default), every cache-miss pull is anonymous from Docker Hub's view — so on a shared machine or CI box, you will hit the cap.

Fix: add a Docker Hub account to the proxy's HTTP Client config.

1. UI → Repositories → `docker-hub-proxy` → HTTP → Authentication type: Username.
2. Enter a Docker Hub username + access token (generate at hub.docker.com → Account Settings → Security).
3. Save. Authenticated pulls give the proxy 200/6h (free tier) or unlimited (paid tier).

The prod Helm chart will mount these credentials from OpenBao (AOH-6742). Laptop profile is fine to point at your personal Docker Hub account while you're testing.

## `docker pull` through the proxy fails with `blob unknown`

Bootstrap's docker-hub-proxy config sets `cacheForeignLayers: false` and an empty `foreignLayerUrlWhitelist`. That's the strict default and works for most Linux images (alpine, ubuntu, nginx, postgres, etc.), but rejects images that reference *foreign layers* — notably Windows base images and some multi-arch images with layers hosted outside Docker Hub's own CDN.

Symptom: `docker pull localhost:8182/<image>` errors with `blob unknown to registry` or `manifest unknown`, and the Nexus log shows `Not allowed to cache foreign layer`. Windows base images use foreign layers deliberately (Microsoft hosts the base layers outside Docker Hub's CDN and marks them with a foreign-layer media type); Linux images typically do not.

Fix (UI): Repositories → `docker-hub-proxy` → Docker Proxy section → tick `Allow Nexus Repository Manager to download and cache foreign layers`, optionally add `.*` to the whitelist. Disk footprint roughly doubles for Windows images; acceptable on laptop, revisit for prod.

## Cleanup policies are a manual UI step in OSS

Cleanup policy REST endpoints (`/service/rest/v1/cleanup-policies`) are a Nexus **Pro** feature — they do not exist in Nexus OSS (verified against the OSS OpenAPI spec for 3.79.0; any POST returns 404, not 401). Bootstrap.sh therefore prints a UI pointer and skips automation.

Configure manually in: Administration → Repository → Cleanup Policies → Create. Suggested default for proxies:

- Name: `default-proxy-cleanup`
- Format: `*`
- Component age criteria: `Last downloaded` > 90 days

Then edit each proxy repo and select this policy from its cleanup dropdown.

The prod-profile Helm chart (AOH-6740) will revisit this via a pre-seeded `nexus.properties` mounted from a ConfigMap, so production hub Nexus has cleanup configured without the UI dance. Laptop profile stays manual — it's a one-time operator action.

## Starting over from scratch

```bash
./scripts/down.sh --volumes
rm -f assets/.env
./scripts/up.sh laptop
```

This wipes Nexus's database, blob store, and all proxy caches. First-run bootstrap takes 2–3 minutes again.

## Why is this separate from `aoh-spoke`?

`aoh-spoke` is the CM-operated project server (compose). `aoh-hub-nexus` is a platform-team-operated service that will eventually run on K8s via Helm. The laptop profile here exists to prototype the Nexus configuration (proxy set, group layout, cleanup policy) on a dev's machine before it ships as a Helm chart — same pattern as aoh-spoke's laptop profile vs team profile, but the "real" deployment of hub Nexus is K8s, not compose. See AOH-6740 for the Helm chart work.
