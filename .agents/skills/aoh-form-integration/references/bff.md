# The shared `/aoh/gateway` BFF

Form has **no SDK-supplied proxy**. The clients target
`{gatewayURL}/{module}/{apiVersion}/{path}` with `gatewayURL` defaulting to
`/aoh/gateway`, so the host owns one **catch-all, module-multiplexing** route (`src/routes/aoh/gateway/[...path]/+server.ts` in your
app) that the form, theme, amm (and dash) modules all share.

## The route (copyable)

```ts
// src/routes/aoh/gateway/[...path]/+server.ts
import type { RequestHandler } from "./$types";
import { env } from "$env/dynamic/private";
import type { AuthResult } from "@mssfoobar/auth-sdk";
import { TEST_BEARER } from "$lib/auth-adapter";   // your dev-identity token

/** Module code → upstream base URL. */
function upstreamFor(module: string): string | undefined {
  switch (module) {
    case "form":
    case "theme": return env.FORM_URL;   // form-service serves both
    case "amm":   return env.AMM_URL;
    default:      return undefined;
  }
}

/** Bearer to forward, or undefined when auth is on but the session is dead. */
function bearer(locals: App.Locals): string | undefined {
  if (env.IAM_URL) {
    const authResult = (locals as { authResult?: AuthResult }).authResult;
    return authResult?.success ? authResult.accessToken : undefined;  // fail closed
  }
  return TEST_BEARER.replace(/^Bearer /, "");   // auth disabled — dev identity
}

const proxy: RequestHandler = async ({ params, request, url, locals }) => {
  const [module, ...rest] = (params.path ?? "").split("/");
  const base = upstreamFor(module);
  if (!base) return new Response(`No upstream for module "${module}"`, { status: 503 });

  const token = bearer(locals);
  if (env.IAM_URL && !token) return new Response("Not authenticated", { status: 401 });

  const target = `${base.replace(/\/$/, "")}/${rest.join("/")}${url.search}`;
  const headers: Record<string, string> = { Accept: "application/json" };
  if (token) headers.Authorization = `Bearer ${token}`;
  const contentType = request.headers.get("content-type");
  if (contentType) headers["Content-Type"] = contentType;

  const method = request.method;
  // Forward the raw request bytes — binary-safe for JSON *and* multipart/form-data
  // (AMM uploads). arrayBuffer preserves the multipart boundary bytes; the original
  // Content-Type (copied above) carries the `boundary=…`. Buffers the whole body,
  // which is fine at this scale.
  const body = method === "GET" || method === "HEAD" ? undefined : await request.arrayBuffer();
  const upstream = await fetch(target, {
    method, headers,
    body: body && body.byteLength ? body : undefined,
  });

  // Stream the response body through unchanged — binary-safe. `await upstream.text()`
  // UTF-8-mangles binary downloads (an AMM image/file: a JPEG's leading 0xFF 0xD8
  // becomes U+FFFD = 0xEF 0xBF 0xBD), corrupting form images + attachment previews.
  // Do NOT forward Content-Length: Node's fetch transparently decompresses gzip/br
  // but leaves the *compressed* length, which would under-declare the stream and
  // truncate it. And set `sandbox` + `nosniff` so a user-uploaded image/svg+xml
  // (a script-capable document) opened top-level can't run script in your origin —
  // it does not affect <img> embedding or fetch() consumers.
  return new Response(upstream.body, {
    status: upstream.status,
    headers: {
      "Content-Type": upstream.headers.get("content-type") ?? "application/json",
      "Content-Security-Policy": "sandbox",
      "X-Content-Type-Options": "nosniff",
    },
  });
};

export const GET = proxy; export const POST = proxy; export const PUT = proxy;
export const PATCH = proxy; export const DELETE = proxy;
```

If your host already has this **hand-rolled** route (GIS/dash ship it), Form
integration is just **adding the `form` / `theme` / `amm` cases** to `upstreamFor`
+ setting the env.

## On an `aoh-web-init` base — register modules, don't paste the route

The `aoh-web-init` scaffold ships a **config-driven** gateway instead: a generic
`(private)/aoh/gateway/[...path]/+server.ts` + a `gateway.config.ts` exposing
`gatewayConfig.modules` and a `resolveModule()` helper. **Do not paste the
hand-rolled route above** onto that base — you'd stand up a second, conflicting
gateway and import a non-existent `$lib/auth-adapter`. Just register the modules:

```ts
// gateway.config.ts → gatewayConfig.modules
export const gatewayConfig: GatewayConfig = {
  modules: {
    form:  { host: process.env.FORM_URL || "http://form-service:8080" },
    theme: { host: process.env.FORM_URL || "http://form-service:8080" },
    amm:   { host: process.env.AMM_URL  || "http://amm-service:8080" },
  },
};
```

That base's gateway forwards the bearer (reading `locals.authResult.access_token`,
**snake_case** — not the hand-rolled route's camelCase `accessToken`), fails closed,
and is binary-safe on both directions — so `multipart/form-data` AMM uploads and
image/file downloads already work, same as the hand-rolled route above.

> **No dev-identity fallback on the `aoh-web-init` gateway.** Unlike the hand-rolled
> route's `TEST_BEARER` path, that base's gateway **always** requires
> `locals.authResult.access_token` and 401s without a live session — there is no
> "auth off" local mode. So a backend-free form dev loop needs a stub/mocked auth or
> a running IAMS; the `TEST_BEARER` dev-identity story applies only to the hand-rolled
> route above.

## Env contract

| Var | Side | Purpose |
|---|---|---|
| `FORM_URL` | server | upstream base for the `form` **and** `theme` modules (`form-service`) |
| `AMM_URL` | server | upstream base for the `amm` module (attachments) |
| `IAM_URL` | server | when set, require a live session and forward its token (fail-closed 401); unset → dev identity (`TEST_BEARER`) |

`bearer()` reads the token off *your* auth module — the exact field name depends on
the auth base your app is built on. On a `@mssfoobar/auth-sdk` base it's
`locals.authResult.accessToken` (camelCase); on an `aoh-web-init` base the shape
differs (`locals.authResult.access_token`, snake_case). Invariant: forward the real
token; fail closed (no token → upstream 401) when auth is on.

## Serving form images — a second host route

SurveyJS form templates reference uploaded images by a **host-relative** path
(`/aoh/api/form/images/<ammAttachmentId>` — the SDK creator's `imageUpload.ts`
`IMAGE_SERVE_PATH`), never an absolute FORM/AMM URL, so a template renders on
whatever host serves it. Your host must proxy that path to its own AMM backend, or
every `<img>` in a rendered form 404s. This is a **separate route** from the gateway
(the renderer builds the URL itself; it does not go through `AmmClient`):

```ts
// src/routes/aoh/api/form/images/[attachmentId]/+server.ts
import { error } from "@sveltejs/kit";
import { env } from "$env/dynamic/private";
import { AmmClient } from "@mssfoobar/form-client/amm";
import type { RequestHandler } from "./$types";

export const GET: RequestHandler = async ({ params, locals, fetch }) => {
  // Same auth posture as the gateway: gated on IAM_URL (open in a backend-free
  // sandbox / Playwright, required when auth is on).
  if (env.IAM_URL && !locals.authResult?.success) error(401, "Not authenticated");

  const client = new AmmClient({ module: "amm", fetch });   // rides /aoh/gateway/amm
  const dl = await client.downloadFile({ attachment_id: params.attachmentId });
  if (!dl.ok) error(502, "Failed to load the image.");

  return new Response(dl.response.body, {
    headers: {
      "Content-Type": dl.response.headers.get("Content-Type") ?? "application/octet-stream",
      "Cache-Control": "private, max-age=3600",   // an attachment id is immutable bytes
      // A form image can be an uploaded image/svg+xml; sandbox + nosniff stop a
      // top-level open from executing embedded script in your origin (harmless to
      // <img> embedding — see the gateway route's rationale).
      "Content-Security-Policy": "sandbox",
      "X-Content-Type-Options": "nosniff",
    },
  });
};
```

Note it fetches through the same-origin `/aoh/gateway/amm` (the `AmmClient` above),
so the bytes ride the binary-safe gateway before this route re-serves them.

## Large uploads (optional)

The route buffers each request body with `arrayBuffer()` — simplest, and correct for
form payloads and normal attachments. If you must stream very large uploads without
buffering, forward `request.body` directly and set `duplex: "half"` on the `fetch`
(only when the body is a stream); keep the multipart `Content-Type` intact either
way so its `boundary=…` survives.

## Deployment (server build & runtime)

When you package the host as a container, ensure the server build:

+ **installs the three form packages so their subpath exports resolve** — a package
  manager that respects each package's `exports` map. You `pnpm add` the published
  packages (`@mssfoobar/form-web-sdk` ships its compiled `./renderer` `dist/`;
  `@mssfoobar/form-types` + `@mssfoobar/form-client` are consumed as source), and
  your app's own `vite build` SSR-bundles all three (Step 1 in SKILL.md — the
  `ssr.noExternal: [/^@mssfoobar\//]` entry). Skip the install and the bundler
  resolves an empty package root and the build fails to find the subpath exports.
+ **has `FORM_URL` + `AMM_URL` (and `IAM_URL` if auth is on) available to the running
  server** — these are read at request time by your `+server.ts` gateway route, so
  set them in the server's runtime environment (container env, `.env`, or your
  platform's secret store).
