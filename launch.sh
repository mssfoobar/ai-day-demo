#!/usr/bin/env bash
# One-shot bootstrap + run for macOS / Linux.
#
#   ./launch.sh              # check prerequisites, install, start db + service + console
#   ./launch.sh --setup-only # check + install, don't start
#
# Everything real happens in scripts/setup.mjs (Node). This wrapper only exists so a fresh
# clone works without pnpm on PATH yet: corepack ships with Node and activates the pinned
# pnpm version for us.
set -euo pipefail
cd "$(dirname "$0")"

if ! command -v node >/dev/null 2>&1; then
  echo "Node.js >= 24 is required and was not found. Install it from https://nodejs.org, then rerun." >&2
  exit 1
fi

corepack enable >/dev/null 2>&1 || true

if [[ "${1:-}" == "--setup-only" ]]; then
  exec node scripts/setup.mjs
else
  exec node scripts/setup.mjs --start
fi
