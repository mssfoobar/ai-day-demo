#!/usr/bin/env bash
# Bootstrap hub Nexus after the container is writable. Idempotent.
#
# Responsibilities:
#   1. Reset admin password from the auto-generated first-run value
#   2. Disable anonymous access
#   3. Create proxy repositories (maven, npm, pypi, go, apt, yum, docker — configurable)
#   4. Create hosted repositories (maven, npm, pypi, raw — no docker; Harbor owns that)
#   5. Create group repositories aggregating proxy + hosted where the format supports it
#   6. Create + attach a default cleanup policy
#
# Called by up.sh. Not typically invoked directly, but safe to run standalone.

set -euo pipefail

# Clean up response-capture temp files on any exit path (including SIGINT
# from a user ctrl-c during cold boot). These hold HTTP bodies that can
# include rejected-password responses — minor hygiene, but worth doing.
trap 'rm -f /tmp/nexus-pw-resp.$$ /tmp/nexus-repo-err.$$' EXIT INT TERM

# Counters populated by create_repo to gate the "Bootstrap complete" banner.
# Fatal-class failures (401/5xx) make this script exit non-zero; warnings
# (400 validation errors, which are usually benign collisions) don't.
BOOTSTRAP_ERRORS=0
BOOTSTRAP_WARNINGS=0

# PROFILE is reserved for future profile-aware setup (laptop vs team).
# Currently unused but accepted as $1 to keep the call signature stable
# alongside the other hub-nexus scripts. shellcheck flags this; suppress.
# shellcheck disable=SC2034
PROFILE="${1:-laptop}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ASSETS_DIR="$(cd "$SCRIPT_DIR/../assets" && pwd)"
ENV_FILE="$ASSETS_DIR/.env"

[[ -f "$ENV_FILE" ]] || { echo "error: missing $ENV_FILE (run up.sh first)" >&2; exit 70; }
# shellcheck disable=SC1090
source "$ENV_FILE"

NEXUS_URL="http://localhost:${NEXUS_HTTP_PORT}"
CONTAINER=aoh-hub-nexus

# Pick a container CLI. Both podman and docker accept the same exec flags
# this script uses, so either works transparently.
if command -v podman >/dev/null 2>&1; then
  CLI=podman
elif command -v docker >/dev/null 2>&1; then
  CLI=docker
else
  echo "error: neither podman nor docker found on PATH" >&2
  exit 69
fi

# --- 1. Admin password reset ---
# Nexus 3 writes an auto-generated admin password to /nexus-data/admin.password
# on first start and removes it after the password is changed via the API. So
# file-present means "first run, use temp creds"; file-absent means "already
# initialised, ensure configured password still works".
#
# `podman exec` can fail for reasons unrelated to file-absent (container dead,
# socket unreachable). Probe for file existence first so we can tell the two
# apart — otherwise a broken exec looks like "past first run" and leads to
# a misleading "cannot authenticate" error later.
if ! "$CLI" exec "$CONTAINER" test -e /dev/null >/dev/null 2>&1; then
  echo "error: '$CLI exec $CONTAINER' is not working — is the container running?" >&2
  exit 72
fi
if "$CLI" exec "$CONTAINER" test -f /nexus-data/admin.password >/dev/null 2>&1; then
  TEMP_PW=$("$CLI" exec "$CONTAINER" cat /nexus-data/admin.password 2>/dev/null || true)
else
  TEMP_PW=""
fi

api_auth_ok() {
  # Returns 0 if `admin:$1` can reach an authenticated endpoint. /users LIST
  # (not /users/admin — that path only accepts PUT/DELETE and 405s on GET).
  # /status/writable is open to anonymous, so it doesn't prove auth.
  curl -fsS -o /dev/null -u "admin:$1" "$NEXUS_URL/service/rest/v1/security/users" 2>/dev/null
}

if [[ -n "$TEMP_PW" ]]; then
  echo "==> Resetting admin password from first-run value"
  # Nexus rejects a PUT where the new password equals the current one with a
  # 400 — for us that's benign (means someone already set this exact pw), but
  # it's unlikely on a true first run where TEMP_PW is random.
  # Use --data-raw, NOT --data: curl's --data treats a leading '@' as
  # a filename-to-read, so a perfectly legal password starting with '@'
  # would either fail cryptically or silently set the admin password to
  # the contents of an unrelated file in cwd.
  HTTP_CODE=$(curl -sS -o /tmp/nexus-pw-resp.$$ -w "%{http_code}" \
    -u "admin:$TEMP_PW" \
    -H "Content-Type: text/plain" \
    -X PUT "$NEXUS_URL/service/rest/v1/security/users/admin/change-password" \
    --data-raw "$NEXUS_ADMIN_PASSWORD" || true)
  if [[ "$HTTP_CODE" != "204" ]]; then
    echo "error: password reset failed (HTTP $HTTP_CODE): $(head -c 400 /tmp/nexus-pw-resp.$$)" >&2
    rm -f /tmp/nexus-pw-resp.$$
    exit 75
  fi
  rm -f /tmp/nexus-pw-resp.$$
elif api_auth_ok "$NEXUS_ADMIN_PASSWORD"; then
  echo "    admin password already set to configured value — skipping"
else
  # admin.password file is gone (past first run) AND configured password
  # doesn't work. Operator changed it in the UI and forgot to update .env,
  # or ran bootstrap against a Nexus someone else initialised. Fail loudly
  # rather than silently continuing with broken auth.
  echo "error: cannot authenticate as admin and /nexus-data/admin.password is absent." >&2
  echo "       either (a) update NEXUS_ADMIN_PASSWORD in .env to match what was set in Nexus," >&2
  echo "       or (b) ./scripts/down.sh --volumes && ./scripts/up.sh laptop to start fresh." >&2
  exit 77
fi

AUTH=(-u "admin:${NEXUS_ADMIN_PASSWORD}")

# --- 2. Disable anonymous access ---
# Nexus 3's headless Docker image ships with anonymous access ENABLED (the
# interactive first-run wizard that can opt out of this is bypassed in
# container deployments). Force it off idempotently — probe explicitly
# rather than relying on `|| true` so a 401/5xx here is loud, not silent.
ANON_CODE=$(curl -sS -o /tmp/nexus-pw-resp.$$ -w "%{http_code}" \
  "${AUTH[@]}" "$NEXUS_URL/service/rest/v1/security/anonymous" || true)
if [[ "$ANON_CODE" != "200" ]]; then
  echo "error: GET /security/anonymous returned HTTP $ANON_CODE (expected 200): $(head -c 300 /tmp/nexus-pw-resp.$$)" >&2
  exit 73
fi
if grep -q '"enabled"[[:space:]]*:[[:space:]]*false' /tmp/nexus-pw-resp.$$; then
  echo "    anonymous access already disabled — skipping"
else
  echo "==> Disabling anonymous access"
  curl -fsS "${AUTH[@]}" -H "Content-Type: application/json" \
    -X PUT "$NEXUS_URL/service/rest/v1/security/anonymous" \
    -d '{"enabled":false,"userId":"anonymous","realmName":"NexusAuthorizingRealm"}' >/dev/null
fi

# --- Helpers for repo creation ---

# repo_exists <name> — 0 if present, 1 if absent.
repo_exists() {
  curl -fsS -o /dev/null "${AUTH[@]}" "$NEXUS_URL/service/rest/v1/repositories/$1" 2>/dev/null
}

# create_repo <format> <type> <name> <json-body>
# Idempotent: skips if the repo already exists. Outcome depends on HTTP
# response class:
#   201        — created
#   400        — benign validation error (e.g. member repo missing on a
#                group create because a prior create warned). Counted as
#                warning; script keeps going so operators see the full set.
#   401/5xx/…  — fatal. Session invalidated, Nexus is crashing, or we've
#                lost the DB. Record an error so bootstrap exits non-zero
#                at the end — don't let up.sh print "Hub Nexus is up"
#                on top of a broken pipeline.
create_repo() {
  local format=$1 type=$2 name=$3 body=$4
  if repo_exists "$name"; then
    echo "    repo '$name' already exists — skipping"
    return 0
  fi
  local http_code
  http_code=$(curl -sS -o /tmp/nexus-repo-err.$$ -w "%{http_code}" \
    "${AUTH[@]}" -H "Content-Type: application/json" \
    -X POST "$NEXUS_URL/service/rest/v1/repositories/$format/$type" \
    -d "$body" || true)
  case "$http_code" in
    201)
      echo "==> Created $format/$type repo '$name'"
      ;;
    400)
      echo "    warn: $format/$type '$name' rejected by Nexus (HTTP 400): $(head -c 300 /tmp/nexus-repo-err.$$)" >&2
      BOOTSTRAP_WARNINGS=$((BOOTSTRAP_WARNINGS + 1))
      ;;
    *)
      # 000 (curl couldn't connect) / 401 (auth lost) / 403 / 5xx — all fatal.
      # If we keep going, every subsequent create_repo will hit the same
      # error and drown the operator in near-identical warns.
      echo "    error: $format/$type '$name' FATAL (HTTP $http_code): $(head -c 300 /tmp/nexus-repo-err.$$)" >&2
      BOOTSTRAP_ERRORS=$((BOOTSTRAP_ERRORS + 1))
      ;;
  esac
  rm -f /tmp/nexus-repo-err.$$
}

format_enabled() {
  # Checks whether $1 is in the space-separated NEXUS_PROXY_FORMATS or
  # NEXUS_HOSTED_FORMATS list.
  local fmt=$1 list=$2
  case " $list " in
    *" $fmt "*) return 0 ;;
    *) return 1 ;;
  esac
}

# --- 3. Proxy repositories ---

echo "==> Creating proxy repositories (requested: ${NEXUS_PROXY_FORMATS})"

if format_enabled maven "$NEXUS_PROXY_FORMATS"; then
  create_repo maven proxy maven-central-proxy "$(cat <<'JSON'
{
  "name": "maven-central-proxy",
  "online": true,
  "storage": {"blobStoreName": "default", "strictContentTypeValidation": false},
  "proxy": {"remoteUrl": "https://repo1.maven.org/maven2/", "contentMaxAge": -1, "metadataMaxAge": 1440},
  "negativeCache": {"enabled": true, "timeToLive": 1440},
  "httpClient": {"blocked": false, "autoBlock": true},
  "maven": {"versionPolicy": "RELEASE", "layoutPolicy": "PERMISSIVE", "contentDisposition": "INLINE"}
}
JSON
)"
fi

if format_enabled npm "$NEXUS_PROXY_FORMATS"; then
  create_repo npm proxy npm-proxy "$(cat <<'JSON'
{
  "name": "npm-proxy",
  "online": true,
  "storage": {"blobStoreName": "default", "strictContentTypeValidation": true},
  "proxy": {"remoteUrl": "https://registry.npmjs.org", "contentMaxAge": 1440, "metadataMaxAge": 1440},
  "negativeCache": {"enabled": true, "timeToLive": 1440},
  "httpClient": {"blocked": false, "autoBlock": true}
}
JSON
)"
fi

if format_enabled pypi "$NEXUS_PROXY_FORMATS"; then
  create_repo pypi proxy pypi-proxy "$(cat <<'JSON'
{
  "name": "pypi-proxy",
  "online": true,
  "storage": {"blobStoreName": "default", "strictContentTypeValidation": true},
  "proxy": {"remoteUrl": "https://pypi.org", "contentMaxAge": 1440, "metadataMaxAge": 1440},
  "negativeCache": {"enabled": true, "timeToLive": 1440},
  "httpClient": {"blocked": false, "autoBlock": true}
}
JSON
)"
fi

if format_enabled go "$NEXUS_PROXY_FORMATS"; then
  create_repo go proxy go-proxy "$(cat <<'JSON'
{
  "name": "go-proxy",
  "online": true,
  "storage": {"blobStoreName": "default", "strictContentTypeValidation": false},
  "proxy": {"remoteUrl": "https://proxy.golang.org", "contentMaxAge": 1440, "metadataMaxAge": 1440},
  "negativeCache": {"enabled": true, "timeToLive": 1440},
  "httpClient": {"blocked": false, "autoBlock": true}
}
JSON
)"
fi

if format_enabled apt "$NEXUS_PROXY_FORMATS"; then
  # APT proxy requires an explicit distribution codename — operators change it
  # in .env to match the Ubuntu version their projects build against. If you
  # need to proxy multiple distributions, create additional apt-proxy-* repos
  # with different names (one-per-distribution is the Nexus model).
  #
  # These values are interpolated raw into a JSON body. Without a guard an
  # operator could inject arbitrary JSON fields via a crafted .env (or a
  # stray quote in a pasted URL would just produce malformed JSON that
  # Nexus silently rejects). Whitelist to the character set these fields
  # legitimately need — alphanumerics + the URL-safe punctuation. If this
  # ever rejects a valid URL, loosen the pattern; do not drop it.
  safe='^[A-Za-z0-9:/._~?&=+@%-]+$'
  [[ "$NEXUS_APT_DISTRIBUTION" =~ $safe ]] \
    || { echo "error: NEXUS_APT_DISTRIBUTION='${NEXUS_APT_DISTRIBUTION}' contains unsafe characters for JSON interpolation" >&2; exit 65; }
  [[ "$NEXUS_APT_REMOTE_URL" =~ $safe ]] \
    || { echo "error: NEXUS_APT_REMOTE_URL='${NEXUS_APT_REMOTE_URL}' contains unsafe characters for JSON interpolation" >&2; exit 65; }
  create_repo apt proxy "apt-ubuntu-${NEXUS_APT_DISTRIBUTION}-proxy" "$(cat <<JSON
{
  "name": "apt-ubuntu-${NEXUS_APT_DISTRIBUTION}-proxy",
  "online": true,
  "storage": {"blobStoreName": "default", "strictContentTypeValidation": false},
  "proxy": {"remoteUrl": "${NEXUS_APT_REMOTE_URL}", "contentMaxAge": 1440, "metadataMaxAge": 1440},
  "negativeCache": {"enabled": true, "timeToLive": 1440},
  "httpClient": {"blocked": false, "autoBlock": true},
  "apt": {"distribution": "${NEXUS_APT_DISTRIBUTION}", "flat": false}
}
JSON
)"
fi

if format_enabled yum "$NEXUS_PROXY_FORMATS"; then
  # Same JSON-injection guard as APT above — NEXUS_YUM_REMOTE_URL is
  # interpolated raw into the body.
  safe='^[A-Za-z0-9:/._~?&=+@%-]+$'
  [[ "$NEXUS_YUM_REMOTE_URL" =~ $safe ]] \
    || { echo "error: NEXUS_YUM_REMOTE_URL='${NEXUS_YUM_REMOTE_URL}' contains unsafe characters for JSON interpolation" >&2; exit 65; }
  create_repo yum proxy yum-rocky-proxy "$(cat <<JSON
{
  "name": "yum-rocky-proxy",
  "online": true,
  "storage": {"blobStoreName": "default", "strictContentTypeValidation": false},
  "proxy": {"remoteUrl": "${NEXUS_YUM_REMOTE_URL}", "contentMaxAge": 1440, "metadataMaxAge": 1440},
  "negativeCache": {"enabled": true, "timeToLive": 1440},
  "httpClient": {"blocked": false, "autoBlock": true}
}
JSON
)"
fi

if format_enabled docker "$NEXUS_PROXY_FORMATS"; then
  # Docker proxy gets its own HTTP connector (Docker v2 protocol can't
  # path-route) — that's why compose.laptop.yml exposes port 8182 directly
  # to the container. Matching `httpPort: 8182` tells Nexus to listen there.
  # indexType HUB = use Docker Hub's search index metadata (enables `docker
  # search` through the proxy).
  create_repo docker proxy docker-hub-proxy "$(cat <<'JSON'
{
  "name": "docker-hub-proxy",
  "online": true,
  "storage": {"blobStoreName": "default", "strictContentTypeValidation": false},
  "proxy": {"remoteUrl": "https://registry-1.docker.io", "contentMaxAge": 1440, "metadataMaxAge": 1440},
  "negativeCache": {"enabled": true, "timeToLive": 1440},
  "httpClient": {"blocked": false, "autoBlock": true},
  "docker": {"v1Enabled": false, "forceBasicAuth": false, "httpPort": 8182},
  "dockerProxy": {"indexType": "HUB", "cacheForeignLayers": false, "foreignLayerUrlWhitelist": []}
}
JSON
)"
fi

# --- 4. Hosted repositories ---
# For internal platform-team-published packages that cross project boundaries.
# Per-project internal packages go to the project's Forgejo registry, not here.
# Docker is intentionally absent — internal images live in Harbor.

echo "==> Creating hosted repositories (requested: ${NEXUS_HOSTED_FORMATS})"

if format_enabled maven "$NEXUS_HOSTED_FORMATS"; then
  create_repo maven hosted maven-hosted "$(cat <<'JSON'
{
  "name": "maven-hosted",
  "online": true,
  "storage": {"blobStoreName": "default", "strictContentTypeValidation": true, "writePolicy": "ALLOW_ONCE"},
  "maven": {"versionPolicy": "MIXED", "layoutPolicy": "STRICT", "contentDisposition": "INLINE"}
}
JSON
)"
fi

if format_enabled npm "$NEXUS_HOSTED_FORMATS"; then
  create_repo npm hosted npm-hosted "$(cat <<'JSON'
{
  "name": "npm-hosted",
  "online": true,
  "storage": {"blobStoreName": "default", "strictContentTypeValidation": true, "writePolicy": "ALLOW_ONCE"}
}
JSON
)"
fi

if format_enabled pypi "$NEXUS_HOSTED_FORMATS"; then
  create_repo pypi hosted pypi-hosted "$(cat <<'JSON'
{
  "name": "pypi-hosted",
  "online": true,
  "storage": {"blobStoreName": "default", "strictContentTypeValidation": true, "writePolicy": "ALLOW_ONCE"}
}
JSON
)"
fi

if format_enabled raw "$NEXUS_HOSTED_FORMATS"; then
  # raw-hosted is the catch-all for generic file artifacts (release bundles,
  # tarballs, model weights snapshots, etc.) that don't fit a format spec.
  create_repo raw hosted raw-hosted "$(cat <<'JSON'
{
  "name": "raw-hosted",
  "online": true,
  "storage": {"blobStoreName": "default", "strictContentTypeValidation": false, "writePolicy": "ALLOW"},
  "raw": {"contentDisposition": "ATTACHMENT"}
}
JSON
)"
fi

# --- 5. Group repositories ---
# Groups aggregate proxy + hosted for a given format so developers configure
# ONE URL in their build tool and Nexus resolves through the group's member
# order. apt/yum/raw have no group format; go has no group format in v1 of
# the format (developers hit the proxy directly).

echo "==> Creating group repositories"

if format_enabled maven "$NEXUS_PROXY_FORMATS" && format_enabled maven "$NEXUS_HOSTED_FORMATS"; then
  create_repo maven group maven-group "$(cat <<'JSON'
{
  "name": "maven-group",
  "online": true,
  "storage": {"blobStoreName": "default", "strictContentTypeValidation": true},
  "group": {"memberNames": ["maven-hosted", "maven-central-proxy"]},
  "maven": {"versionPolicy": "MIXED", "layoutPolicy": "PERMISSIVE", "contentDisposition": "INLINE"}
}
JSON
)"
fi

if format_enabled npm "$NEXUS_PROXY_FORMATS" && format_enabled npm "$NEXUS_HOSTED_FORMATS"; then
  create_repo npm group npm-group "$(cat <<'JSON'
{
  "name": "npm-group",
  "online": true,
  "storage": {"blobStoreName": "default", "strictContentTypeValidation": true},
  "group": {"memberNames": ["npm-hosted", "npm-proxy"]}
}
JSON
)"
fi

if format_enabled pypi "$NEXUS_PROXY_FORMATS" && format_enabled pypi "$NEXUS_HOSTED_FORMATS"; then
  create_repo pypi group pypi-group "$(cat <<'JSON'
{
  "name": "pypi-group",
  "online": true,
  "storage": {"blobStoreName": "default", "strictContentTypeValidation": true},
  "group": {"memberNames": ["pypi-hosted", "pypi-proxy"]}
}
JSON
)"
fi

# --- 6. Cleanup policies: UI-only in Nexus OSS ---
# The /service/rest/v1/cleanup-policies endpoint exists only in Nexus Pro. The
# OSS OpenAPI spec (verified against 3.79.0) has no such path — POSTs return
# 404, not 401. Automating cleanup-policy creation would require either (a)
# going Pro (not happening for AOH), (b) enabling Groovy scripts (disabled by
# default since 3.21; security trade-off), or (c) mounting a pre-seeded
# `nexus.properties` that seeds policies at first boot.
#
# For the laptop profile we just flag that this is a manual UI step. The prod
# profile Helm chart will revisit via option (c).
echo "==> Cleanup policies: configure manually in the UI (OSS limitation)"
echo "    http://localhost:${NEXUS_HTTP_PORT}/#admin/repository/cleanuppolicies"
echo "    Suggested default: last-downloaded > 90d, format '*', apply to all proxies."

if [[ $BOOTSTRAP_ERRORS -gt 0 ]]; then
  echo "error: bootstrap finished with $BOOTSTRAP_ERRORS fatal-class failure(s); Nexus is likely in a broken state." >&2
  echo "       check container logs ($CLI logs $CONTAINER) and ./scripts/check.sh for specifics." >&2
  exit 78
fi
if [[ $BOOTSTRAP_WARNINGS -gt 0 ]]; then
  echo "==> Bootstrap complete with $BOOTSTRAP_WARNINGS warning(s) (non-fatal)."
else
  echo "==> Bootstrap complete."
fi
