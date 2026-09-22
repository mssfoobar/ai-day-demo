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
echo "The platform stack (Postgres, IAMS, SDS, RTUS, GIS) runs as sibling containers;"
echo "the runner waits for them to converge before starting the apps."
echo "The console comes up on http://127.0.0.1.nip.io:5173 - sign in as admin / P@ssw0rd."
echo
