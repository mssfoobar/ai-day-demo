#!/usr/bin/env bash
# Tear down hub Nexus. Preserves volumes by default.
#
# Usage: ./down.sh [--volumes]
#   --volumes    Also delete named volumes (full data wipe; use for a fresh start)

set -euo pipefail

WIPE_VOLUMES=0
for arg in "$@"; do
  case "$arg" in
    --volumes|-v) WIPE_VOLUMES=1 ;;
    *) echo "error: unknown flag '$arg'" >&2; exit 64 ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ASSETS_DIR="$(cd "$SCRIPT_DIR/../assets" && pwd)"

if command -v podman >/dev/null 2>&1; then
  COMPOSE=(podman compose)
elif command -v docker >/dev/null 2>&1; then
  COMPOSE=(docker compose)
else
  echo "error: neither podman nor docker found on PATH" >&2
  exit 69
fi

cd "$ASSETS_DIR"

# Find whichever profile overlay is present; there's only one at a time in
# practice, but be defensive.
PROFILE_FILES=()
for f in compose.laptop.yml compose.prod.yml; do
  [[ -f "$f" ]] && PROFILE_FILES+=(-f "$f")
done

ENV_FLAG=()
[[ -f .env ]] && ENV_FLAG=(--env-file .env)

# macOS ships bash 3.2, which treats `"${arr[@]}"` on an empty array as
# "unbound" under `set -u`. The `${arr[@]+...}` form expands to nothing when
# arr is unset or empty, which is what we want.
if [[ $WIPE_VOLUMES -eq 1 ]]; then
  echo "==> Stopping hub Nexus + wiping volumes"
  "${COMPOSE[@]}" ${ENV_FLAG[@]+"${ENV_FLAG[@]}"} -f compose.yml ${PROFILE_FILES[@]+"${PROFILE_FILES[@]}"} down --volumes
else
  echo "==> Stopping hub Nexus (volumes preserved)"
  "${COMPOSE[@]}" ${ENV_FLAG[@]+"${ENV_FLAG[@]}"} -f compose.yml ${PROFILE_FILES[@]+"${PROFILE_FILES[@]}"} down
fi
