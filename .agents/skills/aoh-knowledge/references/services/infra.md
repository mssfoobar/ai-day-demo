# Infra — Traefik Reverse Proxy

## What It Does

Traefik is the API gateway and reverse proxy for all AOH services. Every HTTP request —
whether from a browser, your app, or an AOH service's web UI — enters through Traefik on
port 80 and is routed to the correct container by hostname.

It is always required. All other services register themselves with Traefik using Docker
labels; without it, nothing is reachable via `*.${DEV_DOMAIN}`.

## Architecture

```
Internet / localhost
       │
  port 80 (web)
  port 5333 (sdstcp)
       │
  ┌────▼────┐
  │ Traefik │  ← reads Docker/Podman labels to discover services
  └────┬────┘
       │  host-based routing
  ┌────┴────────────────────────────────────┐
  │                                         │
  ▼                                         ▼
iams-keycloak.${DEV_DOMAIN}         gis.${DEV_DOMAIN}
rtus-seh.${DEV_DOMAIN}              ...etc
```

## Key Concepts

### Host-based routing

Each service declares its own routing rule via Docker Compose labels:

```yaml
labels:
  - traefik.enable=true
  - traefik.http.routers.<name>.rule=Host(`<name>.${DEV_DOMAIN}`)
  - traefik.http.routers.<name>.entrypoints=web
  - traefik.http.routers.<name>.middlewares=permissive-headers@docker
  - traefik.http.services.<name>.loadBalancer.server.port=8080
```

- `traefik.enable=true` — opt-in (Traefik ignores containers without this label)
- The rule uses `${DEV_DOMAIN}` which defaults to `127.0.0.1.nip.io` in dev
- `loadBalancer.server.port` must match the port the container listens on (most AOH services use 8080)

### Entry points

| Entry point | Port | Used for |
|-------------|------|----------|
| `web` | 80 | All HTTP traffic (services, web UIs, Keycloak) |
| `sdstcp` | 5333 | SDS TCP transport (session store) |

### CORS middleware

Traefik defines a global `permissive-headers@docker` middleware that allows all origins,
all methods, and credentials. This is applied to every AOH service except RTUS-SEH, which
uses its own stricter CORS (SSE with credentials requires explicit origin whitelisting).

```yaml
# Defined in infra compose as a Docker label on the Traefik container
traefik.http.middlewares.permissive-headers.headers.accessControlAllowOriginList=*
traefik.http.middlewares.permissive-headers.headers.accessControlAllowCredentials=true
traefik.http.middlewares.permissive-headers.headers.accessControlAllowHeaders=*
traefik.http.middlewares.permissive-headers.headers.accessControlAllowMethods=*
```

### Docker vs Podman

The default Traefik config mounts the **Podman** socket:

```yaml
volumes:
  - /run/user/1000/podman/podman.sock:/var/run/docker.sock:ro
```

Docker users must override this via `compose.override.yml`:

```yaml
services:
  traefik:
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
```

## Access

| URL | What |
|-----|------|
| `http://traefik.${DEV_DOMAIN}` | Traefik dashboard (service map, routes, health) |

## Adding a new service

To make your own service reachable via Traefik, add these labels to its compose definition:

```yaml
services:
  my-service:
    labels:
      - traefik.enable=true
      - traefik.http.routers.my-service.rule=Host(`my-service.${DEV_DOMAIN}`)
      - traefik.http.routers.my-service.entrypoints=web
      - traefik.http.routers.my-service.middlewares=permissive-headers@docker
      - traefik.http.services.my-service.loadBalancer.server.port=8080
```

Then access your service at `http://my-service.${DEV_DOMAIN}`.

## Dependencies

- None. Traefik is the foundation layer and has no AOH dependencies.
