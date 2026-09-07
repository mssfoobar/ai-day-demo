# Project AAS Bootstrap (development only)

Seeds project-specific authorization state into IAMS-AAS so a freshly cloned
repo + `docker compose up` produces a working local environment with the
roles, groups, resources, scopes, and grants this project's services expect.

> **Dev-only.** This artifact is for local Docker Compose stacks. Production
> AAS configuration is owned by the deployment pipeline and is intentionally
> not the same as this file — different tenant name, different role matrix,
> different secrets management. Do not treat `roles.yaml` here as the
> production source of truth.

## What this is and isn't

| | This `project-aas/` | Platform `iams-aas-init.postman_collection.json` |
|--|--|--|
| Owner | Your project | Upstream IAMS platform |
| Creates | Application roles, groups, resources, scopes, permissions, assignments | The `development` tenant + admin membership |
| When to edit | Adding a new role / group / grant | Never |
| Re-runs on | Every `docker compose up` | Every `docker compose up` |

The two run in order: platform's collection first (creates the tenant), then
this script (populates the tenant). The `project-aas-init` compose service
gates on the platform via
`depends_on: { iams-init: { condition: service_completed_successfully } }`.

## Layout

```
project-aas/
├── README.md          # this file
├── Dockerfile         # python:3.12-slim + bootstrap.py + roles.yaml
├── requirements.txt   # requests, PyYAML
├── bootstrap.py       # idempotent reconciler — do not run by hand
└── roles.yaml         # the data — this is what you edit
```

## How to add a role

1. Append the role under `roles:` in `roles.yaml`:
   ```yaml
   roles:
     - name: operations-officer
       description: Triages and assigns incoming incidents
   ```
2. Add it to whichever user should have it:
   ```yaml
   assignments:
     - username: admin
       roles: [operations-officer]
   ```
3. Apply: `docker compose up -d --build project-aas-init`
4. Verify: obtain a token for the user and decode it — the new role appears
   under `active_tenant.roles`:
   ```bash
   TOKEN=$(curl -s -X POST "http://iams-keycloak.127.0.0.1.nip.io/realms/aoh/protocol/openid-connect/token" \
     -d grant_type=password -d client_id=iams \
     -d username=admin -d password=P@ssw0rd -d scope=openid | jq -r .access_token)
   echo "$TOKEN" | cut -d. -f2 | base64 -d 2>/dev/null | jq .active_tenant.roles
   ```

## Fine-grained permissions (RBAC + GBAC + UBAC)

`roles.yaml` has slots for **every** AAS authorization concept, most empty
by default. Fill the slot you need; `bootstrap.py` walks them in dependency
order:

```
roles → groups → resources → scopes → resource_scopes
      → assignments → permissions
```

The `permissions:` section is where the three authorization models compose —
each entry grants a `(resource, scope)` pair to any combination of `roles`,
`groups`, and `users`. The PUT semantics mean removing a name from a grant
**revokes** it on the next run; the YAML is the source of truth for the
access matrix.

## Idempotency

Every step is GET-then-create, except `permissions:` which is
PUT-the-whole-array. Re-running is safe and is in fact how the system
works — every `docker compose up` re-applies the YAML.

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| `tenant 'development' not found` | Platform `iams-init` didn't run first | Check `docker compose ps` — the `iams-init` container should show exit 0 |
| `user 'xyz' not found in Keycloak` | User isn't in `realm-import.json` | Add the user to the realm export and `docker compose down -v && up -d` |
| 401 from AAS | `DEV_USER` / `DEV_PASSWORD` env vars not passed through | Check the `project-aas-init` `environment:` block in `compose/iams/compose.yml` |
| Role not in JWT after re-run | Token was issued before the role was created | Re-fetch the token; old tokens are not retroactively updated |

## Don't

- **Don't add application roles to `compose/iams/keycloak/realm-import.json`.**
  That file is platform-bootstrap, upstream-owned, and only carries OIDC
  clients + the realm-level platform roles (`system-admin`, `tenant-admin`,
  `default-roles-aoh`). Application roles live in AAS, not the realm.
- **Don't seed via ad-hoc cURL or the AAS Swagger UI.** State created that
  way is invisible to teammates and vanishes on `docker compose down -v`.
- **Don't edit `iams-aas-init.postman_collection.json`.** That's the
  platform's tenant-creation collection — upstream replaces it on upgrades.
- **Don't repurpose this for production.** Production AAS state belongs to
  the deployment pipeline, not a local-dev compose container.
