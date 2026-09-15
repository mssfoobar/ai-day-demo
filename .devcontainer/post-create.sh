#!/usr/bin/env bash
# Trusts the workspace, checks the GitHub Packages token, and installs node
# dependencies when it is present. Always exits 0: a missing token is explained
# rather than failing the container create.
set -uo pipefail

REGISTRY_LINE='//npm.pkg.github.com/:_authToken='
NPMRC="${HOME}/.npmrc"

git config --global --add safe.directory /workspace

echo
if [ -d "${NPMRC}" ]; then
	echo "~/.npmrc is a DIRECTORY, not a file."
	echo "A missing host file is mounted as an empty directory. Remove this container,"
	echo "create ~/.npmrc on your host with the line below, then start again:"
	echo
	echo "    ${REGISTRY_LINE}<your-token>"
	echo
	exit 0
fi

if [ ! -f "${NPMRC}" ] || ! grep -q "${REGISTRY_LINE}" "${NPMRC}"; then
	echo "No GitHub Packages token found in ~/.npmrc."
	echo "Six dependencies come from npm.pkg.github.com, which refuses anonymous reads,"
	echo "so pnpm install cannot run yet. Add this line on your host and start again:"
	echo
	echo "    ${REGISTRY_LINE}<a token with read:packages>"
	echo
	echo "Everything else is ready: Node, pnpm, Go and the Go caches are in the image."
	exit 0
fi

echo "Installing node dependencies..."
if ! pnpm install --frozen-lockfile; then
	echo
	echo "pnpm install failed. If it stopped on a 401, the token in ~/.npmrc is missing"
	echo "read:packages or has expired."
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
