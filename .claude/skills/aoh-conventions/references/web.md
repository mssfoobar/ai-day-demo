# AOH Web App — Developer Conventions

Patterns for working inside an AOH web-base SvelteKit app — the kind scaffolded
by `aoh-web-init`. Read this when adding a feature, page, API route, or
component in such an app. If the project doesn't have `src/lib/aoh/`,
`gateway.config.ts`, and `(private)`/`(public)` route groups, this guide does
not apply — defer to ordinary SvelteKit conventions.

For conventions that aren't web-specific:

- **HTTP wire contract (response envelope shape, pagination params, web page
  routing namespaces)** — see `api.md`. The frontend consumes this contract
  via the gateway proxy.

## Folder structure: `[project]/[module]`

Code is organised by project then module. The project is `aoh`; modules sit under it:

```
src/lib/aoh/                # project root
├── core/                   # shared infrastructure (auth, theme, layout, stores)
│   ├── provider/           # context providers (auth, theme)
│   ├── stores/             # shared stores (breadcrumb, etc.)
│   ├── components/layout/  # Sidebar, Headerbar, nav.ts (NOT primitives — see below)
│   ├── logger/Logger.ts
│   ├── browser_broadcaster/
│   └── constants.ts
├── iams/                   # identity & access management module
└── {module-name}/          # each feature module follows the same pattern
    ├── components/         # module-specific components
    ├── stores/             # module-specific stores
    └── utils/              # module-specific utilities
```

Routes mirror this structure under `src/routes/(private)/aoh/{module-name}/`.
Every module's *code* lives in `src/lib/aoh/{module}/`; its *pages* live in
`src/routes/(private)/aoh/{module}/`. Don't split this — agents tend to put pages
under `src/lib/` or stores under `src/routes/`; both are wrong.

## UI primitives come exclusively from `@mssfoobar/ui`

The design-system contract — *why* primitives come from `@mssfoobar/ui`, page
archetypes, token philosophy, when to compose vs. hand-roll, the full primitive
inventory — lives in the `aoh-design` skill. Engage it for design intent. This
section covers the **code-level mechanics** of consuming the package from a
SvelteKit app.

```ts
import { Button } from "@mssfoobar/ui/button";
import { Dialog, DialogContent } from "@mssfoobar/ui/dialog";
import { cn } from "@mssfoobar/ui/utils";
```

`@mssfoobar/ui`'s `styles/app.css` is already `@import`-ed at the top of the
app's `src/app.css` and owns every theme token — don't redeclare those tokens
in the app.

### Discovering what's available

Before importing, check what the installed version of `@mssfoobar/ui` actually
ships — the primitive set evolves and a hand-maintained inventory in this doc
would go stale. The package's `exports` field is the authoritative list:

```bash
cat node_modules/@mssfoobar/ui/package.json
# look at the "exports" field — every subpath you can import is listed there
```

Don't guess subpaths (`@mssfoobar/ui/Input` vs `@mssfoobar/ui/input` vs
`@mssfoobar/ui/form/input`) — read the exports field and use what's there.

### Theme tokens

When you do need to add classes around or inside a primitive (layout, spacing,
emphasis), use the package's semantic Tailwind classes — never raw colour
utilities like `bg-blue-500`, `text-red-500`, `border-gray-300`. Common tokens:

| Use case | Class |
|---|---|
| Page / surface background | `bg-background`, `text-foreground` |
| Muted surface (cards, sections) | `bg-muted`, `text-muted-foreground` |
| Borders, dividers, input outlines | `border-border` |
| Primary action / accent | `bg-primary`, `text-primary-foreground` |
| Secondary action | `bg-secondary`, `text-secondary-foreground` |
| Destructive (errors, delete) | `bg-destructive`, `text-destructive`, `text-destructive-foreground` |
| Focus rings | `ring-ring`, `ring-offset-background` |

The full list lives in `@mssfoobar/ui/styles/app.css` (already imported by the
app's `src/app.css`). When a design needs a colour that isn't represented by a
semantic token, that's a signal the design has drifted from the system —
escalate, don't reach for `bg-blue-500`.

**Do not** create `src/lib/aoh/core/components/ui/`. There is no app-local
primitive layer. If a *primitive* you need isn't in `@mssfoobar/ui` yet, request
it from the platform team that owns the package — don't work around it locally
by hand-rolling Tailwind. While the upstream addition is in flight, prefer
composing from primitives that do exist (e.g., a Combobox composed from
`Popover` + `Command` + `Input`) over building from scratch.

This rule is about *primitives* (atoms — Button, Input, Select, Dialog, …), not
about app-specific components. Building your own composite components for the
features this app actually needs — a `ReportForm`, a `VehicleCard`, a
`SubmissionWizard` — is exactly the expected pattern. Compose them from
`@mssfoobar/ui` primitives, place them under
`src/lib/aoh/{module}/components/` (or `core/components/layout/` for
infrastructure-level composites like Sidebar / Headerbar / breadcrumb), and
import them via `$lib/...`. They are app components built **from** primitives —
they are not themselves primitives, and they do not live under a `ui/`
directory.

### `@mssfoobar/ui` API quirks to know up front

Some primitives diverge from common shadcn-svelte naming in ways that don't
fail the build — they just silently render wrong or look bare. Patterns worth
internalising before composing forms:

- **`Badge` splits `variant` and `color`.** Unlike vanilla shadcn's single
  `variant` prop, AOH's Badge has two: `variant` for shape (`solid`, `soft`,
  `outline`, `surface`) and `color` for semantic palette (`default`, `info`,
  `success`, `warning`, `destructive`). Writing `<Badge variant="destructive">`
  silently produces a destructively-shaped `solid` badge in the default colour.
  Correct: `<Badge variant="soft" color="destructive">`.
- **`Select` doesn't export `SelectValue`.** The package re-exports `Root`,
  `Trigger`, `Content`, `Item`, `Group`, etc. — but no `SelectValue`. Render
  the displayed value as children of `SelectTrigger` (typically a `$derived`
  string keyed off the current `value`), don't reach for the shadcn pattern.
- **The `Toaster` host is NOT auto-mounted.** `import { toast } from '@mssfoobar/ui/toast'`
  is a no-op renderer unless `<Toaster />` (also from `@mssfoobar/ui/toast`)
  is mounted somewhere in the tree — typically once in `(private)/+layout.svelte`.
  The scaffold ships this mount; if you stripped it out, put it back.
- **Lucide icons import from `@lucide/svelte/icons/<name>`**, not
  `lucide-svelte`. The AOH stack standardises on the scoped `@lucide/svelte`
  package; `lucide-svelte` (the unscoped one) is NOT installed by the
  scaffold. Nav icons (in `nav.ts`):
  `import Siren from '@lucide/svelte/icons/siren';`

## Svelte 5 runes — never legacy reactive syntax

All components use Svelte 5 runes. If you see code that uses the legacy syntax,
rewrite the file you're editing — don't preserve it for "consistency":

| Legacy (don't use)         | Runes (use)         |
| -------------------------- | ------------------- |
| `export let foo`           | `let { foo } = $props()`  |
| `let count = 0` + `$:`     | `let count = $state(0)`   |
| `$: doubled = count * 2`   | `let doubled = $derived(count * 2)` |
| `$: { ... side-effect }`   | `$effect(() => { ... })`  |

`<svelte:options runes={true} />` is set at the route or component level when
needed, but most files in this project already opt in.

### `bind:` + type assertions = silently read-only

Don't put a TypeScript type assertion on the right-hand side of a `bind:`:

```svelte
<!-- BROKEN: binding becomes one-way (read-only). The Select trigger opens,
     the user picks a value, but `category` never updates. svelte-check
     does NOT flag this. -->
<Select bind:value={category as string}>...</Select>

<!-- Correct: keep the state typed as the widest type the binding accepts
     (`string`), and narrow at the call site instead. -->
<script>
  let category = $state('');
</script>
<Select bind:value={category}>...</Select>
```

Svelte's `bind:` requires a writable expression (an l-value) on the right.
A type assertion (`as T`) produces an r-value, so Svelte silently downgrades
to a read-only binding — the property feeds INTO the component but writes
never flow back. The component renders, the user interacts, nothing changes
in your state. This usually surfaces as "the Select opens but submitting
fails validation on a required field" or "the form is permanently disabled
because the submit-button gating sees the initial empty value."

The same caveat applies to bracket access (`bind:value={obj['x']}`) and any
other non-l-value expression. Stick to plain identifiers or property paths.

## Sidebar navigation — one hardcoded list

Sidebar and header navigation come from a single hardcoded list in
`src/lib/aoh/core/components/layout/nav.ts` — there is **no** modlet
auto-discovery. When you add a feature page, append a `NavItem` row:

```ts
// src/lib/aoh/core/components/layout/nav.ts
import SomeIcon from "@lucide/svelte/icons/siren"; // optional, per-icon subpath

export const navItems: NavItem[] = [
    { name: "Getting started", url: "/getting-started" },
    {
        name: "Dashboard",
        url: "/aoh/{module-name}/dashboard",
        group: "Module Name",        // sidebar group label; entries sharing a group render together
        icon: SomeIcon,              // optional
        roles: ["operations-team"]   // optional active-tenant role gate; omit = visible to everyone
    }
];
```

Pair each entry with its page under `src/routes/(private)/aoh/{module-name}/+page.svelte`.
`Sidebar.svelte` role-filters and groups the list; `Headerbar.svelte` derives the
breadcrumb from it. The auth wrapper and layout chrome (sidebar, header, logout) are
inherited from `(private)/+layout.svelte` — don't re-wrap pages in AuthProvider or
add your own sidebar/header.

## Route groups: `(private)` vs `(public)`

- `(private)/` — authenticated. Wrapped in AuthProvider + Sidebar + Headerbar by
  `(private)/+layout.svelte`. New feature pages go here. Don't add auth checks
  to individual pages — `(private)/+layout.server.ts` handles redirect-to-login
  for the whole group.
- `(public)/` — no auth. Reserved for the auth flow (`/aoh/api/auth/*`), health
  endpoints (`/livez`, `/readyz`), and the debug page. New pages should not be
  added here unless they genuinely don't need auth.

## Auth flow — don't reinvent

The OIDC PKCE flow is implemented across five files that work as a single contract:

1. `(private)/+layout.server.ts` — checks `locals.authResult`. If unauthenticated,
   stores intended path in `aoh_redirect_after_auth` cookie and redirects to
   `/aoh/api/auth/login`.
2. `(public)/aoh/api/auth/login/+server.ts` — generates PKCE challenge, redirects
   to Keycloak.
3. `(public)/aoh/api/auth/callback/+server.ts` — exchanges code for tokens,
   reads `aoh_redirect_after_auth`, redirects user back.
4. `(public)/aoh/api/auth/refresh/+server.ts` — periodic token refresh.
5. `AuthProvider` component (in `src/lib/aoh/core/provider/auth/`) — schedules
   refreshes client-side.

When adding a feature, you should never need to touch any of these. If a feature
needs the current user's claims, read them from `data` in a private route's
`+page.server.ts` (load function) or `+layout.server.ts` populates `locals.user`.

### Tokens live in SDS, not browser cookies

The auth flow above runs SDS-backed in every AOH web app. The post-callback
step stores the access + refresh tokens in `sds-server` (backed by Valkey)
and only the resulting session ID lands in a browser cookie. `SDS_URL` is
required env (`tcp://127.0.0.1:5333` for native `pnpm dev`,
`tcp://sds-server:5333` inside the compose network).

`auth.ts` has a cookie-only fallback path for cases where SDS is down and
you need to confirm Keycloak still works — but that path is for emergency
diagnostics only and `hooks.server.ts` logs an `ERROR` if `SDS_URL` is
unset. Don't deploy without SDS, don't introduce code paths that read
`access_token` cookies on the client side, and don't add UI that surfaces
the raw token to the browser. See
`aoh-knowledge/services/sds.md` → "Why SDS is mandatory" for the rationale
(token exfiltration surface, server-side refresh, replay-safe logout,
RTUS-SEH downstream consumers).

### Setting the landing route — `LOGIN_DESTINATION`

The scaffold ships `LOGIN_DESTINATION=/getting-started` everywhere it appears (`.env.template`, `.env.development`, `compose/<app>/compose.yml`, the runtime `Dockerfile`). This is the page Keycloak's callback redirects to when the user has no stored intended destination (e.g. clean login from `/aoh/api/auth/login`).

The repo deliberately does **not** ship a root `+page.svelte`, so visiting `/` directly is a 404. The intent is that every authenticated user enters through `LOGIN_DESTINATION`, never through `/`.

**When you add a feature that introduces a real landing page, update `LOGIN_DESTINATION` in all three places to match.** Example: an incident-reporting capability whose primary page is `/incidents/new`:

```
apps/<app>/.env.template          LOGIN_DESTINATION=/incidents/new
apps/<app>/.env.development       LOGIN_DESTINATION=/incidents/new
compose/<app>/compose.yml         LOGIN_DESTINATION: /incidents/new
```

Leave the `Dockerfile`'s ENV default (`/getting-started`) alone — it's the floor for an unconfigured container.

**Why three places:** they're consumed at different times — `.env.development` by `vite dev`, the compose env by `docker compose up`, and `.env.template` is the source-of-truth that documents what env vars exist.

## API gateway — adding a backend service

`(private)/aoh/gateway/[...path]/+server.ts` proxies authenticated requests to
backend modules. The proxy is configured in `gateway.config.ts` at the app root:

```ts
// gateway.config.ts
export const gatewayConfig: GatewayConfig = {
    modules: {
        mymodule: {
            host: process.env.MY_MODULE_URL ?? "http://localhost:8081",
            basePath: "/api"
        }
    }
};
```

To call it from the frontend: `fetch('/aoh/gateway/mymodule/some/endpoint')`.
The gateway forwards to `${host}${basePath}/some/endpoint` and attaches the
user's access token. Don't write your own auth-aware fetch wrapper — use the
gateway.

> **Call the gateway from the browser only — never from a server `load()`
> (`+page.server.ts` / `+layout.server.ts`).** The proxy reads the bearer from
> `locals.authResult.access_token`, which `hooks.server.ts` populates from the
> `web_auth_session_id` cookie (via SDS) — and only the real browser request
> carries that. A server-side `event.fetch('/aoh/gateway/...')` spawns a
> sub-request whose `locals.authResult` has no token, so the gateway returns
> **401 before it forwards**. For data you need during SSR, skip the proxy and
> call the backend **directly** from `load()` using the token you already hold
> server-side (`locals.authResult.access_token`) plus an `Authorization: Bearer`
> header — routing a server `load` through the browser-oriented gateway is both
> redundant and broken.

### Typing the response

The wire envelope is declared as the ambient global `HTTPResponseBody<T>` in
`src/app.d.ts` — **don't redeclare it per feature**. Narrow `T` to your row
type at the call site:

```ts
// Single record (GET /v1/reports/{id})
const body = (await res.json()) as HTTPResponseBody<Report>;
const report = body.data; // Report | undefined

// Paginated list (GET /v1/reports?page=…&size=…&sort=…)
const body = (await res.json()) as HTTPResponseBody<Report[]>;
const rows = body.data ?? [];
const page = body.page; // { number, size, total_records, count, sort? }
```

`page.sort` is `string[]` (e.g. `["created_at,desc", "name,asc"]`) — see
`api.md` for the full pagination contract.

## Real-time subscriptions — use `@mssfoobar/sse-client`

When a page needs RTUS topic / map / user-map updates, subscribe through
`@mssfoobar/sse-client` (1.2.0+) — NOT a hand-rolled `EventSource` wrapper.
The SDK ships the four URL strategies (topic / user-value-map / json-map /
map), reconnect-with-backoff, refresh-before-reconnect via the SvelteKit
`/aoh/api/auth/refresh` endpoint, and heartbeat-timeout liveness detection.
Hand-rolling all of this is the cookie/EventSource auth bridge that bit
during the first real RTUS integration — don't redo it.

Canonical wrapper shape (Svelte 5 rune store around the SDK):

```ts
import { SSESubscribeClient } from '@mssfoobar/sse-client';

let status = $state<'connecting' | 'live' | 'reconnecting' | 'offline'>('connecting');
let lastEvent = $state<MyEvent | null>(null);
const client = new SSESubscribeClient<MyEvent>({
    tenantId,
    mapName: 'my.topic',     // re-purposed for topic strategy
    eventId: '',             // discriminator — empty string is fine
    domainURL: `http://rtus-seh.${root}`,
    heartbeatTimeout: 15_000,
    onConnected:        () => (status = 'live'),
    OnReconnecting:     () => (status = 'reconnecting'),
    onHeartbeatTimeout: () => (status = 'reconnecting'),
    onDisconnected:     () => (status = 'offline'),
    onAnyEvent: (name, data) => {
        if (name === 'keepalive') return;
        lastEvent = data;
    }
});
client.connect();
// onDestroy: client.disconnect();
```

The cookie that authorises the request is `web_auth_session_id` — see
"Auth flow → Tokens live in SDS" above. rtus-seh resolves it through SDS
to mint a JWT.

### Initial-fetch + SSE dedup

Pages that combine an initial `GET /list` with a live topic of the same
domain (e.g. an incident triage table) MUST deduplicate by `id` before
prepending an SSE event into the rendered list:

```ts
rows = [incoming, ...rows.filter((r) => r.id !== incoming.id)].slice(0, size);
```

Without this filter, an event that races the initial fetch produces two
rows with the same key, and Svelte 5's keyed `{#each}` throws
`each_key_duplicate` — the table stops rendering and looks "stuck on a
stale row" with no obvious console error to clue you in. The SSE
delivery is fast, the initial fetch is HTTP-streamed, and overlap is
expected — apply this dedup at every prepend site.

## Logging — use the AOH logger, never console.log

ESLint's `no-console: error` rule is enforced project-wide. Use the AOH logger:

```ts
import { log } from "$lib/aoh/core/logger/Logger";

log.info({ userId }, "user opened the dashboard");
log.error({ err }, "failed to load module");
```

Pino conventions: pass an **object** as the first arg (mergeable into the log
record), then the **message** as the second. The reverse order (message first,
object second) silently drops the object.

## Playwright pitfalls — what bites you once chrome adds persistent SSE

These two patterns work fine in a greenfield project but break the moment
the `(private)` layout starts holding a long-lived SSE (e.g. an IAN
notification bell, or a live-status indicator). Reach for the alternatives
proactively when the layout has any persistent stream.

### `waitForLoadState('networkidle')` never resolves

A persistent SSE subscription keeps the network permanently busy from
Playwright's point of view, so `page.waitForLoadState("networkidle")`
will hang until the test's own timeout. Symptoms: a test that previously
passed in <1s suddenly times out at exactly 30s on the `waitForLoadState`
line, with no obvious cause.

Use a visibility-anchored wait instead:

```ts
// ❌ hangs forever once layout mounts an SSE subscription
await page.waitForLoadState("networkidle");

// ✅ wait on the actual element you're about to interact with
await expect(page.getByRole("heading", { name: "Report incident" })).toBeVisible({
    timeout: 15_000,
});
```

This is a one-way door: once any feature adds a layout-level SSE (or
WebSocket, or polling timer), every test in the suite that used
`networkidle` needs updating. Fix proactively.

### Role-based selectors get ambiguous as chrome accretes

`page.getByRole("button", { name: "Category" })` works in isolation but
becomes ambiguous once the layout grows additional buttons (a
notification bell, a help icon, a profile menu). Worse, `bits-ui`'s
`Select` trigger has a known actionability hiccup under
`getByRole("button")` when the page hasn't fully settled — the click
silently no-ops and the listbox never opens.

For `bits-ui` Select triggers, prefer the data-slot locator (matches the
proven pattern in `triage-incident.spec.ts`):

```ts
// ❌ ambiguous and flaky under layout chrome
await page.getByRole("button", { name: "Category" }).click();

// ✅ specific to the Select trigger — AND retry the open. The first click on a
// bits-ui trigger often only *focuses* it without opening the listbox (a known
// actionability hiccup, worse on the first interaction after page load), so a
// single click + waitFor times out. Retry the open until the option is visible:
async function chooseOption(page, triggerId: string, optionName: string) {
    const trigger = page.locator(`#${triggerId}[data-slot="select-trigger"]`);
    const option = page.getByRole("option", { name: optionName, exact: true });
    await expect(async () => {
        await trigger.click();                              // toggles; retried if it no-ops
        await expect(option).toBeVisible({ timeout: 1500 });
    }).toPass({ timeout: 15_000 });
    await option.click();
}
await chooseOption(page, "category", "Fire");
```

For other ambiguity, scope the selector to a region or dialog:

```ts
const panel = page.getByRole("dialog", { name: /Notifications/ });
await expect(panel.getByText("New incident: ...")).toBeVisible();
//          ^^^^^^^^ scoped — won't collide with the toast region
```

### When the form is incidental to your test, skip the form

If the test you're writing is about something downstream (e.g. a
notification arriving), don't drive the bits-ui form — POST directly
through the gateway proxy with `page.evaluate`:

```ts
const res = await page.evaluate(async ([title]) => {
    const r = await fetch("/aoh/gateway/incident/incidents", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ title, description: "...", category: "fire", severity: "high" }),
    });
    return { status: r.status, body: await r.text() };
}, [`Notif arrival ${Date.now()}`]);
expect(res.status).toBe(201);
```

Form-interaction reliability is covered by the dedicated form spec
(`report-incident.spec.ts`); your test only needs the side effect.

## SvelteKit error/redirect helpers

Use `error()` and `redirect()` from `@sveltejs/kit` for control flow in load
functions and server endpoints. **Never** wrap a `redirect()` in a try/catch
that swallows it — `redirect()` *throws* the redirect and SvelteKit catches it
upstream. Catching it kills the redirect.

```ts
import { error, redirect } from "@sveltejs/kit";

export const load: PageServerLoad = async ({ locals }) => {
    if (!locals.user) throw redirect(303, "/aoh/api/auth/login");
    if (!locals.user.permissions.includes("foo")) throw error(403, "forbidden");
    return { /* ... */ };
};
```

## Imports

- `$lib/...` → `src/lib/...` (the SvelteKit default alias)
- `$root/...` → repo root (configured in `svelte.config.js`); use for things like
  `$root/gateway.config`
- `@mssfoobar/ui/...` → AOH UI primitives (the only source of visual primitives)

Don't use relative imports across module boundaries inside `aoh/` (e.g., from
`aoh/iams/foo.ts` reaching into `aoh/core/`, write `$lib/aoh/core/logger/Logger`,
not `../../core/logger/Logger`) — relative paths break when files move. Relative
imports within the same module are fine.
