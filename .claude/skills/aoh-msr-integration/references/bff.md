# MSR BFF wiring (`@mssfoobar/msr-web-sdk/server`)

The browser-side store and worker never call `msr-service`. They fetch the
host's **own** routes under `msr_bff_base` (default `/aoh/msr/api`); those
routes proxy to `msr-service` with a **server-side** bearer token. This
file is the complete, copy-pasteable BFF.

The SDK gives you `msrBffHandlers(cfg)` — a factory returning the BFF
handlers: **seven handler groups** (eight methods, since `admin/settings`
carries both GET and PUT) across **seven route files**, built on
`@mssfoobar/msr-client`. Each accepts a structural subset of `RequestEvent`
(`{ request, params? }`) and returns a standard `Response`, so any
fetch-API server can host them.

## The config (`MsrBffConfig`)

`MsrBffConfig` extends `@mssfoobar/msr-client`'s `MsrClientConfig`:

| Field | Type | Notes |
|---|---|---|
| `baseUrl` | `string` | Upstream `msr-service` base URL. |
| `getToken` | `() => string \| Promise<string>` | Returns the **raw** token — the client adds the `Bearer` scheme itself. Don't include it. |
| `fetch` | `typeof fetch` | Optional. The OTEL HTTP instrumentation auto-propagates the active span's trace context (W3C `traceparent`) on outbound calls — no manual header injection. |
| `enrichSessions` | `(sessions: Session[]) => UserSession[] \| Promise<…>` | Optional hook for the `admin.sessions` GET — join IAMS user info (`fullName` / `userName`) onto the raw service sessions. Omit → sessions pass through unenriched and the UI shows `user_id`. |

## The handlers (seven groups, eight methods)

`msrBffHandlers(cfg)` returns a nested object — mount one route file per
group (`admin.settings` exposes GET + PUT from a single file):

| Handler | Route (under the BFF base) | Method | Returns |
|---|---|---|---|
| `session.create.POST` | `/session` | POST | `201` + `{ data: Session, message, sent_at }` |
| `session.terminate.POST` | `/session/[sessionId]/terminate` | POST | `204` |
| `admin.sessions.GET` | `/admin/sessions` | GET | paginated envelope `{ data: Session[]\|UserSession[], page }` (enriched if configured) |
| `admin.settings.GET` | `/admin/settings` | GET | `{ data: AdminSettings, sent_at }` |
| `admin.settings.PUT` | `/admin/settings` | PUT | echoes back **only** the submitted partial (`AdminSettingsUpdate`) |
| `admin.terminate.POST` | `/admin/sessions/terminate` | POST | `204`; body `{ session_ids: string[] }` (non-empty) |
| `replay.state.GET` | `/replay/state` | GET | `{ data: ChangeEvent[] }`; query `timestamp` (required), `filter_tables?`, `filter_min_timestamp?`, `explain?` |
| `replay.events.GET` | `/replay/events` | GET | `{ data: ChangeEvent[] }`; query `start` + `end` (required), `filter_tables?`, `filter_min_timestamp?` |

Notes worth knowing:

- **`admin.sessions` defaults** to `page=1, size=50` if the query omits
  them.
- **`session.terminate`** reads the id from `params.sessionId`, falling
  back to parsing `/session/:id/terminate` out of the URL — so it works
  whether or not your router populates `params`.
- **`admin.settings.PUT`** echoes only the keys you sent (the service
  never returns the full document on a partial update).
- **Missing required params** → a `400 INVALID_REQUEST` error envelope
  from the handler itself (it doesn't round-trip a malformed request).

## Wire contract & observability

- **Successes** are re-wrapped in the aoh envelope `{ data, message?,
  sent_at }` (or the pagination envelope for list endpoints) — exactly
  what the browser API client and worker expect.
- **`msr-service` errors pass through verbatim** with their original
  status code and `{ timestamp, trace_id?, errorCode, errorMessage,
  details? }` envelope; anything unexpected becomes a `500 UNEXPECTED_ERROR`.
- **One trace_id per request**: the active span's `trace_id` is the single
  correlation key. The OTEL HTTP instrumentation propagates trace context
  upstream automatically (W3C `traceparent`) and the same `trace_id` is
  surfaced in any error envelope returned (omitted when no trace is
  active) — so the support id the user sees matches the distributed trace
  and what `msr-service` logged. Handler failures are logged (with the
  trace_id auto-injected by the `@mssfoobar/logger` OTEL mixin) + status
  before the envelope is returned.

## Forwarding the user's token (fail closed)

The BFF forwards the caller's **server-side** access token to `msr-service`
and **fails closed**: if there's no valid session, send an empty token and
let the upstream `401` — never substitute a dev/test identity against a
real service (a fail-open identity swap). **Stub mode (`MSR_URL` unset)
needs no token** — it only matters once you point at a real service.

Where the token lives is **your app's** concern — the SDK prescribes no
auth module, so read the field off your own `App.Locals.authResult` type,
not from this doc. The exact field name depends on the auth base your app
is built on: some bases expose it as **`access_token`** (snake_case),
others as **`accessToken`** (camelCase). The two shapes:

- **Auth enabled** — narrow `authResult` on `success`, return the raw
  token, and fall back to an empty string when there's no valid session so
  the upstream `401`s (fail closed):

  ```ts
  // src/lib/server/msr-bff.ts
  import { env } from "$env/dynamic/private";
  import { msrBffHandlers, type MsrBffHandlers } from "@mssfoobar/msr-web-sdk/server";
  import { msrStubHandlers } from "./msr-stub"; // backend-free dev — see references/stub.md

  /** RAW token (msr-client adds "Bearer "); fail closed. */
  function bearerToken(locals: App.Locals): string {
    const authResult = locals.authResult;                 // your own App.Locals shape
    // snake_case bases: authResult.access_token; camelCase bases: authResult.accessToken
    return authResult.success ? (authResult.access_token ?? "") : "";
  }

  export function msrBff(locals: App.Locals): MsrBffHandlers {
    if (!env.MSR_URL) return msrStubHandlers;             // backend-free dev / e2e
    return msrBffHandlers({
      baseUrl: env.MSR_URL,
      getToken: () => bearerToken(locals),
      // enrichSessions: (sessions) => joinIamsUserInfo(sessions),  // optional
    });
  }
  ```

- **Auth disabled (dev/test tier)** — if your app supports a no-IdP dev
  tier, you may gate it on the presence of an IdP URL (e.g. `IAM_URL`) and
  forward a fixed dev bearer so the pages work without a live IdP. Same
  fail-closed invariant when auth *is* enabled:

  ```ts
  function bearerToken(locals: App.Locals): string {
    if (!env.IAM_URL) return DEV_BEARER;                  // auth disabled (e2e only)
    const r = (locals as { authResult?: { success?: boolean; accessToken?: string } }).authResult;
    return r?.success && r.accessToken ? r.accessToken : "";  // fail closed
  }
  ```

> Read your own locals type — the exact field name (`access_token` vs
> `accessToken`) and whether you support a dev-identity tier depend on the
> auth base your app is built on. Never forward a dev identity against a
> real service; that's a fail-open swap.

## The route files (seven one-liners)

Each route is a pass-through to `msrBff(locals)`:

```ts
// src/routes/aoh/msr/api/session/+server.ts
import type { RequestHandler } from "@sveltejs/kit";
import { msrBff } from "$lib/server/msr-bff";
export const POST: RequestHandler = ({ request, params, locals }) =>
  msrBff(locals).session.create.POST({ request, params });
```

```ts
// src/routes/aoh/msr/api/session/[sessionId]/terminate/+server.ts
export const POST: RequestHandler = ({ request, params, locals }) =>
  msrBff(locals).session.terminate.POST({ request, params });
```

```ts
// src/routes/aoh/msr/api/admin/sessions/+server.ts
export const GET: RequestHandler = ({ request, params, locals }) =>
  msrBff(locals).admin.sessions.GET({ request, params });
```

```ts
// src/routes/aoh/msr/api/admin/sessions/terminate/+server.ts
export const POST: RequestHandler = ({ request, params, locals }) =>
  msrBff(locals).admin.terminate.POST({ request, params });
```

```ts
// src/routes/aoh/msr/api/admin/settings/+server.ts
export const GET: RequestHandler = ({ request, params, locals }) =>
  msrBff(locals).admin.settings.GET({ request, params });
export const PUT: RequestHandler = ({ request, params, locals }) =>
  msrBff(locals).admin.settings.PUT({ request, params });
```

```ts
// src/routes/aoh/msr/api/replay/state/+server.ts
export const GET: RequestHandler = ({ request, params, locals }) =>
  msrBff(locals).replay.state.GET({ request, params });
```

```ts
// src/routes/aoh/msr/api/replay/events/+server.ts
export const GET: RequestHandler = ({ request, params, locals }) =>
  msrBff(locals).replay.events.GET({ request, params });
```

(The two `RequestHandler` imports are elided in the shorter snippets —
add `import type { RequestHandler } from "@sveltejs/kit";` and
`import { msrBff } from "$lib/server/msr-bff";` to each file.)

> **Route group is cosmetic — the URL is what matters.** These paths show
> no SvelteKit route group, but the group prefix (e.g. `(private)/`) never
> appears in the URL. If your app guards authenticated routes with a route
> group, mount them under it — e.g.
> `src/routes/(private)/aoh/msr/api/…` — so they sit behind the auth
> guard; the resolved URLs are still `/aoh/msr/api/…`, which is what
> `msr_bff_base` (default `/aoh/msr/api`) must equal. Keep the URL equal to
> `msr_bff_base`; the group is free to choose.

## Stub mode (develop without a backend)

When `MSR_URL` is unset the adapter returns a stub set shaped exactly like
`MsrBffHandlers`, so the same route files run unchanged. This lets the
whole frontend — lobby, date picker, playback, scrubbing, admin terminate
— run with **no `msr-service` and no CDC pipeline**.

**`references/stub.md` has a complete, copy-pasteable stub plus the exact
wire shapes you must emit** — start there. The non-obvious trap:
`MsrBffHandlers` types only the handler *signatures*, not the response
*body*, so a stub with the wrong field names (`data` vs `entity_state`,
`timestamp` vs `event_timestamp`, or `op` `"r"` vs `"u"`) **compiles and
lints clean** but the worker can't parse it — playback "runs" yet
`replayStore.entities` stays empty. Match the wire table in `stub.md`
exactly, then swap the table name + entity generator for your own domain
data.

> **Why a stub and not just `msr-service`?** The CDC ingestion pipeline
> (Kafka/Redpanda + Debezium) that fills `msr.cdc_event` is heavyweight and
> a deploy concern. Stub mode gives a backend-free dev + e2e loop; flip
> `MSR_URL` to point at a real service when you need real data. The switch
> is one env var — the page never learns which tier it's on.

## Direct-to-service (skip the BFF)

If you don't want a BFF, the SvelteKit server `load`/actions can call
`@mssfoobar/msr-client` directly — e.g. the admin page can load sessions
without a route:

```ts
// +page.server.ts
import { createMsrClient } from "@mssfoobar/msr-client";
import { env } from "$env/dynamic/private";

export const load = async ({ locals }) => {
  const client = createMsrClient({
    baseUrl: env.MSR_URL,
    // Read the token off YOUR auth module, narrowed on `success`, fail closed.
    // Field name depends on your base: `access_token` (snake_case) or `accessToken` (camelCase).
    getToken: () => (locals.authResult.success ? (locals.authResult.access_token ?? "") : ""),
  });
  const { data: sessions } = await client.admin.listSessions({ page: 1, size: 50 });
  return { sessions };
};
```

But the **replay store + worker still need the BFF routes** (they fetch
relative `/aoh/msr/api/*` paths and cannot run server-only code). So a
replay surface always needs at least `replay.state`, `replay.events`, and
`session.create` mounted. Direct-to-service only saves you the admin /
settings routes.
