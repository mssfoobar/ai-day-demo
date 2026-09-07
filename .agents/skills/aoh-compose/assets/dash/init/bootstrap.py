#!/usr/bin/env python3
"""dash bootstrap — optionally seed dashboards into dash-service from dashboard.yaml.

Mirrors the project-aas bootstrap pattern (../iams/init/project-aas): fetch the
desired state from YAML, fetch the current state from dash-service, reconcile. Safe
to re-run — re-running after `docker compose down -v` converges on the same
dashboards.

OPTIONAL: only needed when dashboards must exist before first use. If admins
create dashboards in the UI at runtime, leave the `dash-init` block in
../compose.yml commented out (and you may delete this ./init dir) — seeding is
not required. See dashboard.yaml's header.

Auth: a PASSWORD grant as the dev user. dash-service scopes every read/write by the
token's `active_tenant` claim, and only interactive grants (password / auth-code)
carry it — a `client_credentials` service-account token does NOT (its membership
shows in `all_tenants` but `active_tenant` stays null), so it cannot create a
tenant-scoped dashboard. The dev user is a member of the seed tenant via
iams-init, so its active_tenant resolves.

Scope: DEV-time bootstrap. Production dashboard provisioning is owned by the
deployment pipeline.
"""

from __future__ import annotations
import logging
import os
import sys
import time

import requests
import yaml

log = logging.getLogger("dash-init")


class Env:
    aas = os.environ["IAMS_AAS_ENDPOINT"]       # e.g. http://iams-aas:8080
    iam = os.environ["IAM_ENDPOINT"]            # e.g. http://iams-keycloak:8080
    realm = os.environ.get("IAM_REALM", "aoh")
    user = os.environ["DEV_USER"]               # admin
    password = os.environ["DEV_PASSWORD"]       # P@ssw0rd in dev
    client = os.environ.get("IAM_CLIENT_ID", "iams")
    dash = os.environ["DASH_URL"].rstrip("/")   # e.g. http://dash-service:8080
    tenant_name = os.environ.get("SEED_TENANT", "development")
    timeout = int(os.environ.get("HTTP_TIMEOUT", "30"))


def get_token() -> str:
    """Password grant — the token carries the `active_tenant` claim dash-service
    scopes by (a client_credentials token would not). Same pattern as the
    platform iams-init / project-aas bootstrap."""
    r = requests.post(
        f"{Env.iam}/realms/{Env.realm}/protocol/openid-connect/token",
        data={
            "grant_type": "password",
            "client_id": Env.client,
            "username": Env.user,
            "password": Env.password,
            "scope": "openid",
        },
        timeout=Env.timeout,
    )
    r.raise_for_status()
    return r.json()["access_token"]


def wait_for(url: str, token: str, attempts: int = 40, delay: float = 3.0) -> None:
    """Poll until the endpoint answers (dash-service runs its schema migrations on
    boot, so it may 5xx briefly)."""
    h = {"Authorization": f"Bearer {token}"}
    for i in range(attempts):
        try:
            if requests.get(url, headers=h, timeout=Env.timeout).status_code < 500:
                return
        except requests.RequestException:
            pass
        log.info("waiting for %s (%d/%d)", url, i + 1, attempts)
        time.sleep(delay)
    sys.exit(f"{url} never became reachable")


def assert_tenant(token: str, tenant_name: str) -> None:
    """Guard: confirm the seed tenant exists in AAS before seeding. dash-service
    scopes by the token's active_tenant (we never pass tenant_id), but this
    surfaces a clear error if the environment isn't seeded yet."""
    h = {"Authorization": f"Bearer {token}"}
    r = requests.get(
        f"{Env.aas}/admin/tenants", params={"search": tenant_name},
        headers=h, timeout=Env.timeout,
    )
    r.raise_for_status()
    if not any(t.get("name") == tenant_name for t in (r.json() or [])):
        sys.exit(f"tenant {tenant_name!r} not found in AAS — iams-init must run first")


def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    spec_file = sys.argv[1] if len(sys.argv) > 1 else "dashboard.yaml"
    spec = yaml.safe_load(open(spec_file)) or {}
    tenant_name = spec.get("tenant") or Env.tenant_name

    token = get_token()
    wait_for(f"{Env.dash}/widget/type?size=1", token)
    assert_tenant(token, tenant_name)

    s = requests.Session()
    s.headers.update({"Authorization": f"Bearer {token}", "Content-Type": "application/json"})

    # Widget types first so the dashboards' widget foreign keys resolve.
    for wt in spec.get("widget_types", []):
        r = s.post(f"{Env.dash}/widget/type", json=wt, timeout=Env.timeout)
        if r.status_code == 409:
            log.info("widget type %s: exists", wt["id"])
        elif r.ok:
            log.info("widget type %s: created", wt["id"])
        else:
            sys.exit(f"widget type {wt['id']}: http {r.status_code} {r.text[:300]}")

    # Dashboards — resolve by name, create only on miss (idempotent).
    for d in spec.get("dashboards", []):
        name = d["name"]
        r = s.get(f"{Env.dash}/dashboard/name/{requests.utils.quote(name)}", timeout=Env.timeout)
        if r.status_code == 200:
            log.info("dashboard %r: exists — leaving as-is", name)
            continue
        # dash-service returns 500 "sql: no rows in result set" (not 404) when absent.
        if r.status_code != 404 and "no rows in result set" not in r.text:
            sys.exit(f"resolve dashboard {name!r}: http {r.status_code} {r.text[:300]}")
        body = {
            "name": name,
            "description": d.get("description", ""),
            "favourite": d.get("favourite", False),
            "tags": [],
            "widgets": d.get("widgets", []),
        }
        r = s.post(f"{Env.dash}/dashboard", json=body, timeout=Env.timeout)
        if r.status_code == 409:
            log.info("dashboard %r: created concurrently — ok", name)
        elif r.ok:
            log.info("dashboard %r: created", name)
        else:
            sys.exit(f"create dashboard {name!r}: http {r.status_code} {r.text[:400]}")


if __name__ == "__main__":
    main()
