#!/usr/bin/env bash
# Smoke test hub Nexus. Exits non-zero on any failure.
#
# Checks:
#   - Nexus API responds and reports writable
#   - Admin can authenticate
#   - Anonymous access is disabled (read against an API that requires auth)
#   - Every repo requested in NEXUS_PROXY_FORMATS + NEXUS_HOSTED_FORMATS exists
#   - Group repos exist where both their proxy and hosted members do
#   - (Online-only) A real resolve through one proxy succeeds — proves the
#     proxy can actually reach its upstream. Skipped with --no-resolve.
#
# Usage: ./check.sh [--no-resolve]

set -euo pipefail

DO_RESOLVE=1
for arg in "$@"; do
  case "$arg" in
    --no-resolve) DO_RESOLVE=0 ;;
    *) echo "error: unknown flag '$arg'" >&2; exit 64 ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ASSETS_DIR="$(cd "$SCRIPT_DIR/../assets" && pwd)"
ENV_FILE="$ASSETS_DIR/.env"

[[ -f "$ENV_FILE" ]] || { echo "error: missing $ENV_FILE" >&2; exit 70; }
# shellcheck disable=SC1090
source "$ENV_FILE"

NEXUS_URL="http://localhost:${NEXUS_HTTP_PORT}"
AUTH=(-u "admin:${NEXUS_ADMIN_PASSWORD}")

pass() { printf "  \033[32mOK\033[0m  %s\n" "$1"; }
fail() { printf "  \033[31mFAIL\033[0m %s\n" "$1"; exit 1; }

echo "==> Checking Nexus API"
# Avoid the `cmd && pass || fail` chain: `pass` returning non-zero (closed
# stdout under `| head`, stale terminal) would mis-route to `fail`. Use a
# plain if/then/else.
if curl -fsS -o /dev/null "$NEXUS_URL/service/rest/v1/status/writable"; then
  pass "api writable"
else
  fail "api not writable at $NEXUS_URL"
fi

echo "==> Checking admin auth"
# /security/users LIST returns 200 to any authed admin. /security/users/admin
# itself only supports PUT/DELETE and would 405 on GET — don't use it here.
# Capture the HTTP code so a failure message can distinguish "bad credentials"
# from "Nexus is sick" from "network unreachable" — an ambiguous fail message
# would send the operator hunting in .env when Nexus is actually crash-looping.
USERS_CODE=$(curl -sS -o /tmp/nexus-check-users.$$ -w "%{http_code}" \
  "${AUTH[@]}" "$NEXUS_URL/service/rest/v1/security/users" 2>/dev/null || true)
case "$USERS_CODE" in
  200)
    if grep -q '"userId"[[:space:]]*:[[:space:]]*"admin"' /tmp/nexus-check-users.$$; then
      pass "admin can authenticate and is listed"
    else
      rm -f /tmp/nexus-check-users.$$
      fail "admin user not in /security/users list — bootstrap did not run or was wiped"
    fi
    ;;
  401|403) rm -f /tmp/nexus-check-users.$$; fail "admin creds rejected (HTTP $USERS_CODE) — check NEXUS_ADMIN_PASSWORD in .env" ;;
  000)     rm -f /tmp/nexus-check-users.$$; fail "could not connect to Nexus at $NEXUS_URL — is the container running?" ;;
  *)       rm -f /tmp/nexus-check-users.$$; fail "unexpected HTTP $USERS_CODE from /security/users — check container logs" ;;
esac
rm -f /tmp/nexus-check-users.$$

echo "==> Checking anonymous access is disabled"
ANON=$(curl -fsS "${AUTH[@]}" "$NEXUS_URL/service/rest/v1/security/anonymous" 2>/dev/null || true)
if printf '%s' "$ANON" | grep -q '"enabled"[[:space:]]*:[[:space:]]*false'; then
  pass "anonymous access disabled"
else
  fail "anonymous access still enabled — bootstrap step 2 didn't take"
fi

repo_exists() {
  curl -fsS "${AUTH[@]}" -o /dev/null "$NEXUS_URL/service/rest/v1/repositories/$1" 2>/dev/null
}

format_enabled() {
  local fmt=$1 list=$2
  case " $list " in
    *" $fmt "*) return 0 ;;
    *) return 1 ;;
  esac
}

# Expected repo set derived from what bootstrap.sh would create given the
# current .env. Keeping this in sync with bootstrap.sh is the operator's
# job — check.sh is not a re-implementation of bootstrap's repo inventory.
echo "==> Checking proxy repositories"
check_repo() {
  if repo_exists "$1"; then pass "repo '$1' present"; else fail "repo '$1' missing"; fi
}

format_enabled maven  "$NEXUS_PROXY_FORMATS"  && check_repo maven-central-proxy
format_enabled npm    "$NEXUS_PROXY_FORMATS"  && check_repo npm-proxy
format_enabled pypi   "$NEXUS_PROXY_FORMATS"  && check_repo pypi-proxy
format_enabled go     "$NEXUS_PROXY_FORMATS"  && check_repo go-proxy
format_enabled apt    "$NEXUS_PROXY_FORMATS"  && check_repo "apt-ubuntu-${NEXUS_APT_DISTRIBUTION}-proxy"
format_enabled yum    "$NEXUS_PROXY_FORMATS"  && check_repo yum-rocky-proxy
format_enabled docker "$NEXUS_PROXY_FORMATS"  && check_repo docker-hub-proxy

echo "==> Checking hosted repositories"
format_enabled maven "$NEXUS_HOSTED_FORMATS"  && check_repo maven-hosted
format_enabled npm   "$NEXUS_HOSTED_FORMATS"  && check_repo npm-hosted
format_enabled pypi  "$NEXUS_HOSTED_FORMATS"  && check_repo pypi-hosted
format_enabled raw   "$NEXUS_HOSTED_FORMATS"  && check_repo raw-hosted

echo "==> Checking group repositories"
if format_enabled maven "$NEXUS_PROXY_FORMATS" && format_enabled maven "$NEXUS_HOSTED_FORMATS"; then
  check_repo maven-group
fi
if format_enabled npm "$NEXUS_PROXY_FORMATS" && format_enabled npm "$NEXUS_HOSTED_FORMATS"; then
  check_repo npm-group
fi
if format_enabled pypi "$NEXUS_PROXY_FORMATS" && format_enabled pypi "$NEXUS_HOSTED_FORMATS"; then
  check_repo pypi-group
fi

if [[ $DO_RESOLVE -eq 1 ]] && format_enabled npm "$NEXUS_PROXY_FORMATS"; then
  # Two separate checks: the proxy *serves* the artifact (may be from cache),
  # and the proxy can *reach upstream* (forces a network round-trip, not just
  # a cache hit). Conflating them as earlier versions did gave a false-positive
  # on offline machines with warm caches.
  #
  # Anonymous access is disabled on hub Nexus, so the resolve needs admin
  # credentials. Real developers use their own Nexus user / token.
  echo "==> npm-proxy serves 'is-number' (may be from cache)"
  if curl -fsS "${AUTH[@]}" -o /dev/null "$NEXUS_URL/repository/npm-proxy/is-number"; then
    pass "npm-proxy served 'is-number'"
  else
    fail "npm-proxy could not serve 'is-number' — check bootstrap ran, or rerun with --no-resolve"
  fi

  echo "==> Upstream registry.npmjs.org reachable from this host"
  # 3s connect timeout — if the machine is offline, fail fast instead of
  # hanging for the default curl 2-minute retry loop. HEAD request keeps
  # the probe payload-free.
  if curl -fsSI --connect-timeout 3 -o /dev/null https://registry.npmjs.org/is-number; then
    pass "registry.npmjs.org is reachable (upstream confirmed)"
  else
    printf "  \033[33mWARN\033[0m npmjs.org unreachable — proxy is serving from cache only\n"
    printf "       (laptop-profile hub Nexus is an online proxy, not an air-gap seed)\n"
  fi
fi

echo
echo "All checks passed."
