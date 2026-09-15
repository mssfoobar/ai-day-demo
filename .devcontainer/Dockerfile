# Workshop image: Node, pnpm, Go, Claude Code and both Go caches, preloaded.
#
# Node dependencies stay out. Six of them are under the private @mssfoobar scope and
# this image is public, so `pnpm install` runs in the container against ~/.npmrc.

FROM node:24-bookworm-slim

ARG GO_VERSION=1.25.14
ARG GO_SHA256_AMD64=a21ae5633a269bcd7e90cf767e48225633795e99d831742cbf3397064fee7712
ARG GO_SHA256_ARM64=9bf234ea70ffec9347fdf6b22ce4add51717d3386a38a441e8c8743fceb5eaee
ARG PNPM_VERSION=10.16.1
ARG TARGETARCH

# git for the repo, curl for the Go tarball, procps for the process-tree kill in dev.mjs.
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl git procps \
    && rm -rf /var/lib/apt/lists/*

RUN set -eux; \
    case "${TARGETARCH}" in \
      amd64) sha="${GO_SHA256_AMD64}" ;; \
      arm64) sha="${GO_SHA256_ARM64}" ;; \
      *) echo "unsupported architecture: ${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${TARGETARCH}.tar.gz" -o /tmp/go.tgz; \
    echo "${sha}  /tmp/go.tgz" | sha256sum -c -; \
    tar -C /usr/local -xzf /tmp/go.tgz; \
    rm /tmp/go.tgz; \
    rm -rf /usr/local/go/test /usr/local/go/api

# Login shells rebuild PATH from /etc/profile, dropping the ENV set below.
RUN printf 'export PATH=/usr/local/go/bin:/home/node/go/bin:$PATH\n' \
    > /etc/profile.d/go-path.sh

# World-readable corepack cache: prepared as root, read later as `node`.
ENV COREPACK_HOME=/usr/local/share/corepack
ENV COREPACK_ENABLE_DOWNLOAD_PROMPT=0
RUN corepack enable \
    && corepack prepare "pnpm@${PNPM_VERSION}" --activate \
    && chmod -R a+rX "${COREPACK_HOME}"

RUN npm install -g @anthropic-ai/claude-code && npm cache clean --force

ENV GOPATH="/home/node/go"
ENV GOMODCACHE="/home/node/go/pkg/mod"
ENV GOCACHE="/home/node/.cache/go-build"
ENV PATH="/usr/local/go/bin:/home/node/go/bin:${PATH}"
# Pins the toolchain to the one installed above; `go run` never fetches another version.
ENV GOTOOLCHAIN=local

USER node
WORKDIR /workspace

# Warms both Go caches at the path the bind mount will use, so the dependency tree is a
# cache hit on the first `go run`. `go mod download` would instead fetch the whole
# module graph, roughly four times the size.
COPY --chown=node:node go.work go.work.sum ./
COPY --chown=node:node packages/aoh-golib ./packages/aoh-golib
COPY --chown=node:node apps/dispatch-svc ./apps/dispatch-svc
RUN cd apps/dispatch-svc && go build ./...

# The bind mount supplies the real source at runtime.
RUN rm -rf /workspace/apps /workspace/packages /workspace/go.work /workspace/go.work.sum

CMD ["sleep", "infinity"]
