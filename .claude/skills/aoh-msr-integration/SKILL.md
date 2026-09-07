---
name: aoh-msr-integration
description: >
  Integrate the AOH MSR module (`@mssfoobar/msr-web-sdk`) into a SvelteKit
  consumer app — mount the replay surface, drive the Web-Worker replay
  pipeline, and render backend entities from the reactive replay store.
  MSR ships no renderer (unlike GIS): its job is to deliver recorded
  entity state into `replayStore.entities` frame-by-frame and the host
  draws it. Also wires the BFF routes to `msr-service` and supplies theme
  + BFF base via `<MsrProvider>`. Use whenever a user wants to add replay
  / time-travel / session playback, replay recorded CDC or operational
  data, show historical entity state over a time range, build a replay
  scrubber / timeline, consume `@mssfoobar/msr-web-sdk`, wire the MSR
  BFF, or stand up the MSR session-admin page. Trigger phrases: "add
  replay", "multi-session replay", "integrate MSR", "use msr-web-sdk",
  "replay recorded data", "time-travel the data", "MSR sessions",
  "replay store", "mount MsrProvider", "wire the MSR BFF".
license: Proprietary
metadata:
  owner: AOH Platform Team
  status: v1
---

# AOH MSR Integration

Wire the AOH **MSR module** (Multi-Session Replay) into a SvelteKit
consumer app. The module ships as the npm package
**`@mssfoobar/msr-web-sdk`** (published to GitHub Packages under the
`@mssfoobar` scope); the backend is the Go **`msr-service`** (records
database table changes via CDC into TimescaleDB and serves reconstructed
state + change events). This skill is the integration playbook — the
exact components, props, route setup, and server-side BFF wiring to stand
up a working, themed replay surface in your app.

> **Package naming — read this first.** The published SDK is
> **`@mssfoobar/msr-web-sdk`**. The old **`@mssfoobar/msr-web`** is the
> **retired modlet** (a source-copying installer that mutated host files);
> it is gone. There is no `@mssfoobar/msr-sdk` or `@mssfoobar/msr-app`. The
> backend is the Go service **`msr-service`**, not the legacy `msr-app`
> image. If you see those old names, they're stale.

## The one thing to understand first: MSR has no renderer

This is the single most important difference from GIS, and the thing that
trips people up. GIS ships a map engine that draws entities for you. **MSR
draws nothing of your data.** Its entire contract is:

> A runes store (`replayStore`) drives a Web Worker that polls the host's
> BFF, buffers CDC change events, and applies them frame-by-frame into a
> reactive map — **`replayStore.entities`**. Your app reads that map and
> renders it however it wants (a table, a list, feeding markers to a GIS
> map, a `<canvas>`…). As playback advances, the map mutates and your UI
> re-paints reactively.

The SDK's own components (`<MultiSessionReplay>`, `<ReplayController>`)
render only the **chrome** — the lobby card, the date picker, the
play/pause/scrub controls. They do **not** visualise your entities. If you
mount `<MultiSessionReplay>` and expect to "see the data replay", you'll
see a lobby and a controller over an empty surface. **You must render
`replayStore.entities` yourself.** See `references/replay-store.md`.

## What you're wiring (mental model)

```text
<MsrProvider dark_mode_store msr_bff_base>     ← app-root: theme store + BFF base path
  …your replay route…
    {#each replayStore.entities …}             ← YOUR visualisation (the payoff)
    <MultiSessionReplay lobby* playbackWindowMinutes/>  ← lobby + date picker + controller chrome
  …your admin route…
    <SessionListPage data onRefresh/>          ← session admin table
+ BFF routes: msrBffHandlers(cfg) → msr-service ← server-side proxy (token, OTEL trace)
```

Two halves to every integration:

1. **Client (`.svelte`)** — `<MsrProvider>` at the root; on the replay
   route, `<MultiSessionReplay>` for the chrome **plus your own rendering
   of `replayStore.entities`**.
2. **Server (`+server.ts` BFF)** — seven one-line route files built from
   `msrBffHandlers(cfg)` that proxy session / replay / admin calls to
   `msr-service` with a **server-side** bearer token. The browser never
   sees the token or the service URL.

## Prerequisites & install

The SDK is a Svelte 5 package. Peer requirements:

- **SvelteKit + Svelte 5** (runes). Components ship as compiled `.svelte`
  consumed via the `svelte` export condition.
- **Tailwind v4** — the SDK's look uses Tailwind utility classes; register
  its `@source` bridge (Step 2). Tailwind itself is bootstrapped by the
  host (e.g. via `@mssfoobar/ui/styles/app.css`).
- Peer packages: `@mssfoobar/ui`, `@mssfoobar/logger`, `svelte`.

Authenticate the scope to GitHub Packages (project-level `.npmrc`):

```ini
@mssfoobar:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${NODE_AUTH_TOKEN}
```

…with `NODE_AUTH_TOKEN` exported (a PAT / `gh auth token` with
`read:packages`). Then:

```bash
pnpm add @mssfoobar/msr-web-sdk @mssfoobar/ui @mssfoobar/logger svelte
```

> **`@mssfoobar/msr-client` + `@mssfoobar/msr-types` come transitively.**
> The SDK declares them as runtime **dependencies** (not peers), so you
> don't `pnpm add` them — importing `@mssfoobar/msr-web-sdk/server` pulls
> `@mssfoobar/msr-client` in automatically. (This is a real difference from
> `gis-web-sdk`, whose BFF is self-contained.) A transitive dep is **not
> resolvable by name** under pnpm-strict isolation, so
> `import … from "@mssfoobar/msr-types"` fails `svelte-check`/`tsc` with
> `Cannot find module '@mssfoobar/msr-types'` — import DTO types from the
> SDK's re-export subpath **`@mssfoobar/msr-web-sdk/types`** instead
> (`@mssfoobar/msr-client` is fine — the SDK depends on it directly).

> **No Cesium, no static-asset copy, no SSR-disable.** MSR has no map
> engine, so none of the GIS "black canvas" machinery applies — there is
> no `static/cesium/`, no `CESIUM_BASE_URL`, and the replay route can
> server-render normally (the store guards browser-only code behind a
> `typeof window` check and only spins up the worker in the browser).

## Integration steps

### Step 1. Bundle the SDK's Svelte source for SSR

`@mssfoobar/msr-web-sdk` ships compiled `.svelte`, so SSR must transform
it. Add the `@mssfoobar/` namespace to `ssr.noExternal` in
`vite.config.ts`:

```ts
// vite.config.ts
export default defineConfig({
  // …sveltekit(), tailwindcss()…
  ssr: {
    noExternal: [/^@mssfoobar\//],
  },
});
```

The load-bearing entry is `/^@mssfoobar\//`. Without it, SSR fails to
resolve the SDK's Svelte components. If you already run other AOH modules
(e.g. GIS), your `noExternal` list may carry extra entries
(`@lucide/svelte`, `@cesium/engine`, …) — MSR needs only the
`@mssfoobar/` regex added alongside them.

### Step 2. Register the SDK styles (one CSS-level `@import`)

Chain the SDK's `@source` bridge into your **single** Tailwind v4 CSS
entry — a CSS-level `@import`, not a JS import — so every package's
`@source` directives aggregate in one Tailwind pass:

```css
/* src/app.css */
@import "@mssfoobar/ui/styles/app.css";       /* owns @import 'tailwindcss' + @theme */
@import "@mssfoobar/msr-web-sdk/styles/app.css";
```

The first `@import` must be the one that owns `@import 'tailwindcss'` +
the `@theme` block (here `@mssfoobar/ui`). The MSR line is a thin
`@source` registration that puts the SDK's compiled output under
Tailwind's content scan — without it the SDK's panels render with no
backgrounds.

> **Don't re-introduce the retired modlet's iconify mutation.** The SDK
> uses `@lucide/svelte` components directly; it needs **no**
> `addIconSelectors(["lucide"])` in any Tailwind config. If you migrated
> from the legacy `@mssfoobar/msr-web` modlet, remove that line or the
> icons misbehave.

### Step 3. Mount `<MsrProvider>` (at the layout root)

`<MsrProvider>` hoists app-wide concerns off the per-component surface.
Its props (from `MsrProviderValue`):

| Prop | Type | Required | Purpose |
|---|---|---|---|
| `dark_mode_store` | `Writable<boolean>` | **yes** | Host-owned theme store; SDK panels repaint on change. |
| `msr_bff_base` | `string` | no | Base path of the host's MSR BFF routes. Default `"/aoh/msr/api"`. The store, API client, and worker derive every fetch path from this. |
| `msr_url` | `string` | no | Only if the host lets the SDK call `msr-service` directly instead of via the BFF. The SDK does **not** use it today — leave unset. |
| `auth_adapter` | `() => MsrAuthHeaders \| Promise<MsrAuthHeaders>` | no | Reserved for future SDK-internal fetches; not called today. |
| `feature_flags` | `Record<string, boolean>` | no | SDK-side toggles. |
| `log_level` | `"trace"\|"debug"\|"info"\|"warn"\|"error"\|"silent"` | no | MSR SDK log verbosity. Default `"warn"` (quiet); raise to `"debug"` in dev. |

The host owns the theme store; the SDK only *reacts* to it (no global
singleton, no SDK-internal `localStorage`). Mount it at your root
`+layout.svelte`:

```svelte
<script lang="ts">
  import { writable } from "svelte/store";
  import MsrProvider from "@mssfoobar/msr-web-sdk/provider";

  const darkModeStore = writable(false);  // wire to your real theme toggle
</script>

<MsrProvider dark_mode_store={darkModeStore}>
  {@render children()}
</MsrProvider>
```

> **No `msr_url`.** The SDK talks **only** to the host's BFF at
> `msr_bff_base`; it never calls `msr-service` directly from the browser.
> Leaving `msr_bff_base` at its default means your BFF must live at
> `/aoh/msr/api` (Step 4). If both MSR and GIS are present, nest both
> providers and pass the **same** `dark_mode_store` — one theme, every
> module reacts.

### Step 4. Wire the BFF (one adapter + seven one-line routes)

The browser-side store/worker fetch the host's own routes under
`/aoh/msr/api`; those routes proxy to `msr-service`. Build them from
`msrBffHandlers(cfg)` — **seven route files**, seven handler groups
(eight methods, since `admin/settings` carries both GET and PUT).
Centralise the stub-vs-real switch + token in one adapter, then mount the
routes as one-liners.

**The adapter** (`$lib/server/msr-bff.ts`) — switches on `MSR_URL` (set →
real service; unset → in-memory stub) and forwards the user's server-side
token, **fail-closed**:

```ts
import { env } from "$env/dynamic/private";
import { msrBffHandlers, type MsrBffHandlers } from "@mssfoobar/msr-web-sdk/server";
import { msrStubHandlers } from "./msr-stub";    // backend-free dev — references/stub.md

export function msrBff(locals: App.Locals): MsrBffHandlers {
  if (!env.MSR_URL) return msrStubHandlers;      // backend-free dev / e2e
  return msrBffHandlers({
    baseUrl: env.MSR_URL,
    getToken: () => bearerToken(locals),         // RAW token; the client prepends "Bearer "
  });
}
```

> **`bearerToken` reads the token off *your* auth module — there's no
> single shape.** Where the token lives, and its exact field name, depends
> on the auth base your app is built on: some bases expose it as
> `access_token` (**snake_case**) on `locals.authResult`, others as
> `accessToken` (camelCase). Read your own `App.Locals.authResult` type
> rather than copying a field name blind. The invariant: forward the real
> token; on no valid session send an empty token and let the upstream 401
> (fail closed). Full per-shape policy + the stub: `references/bff.md` and
> `references/stub.md`.

**The routes** — each is a one-line pass-through. Mount these seven files
under `src/routes/aoh/msr/api/`. If your app puts authenticated routes
behind a SvelteKit route group (e.g. `(private)/`), mount them under that
group instead — the group prefix never appears in the URL, so the
resolved URL is still `/aoh/msr/api/…`, which is what `msr_bff_base` must
equal:

| Route file | SDK handler |
|---|---|
| `session/+server.ts` (POST) | `msrBff(locals).session.create.POST` |
| `session/[sessionId]/terminate/+server.ts` (POST) | `msrBff(locals).session.terminate.POST` |
| `admin/sessions/+server.ts` (GET) | `msrBff(locals).admin.sessions.GET` |
| `admin/sessions/terminate/+server.ts` (POST) | `msrBff(locals).admin.terminate.POST` |
| `admin/settings/+server.ts` (GET + PUT) | `msrBff(locals).admin.settings.GET` / `.PUT` |
| `replay/state/+server.ts` (GET) | `msrBff(locals).replay.state.GET` |
| `replay/events/+server.ts` (GET) | `msrBff(locals).replay.events.GET` |

```ts
// e.g. src/routes/aoh/msr/api/session/+server.ts
import type { RequestHandler } from "@sveltejs/kit";
import { msrBff } from "$lib/server/msr-bff";

export const POST: RequestHandler = ({ request, params, locals }) =>
  msrBff(locals).session.create.POST({ request, params });
```

The env vars in play:

| Var | Side | Purpose |
|---|---|---|
| `MSR_URL` | server | BFF → `msr-service` base URL. **Unset → BFF stub mode** (no backend needed). |
| `IAM_URL` | server | *Optional.* Only if you gate an auth-disabled dev tier on the presence of an IdP URL (see `references/bff.md`). If your app has auth always on and reads the token from `locals.authResult`, you can ignore it. |

Authn/authz gating stays a host concern; `enrichSessions` lets you join
IAMS user info onto the admin list. The handlers preserve the aoh wire
envelope; trace context propagates automatically across the proxy via the
OTEL HTTP instrumentation's W3C `traceparent` header. **Full copy-pasteable
adapter + per-base token policy: `references/bff.md`. The complete stub +
the wire shapes it must emit: `references/stub.md`.**

### Step 5. Build the replay page (chrome + your rendering)

Two things on this route: `<MultiSessionReplay>` (chrome) and **your own
rendering of `replayStore.entities`** (the payoff). The layout is the
part people get wrong — get it from the recipe below.

```svelte
<script lang="ts">
  import MultiSessionReplay from "@mssfoobar/msr-web-sdk/multi-session-replay";
  import { replayStore } from "@mssfoobar/msr-web-sdk/store";

  // The store getters are $state-backed, so plain $derived reads stay reactive.
  const status = $derived(replayStore.status);
  const entities = $derived([...replayStore.entities.values()]);
</script>

<!--
  Layout recipe (copy it): a single-cell centring grid that bounds height
  and clips overflow. `isolate` scopes z-index so your backdrop can sit
  BEHIND the SDK's ReplayController (which has NO z-index of its own).
-->
<div class="relative isolate grid h-full min-h-0 w-full place-items-center overflow-hidden">

  <!-- YOUR visualisation — rendered BEFORE <MultiSessionReplay> and pushed
       behind it with -z-10 so the controller paints on top and stays usable.
       Use a near-opaque bg, NOT backdrop-blur (blur makes a stacking context
       that paints back OVER the controller). pointer-events-none = read-only. -->
  {#if entities.length > 0}
    <div class="pointer-events-none absolute inset-x-4 bottom-32 top-14 -z-10 overflow-auto rounded-lg border bg-card/95">
      <!-- table / list / map of `entities` — see references/replay-store.md -->
    </div>
  {/if}

  <MultiSessionReplay
    lobbyTitle="Multi-Session Replay"
    lobbyDescription="Replay recorded operational data from a point in time."
    lobbyButtonText="Start replay"
    playbackWindowMinutes={120}
  />
</div>
```

**Why the layout matters:** `<MultiSessionReplay>` renders the lobby /
init card / controller **in normal flow**, one in-flow child per status.
Without `grid place-items-center` (or equivalent centring) the lobby card
clips to the top-left; without a bounded, clipped height the controller
drifts. The ReplayController positions itself bottom-centre and is
draggable, with no z-index — so anything you want *under* it needs `-z-10`
within an `isolate` container. (A complete live-entity-table rendering is
in `references/replay-store.md`.)

> **Shortcut — `selfContained`.** If you *don't* need a custom backdrop behind
> the controller, pass `<MultiSessionReplay selfContained>`: the component owns
> the whole layout context itself (the centring grid + `relative isolate` +
> bounded/clipped height), so the host only needs a height-bounded box — no
> `grid place-items-center` wrapper of its own. Default is `false`
> (host-controlled, as above) so existing hosts are unchanged. For the
> backdrop-behind-controller pattern shown above (your visualisation under the
> chrome via `-z-10`), keep the host-controlled layout — the host must own the
> `isolate` context the backdrop shares.

**Key `<MultiSessionReplay>` props:**

| Prop | Type | Notes |
|---|---|---|
| `lobbyTitle` / `lobbyDescription` / `lobbyButtonText` | `string` | Required. The lobby card copy. |
| `playbackWindowMinutes` | `number` | Per-session replay **duration**, default `120`. Drives the derived end time, the DateTimePicker bounds, and the default buffers. |
| `startBufferMinutes` / `endBufferMinutes` | `number` | Safety buffers shrinking the selectable range. Default = `playbackWindowMinutes`. End buffer keeps users off "now" where CDC may lag; start buffer keeps them off the cleanup boundary. |
| `performanceOptions` | `MsrPerformanceOptions` | `pollWindowMs` (3000), `frameRate` (30), `maxBufferSize` (1e6), `minPollingInterval` (100). |
| `stateFilterConfig` / `playbackFilterConfig` | filter config | Per-table stale-data exclusion via a `getMinTimestamp` callback. Advanced; see `references/replay-store.md`. |
| `errorHandling` | `ErrorHandlingConfig` | `onError` hook + `defaultOptions` (toast/log/throw). Defaults to a Sonner toast handler. |
| `blurTargetElement` | `HTMLElement` | Optional element to blur while the lobby / error / terminated card is up — e.g. point it at your background entity visualisation so it dims behind the lobby. |
| `selfContained` | `boolean` | Default `false`. When `true`, the component supplies its own centring + `relative isolate` + bounded/clipped layout, so the host needs only a height-bounded box (no `grid place-items-center` wrapper). Best for chrome-only use; for a custom backdrop *behind* the controller, keep the host-controlled layout above. |

Most consumers never touch the store's control functions — `<MultiSessionReplay>`
drives them. Your job is to **read** `replayStore.entities` / `.status` /
`.playbackTime`. The full store contract (status state machine, entity
model, control surface, performance + filter tuning) is in
`references/replay-store.md`.

### Step 6. The session-admin page

`<SessionListPage>` is a ready-made admin table (search, terminate one /
many). It takes the session list as a **required `data` prop** — the host
loads it server-side and re-runs the load on `onRefresh`.

```svelte
<script lang="ts">
  import { invalidateAll } from "$app/navigation";
  import { SessionListPage } from "@mssfoobar/msr-web-sdk/pages/session-list";
  let { data } = $props();
</script>

{#if data.loadError}
  <!-- explicit "couldn't load" + retry — see below -->
{:else}
  <SessionListPage data={data.sessions} onRefresh={() => invalidateAll()} />
{/if}
```

Load it via your own BFF route (one code path for stub + real tiers):

```ts
// +page.server.ts
export const load = async ({ fetch }) => {
  try {
    const res = await fetch("/aoh/msr/api/admin/sessions");
    if (!res.ok) return { sessions: [], loadError: true };
    const body = await res.json();
    // A contract-conformant 200 always carries a `data` array. Anything else
    // (e.g. a gateway returning an HTML login page as 200) is a load failure,
    // NOT a genuine empty list.
    if (!Array.isArray(body.data)) return { sessions: [], loadError: true };
    return { sessions: body.data, loadError: false };
  } catch {
    // A connection-level failure (fetch threw) must ALSO land in loadError,
    // not bubble to SvelteKit's error page — same "outage ≠ empty" guarantee.
    return { sessions: [], loadError: true };
  }
};
```

> **Distinguish outage from "zero sessions".** A backend failure must not
> render as the SDK table's "No Active Sessions" empty state — to an
> operator that reads as a confident "zero active sessions" while the
> service is down. Set a `loadError` flag and show an explicit retry.

`fullName` / `userName` columns are optional IAMS enrichment — without it
the table falls back to the raw `user_id`. Populate them via the
`enrichSessions` hook on `msrBffHandlers` (or in your load). See
`references/bff.md`.

### Step 7. Navigation (optional)

```ts
import { msrNav } from "@mssfoobar/msr-web-sdk/nav";
// msrNav = { code: "MSR", header: { name, url }, sidebar: [{ name, url, icon }] }
const moduleNavs = [/* …, */ msrNav];   // feed to your sidebar chrome
```

`msrNav` is a **structured object**, not a flat list — if your sidebar
consumes a flat item array (e.g. a `NavItem[]`), map `msrNav.sidebar` into
your item type:

```ts
const items = msrNav.sidebar.map((e) => ({ name: e.name, url: e.url, icon: e.icon }));
```

`icon`s are `@lucide/svelte` component references (cast/adapt to your
sidebar's icon prop); `url`s default to `/aoh/msr` and `/aoh/msr/sessions`.
Override if you mounted MSR elsewhere.

### Step 8. Backend — stub first, then `msr-service` + CDC

You do **not** need a backend to develop the frontend: leave `MSR_URL`
unset and the BFF serves an in-memory stub. **`references/stub.md` ships a
complete, copyable stub + the exact wire shapes it must emit** — don't
hand-roll one blind; the response field names aren't type-checked, so a
wrong guess (`data` vs `entity_state`) silently renders nothing.

Bring up the real service when you need real data. In deployed
environments your platform provides `msr-service`; for local dev you run
it yourself (it needs a session database + a TimescaleDB CDC store). Once
it's up, point your app at it and confirm readiness — `/readyz` is
auth-free and pings both DBs, so a `200` means it's fully wired:

```bash
export MSR_URL=http://localhost:8082
curl -fsSL "$MSR_URL/readyz"     # auth-free readiness — 200 = both DBs reachable
```

> **`/readyz` green does not mean *authenticated* traffic works.**
> `msr-service` does a Keycloak userinfo round-trip on every request, so
> real authed calls need a reachable IAMS/Keycloak issuer
> (`IAMS_KEYCLOAK_URL`, e.g. `.../realms/aoh`). `/readyz` only checks the
> DBs, so it returns `200` even without a reachable issuer — don't read a
> green `/readyz` as "auth works". Pure frontend dev sidesteps this
> entirely via stub mode (`MSR_URL` unset).

> **Live replay needs the CDC pipeline.** `msr-service` only **reads**
> `msr.cdc_event` in TimescaleDB; it never consumes Kafka directly. That
> table is populated by a CDC ingestion pipeline (Kafka/Redpanda +
> Debezium source + JDBC sink) that is a deploy concern separate from the
> service itself. If replay "starts" but no entities ever appear, the
> event store is empty (`select count(*) from msr.cdc_event`) or your
> timestamp is outside the recorded range.

## Verify it works

1. `pnpm dev` with `MSR_URL` **unset** → visit `/aoh/msr`. You see the
   **lobby card** (centred). Click "Start replay" → date picker → pick a
   time → the controller appears and `replayStore.status` walks
   `selecting → initializing → ready`; press play → entities stream into
   your rendering and re-paint as `playbackTime` advances.
2. **Admin**: visit `/aoh/msr/sessions` → the active session from step 1
   is listed; terminate it → the table refreshes.
3. **Empty surface but a working lobby/controller?** That's expected if
   you didn't render `replayStore.entities` — MSR draws no data. Add your
   visualisation (Step 5).
4. **Real backend**: set `MSR_URL` (+ `IAM_URL` for auth), confirm
   `msr.cdc_event` has rows, pick a timestamp inside the recorded range.

## Common failure modes

| Symptom | Cause | Fix |
|---|---|---|
| Mounted `<MultiSessionReplay>` but **no data shows** — only a lobby/controller over an empty surface | MSR ships no renderer; nothing draws your entities | Render `replayStore.entities` yourself (Step 5; `references/replay-store.md`). This is by design, not a bug. |
| Lobby card **clips to the top-left** / controller drifts off-screen | Replay route container isn't a bounded, centring, clipped box | Use `relative isolate grid h-full min-h-0 w-full place-items-center overflow-hidden` (Step 5 recipe). |
| Your entity panel **paints over** the play/scrub controls (controls unclickable) | ReplayController has **no z-index**; a `backdrop-blur` backdrop made a stacking context above it | Put your backdrop BEHIND with `-z-10` inside an `isolate` container; use a near-opaque `bg-card/95`, **not** `backdrop-blur`. |
| SSR build fails resolving the SDK's `.svelte` components | `ssr.noExternal` missing the `@mssfoobar/` regex | Add `noExternal: [/^@mssfoobar\//]` (Step 1). |
| SDK panels render **unstyled** (no backgrounds) | The `@source` bridge isn't in the Tailwind pipeline | CSS-level `@import "@mssfoobar/msr-web-sdk/styles/app.css"` into the single CSS entry, after the `tailwindcss`-owning import (Step 2). |
| Controller **icons are blank** | Leftover `addIconSelectors(["lucide"])` / iconify expectation from the retired modlet | The SDK uses `@lucide/svelte` directly — remove the iconify selector; no iconify needed. |
| `401` from `msr-service` on BFF calls | BFF sent no / a dev token against real auth, **or read the wrong token field** | Forward the real token, **fail closed** (Step 4). The token field name (`access_token` **snake_case** vs `accessToken`) depends on your auth base — read your own `App.Locals.authResult` type (`references/bff.md`). |
| **Stub mode "runs" but `replayStore.entities` stays empty** / replay shows nothing on the stub | Stub response used wrong field names — `MsrBffHandlers` types the signature, not the body, so it compiled clean | Match the wire shapes exactly: `entity_state` (not `data`), `event_timestamp` (not `timestamp`), `op` `"r"` for state / `"u"` for events, plus `table_name` / `entity_id`. Use the copyable stub in `references/stub.md`. |
| Replay **starts but no entities ever appear** (real backend) | CDC event store empty (no pipeline) or timestamp outside recorded range | `select count(*), max(event_timestamp) from msr.cdc_event;`; populate the pipeline, or use stub mode (`MSR_URL` unset). |
| Admin page shows **"No Active Sessions"** during an outage | Load collapsed a backend failure into an empty list | Set a `loadError` flag when the response isn't a contract-conformant `{ data: [...] }`; show an explicit retry (Step 6). |
| Session rows show a **raw `user_id`** instead of names | No IAMS enrichment | Configure the `enrichSessions` hook on `msrBffHandlers` (or enrich in your load). |
| DateTimePicker shows **nothing selectable** / everything disabled | `startBufferMinutes + endBufferMinutes` exceed the available data window | Lower the buffers, or widen the data window (`earliest_valid_timestamp` further back / larger `max_playback_range`). The selectable range is `[startDate + startBuffer, endDate − endBuffer]`. |
| "Start replay" / lobby button **does nothing** | Status machine not advancing | Lobby dismiss → `selecting` → pick a date in the dialog. Confirm `replayStore.status` transitions (it's reactive). |
| Worker fails to load in the **packaged build** | A `?worker` import (Vite-only) crept in | The SDK already creates the worker via `createMsrWorker()` (`new URL(...)`, statically analysable). Never replace it with `?worker`. |

## Settings semantics (don't conflate the two windows)

- **`max_playback_range`** (admin settings) is in **DAYS** — how far back
  in time a replay may start. With `earliest_valid_timestamp` it bounds
  the DateTimePicker's outer range.
- **`playbackWindowMinutes`** (a `<MultiSessionReplay>` prop) is the
  **per-session duration in minutes** — how long a single replay runs from
  the chosen start. These are unrelated; mixing them up yields a picker
  that's wrong by orders of magnitude.

## References

- `references/components.md` — the full export catalog: every public
  component / subpath, what it does, and its props (`MsrProvider`,
  `MultiSessionReplay`, `ReplayController`, `SessionListPage`,
  `DateTimePicker`, `Combobox`, `Input`, `ConfirmDialog`, `TimelineSlider`,
  plus `replayStore`, `msrNav`, the API client, and the `/server` BFF
  factory).
- `references/bff.md` — the complete BFF wiring: the `msrBffHandlers`
  factory, all seven routes, the per-shape fail-closed token policy
  (`access_token` snake_case vs `accessToken` camelCase), the
  `enrichSessions` IAMS join, the wire envelope, and OTEL `traceparent`
  trace-context propagation.
- `references/stub.md` — backend-free dev: the **exact wire shapes** the
  worker parses (envelopes, `entity_state`/`event_timestamp`, `op` codes,
  `Session`, `AdminSettings`, the `403 SESSION_TERMINATED` signal) and a
  **complete, copyable in-memory stub** to adapt to your domain data.
- `references/replay-store.md` — the replay store + worker pipeline deep
  dive: **how to render entities** (the payoff), the entity model, the
  status state machine, the reactive read surface, the imperative control
  surface (for custom controllers), and performance + per-table filter
  tuning.
