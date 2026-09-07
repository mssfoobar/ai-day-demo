# syntax=docker/dockerfile:1
#
# Build from this service directory. Run `go mod tidy` first so go.sum exists.
# aoh-golib is a private module — build with ONE of:
#
#   1. Access token with repo read access, in a file (`.secrets/` is gitignored):
#        podman build --secret id=access_token,src=.secrets/access_token -f Dockerfile .
#        docker  build --secret id=access_token,src=.secrets/access_token -f Dockerfile .
#
#   2. Registry mirror (air-gapped spokes):
#        podman build --build-arg GOPROXY=<mirror-url> --build-arg GOSUMDB=off -f Dockerfile .
FROM --platform=$BUILDPLATFORM golang:{GO_VERSION} AS builder

ARG GOPROXY
ARG GOSUMDB
ARG GONOSUMDB=github.com/mssfoobar/*
ENV GOWORK=off CGO_ENABLED=0 \
    GOPROXY=${GOPROXY} GOSUMDB=${GOSUMDB} GONOSUMDB=${GONOSUMDB}
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=secret,id=access_token,required=false \
    if [ -f /run/secrets/access_token ]; then \
      printf 'machine github.com login x-access-token password %s\n' \
        "$(cat /run/secrets/access_token)" > /root/.netrc; \
    fi; \
    go mod download; rc=$?; rm -f /root/.netrc; exit $rc

# Service source.
COPY internal/ internal/
COPY schema/ schema/
COPY cmd/ cmd/
COPY docs/ docs/

ARG TARGETOS TARGETARCH
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -v -a -o /build/server ./cmd/server

FROM alpine:{ALPINE_VERSION}

LABEL org.opencontainers.image.title="__AOH_NAME__"

RUN addgroup -g 1000 app && \
    adduser -u 1000 -G app -D app && \
    mkdir /app && \
    chown -R app:app /app
USER app

WORKDIR /app

COPY --from=builder /build/server .
COPY --from=builder --chown=app:app /src/schema/ schema/

CMD ["/app/server"]
