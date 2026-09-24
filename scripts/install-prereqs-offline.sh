#!/usr/bin/env bash
# Installs the workshop prerequisites from the offline bundle, on macOS and Linux.
#
#   ./install-prereqs-offline.sh              # install what is missing
#   ./install-prereqs-offline.sh --check      # report only, install nothing
#   ./install-prereqs-offline.sh --bundle DIR # the bundle is somewhere else
#
# Needs no network. Bash rather than Node, because Node is one of the things it
# installs.
set -uo pipefail

NODE_MIN=24
GO_MIN=1.25
PNPM_MAJOR=10

CHECK_ONLY=0
BUNDLE=""
for arg in "$@"; do
	case "$arg" in
		--check) CHECK_ONLY=1 ;;
		--bundle) BUNDLE="__next__" ;;
		--help|-h) sed -n '2,9p' "$0" | sed 's/^# \?//'; exit 0 ;;
		*)
			if [ "$BUNDLE" = "__next__" ]; then BUNDLE="$arg"
			else echo "unknown option: $arg" >&2; exit 2; fi
			;;
	esac
done
[ "$BUNDLE" = "__next__" ] && { echo "--bundle needs a directory" >&2; exit 2; }

OS=$(uname -s)
ARCH=$(uname -m)

BOLD=$(tput bold 2>/dev/null || true)
DIM=$(tput dim 2>/dev/null || true)
RED=$(tput setaf 1 2>/dev/null || true)
GREEN=$(tput setaf 2 2>/dev/null || true)
YELLOW=$(tput setaf 3 2>/dev/null || true)
RESET=$(tput sgr0 2>/dev/null || true)

FAILED=0
SUMMARY=()

step() { printf '\n%s==> %s%s\n' "$BOLD" "$1" "$RESET"; }
info() { printf '    %s\n' "$1"; }
warn() { printf '    %s%s%s\n' "$YELLOW" "$1" "$RESET"; }
fail() { printf '    %s%s%s\n' "$RED" "$1" "$RESET"; FAILED=1; }

have() { command -v "$1" >/dev/null 2>&1; }

# True when $1 is greater than or equal to $2, compared as dotted versions.
version_ge() {
	[ "$(printf '%s\n%s\n' "$2" "$1" | sort -V | head -1)" = "$2" ]
}

# Records a tool's outcome for the closing summary.
record() { SUMMARY+=("$1|$2|$3"); }

# Runs a command, or prints it when --check is set.
run() {
	if [ "$CHECK_ONLY" = 1 ]; then
		info "${DIM}would run: $*${RESET}"
		return 1
	fi
	"$@"
}

# ---------------------------------------------------------------- platform

case "$OS-$ARCH" in
	Darwin-arm64) PLATFORM=darwin-arm64 ;;
	Linux-x86_64) PLATFORM=linux-x64 ;;
	Darwin-x86_64)
		echo "This bundle carries the Apple Silicon build. On an Intel Mac, rebuild it with" >&2
		echo "  pnpm bundle:prereqs --platform darwin-x64" >&2
		exit 2
		;;
	*) echo "unsupported platform: $OS $ARCH. On Windows use install-prereqs-offline.ps1" >&2; exit 2 ;;
esac

# ---------------------------------------------------------------- bundle

SCRIPT_DIR=$(cd -- "$(dirname -- "$0")" && pwd)

locate_bundle() {
	[ -n "$BUNDLE" ] && return 0
	for candidate in "$SCRIPT_DIR" "$SCRIPT_DIR/prereq-bundle" "$PWD/prereq-bundle" "$SCRIPT_DIR/../prereq-bundle"; do
		if [ -d "$candidate/$PLATFORM" ] && [ -d "$candidate/common" ]; then
			BUNDLE=$(cd -- "$candidate" && pwd)
			return 0
		fi
	done
	return 1
}

if ! locate_bundle; then
	echo "no prereq-bundle found. Pass --bundle /path/to/prereq-bundle" >&2
	exit 2
fi
if [ ! -d "$BUNDLE/$PLATFORM" ]; then
	echo "$BUNDLE has no $PLATFORM directory. This bundle was built for other machines." >&2
	exit 2
fi

ASSETS="$BUNDLE/$PLATFORM"
COMMON="$BUNDLE/common"

sha_of() {
	if have sha256sum; then sha256sum "$1" | cut -d' ' -f1
	else shasum -a 256 "$1" | cut -d' ' -f1; fi
}

# Prints the path of the first file in $1 matching glob $2.
find_asset() {
	local file
	for file in "$1"/$2; do
		[ -e "$file" ] && { printf '%s' "$file"; return 0; }
	done
	return 1
}

# Compares a bundled file against the SHA256SUMS beside it.
verify() {
	local file=$1 dir name expected actual
	dir=$(dirname "$file")
	name=$(basename "$file")
	expected=$(awk -v n="$name" '$2 == n { print $1 }' "$dir/SHA256SUMS" 2>/dev/null)
	if [ -z "$expected" ]; then
		fail "$name is not listed in $dir/SHA256SUMS"
		return 1
	fi
	actual=$(sha_of "$file")
	if [ "$actual" != "$expected" ]; then
		fail "$name is corrupt. Copy the bundle across again."
		return 1
	fi
	info "${DIM}verified $name${RESET}"
}

# ---------------------------------------------------------------- prefix

# /usr/local is where nodejs.org and go.dev tell you to put these. A machine
# with neither write access nor sudo gets ~/.local instead.
PREFIX=/usr/local
SUDO=""
if [ ! -w "$PREFIX" ]; then
	if have sudo; then SUDO=sudo
	else PREFIX="$HOME/.local"; fi
fi
[ "$CHECK_ONLY" = 1 ] || $SUDO mkdir -p "$PREFIX/bin"

# The startup file the login shell reads in a new terminal.
case "${SHELL:-}" in
	*/zsh) SHELL_RC="$HOME/.zshrc" ;;
	*/bash) [ "$OS" = Darwin ] && SHELL_RC="$HOME/.bash_profile" || SHELL_RC="$HOME/.bashrc" ;;
	*) SHELL_RC="$HOME/.profile" ;;
esac

# Appends a PATH line to the shell startup file once.
add_to_path() {
	[ "$CHECK_ONLY" = 1 ] && return 0
	grep -qs "$1" "$SHELL_RC" 2>/dev/null && return 0
	echo "export PATH=$1:\$PATH" >> "$SHELL_RC"
	info "added $1 to $(basename "$SHELL_RC")"
}

# ---------------------------------------------------------------- node

install_node() {
	step "Node ${NODE_MIN}+"
	if have node; then
		local current
		current=$(node --version | tr -d 'v')
		if version_ge "$current" "$NODE_MIN"; then
			info "node $current"
			record Node ok "$current"
			return
		fi
		warn "node $current is older than $NODE_MIN"
	else
		info "not installed"
	fi

	local tarball
	if ! tarball=$(find_asset "$ASSETS" 'node-v*.tar.gz'); then
		fail "no node tarball in $ASSETS"
		record Node missing "not in the bundle"
		return
	fi
	verify "$tarball" || { record Node missing "corrupt download"; return; }

	# The tarball's top level is node-vX-platform/, whose bin, include, lib and
	# share go straight into the prefix; its three doc files do not.
	run $SUDO tar -C "$PREFIX" --strip-components=1 --no-same-owner \
		--exclude CHANGELOG.md --exclude LICENSE --exclude README.md -xf "$tarball"

	if [ "$CHECK_ONLY" = 1 ]; then
		record Node missing "$(basename "$tarball")"
		return
	fi

	export PATH="$PREFIX/bin:$PATH"
	[ "$PREFIX" = /usr/local ] || add_to_path "$PREFIX/bin"

	if have node && version_ge "$(node --version | tr -d 'v')" "$NODE_MIN"; then
		record Node installed "$(node --version | tr -d 'v')"
	else
		record Node missing "$(basename "$tarball")"
		fail "node is still missing or too old"
	fi
}

# ---------------------------------------------------------------- pnpm

install_pnpm() {
	step "pnpm ${PNPM_MAJOR}"
	if have pnpm; then
		local current
		current=$(pnpm --version)
		if [ "${current%%.*}" = "$PNPM_MAJOR" ]; then
			info "pnpm $current"
			record pnpm ok "$current"
			return
		fi
		warn "pnpm $current is not $PNPM_MAJOR.x, and this repo's lockfile was written for $PNPM_MAJOR"
	else
		info "not installed"
	fi

	if ! have npm; then
		fail "npm is missing, so Node did not install. Fix Node first."
		record pnpm missing "needs Node"
		return
	fi

	local tarball
	if ! tarball=$(find_asset "$COMMON" 'pnpm-*.tgz'); then
		fail "no pnpm tarball in $COMMON"
		record pnpm missing "not in the bundle"
		return
	fi
	verify "$tarball" || { record pnpm missing "corrupt download"; return; }

	# The package bundles its own dependencies, so --offline resolves nothing.
	run $SUDO npm install -g --offline --no-audit --no-fund "$tarball"

	if [ "$CHECK_ONLY" = 1 ]; then
		record pnpm missing "$(basename "$tarball")"
		return
	fi

	hash -r 2>/dev/null
	if have pnpm && [ "$(pnpm --version | cut -d. -f1)" = "$PNPM_MAJOR" ]; then
		record pnpm installed "$(pnpm --version)"
	else
		record pnpm missing "$(basename "$tarball")"
		fail "pnpm is still missing"
	fi
}

# ---------------------------------------------------------------- go

install_go() {
	step "Go ${GO_MIN}+"
	if have go; then
		local current
		current=$(go version | awk '{print $3}' | tr -d 'go')
		if version_ge "$current" "$GO_MIN"; then
			info "go $current"
			record Go ok "$current"
			return
		fi
		warn "go $current is older than $GO_MIN"
	else
		info "not installed"
	fi

	local tarball
	if ! tarball=$(find_asset "$ASSETS" 'go*.tar.gz'); then
		fail "no go tarball in $ASSETS"
		record Go missing "not in the bundle"
		return
	fi
	verify "$tarball" || { record Go missing "corrupt download"; return; }

	# go.dev ships a whole tree under go/, and refuses to run when merged into an old one.
	run $SUDO rm -rf "$PREFIX/go"
	run $SUDO tar -C "$PREFIX" --no-same-owner -xf "$tarball"

	if [ "$CHECK_ONLY" = 1 ]; then
		record Go missing "$(basename "$tarball")"
		return
	fi

	export PATH="$PREFIX/go/bin:$PATH"
	add_to_path "$PREFIX/go/bin"

	if have go && version_ge "$(go version | awk '{print $3}' | tr -d 'go')" "$GO_MIN"; then
		record Go installed "$(go version | awk '{print $3}' | tr -d 'go')"
	else
		record Go missing "$(basename "$tarball")"
		fail "go is still missing or too old"
	fi
}

# ---------------------------------------------------------------- claude code

install_claude() {
	step "Claude Code"
	if have claude; then
		info "$(claude --version 2>/dev/null || echo present)"
		record "Claude Code" ok "$(claude --version 2>/dev/null | awk '{print $1}')"
		return
	fi
	info "not installed"

	local binary
	if ! binary=$(find_asset "$ASSETS" 'claude'); then
		fail "no claude binary in $ASSETS"
		record "Claude Code" missing "not in the bundle"
		return
	fi
	verify "$binary" || { record "Claude Code" missing "corrupt download"; return; }

	if [ "$OS" = Linux ] && ldd /bin/ls 2>&1 | grep -q musl; then
		warn "this looks like a musl system, and the bundled binary is the glibc build"
	fi

	local target="$HOME/.local/bin/claude"
	# `claude install` reads the release list over the network, so the offline
	# path is the binary on its own. It updates itself once there is internet.
	if ! run bash -c "mkdir -p '$HOME/.local/bin' && cp '$binary' '$target' && chmod +x '$target'"; then
		record "Claude Code" missing "$(basename "$binary")"
		return
	fi
	add_to_path "$HOME/.local/bin"

	export PATH="$HOME/.local/bin:$PATH"
	hash -r 2>/dev/null
	if have claude; then
		record "Claude Code" installed "$(claude --version 2>/dev/null | awk '{print $1}')"
	else
		record "Claude Code" missing "open a new terminal, then run claude --version"
	fi
}

# ---------------------------------------------------------------- not bundled

check_python() {
	step "Python 3.9+"
	if python3 --version >/dev/null 2>&1; then
		info "$(python3 --version)"
		record Python ok "$(python3 --version | awk '{print $2}')"
		return
	fi
	# macOS and Linux both ship one, so the bundle carries the Windows installer only.
	fail "python3 is missing, and only the Windows installer is bundled"
	record Python missing "install python3 from your package manager"
}

check_podman() {
	step "Podman, with a compose provider"
	if ! have podman; then
		fail "not installed, and Podman is not in this bundle"
		record Podman missing "https://podman-desktop.io"
		return
	fi
	info "$(podman --version)"
	if podman compose version 2>&1 | grep -i 'docker compose' >/dev/null; then
		record Podman ok "$(podman --version | awk '{print $3}')"
	else
		warn "no docker-compose provider. Install docker-compose while you still have internet."
		record Podman missing "podman compose version does not report docker-compose"
	fi
}

# ---------------------------------------------------------------- summary

print_summary() {
	printf '\n%s==> Summary%s\n\n' "$BOLD" "$RESET"
	printf '    %-14s %-11s %s\n' TOOL STATUS DETAIL
	printf '    %-14s %-11s %s\n' "-----" "------" "------"
	local name status detail colour
	# bash 3.2 treats an empty array as unbound under `set -u`.
	[ "${#SUMMARY[@]}" -eq 0 ] && return
	for row in "${SUMMARY[@]}"; do
		IFS='|' read -r name status detail <<< "$row"
		case "$status" in
			ok|installed) colour=$GREEN ;;
			*) colour=$RED ;;
		esac
		printf '    %-14s %s%-11s%s %s\n' "$name" "$colour" "$status" "$RESET" "$detail"
	done

	if [ "$CHECK_ONLY" = 1 ]; then
		printf '\n    Check only. Drop --check to install anything marked missing.\n'
	elif [ "$FAILED" = 1 ]; then
		printf '\n    %sSomething is still missing. Fix the red rows above, then rerun.%s\n' "$RED" "$RESET"
	else
		cat <<'EOF'

    Open a new terminal so every PATH change takes effect, then:

        cd path/to/ai-day-demo
        pnpm start
EOF
	fi
}

# ---------------------------------------------------------------- main

printf '%sWorkshop prerequisites, offline%s  (%s %s)\n' "$BOLD" "$RESET" "$OS" "$ARCH"
info "bundle $BUNDLE"
info "prefix $PREFIX"
[ "$CHECK_ONLY" = 1 ] && info "check only, nothing will be installed"

install_node
install_pnpm
install_go
install_claude
check_python
check_podman
print_summary

exit "$FAILED"
