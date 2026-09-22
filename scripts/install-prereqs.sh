#!/usr/bin/env bash
# Installs the workshop prerequisites on macOS and Linux.
#
#   ./scripts/install-prereqs.sh          # install what is missing
#   ./scripts/install-prereqs.sh --check  # report only, install nothing
#   ./scripts/install-prereqs.sh --yes    # no prompts
#
# Bash rather than Node, because Node is one of the things it installs.
set -uo pipefail

NODE_MIN=24
GO_MIN=1.25
PNPM_MAJOR=10
GO_FALLBACK=1.25.14

CHECK_ONLY=0
ASSUME_YES=0
for arg in "$@"; do
	case "$arg" in
		--check) CHECK_ONLY=1 ;;
		--yes|-y) ASSUME_YES=1 ;;
		--help|-h) sed -n '2,8p' "$0" | sed 's/^# \?//'; exit 0 ;;
		*) echo "unknown option: $arg" >&2; exit 2 ;;
	esac
done

OS=$(uname -s)
ARCH=$(uname -m)
case "$ARCH" in
	x86_64|amd64) GOARCH=amd64 ;;
	arm64|aarch64) GOARCH=arm64 ;;
	*) GOARCH="" ;;
esac

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

confirm() {
	[ "$ASSUME_YES" = 1 ] && return 0
	printf '    %s [Y/n] ' "$1"
	read -r reply </dev/tty || return 1
	[ -z "$reply" ] || [ "$reply" = y ] || [ "$reply" = Y ]
}

# Runs a command, or prints it when --check is set.
run() {
	if [ "$CHECK_ONLY" = 1 ]; then
		info "${DIM}would run: $*${RESET}"
		return 1
	fi
	"$@"
}

# ---------------------------------------------------------------- package manager

PM=""
PM_INSTALL=""

detect_package_manager() {
	if [ "$OS" = Darwin ]; then
		if have brew; then
			PM=brew
			PM_INSTALL="brew install"
			return
		fi
		step "Homebrew"
		warn "not installed, and it is how this script installs everything on macOS"
		if [ "$CHECK_ONLY" = 1 ]; then
			info "install it from https://brew.sh, then run this script again"
			return
		fi
		if confirm "Install Homebrew now?"; then
			/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
			# A fresh install is not on PATH until the shell is re-evaluated.
			for candidate in /opt/homebrew/bin/brew /usr/local/bin/brew; do
				[ -x "$candidate" ] && eval "$("$candidate" shellenv)"
			done
			have brew && PM=brew && PM_INSTALL="brew install"
		else
			info "install it from https://brew.sh, then run this script again"
		fi
	elif have apt-get; then
		PM=apt
		PM_INSTALL="sudo apt-get install -y"
	elif have dnf; then
		PM=dnf
		PM_INSTALL="sudo dnf install -y"
	elif have pacman; then
		PM=pacman
		PM_INSTALL="sudo pacman -S --noconfirm"
	fi
}

# ---------------------------------------------------------------- xcode clt

check_xcode_tools() {
	[ "$OS" = Darwin ] || return 0
	step "Xcode Command Line Tools"
	if xcode-select -p >/dev/null 2>&1; then
		info "present at $(xcode-select -p)"
		record "Xcode CLT" ok "$(xcode-select -p)"
		return
	fi
	warn "missing. git and python3 both prompt for this on first use."
	if run xcode-select --install; then
		info "accept the dialog, wait for it to finish, then run this script again"
	fi
	record "Xcode CLT" missing "run xcode-select --install"
}

# ---------------------------------------------------------------- node

install_node() {
	step "Node ${NODE_MIN}+"
	if have node; then
		local v
		v=$(node --version | tr -d 'v')
		if version_ge "$v" "$NODE_MIN"; then
			info "node $v"
			record Node ok "$v"
			return
		fi
		warn "node $v is older than $NODE_MIN"
	else
		info "not installed"
	fi

	case "$PM" in
		brew) run brew install node ;;
		apt)
			# Debian and Ubuntu package a Node far older than 24, so use NodeSource.
			run bash -c "curl -fsSL https://deb.nodesource.com/setup_${NODE_MIN}.x | sudo -E bash -" \
				&& run sudo apt-get install -y nodejs
			;;
		dnf) run bash -c "curl -fsSL https://rpm.nodesource.com/setup_${NODE_MIN}.x | sudo -E bash -" \
				&& run sudo dnf install -y nodejs ;;
		pacman) run sudo pacman -S --noconfirm nodejs npm ;;
		*) fail "no package manager found. Install Node from https://nodejs.org" ;;
	esac

	if have node && version_ge "$(node --version | tr -d 'v')" "$NODE_MIN"; then
		record Node installed "$(node --version | tr -d 'v')"
	else
		record Node missing "https://nodejs.org"
		[ "$CHECK_ONLY" = 1 ] || fail "node is still missing or too old"
	fi
}

# ---------------------------------------------------------------- pnpm

install_pnpm() {
	step "pnpm ${PNPM_MAJOR}"
	if have pnpm; then
		local v major
		v=$(pnpm --version)
		major=${v%%.*}
		if [ "$major" = "$PNPM_MAJOR" ]; then
			info "pnpm $v"
			record pnpm ok "$v"
			return
		fi
		warn "pnpm $v is not $PNPM_MAJOR.x, and this repo's lockfile was written for $PNPM_MAJOR"
	else
		info "not installed"
	fi

	if ! have npm; then
		fail "npm is missing, so Node did not install. Fix Node first."
		record pnpm missing "needs Node"
		return
	fi

	# Pinned: an unpinned install pulls pnpm 12, which reads the workspace
	# settings block differently.
	run npm install -g "pnpm@${PNPM_MAJOR}"

	if have pnpm && [ "$(pnpm --version | cut -d. -f1)" = "$PNPM_MAJOR" ]; then
		record pnpm installed "$(pnpm --version)"
	else
		record pnpm missing "npm install -g pnpm@${PNPM_MAJOR}"
		[ "$CHECK_ONLY" = 1 ] || fail "pnpm is still missing. Do not retry with sudo."
	fi
}

# ---------------------------------------------------------------- go

# Latest Go release, or the pinned fallback when go.dev is unreachable.
latest_go() {
	local v
	v=$(curl -fsSL --max-time 10 https://go.dev/VERSION?m=text 2>/dev/null | head -1 | tr -d 'go')
	[ -n "$v" ] && printf '%s' "$v" || printf '%s' "$GO_FALLBACK"
}

install_go() {
	step "Go ${GO_MIN}+"
	if have go; then
		local v
		v=$(go version | awk '{print $3}' | tr -d 'go')
		if version_ge "$v" "$GO_MIN"; then
			info "go $v"
			record Go ok "$v"
			return
		fi
		warn "go $v is older than $GO_MIN"
	else
		info "not installed"
	fi

	if [ "$PM" = brew ]; then
		run brew install go
	elif [ -n "$GOARCH" ]; then
		# Distribution packages lag well behind 1.25, so take the official tarball.
		local v url
		v=$(latest_go)
		url="https://go.dev/dl/go${v}.$(echo "$OS" | tr '[:upper:]' '[:lower:]')-${GOARCH}.tar.gz"
		info "downloading go $v"
		if run bash -c "curl -fsSL '$url' -o /tmp/go.tgz"; then
			run sudo rm -rf /usr/local/go
			run sudo tar -C /usr/local -xzf /tmp/go.tgz
			rm -f /tmp/go.tgz
			export PATH=/usr/local/go/bin:$PATH
			if ! grep -qs '/usr/local/go/bin' "$HOME/.profile" 2>/dev/null; then
				echo 'export PATH=/usr/local/go/bin:$PATH' >> "$HOME/.profile"
				info "added /usr/local/go/bin to ~/.profile"
			fi
		fi
	else
		fail "unsupported architecture $ARCH. Install Go from https://go.dev/dl/"
	fi

	if have go && version_ge "$(go version | awk '{print $3}' | tr -d 'go')" "$GO_MIN"; then
		record Go installed "$(go version | awk '{print $3}' | tr -d 'go')"
	else
		record Go missing "https://go.dev/dl/"
		[ "$CHECK_ONLY" = 1 ] || fail "go is still missing or too old"
	fi
}

# ---------------------------------------------------------------- podman

install_podman() {
	step "Podman, with a compose provider"
	if have podman; then
		info "$(podman --version)"
	else
		info "not installed"
		case "$PM" in
			brew) run brew install podman ;;
			apt) run sudo apt-get install -y podman ;;
			dnf) run sudo dnf install -y podman ;;
			pacman) run sudo pacman -S --noconfirm podman ;;
			*) fail "install Podman from https://podman-desktop.io" ;;
		esac
	fi

	# `podman compose` is a dispatcher; without a provider it cannot start anything.
	if podman compose version >/dev/null 2>&1; then
		info "compose provider present"
	else
		info "no compose provider, installing podman-compose"
		case "$PM" in
			brew) run brew install podman-compose ;;
			apt) run sudo apt-get install -y podman-compose ;;
			dnf) run sudo dnf install -y podman-compose ;;
			pacman) run sudo pacman -S --noconfirm podman-compose ;;
			*) fail "install podman-compose, or docker-compose" ;;
		esac
	fi

	# macOS and Windows run containers in a VM that has to exist and be started.
	if [ "$OS" = Darwin ] && have podman; then
		if ! podman machine list --format '{{.Name}}' 2>/dev/null | grep -q .; then
			info "no podman machine, creating one"
			run podman machine init
		fi
		if ! podman machine list --format '{{.Running}}' 2>/dev/null | grep -q true; then
			info "starting the podman machine"
			run podman machine start
		else
			info "podman machine running"
		fi
	fi

	if have podman && podman compose version >/dev/null 2>&1; then
		record Podman ok "$(podman --version | awk '{print $3}')"
	else
		record Podman missing "https://podman-desktop.io"
		[ "$CHECK_ONLY" = 1 ] || fail "podman or its compose provider is still missing"
	fi
}

# ---------------------------------------------------------------- python

install_python() {
	step "Python 3.9+"
	if python3 --version >/dev/null 2>&1; then
		info "$(python3 --version)"
		record Python ok "$(python3 --version | awk '{print $2}')"
		return
	fi
	info "not installed"
	case "$PM" in
		brew) run brew install python ;;
		apt) run sudo apt-get install -y python3 ;;
		dnf) run sudo dnf install -y python3 ;;
		pacman) run sudo pacman -S --noconfirm python ;;
		*) fail "install Python from https://www.python.org" ;;
	esac
	if python3 --version >/dev/null 2>&1; then
		record Python installed "$(python3 --version | awk '{print $2}')"
	else
		record Python missing "https://www.python.org"
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
	if run bash -c "curl -fsSL https://claude.ai/install.sh | bash"; then
		# The installer puts the binary here and edits the shell profile, which
		# this shell has already read.
		export PATH="$HOME/.local/bin:$PATH"
	fi
	if have claude; then
		record "Claude Code" installed "$(claude --version 2>/dev/null | awk '{print $1}')"
	else
		record "Claude Code" missing "open a new terminal, then run claude --version"
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

	if [ "$FAILED" = 1 ]; then
		printf '\n    %sSomething is still missing. Fix the red rows above, then rerun.%s\n' "$RED" "$RESET"
	elif [ "$CHECK_ONLY" = 1 ]; then
		printf '\n    Check only. Drop --check to install anything marked missing.\n'
	else
		cat <<'EOF'

    Open a new terminal so every PATH change takes effect, then:

        cd path/to/ai-day-demo
        pnpm start

    Then open http://localhost:5173
EOF
	fi
}

# ---------------------------------------------------------------- main

case "$OS" in
	Darwin|Linux) ;;
	*) echo "unsupported platform: $OS. On Windows use scripts/install-prereqs.ps1" >&2; exit 2 ;;
esac

printf '%sWorkshop prerequisites%s  (%s %s)\n' "$BOLD" "$RESET" "$OS" "$ARCH"
[ "$CHECK_ONLY" = 1 ] && info "check only, nothing will be installed"

detect_package_manager
[ "$PM" = apt ] && [ "$CHECK_ONLY" = 0 ] && run sudo apt-get update -qq

check_xcode_tools
install_node
install_pnpm
install_go
install_podman
install_python
install_claude
print_summary

exit "$FAILED"
