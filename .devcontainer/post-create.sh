#!/usr/bin/env bash
# Runs once when the devcontainer is created: trusts the workspace, checks the
# GitHub Packages token, and installs node dependencies when it is present.
#
# Always exits 0. A missing token is a thing to explain, not a reason to fail the
# container create and leave the participant with no shell.
set -uo pipefail

REGISTRY_LINE='//npm.pkg.github.com/:_authToken='
NPMRC="${HOME}/.npmrc"

git config --global --add safe.directory /workspace

echo
if [ -d "${NPMRC}" ]; then
	echo "~/.npmrc is a DIRECTORY, not a file."
	echo "Docker creates one when the host file is missing. Remove the container, create"
	echo "~/.npmrc on your host with the line below, then reopen:"
	echo
	echo "    ${REGISTRY_LINE}<your-token>"
	echo
	exit 0
fi

if [ ! -f "${NPMRC}" ] || ! grep -q "${REGISTRY_LINE}" "${NPMRC}"; then
	echo "No GitHub Packages token found in ~/.npmrc."
	echo "Six dependencies come from npm.pkg.github.com, which refuses anonymous reads,"
	echo "so pnpm install cannot run yet. Add this line on your host and reopen:"
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
