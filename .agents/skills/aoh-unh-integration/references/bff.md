# UNH BFF wiring (`@mssfoobar/unh-web-sdk/server`)

The browser-side `@mssfoobar/unh-client` never calls `unh-service` directly.
It fetches your app's **own** routes under the BFF base (default
`/aoh/unh/api`); those routes proxy to `unh-service` with a **server-side**
bearer token. This file is the complete, copy-pasteable BFF.

## One transparent catch-all proxy — not per-route

The UNH SDK ships **ONE generic catch-all proxy** — `createUnhProxy(cfg)`
from `@mssfoobar/unh-web-sdk/server`. You mount it at a single
`[...path]/+server.ts` in your app.

Why one proxy and not per-route: `@mssfoobar/unh-client` already speaks the
full `unh-service` `/v1` contract (channels, distribution lists + members,
templates, dispatch). The proxy needs no per-endpoint logic — it strips its
mount prefix, forwards the remaining path + query upstream with the caller's
bearer + a correlation id, and passes the `aohhttp` success/pagination AND
error envelopes back **verbatim** with their original status code. The same
`@mssfoobar/unh-client` — pointed at this BFF base by `<UnhProvider>` —
serves the browser, so the proxy is a pure pass-through; there is no second
client and no enrichment hook.

### `createUnhProxy(cfg)` signature

The config object (`UnhProxyConfig`):

| Field | Type | Notes |
|---|---|---|
| `baseUrl` | `string` | Upstream `unh-service` base URL (e.g. `env.UNH_URL`). Trailing slash tolerated. |
| `getToken` | `(event) => string \| undefined \| Promise<…>` | Per-request bearer supplier. Returns the **RAW** token — the proxy prepends the `Bearer` scheme itself (it sets the `authorization` header to `Bearer <token>`). Return `undefined`/empty to forward **no** Authorization header. |
| `bffBase?` | `string` | Mount prefix to strip from the inbound path. Defaults to `DEFAULT_UNH_BFF_BASE` = `"/aoh/unh/api"`. |
| `fetch?` | `typeof fetch` | Override fetch (tests). |

It returns `UnhProxyHandlers` — `{ GET, POST, PUT, PATCH, DELETE }`, **all
the same `handle` function** (it dispatches on `event.request.method`
internally), so any method entry handles any request. `createUnhProxy` is
**not itself callable** — you call one of the returned method handlers, e.g.
`.GET(event)`. Each handler takes a `UnhProxyEvent`
(`{ request: Request; url: URL; locals?: unknown }`) — a structural subset of
SvelteKit's `RequestEvent`, which is assignable.

### The catch-all route

In your app, add the catch-all route (path is yours to choose — it just has
to match the `bffBase` the SDK forwards to; the default is `/aoh/unh/api`):

```ts
// src/routes/aoh/unh/api/[...path]/+server.ts
import type { RequestHandler } from "./$types";
import { env } from "$env/dynamic/private";
import { createUnhProxy } from "@mssfoobar/unh-web-sdk/server";

// Built per-request so a deploy-time UNH_URL is honoured without a rebuild.
// Any method entry works — the handler dispatches on event.request.method.
const handle: RequestHandler = (event) =>
  // $env/dynamic/private is string|undefined; `?? ""` satisfies the string baseUrl.
  createUnhProxy({ baseUrl: env.UNH_URL ?? "", getToken }).GET(event);

export const GET = handle;
export const POST = handle;
export const PUT = handle;
export const PATCH = handle;
export const DELETE = handle;
```

An even simpler form destructures the handlers straight off the factory (this
reads `UNH_URL` once at module load, rather than per-request):

```ts
export const { GET, POST, PUT, PATCH, DELETE } = createUnhProxy({
  baseUrl: env.UNH_URL ?? "",   // $env/dynamic/private is string|undefined
  getToken: (event) => event.locals.authResult?.access_token,
});
```

## `UNH_URL` — the upstream base

`createUnhProxy` needs `baseUrl` — your `unh-service` URL, typically
`env.UNH_URL`. Read it **inside** the handler (the `handle` form above) rather
than at module top-level, so a deploy-time `UNH_URL` is honoured without a
rebuild. The browser never learns the upstream URL — it only ever calls your
same-origin BFF base.

## Forwarding the user's token (fail closed)

The proxy forwards the caller's **server-side** access token to
`unh-service` and **fails closed**: if there's no valid session, forward
**no** token and let the upstream `401` — never substitute a dev/test
identity against a real service (a fail-open identity swap).

`getToken` returns the **RAW** token — the proxy adds the `Bearer` scheme.
Where the token lives is **your app's** concern; read the field off your own
`App.Locals` type. The `getToken(event: { locals?: unknown })` signature
takes the request event and returns the raw token string (or `undefined`).

- **Auth enabled** — return the logged-in user's access token off your
  locals, falling back to `undefined` when there's no successful session so
  the upstream `401`s (fail closed):

  ```ts
  function getToken(event: { locals: App.Locals }): string | undefined {
    const r = event.locals.authResult;
    return r?.success ? (r.access_token ?? undefined) : undefined; // fail closed
  }
  ```

- **Auth disabled (dev/test tier)** — you may forward a fixed dev/test
  bearer so the pages work without a live IdP. Gate this on your own env
  (e.g. presence of an auth/IdP URL) and return the RAW token — strip any
  `Bearer` prefix, since the proxy prepends the scheme:

  ```ts
  const TEST_BEARER = "Bearer <your-dev-token>";

  function getToken(event: { locals?: unknown }): string | undefined {
    if (authEnabled) {
      const r = (event.locals as { authResult?: { success?: boolean; access_token?: string } } | undefined)?.authResult;
      return r?.success ? r.access_token : undefined; // fail closed
    }
    return TEST_BEARER.slice("Bearer ".length); // auth disabled — dev identity
  }
  ```

> Read your own locals type — the exact field name (`access_token` vs
> `accessToken`, etc.) depends on the auth base your app is built on. Never
> forward a dev identity against a real service; that's a fail-open swap.

When `getToken` returns `undefined`/empty the proxy sets no Authorization
header; `unh-service` answers `401` and the BFF passes that through verbatim.

## Server-side `load` over the same BFF

A `+page.server.ts` `load` can reuse the transparent proxy instead of calling
`unh-service` directly: build a `@mssfoobar/unh-client` pointed at your app's
**own** same-origin BFF with the event `fetch`, so the load loops back through
the proxy (which injects the bearer):

```ts
import { createUnhClient, type UnhClient } from "@mssfoobar/unh-client";

export function serverUnhClient(event: { url: URL; fetch: typeof fetch }): UnhClient {
  // unh-client needs an absolute base (new URL(base + path)); resolve against event.url.
  return createUnhClient({ baseUrl: new URL("/aoh/unh/api", event.url).href, fetch: event.fetch });
}
```

This is how you compose your own lists (the SDK ships no list components —
see the skill overview and `references/components.md`).

## The IAMS directory route (distribution-member picker)

`<DistributionMembers>` and `<NotificationTemplateForm>` take an optional
`loadDirectory` prop so an operator picks recipients (users/roles/groups)
from a searchable table instead of typing raw ids. This is a **separate
route from the transparent proxy** — it talks to your identity/access service
(IAMS-AAS), not `unh-service`. Add it in your own app under a path of your
choosing (e.g. `src/routes/aoh/unh/iams/[type]/+server.ts`):

```ts
import { error, json } from "@sveltejs/kit";
import type { RequestHandler } from "./$types";
import { loadIamsDirectory, type DirectoryType } from "$lib/server/iams";

const TYPES = new Set<DirectoryType>(["user", "role", "group"]);

export const GET: RequestHandler = async ({ params, locals, fetch }) => {
  const type = params.type as DirectoryType;
  if (!TYPES.has(type)) error(404, "Unknown directory type");
  return json(await loadIamsDirectory(type, locals, fetch)); // {value,label,hint}[]
};
```

**Loader** — your own `$lib/server/iams.ts` forwards the logged-in user's
bearer to AAS (server-only; the token never reaches the browser), against the
collection path per type:

```text
GET {IAMS_AAS_URL}/admin/tenants/{tenantId}/{memberships|roles|groups}?first=0&max=500
Authorization: Bearer <user token>   →   a bare array of directory representations
```

Read the tenant from your session claims (e.g.
`authResult.claims.active_tenant.tenant_id`) and map each row to
`{ value, label, hint }`. The `value` must be the key the dispatch resolver looks
the member up by, and **it differs by `type`**: `user` → the resolvable **user
id** (`user.id`, not the membership id), `role` → the role **name** (AAS addresses
roles by name), `group` → the group **id**. A `group` keyed by name (mirroring
`role`) silently fails at send time — AAS 404s, the group drops into the send's
`unresolved[]`, and the send still returns 200. **Fail SOFT**: with `IAMS_AAS_URL`
unset, no live session, or any AAS error, return `[]`.

**Why load-then-filter, not search:** the AAS `memberships` `search` is an
**exact-username match** (no substring / prefix / email match), so it's
unusable for type-ahead. Pull the directory once (cap it, e.g. `MAX = 500`)
and let the SDK component filter it client-side.

**Client glue** — a small `DirectoryLoad` the SDK component calls; it fetches
your BFF route and fails soft to `[]`. Keep it in a **client** path (NOT
`$lib/server/*` — SvelteKit forbids importing that into a component) and give it
a **distinct name** from the server loader above, so a component never imports
the server one by mistake:

```ts
// $lib/unh/iams-directory.ts  (client — safe to import into a .svelte)
import type { DirectoryLoad } from "@mssfoobar/unh-web-sdk";

export const iamsDirectory: DirectoryLoad = async (type) => {
  const res = await fetch(`/aoh/unh/iams/${type}`);
  if (!res.ok) return [];
  return res.json();
};
```

**Gating:** only wire `loadDirectory` when IAMS is configured — e.g.
`iamsEnabled: Boolean(env.IAMS_AAS_URL)` in your `+page.server.ts`, then
`loadDirectory={data.iamsEnabled ? iamsDirectory : undefined}` (the **client**
`iamsDirectory`, not the server `loadIamsDirectory`). With `IAMS_AAS_URL` unset
the picker is omitted and the component degrades to **free-text member ids**.

## Environment variables

| Var | Read in | Purpose |
|---|---|---|
| `UNH_URL` | your catch-all `+server.ts` | **Required** — upstream `unh-service` base URL the proxy forwards to. |
| `IAMS_AAS_URL` | your `iams.ts` loader | IAMS picker directory. Unset → picker omitted, free-text member ids. |

Your app's own auth/IdP URL gate is not UNH-specific: set → forward the real
session bearer; unset → dev-identity tier. If your picker needs a live auth
session too, co-gate `iamsEnabled` on it alongside `IAMS_AAS_URL`.

## Trace-context propagation (W3C `traceparent`)

You inject **no** trace headers manually. If your app has OpenTelemetry HTTP
instrumentation, it propagates the active span's trace context (W3C
`traceparent` + baggage, the SDK-default propagators) on outbound `fetch`
automatically — a fetch-level instrumentation (e.g. undici) that hooks
outbound `fetch` keeps client spans + W3C propagation working through
bundling with no module patch. The proxy's own `x-correlation-id` (preserved
from the inbound request or a fresh `crypto.randomUUID()`) rides alongside as
the app-level correlation key.

## Deployment note (server build)

`@mssfoobar/unh-web-sdk` ships subpath exports (`./server`, `./provider`,
`./distribution-members`, …) that resolve to its built `dist/`. Ensure your
server build:

- **installs and builds the SDK** so those subpath exports resolve — a
  package manager that respects the SDK's `exports` map, and (if you build
  from source in a workspace rather than a published tarball) a build step
  that produces the SDK's `dist/` before your app build runs. Skip it and the
  bundler resolves an empty package root and your build fails to find the
  SDK's subpath exports.
- **has `UNH_URL` available to the running server** (and `IAMS_AAS_URL` if
  you use the member picker) — these are read at request time by your
  `+server.ts` routes, so set them in the server's runtime environment
  (container env, `.env`, or your platform's secret store).

If you containerize, that means the SDK (and its peer `@mssfoobar/ui`) must
be present and built in the image's build chain before your app build, and
`UNH_URL` must be set on the running container.
