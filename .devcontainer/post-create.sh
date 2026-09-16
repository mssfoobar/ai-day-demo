#!/usr/bin/env bash
# Trusts the workspace and installs node dependencies. Always exits 0: a failed
# install is explained rather than failing the container create.
set -uo pipefail

git config --global --add safe.directory /workspace

echo
echo "Installing node dependencies..."
if ! pnpm install; then
	echo
	echo "pnpm install failed. If it stopped on a 401, the token in .npmrc has expired."
	exit 0
fi

echo
echo "Ready. Start the stack with:"
echo
echo "    pnpm start"
echo
echo "Postgres already runs as a sibling container, and the runner skips it."
echo "The console comes up on http://localhost:5173"
echo
