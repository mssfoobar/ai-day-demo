# The same-origin BFF + server-side token flow

The wfe-web-sdk's `<WorkflowProvider>` can call the workflow-manager **directly from
the browser** (`wfe_url` + `getToken`). The recommended posture is the opposite:
keep the access token **server-side** and route every call through a same-origin
BFF proxy. The browser never holds the manager URL or a token, and the proxy
fails closed when auth is on (see "The auth gate" below).

## The proxy (copy + adapt)

Create `src/routes/<mount>/api/[...path]/+server.ts` (e.g. `<mount>` = `aoh/wfe`).
Adjust `WFE_URL` to your env var name and `TEST_BEARER` to taste; the rest is
portable:

```ts
import type { RequestHandler } from "./$types";
import { env } from "$env/dynamic/private";
import type { AuthResult } from "@mssfoobar/auth-sdk";

// Dev identity, used ONLY when auth is disabled (IAM_URL unset). Never sent when
// IAM_URL is set.
const TEST_BEARER = (env.TEST_BEARER ?? "Bearer dev-user").replace(/^Bearer /, "");

/** Resolve the Bearer to forward, or undefined when auth is on but the session is dead. */
function bearer(locals: App.Locals): string | undefined {
	if (env.IAM_URL) {
		// auth-sdk's hooks.server.ts populates locals.authResult (SDS-backed, server-side).
		const authResult = (locals as { authResult?: AuthResult }).authResult;
		return authResult?.success ? authResult.accessToken : undefined; // fail closed
	}
	return TEST_BEARER; // auth disabled — dev identity
}

const proxy: RequestHandler = async ({ params, request, url, locals }) => {
	const base = env.WFE_URL;
	if (!base) return new Response("WFE_URL is not configured", { status: 503 });

	const token = bearer(locals);
	if (env.IAM_URL && !token) return new Response("Not authenticated", { status: 401 });

	const target = `${base.replace(/\/$/, "")}/${params.path}${url.search}`;
	const headers: Record<string, string> = { Accept: "application/json" };
	if (token) headers.Authorization = `Bearer ${token}`;
	const contentType = request.headers.get("content-type");
	if (contentType) headers["Content-Type"] = contentType;

	const method = request.method;
	// Forward the raw request bytes — binary-safe for JSON and any multipart body
	// (arrayBuffer preserves a multipart boundary; the Content-Type copied above
	// carries it). Buffers the whole body, fine at this scale.
	const body = method === "GET" || method === "HEAD" ? undefined : await request.arrayBuffer();

	const upstream = await fetch(target, {
		method, headers,
		body: body && body.byteLength ? body : undefined,
	});
	// Stream the response through unchanged — binary-safe. `await upstream.text()`
	// UTF-8-mangles any binary body; and do NOT forward Content-Length (Node's fetch
	// decompresses gzip/br but leaves the compressed length, which truncates the stream).
	return new Response(upstream.body, {
		status: upstream.status,
		headers: { "Content-Type": upstream.headers.get("content-type") ?? "application/json" },
	});
};

export const GET = proxy;
export const POST = proxy;
export const PUT = proxy;
export const DELETE = proxy;
```

## How the path maps

The wfe-client builds `${baseUrl}/v1/<resource>`. With the client created as
`createWfeClient({ baseUrl: "/<mount>/api", getToken: () => "" })`, a list call hits:

```
browser:   GET /<mount>/api/v1/workflow_template
           │  (same-origin — no token in the browser)
           ▼
+server.ts catch-all `[...path]`  →  params.path = "v1/workflow_template"
           │  attaches Authorization: Bearer <server-held token>
           ▼
manager:   GET ${WFE_URL}/v1/workflow_template
```

`WFE_URL` is the manager **base** URL — **no `/v1`** (the client already adds it).
The catch-all `[...path]` route forwards GET/POST/PUT/DELETE verbatim, passing the
query string and request body through.

## The auth gate — fail closed

| Mode | Condition | Token forwarded |
|---|---|---|
| Real auth | `IAM_URL` set | The user's token from `locals.authResult` (auth-sdk, SDS-backed, server-side). **No valid session → no `Authorization` header → upstream 401.** Never a dev identity. |
| Auth disabled | `IAM_URL` unset | A deterministic dev bearer (`TEST_BEARER` env, default `Bearer dev-user`). |

Failing **closed** is the rule: when auth is on and the session is dead, send no
credentials and let the manager reject — never substitute a dev/test identity
against a real service (that's a fail-open identity swap). The proxy returns 401
itself in that case rather than making an unauthenticated upstream call.

`locals.authResult` is populated by the host app's `hooks.server.ts` (the auth-sdk
wiring `aoh-web-init` scaffolds). This skill assumes that's in place; it does not
set up auth.

The proxy above inlines the dev `TEST_BEARER` so it's self-contained; a host that
already centralises a dev identity (e.g. in a `$lib/auth-adapter`) can import that
instead — the behaviour is identical.

## Env vars

| Var | Side | Purpose |
|---|---|---|
| `WFE_URL` (or your `--wfe-url-env`) | server | workflow-manager base URL, no `/v1`. Unset → the BFF returns 503. |
| `IAM_URL` | server | Presence flips the auth gate from dev-bearer to fail-closed real-token. |
| `TEST_BEARER` | server | Optional dev identity used only when `IAM_URL` is unset. |

No `PUBLIC_*` WFE var exists — the browser never sees the manager URL or a token.

## Why not call the manager from the browser?

`<WorkflowProvider wfe_url=… getToken=…>` works, but it puts the bearer token in
the browser and exposes the manager URL/CORS surface. The BFF keeps the token in
the server session and presents a same-origin API, which is the AOH default for
authenticated module traffic. Use the browser-direct mode only for throwaway demos
with auth disabled.
