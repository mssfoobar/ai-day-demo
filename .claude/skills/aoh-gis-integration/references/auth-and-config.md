# Auth & config: the server-side BFF + SDS token pattern

How a GIS consumer wires authentication and configuration — the division
of responsibility between the SDK (config-agnostic) and your host app
(owns config, auth, and the server-side BFF).

## The division of responsibility

The SDK is deliberately **config-agnostic**: it reads no env vars, owns no
token, maintains no global theme. The **host app** owns config and auth.

Crucially, the SDK makes **no GIS REST calls from the browser** — there is no
client-side GIS base URL or auth adapter to supply. All authenticated GIS
traffic (bookmarks, geo-entity reads) goes through the host's **server-side
BFF**, which holds the token; live entity data arrives over RTUS or via
`map.upsert_entities`. The only thing the host hands `<GisProvider>` is the
theme store (`dark_mode_store`).

| Concern | Host supplies | Via |
|---|---|---|
| Theme | `Writable<boolean>` | `<GisProvider dark_mode_store=…>` |
| Upstream GIS base URL | `GIS_URL` (server-only) | the BFF's `fetch(GIS_URL)` |
| Upstream auth | server-held token | the BFF's `authHeaders()` |

## Env var inventory

| Var | Side | Required | Purpose |
|---|---|---|---|
| `GIS_URL` | server (private) | for real backend | BFF → `gis-service` base URL. Unset → BFF stub mode. |
| `PUBLIC_RTUS_SEH_URL` | browser (public) | for live feed | RTUS SEH SSE endpoint. |
| `PUBLIC_CESIUM_TOKEN` | browser (public) | for 3D imagery | Cesium ion token; OSM-only 2D without it. |
| `IAM_URL` | server (private) | for real auth | When set, auth is enabled (OIDC + SDS). Unset → open sandbox / test identity. |

`PUBLIC_`-prefixed vars are exposed to the browser by SvelteKit; the rest
stay server-side. The token-bearing values (`GIS_URL` upstream calls,
`IAM_URL`) are **never** public.

## The SDS server-side token pattern (production auth)

The recommended setup wires the real AOH auth stack via
`@mssfoobar/auth-sdk/sveltekit`: an OIDC Authorization-Code + PKCE flow
against Keycloak, with **SDS** holding the session **server-side**.
The browser only ever carries an opaque, httpOnly session-id cookie — the
access/refresh tokens never leave the server.

Consequences for GIS integration:

- **Upstream GIS calls read the token server-side**, from
  `locals.authResult` (populated by your auth hook in `hooks.server.ts`),
  **not** from page data threaded to the browser. See
  `references/bff-bookmarks.md` → `authHeaders()`.
- The layout guard (`+layout.server.ts`) redirects unauthenticated
  `/aoh/*` requests to login and exposes the user's **claims** — not the
  token — to the UI. On an `aoh-web-init` scaffold the (private) layout
  returns `{ user: authResult.claims }`, i.e. the decoded JWT, so the id
  is the `sub` claim and the tenant is `active_tenant.tenant_id`. Those
  feed `<GisMap user_id={data.user?.sub}
  tenant_id={data.user?.active_tenant?.tenant_id}>`.
- **Fail closed:** if auth is enabled but there's no valid token, send no
  `Authorization` and let the service 401. Never substitute a dev
  identity against a real backend.

Auth is **gated on `IAM_URL`**: unset → the hook is a passthrough (open
sandbox / tests); set → full OIDC, and `ORIGIN`, `IAM_CLIENT_ID`,
`PUBLIC_COOKIE_PREFIX`, `SDS_URL` become required (the app refuses to
start half-configured).

## Test/dev identity (auth disabled only)

For local dev and E2E tiers (`IAM_URL` unset), you can ship a
deterministic dev bearer: a **parseable, unsigned** JWT (`alg: none`)
carrying a canonical test identity (`sub`, `name`, `active_tenant.tenant_id`).
A mock-OIDC test stack validates via a userinfo round-trip (no
signature check), so an unsigned token suffices and avoids shipping a
signing key.

```ts
// minted once; sent by every server-side route that calls fetch(GIS_URL),
// so the upstream sees one identity.
function mintTestJwt(): string {
  const b64 = (s: string) =>
    (typeof Buffer !== "undefined" ? Buffer.from(s, "utf-8").toString("base64") : btoa(s))
      .replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
  const header = b64(JSON.stringify({ alg: "none", typ: "JWT" }));
  const payload = b64(JSON.stringify({
    sub: "dev-test-user",
    name: "Dev Test User",
    active_tenant: { tenant_id: "test-tenant", tenant_name: "Test Tenant" },
  }));
  return `${header}.${payload}.unsigned`;
}
export const TEST_BEARER = `Bearer ${mintTestJwt()}`;
```

> **TEST INFRA ONLY.** Never use an unsigned token against a real issuer.
> Have every server route that talks to `gis-service` forward the **same**
> `TEST_BEARER` so the upstream sees a single identity across them.

## Styles & theme together

The whole shell (sidebar/navbar) and the SDK panels theme off the `.dark`
class on `<html>`. The host toggles the class **and** updates the
`dark_mode_store` it passed to `<GisProvider>` so both the design tokens
and the SDK's map palette flip together. Keep a single CSS entry
(`src/app.css`) that chains the `ui` and `gis-web-sdk` stylesheets so
Tailwind `@source` directives aggregate in one pass (see SKILL Step 8).
