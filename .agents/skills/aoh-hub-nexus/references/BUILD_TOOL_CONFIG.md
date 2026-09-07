# Build tool configuration

How to point each supported build tool at hub Nexus. URLs below assume the laptop profile (`http://localhost:8181`, docker on `8182`) and the default admin creds (`admin:aohadmin`). For the prod profile, substitute the real hub hostname and an IAMS-issued user token.

**Anonymous access is disabled** by bootstrap, so every client below needs credentials — even read-only fetches. Swap the `admin:aohadmin` shown in examples for a scoped Nexus user once you're past the initial dress rehearsal.

**Rule of thumb:** configure your build tool against the **group** repository for a given format, not the proxy or hosted directly. The group transparently resolves through both (hosted-first, so internally published packages shadow proxied ones when names collide).

## Maven

`~/.m2/settings.xml`:

```xml
<settings>
  <mirrors>
    <mirror>
      <id>nexus</id>
      <mirrorOf>*</mirrorOf>
      <url>http://localhost:8181/repository/maven-group/</url>
    </mirror>
  </mirrors>
  <servers>
    <server>
      <id>nexus</id>
      <username>admin</username>
      <password>aohadmin</password>
    </server>
  </servers>
</settings>
```

To publish to `maven-hosted` (deploy goal), add a `<distributionManagement>` block to the project POM pointing at `http://localhost:8181/repository/maven-hosted/`.

## npm

Project `.npmrc` (or `~/.npmrc`). Anonymous reads are blocked; embed basic-auth creds in the registry URL or generate a token (next subsection):

```
registry=http://admin:aohadmin@localhost:8181/repository/npm-group/
```

To publish to `npm-hosted`:

```
# .npmrc (project-local)
@mssfoobar:registry=http://localhost:8181/repository/npm-hosted/
//localhost:8181/repository/npm-hosted/:_authToken=NpmToken.<token>
```

Generate the token via `npm login --registry=http://localhost:8181/repository/npm-hosted/` (basic-auth backed; Nexus stores it).

## PyPI

`~/.config/pip/pip.conf` (or `pip.conf` / `pip.ini`). pip takes creds inline in the URL:

```ini
[global]
index-url = http://admin:aohadmin@localhost:8181/repository/pypi-group/simple/
```

For Poetry:

```toml
# pyproject.toml
[[tool.poetry.source]]
name = "nexus"
url = "http://localhost:8181/repository/pypi-group/simple/"
priority = "primary"
```

To publish to `pypi-hosted` via `twine`:

```bash
twine upload --repository-url http://localhost:8181/repository/pypi-hosted/ dist/*
```

## Go modules

```bash
# Go's module fetcher accepts creds inline in GOPROXY.
export GOPROXY=http://admin:aohadmin@localhost:8181/repository/go-proxy/,direct
# Keep GOSUMDB pointing at the real sum db — Nexus doesn't cache that:
export GOSUMDB=sum.golang.org
# Or turn it off entirely for air-gap:
# export GOSUMDB=off
```

Go has no "hosted" format in Nexus; the platform team publishes internal Go modules to Forgejo's built-in Go registry instead (per the Nexus/Forgejo split).

## APT (Ubuntu)

`/etc/apt/sources.list.d/nexus.list`. APT respects the `http://user:pass@host/` form for basic auth:

```
deb http://admin:aohadmin@localhost:8181/repository/apt-ubuntu-noble-proxy/ noble main restricted universe multiverse
deb http://admin:aohadmin@localhost:8181/repository/apt-ubuntu-noble-proxy/ noble-updates main restricted universe multiverse
```

If the upstream has signed repositories (archive.ubuntu.com does), import the signing key the same way as normal APT setup — Nexus proxies the signed Release file and index untouched. It does **not** re-sign.

## YUM (RHEL / Rocky)

`/etc/yum.repos.d/nexus-rocky.repo`:

```ini
[nexus-rocky-baseos]
name=Nexus proxied Rocky 9 BaseOS
baseurl=http://admin:aohadmin@localhost:8181/repository/yum-rocky-proxy/
gpgcheck=1
gpgkey=https://download.rockylinux.org/pub/rocky/RPM-GPG-KEY-Rocky-9
```

## Docker

Docker's client doesn't support path-based routing for registries, so the Docker Hub proxy uses its own port (`8182` in the laptop profile).

Authenticate first — anonymous pulls against the docker-hub-proxy return 401 because bootstrap disables anonymous access globally:

```bash
# One-time login (credentials cached in ~/.docker/config.json).
# macOS note: the Docker Desktop keychain integration sometimes fails for
# plain-HTTP localhost registries — if you hit a "credential helper" error,
# remove credsStore from ~/.docker/config.json and retry.
docker login localhost:8182 -u admin
# password: aohadmin  (or whatever you set NEXUS_ADMIN_PASSWORD to)

# Pull through the proxy:
docker pull localhost:8182/library/alpine:3.19
docker pull localhost:8182/mssfoobar/my-image:latest   # → proxies to docker.io/mssfoobar/my-image
```

To use the proxy as a default mirror, add it to Docker / Podman's config:

```json
# ~/.config/containers/registries.conf  (Podman)
[[registry]]
location = "docker.io"
[[registry.mirror]]
location = "localhost:8182"
insecure = true     # laptop profile only; prod profile will use TLS
```

**Internal images do NOT go here** — they go to Harbor. Nexus's Docker support in the AOH stack is strictly pull-through caching of Docker Hub; anything you build yourself publishes to Harbor.

## Raw (generic file artifacts)

For uploading a tarball/bundle to `raw-hosted`:

```bash
curl -u admin:aohadmin --upload-file release-v1.2.3.tar.gz \
  http://localhost:8181/repository/raw-hosted/platform/release-v1.2.3.tar.gz
```

Fetching:

```bash
curl -fsS http://localhost:8181/repository/raw-hosted/platform/release-v1.2.3.tar.gz -O
```

Useful for: offline bundles, model weight snapshots, arbitrary file artifacts that don't fit a language-registry format.
