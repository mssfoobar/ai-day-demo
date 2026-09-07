---
name: aoh-unh-integration
description: >
  Integrate the AOH UNH module (`@mssfoobar/unh-web-sdk`) into a SvelteKit
  consumer app — mount the notification admin/authoring UI (channel settings,
  distribution lists with an IAMS member picker, notification templates with a
  built-in email editor), wire a transparent server-side BFF proxy to
  `unh-service`, and supply theme + BFF base via `<UnhProvider>`. Unlike GIS/MSR
  the SDK ships forms/editors/managers + the IAMS picker but NO list components —
  the host composes lists from `@mssfoobar/unh-client` + a `@mssfoobar/ui`
  DataTable. Use whenever a user wants to add notification management, send
  templated email/push/custom notifications, build channel-settings /
  distribution-list / notification-template pages, embed the UNH email editor,
  consume `@mssfoobar/unh-web-sdk`, or wire the UNH BFF. Trigger phrases: "add
  notifications", "integrate UNH", "use unh-web-sdk", "notification templates UI",
  "distribution lists", "send email/push notifications", "mount UnhProvider",
  "wire the UNH BFF".
license: Proprietary
metadata:
  owner: AOH Platform Team
  status: v1
---

# AOH UNH Integration

Wire the AOH **UNH module** (Unified Notification Hub) into a SvelteKit consumer
app. UNH ships as the npm package **`@mssfoobar/unh-web-sdk`** (published to
GitHub Packages under the `@mssfoobar` scope), backed by the **`unh-service`**
REST API. This skill is the integration playbook — the components, provider,
routes, and server-side BFF wiring to stand up a working, themed
notification-management surface in your app.

> **Package map.** The SDK **`@mssfoobar/unh-web-sdk`** depends on
> **`@mssfoobar/unh-client`** (typed transport client) + **`@mssfoobar/unh-types`**
> (DTOs), both pulled in transitively. The backend is **`unh-service`**. (A
> legacy standalone `unh-app` / `unh-web` predates this; if you encounter
> `unh-app`, `APP_PORT`, `AES_256_KEY`, or `{{api.distribution.user_ids}}` that
> is the old service with a different contract — this skill targets the current
> `unh-service`.)

## Two things to understand first

**1. You compose the CRUD; the SDK ships the hard parts.** UNH gives you the
pieces that carry real module logic — the **IAMS member picker, the
notification-template form + email editor, the custom-channel parameters
manager, the send dialog, and the provider**. It ships **no list/table
components and no simple channel/distribution create-edit forms**. Listing,
deletion, and the plain channel/distribution forms are **yours to compose**:
call `@mssfoobar/unh-client`'s `.list()` / `.create()` / `.update()` /
`.remove()`, validate with the SDK's **exported** helpers (`validateEmailChannel`,
`initialEmailValues`, …), and render with your own `@mssfoobar/ui` `DataTable`,
`Input`, `Sheet`, etc. So a "channels page" = *your* DataTable over
`client.emailChannels.list()` + *your* form (validators + `client.emailChannels`)
in a sheet/dialog.

**2. The BFF is a transparent 1:1 proxy, not a per-route adapter.**
`@mssfoobar/unh-client` speaks the full `unh-service` `/v1` REST contract, so
your server-side BFF is a **single catch-all** (`[...path]`) that forwards every
call through `createUnhProxy` — one route file, plus one small route for the
IAMS directory picker.

## What you're wiring (mental model)

```text
<UnhProvider dark_mode_store bff_base>       ← app root: theme store + BFF base path (default /aoh/unh/api)
<Toaster/>                                   ← you mount it ONCE (the SDK does NOT)
  settings page       YOUR DataTable(client.*Channels.list()) + YOUR channel forms (validators + client) + <CustomChannelParameters>
  distributions page  YOUR DataTable(client.distributionLists.list()) + YOUR name form (validateDistributionList) + <DistributionMembers loadDirectory>
  templates page      YOUR DataTable(client.templates.list()) + <NotificationTemplateForm> (built-in email editor) + <SendTemplateDialog>
BFF /aoh/unh/api/[...path] → createUnhProxy → unh-service   ← transparent proxy (server-side token, fail-closed)
BFF /aoh/unh/iams/[type]   → your IAMS/AAS directory        ← feeds the member picker
```

Every integration has two halves:

1. **Client (`.svelte`)** — `<UnhProvider>` + your `<Toaster>` at the app root;
   per-section pages that combine SDK forms/editors with **your own**
   `@mssfoobar/unh-client`-driven lists.
2. **Server (`+server.ts`)** — a transparent catch-all proxy to `unh-service`
   with a server-side bearer (the browser never sees the token or the service
   URL), plus an IAMS directory route for the picker.

## Prerequisites & install

`@mssfoobar/unh-web-sdk` is a Svelte 5 package. You need:

+ **SvelteKit + Svelte 5** (runes). Components ship compiled, consumed via the
  `svelte` export condition.
+ **Tailwind v4** — register the SDK's `@source` bridge (Step 2). Tailwind itself
  is bootstrapped by your app (e.g. via `@mssfoobar/ui/styles/app.css`).
+ Peer packages: `@mssfoobar/ui`, `@mssfoobar/logger`, `svelte`.

Authenticate the `@mssfoobar` scope to GitHub Packages (project-level `.npmrc`):

```ini
@mssfoobar:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${NODE_AUTH_TOKEN}
```

…with `NODE_AUTH_TOKEN` exported (a token with `read:packages`). Then:

```bash
pnpm add @mssfoobar/unh-web-sdk @mssfoobar/ui @mssfoobar/logger svelte
```

> **`@mssfoobar/unh-client` + `@mssfoobar/unh-types` arrive transitively** as
> runtime dependencies of the SDK — you don't add them explicitly. Importing
> `@mssfoobar/unh-client` (for your lists) or `@mssfoobar/unh-web-sdk/server`
> pulls them in. But a transitive dep is **not resolvable by name** under
> pnpm-strict isolation — `import … from "@mssfoobar/unh-types"` fails
> `svelte-check`/`tsc` with `Cannot find module '@mssfoobar/unh-types'`. Import
> DTO types from the SDK's re-export subpath **`@mssfoobar/unh-web-sdk/types`**
> instead (`@mssfoobar/unh-client` is fine — the SDK depends on it directly).

> **No map engine / static-asset copy / SSR-disable.** The only browser-only
> piece is the built-in email editor, which the SDK loads via a dynamic
> `import()` in `onMount` — so your template pages **server-render normally**;
> do NOT set `ssr = false`.

## Integration steps

### Step 1. Bundle the SDK's Svelte source for SSR

The SDK ships compiled `.svelte` + a vendored TipTap-v3 editor, so SSR must
transform them. In `vite.config.ts`:

```ts
export default defineConfig({
  // …sveltekit(), tailwindcss()…
  ssr: {
    noExternal: [/^@mssfoobar\//, /^@tiptap\//, "svelte-tiptap"],
  },
});
```

`/^@mssfoobar\//` is the load-bearing entry; the `@tiptap/*` + `svelte-tiptap`
entries are needed because the built-in email editor pulls them (without them
its SSR/bundle step fails).

### Step 2. Register the SDK styles (one CSS `@import`) — DO NOT SKIP

Chain the SDK's `@source` bridge into your **single** Tailwind v4 CSS entry — a
CSS-level `@import`, not a JS import:

```css
/* src/app.css */
@import "@mssfoobar/ui/styles/app.css";          /* owns @import 'tailwindcss' + @theme */
@import "@mssfoobar/unh-web-sdk/styles/app.css";
```

The first `@import` must own `@import 'tailwindcss'` + the `@theme` block (here
`@mssfoobar/ui`). The UNH line is a thin `@source` that puts the SDK's compiled
output under Tailwind's content scan.

> **This is the #1 silent failure.** Tailwind v4 excludes `node_modules` from
> auto-detection, so without this import the SDK's utility classes are **never
> generated** — and it fails *partially* and confusingly: most of the UI looks
> fine, but the editor toolbar's rarer utilities vanish (dropdown chevrons
> balloon to the icon default, the colour-swatch grid collapses to one column).
> If the UI is "mostly right but subtly broken," this import is missing.

### Step 3. Mount `<UnhProvider>` + your `<Toaster>` (at the layout root)

`<UnhProvider>` hoists app-wide concerns. Its props:

| Prop | Type | Required | Purpose |
|---|---|---|---|
| `client` | `UnhClient` | no | Inject a client directly (tests/SSR). Otherwise built from `bff_base`. |
| `bff_base` | `string` | no | Base path of your UNH BFF routes. Default `"/aoh/unh/api"`. |
| `dark_mode_store` | `Writable<boolean>` | no | Your theme store; the SDK repaints on change. |
| `feature_flags` | `Record<string, boolean>` | no | SDK-side toggles. |
| `log_level` | `"trace"\|…\|"silent"` | no | Default `"warn"`; raise to `"debug"` in dev. |

```svelte
<script lang="ts">
  import { writable } from "svelte/store";
  import UnhProvider from "@mssfoobar/unh-web-sdk/provider"; // DEFAULT export (component subpath)
  import { Toaster } from "@mssfoobar/ui/toast";
  const darkModeStore = writable(false); // wire to your real theme toggle
</script>

<UnhProvider dark_mode_store={darkModeStore}>
  {@render children()}
</UnhProvider>
<Toaster />
```

> **The SDK does NOT mount its own `Toaster`.** Its components emit toasts via
> `@mssfoobar/ui/toast`; **you must render `<Toaster/>` once** at the app root or
> success/error toasts silently no-op. Running other AOH providers (GIS/MSR)
> too? Nest them and share the **same** `dark_mode_store`.

### Step 4. Wire the BFF (transparent proxy + IAMS picker)

`@mssfoobar/unh-client` calls your own routes under `bff_base` (`/aoh/unh/api`),
which proxy to `unh-service`. Because the client speaks the full `/v1` contract,
the proxy is a **single catch-all**, not per-route handlers.

Build it with `createUnhProxy` from `@mssfoobar/unh-web-sdk/server` and forward
the signed-in user's token **fail-closed**:

```ts
// src/lib/server/unh-proxy.ts
import { env } from "$env/dynamic/private";
import { createUnhProxy } from "@mssfoobar/unh-web-sdk/server";

const proxy = createUnhProxy({
  baseUrl: env.UNH_URL ?? "",                        // your unh-service base URL ($env is string|undefined)
  getToken: (event) => bearerToken(event.locals),   // RAW token; the proxy prepends "Bearer"
});
// createUnhProxy returns { GET, POST, PUT, PATCH, DELETE } — all bound to one
// handler that dispatches on event.request.method internally, so any entry
// handles any method. It is NOT itself callable; call e.g. `.GET(event)`.
export const unhBff = (event) => proxy.GET(event);
```

The catch-all route forwards every method:

```ts
// src/routes/aoh/unh/api/[...path]/+server.ts
import type { RequestHandler } from "@sveltejs/kit";
import { unhBff } from "$lib/server/unh-proxy";
export const GET: RequestHandler = (e) => unhBff(e);
export const POST: RequestHandler = (e) => unhBff(e);
export const PUT: RequestHandler = (e) => unhBff(e);
export const PATCH: RequestHandler = (e) => unhBff(e);
export const DELETE: RequestHandler = (e) => unhBff(e);
```

`bearerToken` reads the access token off **your** auth module — there's no single
shape (e.g. `event.locals.accessToken`, or `event.locals.authResult?.access_token`
on a snake-case base). The invariant: forward the real token; on no session
forward empty and let `unh-service` return 401 (**fail closed** — never
substitute a dev identity against a real backend). The `getToken` callback
receives the request event (`{ locals?: unknown }`). Full copy-pasteable proxy +
token policy: `references/bff.md`.

The **IAMS picker route** feeds `<DistributionMembers>` by proxying to your IAMS
directory (AAS) with the user's bearer:

```ts
// src/routes/aoh/unh/iams/[type]/+server.ts  →  returns the directory for the picker
```

Directory search in AAS is **exact-match only**, so this route returns the
directory and the SDK filters it client-side (that's the `loadDirectory` prop on
`<DistributionMembers>`). If you don't wire it, the member editor degrades to
free-text member IDs.

| Env var | Side | Purpose |
|---|---|---|
| `UNH_URL` | server | Your BFF → `unh-service` base URL. |
| `IAMS_AAS_URL` | server | IAMS/AAS base for the member-picker directory. Unset → free-text member IDs. |

### Step 5. Build the section pages (SDK forms + YOUR lists)

Each page = **your** `@mssfoobar/ui` `DataTable` over `@mssfoobar/unh-client`
`.list()` results + the SDK's form in a `Sheet`/`Dialog` for create/edit. Get
the client from the provider:

```svelte
<script lang="ts">
  import { GetUnhClient, initialEmailValues, validateEmailChannel, hasErrors } from "@mssfoobar/unh-web-sdk";
  const client = GetUnhClient();
  let channels = $state([]);
  $effect(() => { client.emailChannels.list({ size: 200 }).then((r) => (channels = r.data)); });
  // render `channels` in a ui DataTable; delete via client.emailChannels.remove(id), then re-list.
  // create/edit = YOUR form in a Sheet, backed by the exported validators + client:
  let form = $state(initialEmailValues());        // exported seed (secrets blank)
  async function save() {
    const errors = validateEmailChannel(form);     // exported validator — same rules as the service
    if (hasErrors(errors)) return;                  // show `errors` beside the fields
    const body = { ...form, port: Number(form.port) };  // create wants port: number (the seed is number|string)
    await client.emailChannels.create(body);        // edit: .update(id, { ...body, occ_lock: channel.occ_lock })
  }
</script>
```

The client exposes one resource accessor per resource — `client.emailChannels`,
`client.pushChannels`, `client.customChannels` (+ `.params(channelId)`),
`client.distributionLists` (+ `.members(listId)`), and **`client.templates`** —
each with `.list()` / `.get(id)` / `.create()` / `.update(id, …)` / `.remove(id)`.
(The template accessor is **`templates`**, not `notificationTemplates`. Component
subpaths like `./notification-template-form` are **default** exports; `GetUnhClient`,
`unhNav`, and `createUnhProxy` are **named**.)

The three sections:

+ **settings** — tabs per channel kind: *your* channel forms (email fields incl.
  TLS toggle + sender name; push service-account-key upload; custom webhook
  endpoint), each backed by the exported validators + `client.{email,push,custom}Channels`,
  plus the SDK's `<CustomChannelParameters>` for the custom-channel param grid.
+ **distributions** (+ a `[id]` detail page) — *your* name form
  (`validateDistributionList` + `client.distributionLists`) +
  `<DistributionMembers loadDirectory={…}>` (the IAMS picker; To/Cc/Bcc delivery
  mode).
+ **notification-templates** (+ `add`, `[id]/edit`) — `<NotificationTemplateForm>`
  (with the built-in email-body editor) + `<SendTemplateDialog>`.

Channel secrets (email password, push `service_account_key`) are **write-only** —
edit forms never prefill them and require re-entry to change. Full export catalog
+ props: `references/components.md`.

### Step 6. The email body editor

`<NotificationTemplateForm>` defaults the email body to a **built-in rich-text
editor** trimmed to email-safe formatting; it round-trips the rendered `body`
HTML plus an opaque `editor_json` source. You can **replace** it via the
`bodyEditor` snippet prop (e.g. a drag-and-drop email builder) — the seam hands
you both `body` + `editorJson` via `onChange`. Full contract + override seam:
`references/editor.md`.

### Step 7. Navigation (optional)

```ts
import { unhNav } from "@mssfoobar/unh-web-sdk/nav";
const moduleNavs = [/* …, */ unhNav]; // feed to your sidebar chrome
```

`unhNav` is a structured object
(`{ code, header: { name, url }, sidebar: [{ name, url, icon }] }`); map
`unhNav.sidebar` into your sidebar's item type if it wants a flat array. URLs
default to `/aoh/unh/*`.

### Step 8. Point `UNH_URL` at a running `unh-service`

Your BFF needs a reachable `unh-service`. In deployed environments your platform
provides it; for local dev you run it yourself. It **requires an AES-256 key**
(channel secrets are encrypted at rest) and reaches IAMS for auth + recipient
resolution:

```bash
export ENCRYPT_AES_256_KEY=$(openssl rand -hex 32)   # 64-char hex, 32 bytes
# run unh-service (with a Postgres + your IAMS_* config), then point your app at it:
export UNH_URL=http://localhost:8083
curl -fsSL "$UNH_URL/readyz"                          # auth-free readiness
```

> `unh-service` validates a Keycloak bearer with `active_tenant` on every `/v1`
> call, then **authorizes it against UNH-local tenant roles** (AOH-8280). A valid
> token alone gets you 403.
>
> Two capabilities, configured per deployment — `AUTHZ_ADMIN_ROLES` (the
> configuration surface) and `AUTHZ_SENDER_ROLES` (the send surface), recommended
> names `unh_admin` / `unh_sender`. **Both fail closed**, so a service with no
> `authz` config answers 403 to everything.
>
> Authorization is **by path prefix**, and the prefix is the unit:
> `/v1/admin/*` needs administration, `/v1/notification/*` needs send — and
> **administration confers send**, so a console built on `unh_admin` alone can
> configure AND test-send with no second role. Every method on a resource shares
> its prefix's requirement.
>
> A **sender-only** caller can therefore reach `POST /v1/notification/send/...`
> and nothing else: it cannot list templates to discover an id, cannot read a
> distribution list, and cannot label a channel. That fits a service-triggered
> send holding the template id in its own config; it does not fit a human
> sender-only console. Build the console with the admin capability.
>
> A 403 names the missing capability and the config key that grants it, so read
> the message before reaching for these docs. Member display names and the member
> picker come from the host calling IAMS-AAS with the signed-in operator's token,
> via the `loadDirectory` prop the SDK still takes.
>
> **On `tenant-admin`:** it is no longer needed to SEND — recipient resolution uses
> unh-service's own IAMS identity, so a sender needs no IAMS privilege. But it is
> still needed to *display* IAMS names: AAS forwards the operator's token to
> Keycloak, so the directory reads want `realm-management:view-users`, which arrives
> only via `realm-admin` (which `tenant-admin` composites). Measured — a principal
> holding only `unh_admin` gets 403 on every directory read. So the member picker
> and member-name columns work for realm administrators and silently render nothing
> for anyone else. `/readyz` is auth-free.

## Verify it works

1. With `UNH_URL` pointed at a running `unh-service`, `pnpm dev` → visit your UNH
   routes and create an email channel via the settings form; it appears in your
   DataTable.
2. Open a template → the **email editor renders with a styled toolbar** (if the
   toolbar looks broken, Step 2's `@import` is missing).
3. Distribution members: with `IAMS_AAS_URL` unset you get free-text IDs; set it
   (and be signed in) → the picker lists users/roles/groups.
4. Send a template via `<SendTemplateDialog>` and confirm delivery.

## Common failure modes

| Symptom | Cause | Fix |
|---|---|---|
| UI "mostly right but subtly broken" — editor chevrons huge, colour grid one column | SDK utility classes never generated | CSS `@import "@mssfoobar/unh-web-sdk/styles/app.css"` after the `tailwindcss`-owning import (Step 2). |
| SSR build fails resolving the SDK / TipTap | `ssr.noExternal` missing the regexes | Add `[/^@mssfoobar\//, /^@tiptap\//, "svelte-tiptap"]` (Step 1). |
| Success/error **toasts never appear** | The SDK doesn't mount `Toaster` | Render `<Toaster/>` once at the app root (Step 3). |
| `GetUnhClient()` **throws** | Used with no `<UnhProvider>` ancestor | Mount `<UnhProvider>` at the layout root (Step 3) — fail-fast by design. |
| Pages **empty / no list** even with the SDK mounted | The SDK ships no list components | Compose the list yourself: `client.*.list()` + a `@mssfoobar/ui` DataTable (Step 5). |
| `401` from `unh-service` on BFF calls | BFF sent no/wrong token | Forward the real token fail-closed (Step 4). |
| `403` from `unh-service` on **every** call, including reads | `AUTHZ_ADMIN_ROLES` / `AUTHZ_SENDER_ROLES` unset on the service — both fail closed | Configure them, and create the named roles on the tenant (they are AAS tenant roles, not realm roles). |
| `403` on any `/v1/admin/*` read, including templates and lists | The whole `/admin` prefix needs the administration capability — reads included | Give the console operator the admin capability. Read the 403 message: it names the capability and the config key. |
| `403` on send while the configuration pages work | Should not happen — administration confers send. Suspect `AUTHZ_ADMIN_ROLES` not matching the caller's tenant role | Compare the role in the token's `active_tenant.roles` against the configured list. |
| Send returns **500** naming `iams.keycloak_client_id` | UNH's own IAMS service identity is unset or rejected | Set `IAMS_KEYCLOAK_CLIENT_ID`/`_SECRET`; its service account needs realm role `realm-tenant-admin` + `realm-management` `view-users` and `view-clients`. |
| Send returns **503** | IAMS unreachable — retryable, unlike the 500 above | Check AAS/Keycloak reachability; an external-email-only list still sends. |
| Push channel always `skipped` though members resolve | `fcmToken` (and `phone`) are **optional** user-profile attributes; absent → no endpoints | Declare them in the realm user profile and populate them. The service logs `notification.endpoint_gap` with both counts. |
| Member picker **shows no users** after sign-in | Either AAS directory search is exact-match-only, or the operator lacks `realm-admin` — the loader fails soft to `[]` for both | Use the load-then-filter `loadDirectory` shape (Step 4); and check the operator's `realm-management` roles before assuming it is the search. |
| Monorepo Docker build can't resolve `@mssfoobar/unh-web-sdk/*` subpaths | The build didn't include + build the SDK before install | See the deployment note in `references/bff.md`. |

## Template binding (`{{key}}`)

`<SendTemplateDialog>`'s `data` fills the `{{key}}` placeholders in the template.
Three tokens are auto-provided from the resolved distribution list —
`{{distribution.email}}`, `{{distribution.phone}}`, `{{distribution.fcm_token}}` —
each a **comma-joined union over the whole list, rendered once** (not
per-recipient). Any other `{{key}}` comes from the caller-supplied `data`.

## References

+ `references/components.md` — the full export catalog: every public subpath +
  props (`UnhProvider` / `GetUnhClient`, the channel/distribution/template forms,
  `CustomChannelParameters`, `DistributionMembers` + the `loadDirectory` picker,
  `NotificationTemplateForm`, `SendTemplateDialog`, `unhNav`), and the
  host-composes-lists pattern with `@mssfoobar/unh-client`.
+ `references/bff.md` — the transparent `createUnhProxy` catch-all, the IAMS
  directory route, the fail-closed token policy, `UNH_URL` / `IAMS_AAS_URL`, and
  the deployment note for monorepo Docker builds.
+ `references/editor.md` — the built-in email-body editor: the email-safe command
  set, the `editor_json` / `body` round-trip, and the `bodyEditor` override seam
  (swap in your own editor).
