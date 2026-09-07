#!/usr/bin/env bash
# Bring up hub Nexus for a given profile.
#
# Usage: ./up.sh <profile>
#   profile: laptop | prod (prod not yet implemented — that's the Helm chart)
#
# Idempotent. Creates .env from the profile example on first run. Bootstraps
# admin password reset + repos on first run. Re-runs are safe.

set -euo pipefail

PROFILE="${1:-laptop}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ASSETS_DIR="$(cd "$SCRIPT_DIR/../assets" && pwd)"

case "$PROFILE" in
  laptop) ;;
  prod)
    echo "error: prod profile is not implemented — it is the Helm chart for K8s (AOH-6740)" >&2
    exit 64
    ;;
  *)
    echo "error: unknown profile '$PROFILE' (expected: laptop, prod)" >&2
    exit 64
    ;;
esac

ENV_EXAMPLE="$ASSETS_DIR/.env.${PROFILE}.example"
ENV_FILE="$ASSETS_DIR/.env"
COMPOSE_BASE="$ASSETS_DIR/compose.yml"
COMPOSE_PROFILE="$ASSETS_DIR/compose.${PROFILE}.yml"

[[ -f "$ENV_EXAMPLE" ]] || { echo "error: missing $ENV_EXAMPLE" >&2; exit 70; }
[[ -f "$COMPOSE_BASE" ]] || { echo "error: missing $COMPOSE_BASE" >&2; exit 70; }
[[ -f "$COMPOSE_PROFILE" ]] || { echo "error: missing $COMPOSE_PROFILE" >&2; exit 70; }

if [[ ! -f "$ENV_FILE" ]]; then
  echo "==> Creating .env from .env.${PROFILE}.example"
  cp "$ENV_EXAMPLE" "$ENV_FILE"
fi

# Pick a compose CLI. Prefer podman compose since the platform target is podman.
if command -v podman >/dev/null 2>&1; then
  COMPOSE=(podman compose)
elif command -v docker >/dev/null 2>&1; then
  COMPOSE=(docker compose)
else
  echo "error: neither podman nor docker found on PATH" >&2
  exit 69
fi

cd "$ASSETS_DIR"

echo "==> Starting Nexus (profile=$PROFILE)"
"${COMPOSE[@]}" --env-file "$ENV_FILE" \
  -f "$COMPOSE_BASE" -f "$COMPOSE_PROFILE" \
  up -d nexus

echo "==> Waiting for Nexus to be writable (first boot typically 60-180s; timeout at 5 min)"
# Probe the HTTP endpoint directly — /status/writable returns 200 only once
# the DB and blob store are fully initialised. On a cold start this takes
# 60-180s depending on disk + memory. Longer timeout than aoh-spoke's
# Forgejo (2 min) because Nexus is a much heavier JVM service.
# shellcheck disable=SC1090
source "$ENV_FILE"
for i in $(seq 1 60); do
  if curl -fsS -o /dev/null "http://localhost:${NEXUS_HTTP_PORT}/service/rest/v1/status/writable" 2>/dev/null; then
    echo "    writable"
    break
  fi
  sleep 5
  [[ $i -eq 60 ]] && { echo "error: Nexus did not become writable on :${NEXUS_HTTP_PORT} in 300s" >&2; exit 75; }
done

echo "==> Running bootstrap (idempotent)"
"$SCRIPT_DIR/bootstrap.sh" "$PROFILE"

# shellcheck disable=SC1090
source "$ENV_FILE"
echo
echo "==> Hub Nexus is up."
echo "    UI / API:            http://localhost:${NEXUS_HTTP_PORT}"
echo "    Admin:               admin / ${NEXUS_ADMIN_PASSWORD}"
echo "    Docker Hub proxy:    http://localhost:${NEXUS_DOCKER_PROXY_HOST_PORT}"
echo
echo "    Run ./scripts/check.sh to verify proxy resolution."
