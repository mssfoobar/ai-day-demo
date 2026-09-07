#!/usr/bin/env python3
"""
Project-owned AAS bootstrap (development only).

Reads roles.yaml and applies it idempotently to IAMS-AAS by walking the
authz dependency graph in order:

    roles -> groups -> resources -> scopes -> resource_scopes
          -> assignments (user->role, user->group)
          -> tenant_admins (user -> full admin of the tenant)
          -> permissions (role|group|user -> resource+scope)

Each step is "fetch desired state from YAML, fetch current state from AAS,
reconcile."  Re-running this script after `docker compose down -v` (or after
a teammate adds a role to roles.yaml) converges on the same end state.

Scope:  this is a DEV-time bootstrap. Production AAS configuration is owned
        by the deployment pipeline and uses a different tenant / role matrix
        / secrets-management approach. Do not extend this script to handle
        production concerns.
"""

from __future__ import annotations
import os, sys, time, logging, argparse
from typing import Any
import requests, yaml

log = logging.getLogger("project-aas")

# --------------------------------------------------------------------------- #
# Config / env                                                                #
# --------------------------------------------------------------------------- #

class Env:
    aas      = os.environ["IAMS_AAS_ENDPOINT"]        # e.g. http://iams-aas:8080
    iam      = os.environ["IAM_ENDPOINT"]             # e.g. http://iams-keycloak:8080
    realm    = os.environ.get("IAM_REALM", "aoh")
    user     = os.environ["DEV_USER"]                 # admin
    password = os.environ["DEV_PASSWORD"]             # P@ssw0rd in dev
    client   = os.environ.get("IAM_CLIENT_ID", "iams")
    timeout  = int(os.environ.get("HTTP_TIMEOUT", "30"))

# --------------------------------------------------------------------------- #
# HTTP helpers                                                                #
# --------------------------------------------------------------------------- #

def _entity_id(obj: dict) -> str:
    """AAS resource/scope responses use `_id` (Mongo-style); other endpoints
    use `id`. Accept either."""
    return obj.get("id") or obj.get("_id")


def get_token() -> str:
    """Password grant against Keycloak — same pattern as platform iams-init."""
    r = requests.post(
        f"{Env.iam}/realms/{Env.realm}/protocol/openid-connect/token",
        data={
            "grant_type": "password",
            "client_id":  Env.client,
            "username":   Env.user,
            "password":   Env.password,
            "scope":      "openid",
        },
        timeout=Env.timeout,
    )
    r.raise_for_status()
    return r.json()["access_token"]


def wait_for_aas(token: str, attempts: int = 30, delay: float = 2.0) -> None:
    """AAS may not be ready immediately even with depends_on. Poll briefly."""
    h = {"Authorization": f"Bearer {token}"}
    for i in range(attempts):
        try:
            r = requests.get(f"{Env.aas}/admin/tenants?max=1", headers=h, timeout=Env.timeout)
            if r.ok:
                return
        except requests.RequestException:
            pass
        log.info("AAS not ready, retry %d/%d", i + 1, attempts)
        time.sleep(delay)
    sys.exit("AAS never became reachable")


class AAS:
    """Thin wrapper around the AAS admin API. All methods are idempotent."""

    def __init__(self, token: str):
        self.s = requests.Session()
        self.s.headers["Authorization"] = f"Bearer {token}"
        self.s.headers["Content-Type"]  = "application/json"

    # ---- generic ---------------------------------------------------------- #

    @staticmethod
    def _safe_json(r: requests.Response) -> Any:
        """Return parsed JSON or None.

        AAS sometimes responds 2xx with a non-JSON body (e.g. a plain
        confirmation string after a successful POST). The previous
        ``r.json() if r.content else None`` raised JSONDecodeError on those
        responses and aborted the whole bootstrap. We log the offending
        body once at debug and return None — the caller's idempotent retry
        path will then proceed via the matching GET.
        """
        if not r.content:
            return None
        try:
            return r.json()
        except ValueError:
            preview = r.text[:200].replace("\n", " ")
            log.debug(
                "non-JSON response from %s %s (status=%s): %r",
                r.request.method,
                r.request.url,
                r.status_code,
                preview,
            )
            return None

    def _get(self, path: str, **params) -> Any:
        r = self.s.get(f"{Env.aas}{path}", params=params, timeout=Env.timeout)
        if r.status_code == 404:
            return None
        r.raise_for_status()
        return self._safe_json(r)

    def _post(self, path: str, body: Any) -> Any:
        r = self.s.post(f"{Env.aas}{path}", json=body, timeout=Env.timeout)
        r.raise_for_status()
        return self._safe_json(r)

    def _put(self, path: str, body: Any = None) -> Any:
        r = self.s.put(f"{Env.aas}{path}", json=body, timeout=Env.timeout)
        r.raise_for_status()
        return self._safe_json(r)

    # ---- lookups ---------------------------------------------------------- #

    def tenant_id(self, name: str) -> str:
        tenants = self._get("/admin/tenants", search=name) or []
        match = next((t for t in tenants if t["name"] == name), None)
        if not match:
            sys.exit(f"tenant {name!r} not found — platform iams-init must run first")
        return match["id"]

    def user_id(self, username: str) -> str:
        users = self._get("/admin/users", search=username) or []
        match = next((u for u in users if u["username"] == username), None)
        if not match:
            sys.exit(f"user {username!r} not found in Keycloak — add to realm-import.json")
        return match["id"]

    # ---- roles (RBAC) ----------------------------------------------------- #

    def ensure_role(self, tid: str, name: str, description: str, required: bool = False):
        existing = self._get(f"/admin/tenants/{tid}/roles/{name}")
        if existing:
            log.info("role %s: exists", name)
            return existing
        log.info("role %s: creating", name)
        return self._post(
            f"/admin/tenants/{tid}/roles",
            {"name": name, "description": description, "required": required},
        )

    def ensure_membership(self, tid: str, user_id: str):
        """AAS rejects role assignment with 409 if the user isn't a tenant member.
        iams-init only adds the admin to the `development` tenant, so any
        non-admin user referenced in roles.yaml (e.g. seed users from
        realm-import.json) must be added here. Idempotent: a 409 from the
        POST means "already a member" and is treated as success."""
        members = self._get(f"/admin/tenants/{tid}/memberships") or []
        # A membership record nests the user under `user`; the record's own `id` is the
        # membership id, NOT the user id — so match on user.id or the skip-check never
        # fires and we needlessly re-POST every run (the 409 below is the safety net).
        member_ids = {(m.get("user") or {}).get("id") for m in members if isinstance(m, dict)}
        if user_id in member_ids:
            return
        log.info("membership %s -> tenant %s: adding", user_id, tid)
        try:
            self._post(f"/admin/tenants/{tid}/memberships/{user_id}", {})
        except requests.HTTPError as e:
            if e.response is not None and e.response.status_code == 409:
                return
            raise

    def assign_role(self, tid: str, user_id: str, role_name: str):
        self.ensure_membership(tid, user_id)
        current = {r["name"] for r in self._get(f"/admin/tenants/{tid}/users/{user_id}/roles") or []}
        if role_name in current:
            log.info("assignment %s -> %s: exists", user_id, role_name)
            return
        log.info("assignment %s -> %s: creating", user_id, role_name)
        self._post(
            f"/admin/tenants/{tid}/users/{user_id}/roles",
            [{"name": role_name}],
        )

    def ensure_tenant_admin(self, tid: str, user_id: str) -> None:
        """Make the user a full administrator of this tenant (manage members,
        roles, resources, permissions) — the same grant you'd otherwise set by
        hand in the iams-web admin console. AAS requires tenant membership first,
        so ensure it. Idempotent: skip if already an admin; a 409 from the POST
        means 'already an admin' and is treated as success."""
        self.ensure_membership(tid, user_id)
        admins = self._get(f"/admin/tenants/{tid}/tenant-admin/users") or []
        admin_ids = {a.get("id") or a.get("userId") for a in admins if isinstance(a, dict)}
        if user_id in admin_ids:
            log.info("tenant-admin %s: exists", user_id)
            return
        log.info("tenant-admin %s -> tenant %s: assigning", user_id, tid)
        try:
            self._post(f"/admin/tenants/{tid}/tenant-admin/users/{user_id}", {})
        except requests.HTTPError as e:
            if e.response is not None and e.response.status_code == 409:
                return
            raise

    # ---- groups (GBAC) ---------------------------------------------------- #

    def ensure_group(self, tid: str, name: str, parent_id: str | None) -> str:
        """Idempotent. Top-level if parent_id is None, otherwise child of parent."""
        list_path = (
            f"/admin/tenants/{tid}/groups/{parent_id}/children"
            if parent_id
            else f"/admin/tenants/{tid}/groups"
        )
        existing = next(
            (g for g in (self._get(list_path) or []) if g["name"] == name),
            None,
        )
        if existing:
            log.info("group %s: exists", name)
            return existing["id"]
        log.info("group %s: creating (parent=%s)", name, parent_id or "<root>")
        # GroupRepresentation has no `description` (it's roles-only) — send name only.
        created = self._post(list_path, {"name": name})
        return created["id"]

    def assign_group_role(self, tid: str, group_id: str, role_name: str) -> None:
        current = {r["name"] for r in self._get(f"/admin/tenants/{tid}/groups/{group_id}/roles") or []}
        if role_name in current:
            log.info("group-role %s -> %s: exists", group_id, role_name)
            return
        log.info("group-role %s -> %s: creating", group_id, role_name)
        self._post(f"/admin/tenants/{tid}/groups/{group_id}/roles", [{"name": role_name}])

    def add_group_member(self, tid: str, group_id: str, user_id: str) -> None:
        current = {g["id"] for g in self._get(f"/admin/tenants/{tid}/users/{user_id}/groups") or []}
        if group_id in current:
            log.info("group-member %s -> %s: exists", group_id, user_id)
            return
        log.info("group-member %s -> %s: adding", group_id, user_id)
        self._put(f"/admin/tenants/{tid}/users/{user_id}/groups/{group_id}")

    # ---- resources + scopes ---------------------------------------------- #

    # NOTE: AAS resource/scope endpoints return the entity ID as `_id` (Mongo
    # style), NOT `id`. Use _entity_id() to normalize across both.
    def ensure_resource(self, tid: str, name: str, rtype: str, display_name: str) -> str:
        def find():
            return next(
                (r for r in (self._get(f"/admin/tenants/{tid}/resources", name=name) or []) if r["name"] == name),
                None,
            )
        existing = find()
        if existing:
            log.info("resource %s: exists", name)
            return _entity_id(existing)
        log.info("resource %s: creating", name)
        # AAS ResourceRepresentation takes `displayName` (a human-friendly label),
        # NOT `description` (that field is roles-only). Omit it when unset rather
        # than sending "".
        body: dict = {"name": name}
        if rtype:
            body["type"] = rtype
        if display_name:
            body["displayName"] = display_name
        # _post returns None when AAS answers 2xx with a non-JSON body (see
        # _safe_json) — recover the id via the matching GET, as that contract
        # promises, instead of feeding None to _entity_id.
        created = self._post(f"/admin/tenants/{tid}/resources", body) or find()
        if created is None:
            sys.exit(f"resource {name!r} created but could not be read back from AAS")
        return _entity_id(created)

    def ensure_scope(self, tid: str, name: str) -> str:
        def find():
            return next(
                (s for s in (self._get(f"/admin/tenants/{tid}/scopes") or []) if s["name"] == name),
                None,
            )
        existing = find()
        if existing:
            log.info("scope %s: exists", name)
            return _entity_id(existing)
        log.info("scope %s: creating", name)
        created = self._post(f"/admin/tenants/{tid}/scopes", {"name": name}) or find()
        if created is None:
            sys.exit(f"scope {name!r} created but could not be read back from AAS")
        return _entity_id(created)

    def bind_scope_to_resource(self, tid: str, resource_id: str, scope_id: str) -> None:
        current = {_entity_id(s) for s in self._get(f"/admin/tenants/{tid}/resources/{resource_id}/scopes") or []}
        if scope_id in current:
            log.info("resource-scope %s -> %s: bound", resource_id, scope_id)
            return
        log.info("resource-scope %s -> %s: binding", resource_id, scope_id)
        self._put(f"/admin/tenants/{tid}/resources/{resource_id}/scopes/{scope_id}")

    # ---- permissions: PUT-the-whole-array (naturally idempotent) ---------- #
    #
    # PUT semantics — the array supplied REPLACES the current grant set on that
    # resource+scope. Removing a name/id from roles.yaml revokes it on next run.
    # That is intentional: YAML is the source of truth for the access matrix.

    def set_role_permissions(self, tid: str, resource_id: str, scope_id: str, role_names: list[str]) -> None:
        # AAS RoleDefinition is `{id: <uuid>}` per iams-aas-openapi.json — NOT
        # `{name: <role>}`. The role name → id lookup happens here so callers
        # stay declarative.
        body = []
        for n in role_names:
            r = self._get(f"/admin/tenants/{tid}/roles/{n}")
            if not r:
                sys.exit(f"role {n!r} not found while granting permissions")
            body.append({"id": _entity_id(r)})
        log.info("perm roles %s/%s: %s", resource_id, scope_id, role_names)
        self._put(
            f"/admin/tenants/{tid}/resources/{resource_id}/scopes/{scope_id}/permissions/roles",
            body,
        )

    def set_group_permissions(self, tid: str, resource_id: str, scope_id: str, group_ids: list[str]) -> None:
        body = [{"id": gid} for gid in group_ids]
        log.info("perm groups %s/%s: %s", resource_id, scope_id, group_ids)
        self._put(
            f"/admin/tenants/{tid}/resources/{resource_id}/scopes/{scope_id}/permissions/groups",
            body,
        )

    def set_user_permissions(self, tid: str, resource_id: str, scope_id: str, user_ids: list[str]) -> None:
        log.info("perm users %s/%s: %s", resource_id, scope_id, user_ids)
        self._put(
            f"/admin/tenants/{tid}/resources/{resource_id}/scopes/{scope_id}/permissions/users",
            user_ids,
        )

# --------------------------------------------------------------------------- #
# Reconcile — the dependency-ordered walk                                     #
# --------------------------------------------------------------------------- #

def reconcile(aas: AAS, data: dict) -> None:
    tid = aas.tenant_id(data["tenant"])

    # 1. Roles
    for r in data.get("roles") or []:
        aas.ensure_role(tid, r["name"], r.get("description", ""), r.get("required", False))

    # 2. Groups (must exist before assigning roles/members to them)
    group_ids: dict[str, str] = {}
    for g in data.get("groups") or []:
        parent_id = group_ids.get(g["parent"]) if g.get("parent") else None
        gid = aas.ensure_group(tid, g["name"], parent_id)
        group_ids[g["name"]] = gid
        for role_name in g.get("roles") or []:
            aas.assign_group_role(tid, gid, role_name)
        for username in g.get("members") or []:
            aas.add_group_member(tid, gid, aas.user_id(username))

    # 3. Resources
    resource_ids: dict[str, str] = {}
    for res in data.get("resources") or []:
        rid = aas.ensure_resource(tid, res["name"], res.get("type", ""), res.get("displayName", ""))
        resource_ids[res["name"]] = rid

    # 4. Scopes
    scope_ids: dict[str, str] = {}
    for sc in data.get("scopes") or []:
        sid = aas.ensure_scope(tid, sc["name"])
        scope_ids[sc["name"]] = sid

    # 5. Resource <-> scope bindings (must come after both exist)
    for binding in data.get("resource_scopes") or []:
        rid = resource_ids[binding["resource"]]
        for scope_name in binding["scopes"]:
            aas.bind_scope_to_resource(tid, rid, scope_ids[scope_name])

    # 6. User -> role assignments
    for a in data.get("assignments") or []:
        uid = aas.user_id(a["username"])
        for role_name in a.get("roles") or []:
            aas.assign_role(tid, uid, role_name)

    # 7. Tenant admins — full admin over THIS tenant only (membership ensured first).
    for username in data.get("tenant_admins") or []:
        aas.ensure_tenant_admin(tid, aas.user_id(username))

    # 8. Permissions — RBAC + GBAC + UBAC compose here.
    #    PUT-the-whole-array means YAML is the source of truth; removing
    #    a role/group/user from YAML revokes it on next run.
    for p in data.get("permissions") or []:
        rid = resource_ids[p["resource"]]
        sid = scope_ids[p["scope"]]
        aas.set_role_permissions(tid,  rid, sid, p.get("roles")  or [])
        aas.set_group_permissions(tid, rid, sid, [group_ids[g] for g in p.get("groups") or []])
        aas.set_user_permissions(tid,  rid, sid, [aas.user_id(u) for u in p.get("users") or []])

# --------------------------------------------------------------------------- #
# Entry point                                                                 #
# --------------------------------------------------------------------------- #

def main() -> int:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    ap = argparse.ArgumentParser()
    ap.add_argument("roles_file", help="Path to roles.yaml")
    args = ap.parse_args()

    with open(args.roles_file) as f:
        data = yaml.safe_load(f)

    token = get_token()
    wait_for_aas(token)
    reconcile(AAS(token), data)
    log.info("bootstrap complete")
    return 0

if __name__ == "__main__":
    sys.exit(main())
