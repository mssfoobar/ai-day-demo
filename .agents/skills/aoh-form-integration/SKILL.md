---
name: aoh-form-integration
description: >
  Integrate the AOH Form module into a SvelteKit consumer app. Form follows the
  standard AOH trio: `@mssfoobar/form-types` (wire DTOs), `@mssfoobar/form-client`
  (typed REST clients — `FormClient` admin, `FormSubmitterClient` read+submit,
  `AmmClient` attachments, via the host's shared `/aoh/gateway` BFF), and
  `@mssfoobar/form-web-sdk` (the SurveyJS authoring Creator + read/submit renderer).
  Unlike GIS/MSR/UNH the web SDK ships NO Provider, nav, or transparent proxy — you
  `new` the clients and route them through the module-multiplexing `/aoh/gateway`.
  Use whenever a user wants to add forms, build a form designer/studio, render or
  submit a form, embed the SurveyJS creator, handle form attachments, or consume
  `@mssfoobar/form-client` / `@mssfoobar/form-web-sdk`. Triggers: "add forms",
  "integrate form", "use form-client", "use form-web-sdk", "form designer",
  "SurveyJS creator", "render/submit a form", "FormClient", "createSurveyCreator",
  "wire the form gateway".
license: Proprietary
metadata:
  owner: AOH Platform Team
  status: v1
---

# AOH Form Integration

Wire the AOH **Form module** into a SvelteKit consumer app. Form follows the
standard AOH `types ← client ← web` trio:

- **`@mssfoobar/form-types`** — the wire DTOs (`Form`, `Submission`, `FormVersion`,
  `FormDraft`, `FormTheme`, `FormTable`, `Changelog`, `Attachment`, the `*Params`/
  `*Response` shapes). Source-only, zero runtime.
- **`@mssfoobar/form-client`** — the typed fetch clients: `FormClient` (`.`, admin),
  `FormSubmitterClient` (`./submitter`, read+submit), `AmmClient` (`./amm`,
  attachments). Depends on form-types; source-only (consumed by your bundler).
- **`@mssfoobar/form-web-sdk`** — the SurveyJS authoring Creator + read/submit
  renderer (`./renderer`). Ships compiled Svelte components. Depends on
  form-client + form-types.

The backend is the Go **`form-service`**, with file attachments served by the
**AMM** service. This skill is the integration playbook — the clients, the
SurveyJS Creator/renderer, and the shared gateway BFF wiring to stand up working
form authoring + fill/submit pages in your app.

## Three things to understand first

**1. No Provider, no nav, no transparent proxy — you `new` the clients yourself.**
GIS gives you `<GisProvider>` + a map; UNH gives you `<UnhProvider>` + a transparent
`[...path]` proxy. Form gives you **plain typed client classes** (from
`@mssfoobar/form-client`) you construct directly (`new FormClient({ module: "form" })`)
plus a **SurveyJS renderer** (from `@mssfoobar/form-web-sdk/renderer`). No context
provider, no SDK-supplied BFF — requests ride the host's **shared `/aoh/gateway`
BFF** that multiplexes *all* modules by code.

**2. Clients (`form-client`) vs DTOs (`form-types`) vs renderer (`form-web-sdk`).**
Import the client classes from `@mssfoobar/form-client` (`.` / `/submitter` /
`/amm`), the DTO types from `@mssfoobar/form-types`, and the SurveyJS renderer from
`@mssfoobar/form-web-sdk/renderer`. `FormClient` has the full admin surface;
`FormSubmitterClient` is the **browser-safe** read+submit subset (no admin) for
end-user pages; `AmmClient` handles file upload/download/delete. All three share one
constructor and the same `Result<T>` envelope (`{ ok: true, … } | { ok: false, error }`).

**3. The renderer is the centerpiece, and it's browser-only.** Authoring uses
`createSurveyCreator`; filling/editing/viewing uses the high-level
`newSubmission` / `editSubmission` / `viewSubmission` flows (they take a
`formClient` + `ammClient`, fetch the version, wire file handlers, and apply
state-based permissions for you). Both render imperatively into a DOM node
(`.render(container)`), so renderer/creator pages **disable SSR**
(`export const ssr = false`).

## What you're wiring (mental model)

```text
src/
  routes/aoh/form/                     ← YOUR pages (no SDK Provider/nav; all ssr=false)
    +page.svelte          FormClient.listForms()                                              ← form templates list
    [id]/+page.svelte     FormClient — manage one form's drafts/versions (rename, unpublish) ← form detail
    [id]/[resourceType]/[resourceId]/   createSurveyCreator + load draft|version + autosave  ← authoring editor
    create/+page.svelte   FormSubmitterClient — own submissions list + create-submission dialog ← my submissions
    edit/[id]/+page.svelte  editSubmission(...) { useSubmissionData / useDraftData }          ← fill / edit a submission
    submission/+page.svelte       FormClient.adminListSubmissions()                            ← admin submissions list
    submission/[id]/+page.svelte  FormClient — read-only submission + audit log               ← submission detail
  routes/aoh/gateway/[...path]/+server.ts        ← SHARED module-multiplexing BFF (binary-safe)
        form|theme → FORM_URL,  amm → AMM_URL   (+ Bearer token, fail-closed)
  routes/aoh/api/form/images/[attachmentId]/+server.ts   ← image reverse-proxy (form <img> URLs)
  hooks.server.ts      event.locals.ammClient = new AmmClient({module:"amm", fetch, timeout})

  imports:  clients → @mssfoobar/form-client(/submitter|/amm)
            DTOs    → @mssfoobar/form-types
            renderer→ @mssfoobar/form-web-sdk/renderer (+ /renderer/styles.css)
```

Two halves to every integration:

1. **Client (`.svelte` / load)** — `new FormClient(...)` etc. from
   `@mssfoobar/form-client`, then either call typed methods directly or hand them to
   a renderer flow. The client's default `gatewayURL` (`/aoh/gateway`) points at the
   host's own BFF route.
2. **Server (`+server.ts` BFF)** — the **shared** `/aoh/gateway/[...path]` proxy
   that maps a leading **module code** (`form`, `theme`, `amm`) to an upstream base
   URL and attaches the user's bearer. This is the **same** BFF GIS/dash use — Form
   adds no route of its own beyond registering the module bases.

## Prerequisites & install

Peer requirements:

- **SvelteKit + Svelte 5** (runes). Peer packages: `@mssfoobar/ui`,
  `@mssfoobar/logger`, `svelte`.
- **`survey-js-ui` must be a DIRECT dependency of your app.** `form-web-sdk` bundles
  the SurveyJS *engine* (`survey-core`, `survey-creator-core`, `survey-creator-js`)
  as its own deps, but the render/submit pages **side-effect-import `survey-js-ui` at
  the host** (`import "survey-js-ui"`, Step 5). Under strict pnpm a transitive dep is
  **not** hoisted into your app's `node_modules`, so that import fails unless you
  declare it. Pin it to the SDK's range (`^2.5.14`).

Authenticate the `@mssfoobar` scope to GitHub Packages (project-level `.npmrc`):

```ini
@mssfoobar:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${NODE_AUTH_TOKEN}
```

…with `NODE_AUTH_TOKEN` exported (a token with `read:packages`). Then install all
three form packages (+ peers):

```bash
pnpm add @mssfoobar/form-web-sdk @mssfoobar/form-client @mssfoobar/form-types \
         @mssfoobar/ui @mssfoobar/logger svelte survey-js-ui
```

`form-client` pulls `form-types` transitively, and `form-web-sdk` pulls both — but
declare the ones you import directly (importing a transitive dep by name fails to
resolve under pnpm-strict isolation).

## Integration steps

### Step 1. SSR config — bundle the SDK source, disable SSR on form pages

`@mssfoobar/form-web-sdk` ships **compiled Svelte components**, and `form-client` /
`form-types` are consumed **as source**. Both need Vite to SSR-transform the
`@mssfoobar/*` graph. In `vite.config.ts`:

```ts
export default defineConfig({
  // …sveltekit()…
  ssr: {
    noExternal: [/^@mssfoobar\//],   // form-web-sdk .svelte + form-client/types source
  },
});
```

The clients run **in the browser** and the SurveyJS Creator/renderer paint into a
DOM node (`.render(el)`), so **every `/aoh/form/*` page that builds a client in
`onMount`** (rather than a server `load`) **disables SSR** — including the admin
**list** page. Add a sibling `+page.ts`:

```ts
// +page.ts for: the templates list (/aoh/form), the authoring editor, create/ (own
// submissions + create dialog), and edit/[id]/ (fill/edit a submission)
export const ssr = false;
```

On submission **render** pages also import the SurveyJS UI runtime once so the model
can paint (`survey-js-ui` must be a direct dep — see Prerequisites):

```ts
import "survey-js-ui";
```

### Step 2. Wire the shared `/aoh/gateway` BFF

The clients default `gatewayURL = "/aoh/gateway"` and build every request as
`{gatewayURL}/{module}/{apiVersion}/{path}` — e.g. `new FormClient({ module: "form" })`
→ `GET /aoh/gateway/form/v1/forms`. The version segment is per module (`v1` for
`form`/`theme`, none for `amm`) and needs nothing from the host: the route below
forwards whatever follows the module code. It owns one **catch-all** route that maps
the leading module code to an upstream base and attaches the bearer:

```ts
// src/routes/aoh/gateway/[...path]/+server.ts  (abridged — see references/bff.md)
import { env } from "$env/dynamic/private";
import { TEST_BEARER } from "$lib/auth-adapter";

function upstreamFor(module: string): string | undefined {
  switch (module) {
    case "form":
    case "theme": return env.FORM_URL;   // form-service
    case "amm":   return env.AMM_URL;    // attachment service
    default:      return undefined;
  }
}
// IAM_URL set → require a live session, forward locals.authResult.accessToken
// (fail closed → 401); unset → forward TEST_BEARER (dev identity). Then:
//   headers.Authorization = `Bearer ${token}`  and forward to `${base}/${rest}`.
```

| Var | Side | Purpose |
|---|---|---|
| `FORM_URL` | server | gateway → `form-service` base (serves `form` **and** `theme` modules) |
| `AMM_URL` | server | gateway → attachment service base (the `amm` module) |
| `IAM_URL` | server | when set, the BFF requires a live session and forwards its token (fail-closed); unset → dev identity |

**On an `aoh-web-init` base, do NOT paste the route above** — that base already
ships a **config-driven** gateway (a generic `[...path]/+server.ts` plus a
`resolveModule` helper). You wire Form by **registering modules in
`gateway.config.ts`**, not by editing an `upstreamFor` switch:

```ts
// gateway.config.ts → gatewayConfig.modules
modules: {
  form:  { host: process.env.FORM_URL || "http://form-service:8080" },
  theme: { host: process.env.FORM_URL || "http://form-service:8080" },
  amm:   { host: process.env.AMM_URL  || "http://amm-service:8080" },
}
```

That base's gateway already forwards the bearer (reading
`locals.authResult.access_token`, **snake_case**), fails closed, and is binary-safe
on both directions — so multipart AMM uploads and image/file downloads work. If your
host instead carries the **hand-rolled** route shown above (the same gateway other
AOH modules like GIS and dash use), just add the `form`/`theme`/`amm` cases to its
`upstreamFor` — it's binary-safe the same way. Both bases + the full proxy, token
policy, and the image reverse-proxy route: `references/bff.md`.

### Step 3. Construct the clients

Construct directly from `@mssfoobar/form-client` — no provider. The `module` code
selects the gateway upstream; in a **server `load`** pass `fetch` so relative URLs +
cookies resolve. Import DTO types from `@mssfoobar/form-types`:

```ts
import { FormClient } from "@mssfoobar/form-client";                 // admin
import { FormSubmitterClient } from "@mssfoobar/form-client/submitter"; // read+submit
import { AmmClient } from "@mssfoobar/form-client/amm";              // attachments
import type { Form, Submission } from "@mssfoobar/form-types";       // DTOs

const admin   = new FormClient({ module: "form" });                  // browser
const load    = new FormClient({ module: "form", fetch });           // server load()
const submit  = new FormSubmitterClient({ module: "form" });         // end-user pages
const amm     = new AmmClient({ module: "amm", timeout: 300_000 });  // large uploads
```

Every method returns a `Result<T>` — **always branch on `.ok`** (errors are values,
not throws):

```ts
const res = await admin.listForms({ page_size: 10 });
if (!res.ok) { toast.error(res.error.displayMessage); return; }
forms = res.data;
```

A common host pattern is one shared `AmmClient` per request on `event.locals`
(bound to `event.fetch` for SSR-correct gateway routing):

```ts
// hooks.server.ts
event.locals.ammClient = new AmmClient({ module: "amm", fetch: event.fetch, timeout: 300_000 });
// app.d.ts → interface Locals { ammClient?: AmmClient }  (import type from @mssfoobar/form-client/amm)
```

Full method catalogue (admin / read / submission / theme / amm) + the `Result`
envelope + `FormClientApiError`: `references/clients.md`.

### Step 4. Authoring — the SurveyJS Creator

`createSurveyCreator` (from `@mssfoobar/form-web-sdk/renderer`) returns a configured
SurveyJS Creator; the `FormClient` (from `@mssfoobar/form-client`) persists it. Mount
it wherever you author — a dedicated greenfield page or (as the reference host does)
the draft/version editor that loads an existing `form_json` and autosaves it back:

```svelte
<script lang="ts">
  import { onMount } from "svelte";
  import {
    createSurveyCreator,
    customDesignerLightTheme, customDesignerDarkTheme,   // Creator CHROME themes
    defaultLight, defaultDark,                           // designed-FORM surface themes
  } from "@mssfoobar/form-web-sdk/renderer";
  import { FormClient } from "@mssfoobar/form-client";
  import "@mssfoobar/form-web-sdk/renderer/styles.css";   // SurveyJS + AOH creator CSS

  const client = new FormClient({ module: "form" });
  let el: HTMLElement;
  let creator: ReturnType<typeof createSurveyCreator>;

  onMount(() => {
    creator = createSurveyCreator({ isEditMode: true });   // + customThemes?, tableSchema?

    // Theme BOTH layers to the host's mode, or the Creator chrome AND the designed
    // form fall back to SurveyJS's light default and render white in dark mode:
    const isDark = document.documentElement.classList.contains("dark");
    creator.applyCreatorTheme(isDark ? customDesignerDarkTheme : customDesignerLightTheme);
    creator.theme = isDark ? defaultDark : defaultLight;

    creator.JSON = STARTER_FORM;
    creator.render(el);
    return () => creator?.dispose();        // dispose on unmount or the Creator leaks
  });

  // Publish flow: form → draft → version
  async function publish(name: string) {
    const f = await client.createForm({ name });
    if (!f.ok) return;
    const d = await client.createFormDraft({
      form_id: f.data.id, name,
      form_json: creator.JSON,
      theme_json: creator.theme as unknown as Record<string, unknown>,  // creator.theme is ITheme
    });
    if (!d.ok) return;
    await client.createFormVersion({ form_id: f.data.id, draft_id: d.data.id, name: "v1" });
  }
</script>
<!-- the Creator needs a sized, bounded container -->
<div bind:this={el} class="min-h-0 flex-1"></div>
```

Editing an existing draft: load it server-side (`client.getFormDraft` /
`getFormVersion`, `listThemes`, `getFormTable`), then `creator.JSON =
data.resource.form_json` on mount and auto-save via `client.updateFormDraft`. Full
authoring + publish + table-schema flow: `references/renderer.md`.

**Date/time questions render the SDK's own picker, not the native input** (≥ 1.2.0),
so the displayed format follows the schema rather than the operator's OS locale.
Authors set `dateFormat` / `timeFormat` / `timezone` form-wide, per question, or per
matrix column — nothing to wire in the host. One consequence for a host that reads
values itself: a `datetime-local` answer is now an instant **with an offset**
(`2026-07-01T10:30:00Z`), where it used to be a bare `2026-07-01T18:30`. Choices,
defaults and the other answer shapes: `references/renderer.md`.

### Step 5. Fill / edit / view submissions

Use the high-level flows — they fetch the published version, build the model, wire
AMM file handlers, and apply state permissions. Clients come from
`@mssfoobar/form-client`, the flow from `@mssfoobar/form-web-sdk/renderer`. **Create:**

```svelte
<script lang="ts">
  import { onMount } from "svelte";
  import { page } from "$app/state";        // runes page state (or use your layout data)
  import { goto } from "$app/navigation";
  import { FormSubmitterClient } from "@mssfoobar/form-client/submitter";
  import { AmmClient } from "@mssfoobar/form-client/amm";
  import type { Form } from "@mssfoobar/form-types";
  import { newSubmission, type FormModel } from "@mssfoobar/form-web-sdk/renderer";
  import "@mssfoobar/form-web-sdk/renderer/styles.css";
  import "survey-js-ui";

  let { data } = $props();                  // { form } from load (Svelte 5 runes)
  // userRoles drives field-level permissions. Source them from your auth layer's
  // active-tenant roles, surfaced through layout data — roles live UNDER the active
  // tenant, there is NO top-level `claims.roles`. The exact path depends on your
  // auth base: an aoh-web-init base exposes them at `page.data.user.active_tenant.roles`;
  // an auth-sdk base surfaces them at `authResult.claims.active_tenant.roles`. Pass []
  // when auth is off; empty roles render every field read-only/hidden per the form's permissions.
  const userRoles: string[] = page.data.user?.active_tenant?.roles ?? [];

  const client = new FormSubmitterClient({ module: "form" });
  const ammClient = new AmmClient({ module: "amm", timeout: 300_000 });
  let el: HTMLElement, model: FormModel;

  onMount(async () => {
    const ns = await newSubmission({
      formId: data.form.id,
      userRoles,
      formClient: client,
      ammClient,                             // same AmmClient the renderer expects — no cast needed
      onComplete: () => goto("/aoh/form"),   // on an AOH base, eslint may want goto(resolve("/aoh/form"))
    });
    model = ns.model;                    // ns.save() → wire to a "Save Draft" button
    model.render(el);
  });
</script>
<div bind:this={el}></div>
```

> **No `AmmClient` cast needed.** The renderer flows type their `ammClient` param
> from `@mssfoobar/form-client/amm` — the exact package you construct it from — so
> `ammClient` passes straight through. (Pre-split, the client shipped as `dist` while
> the renderer shipped as source, forcing an `as unknown as …` cast; the trio split
> removed that.)

- **Edit:** `editSubmission(...)` returns `{ model, save, useSubmissionData, useDraftData }`
  — call `useSubmissionData()` (committed answers, protects existing files) or
  `useDraftData()` before `model.render(el)`.
- **View (read-only):** `viewSubmission(...)` resolves to a `FormModel` (a
  `Promise<FormModel>` — the bare model, not the `{ model, save, … }` object). `await` it.
- **Low-level:** if you need the model without the fetch/permission plumbing, use
  `createFormModel({ formJson, themeJson, context, initialData, editable, onComplete })`
  then `setupFileHandlersAndData(model, { ammClient, initialData })` **before**
  setting `model.data` (handlers must be registered first or initial attachments
  don't load). This is the path a WFE human-task step takes — see cross-link below.

Submission method shapes, the `useSubmissionData`/`useDraftData` toggle, and the
`setupFileHandlersAndData` ordering rule: `references/renderer.md`.

### Step 6. Backend — `form-service` + AMM

Point `FORM_URL` at `form-service` and `AMM_URL` at the attachment service. In
deployed environments your platform provides them; for local dev you run them
yourself (the `aoh-compose` `form` fragment stands both up and also wires the OTEL
gateway env). `form-service` validates a Keycloak bearer with `active_tenant` on
every call, so real authed calls need a reachable IAMS; the `/livez` / `/readyz`
probes are auth-free.

> **SurveyJS license banner (optional).** `survey-creator-js` is commercial. The SDK
> **does not** set a license key, so the Creator shows SurveyJS's "developer license
> required" banner in the designer. This is cosmetic — authoring still works. If your
> org holds a SurveyJS license, call `survey-creator-core`'s `setLicenseKey(...)`
> yourself in your app bootstrap; there is **no** SDK env var for it (the legacy
> standalone `form-web` app's `PUBLIC_SURVEYJS_KEY` is unrelated).

## Cross-link: forms inside a WFE workflow

If you're rendering forms as **WFE human-task steps**, do **not** re-derive this —
`@mssfoobar/wfe-web-sdk` already embeds the Form renderer (it injects `form_client` +
`amm_client` on `<WorkflowProvider>` and dual-submits to both form-service and the
workflow). Use the **`aoh-wfe-designer`** skill for that path; this skill is for
embedding the Form SDK standalone.

## Verify it works

1. `pnpm dev` with `FORM_URL` + `AMM_URL` set (+ `IAM_URL` if auth is on). Visit
   `/aoh/form` → the form templates list renders via `FormClient`.
2. Open a draft in the authoring editor → the SurveyJS **Creator renders** with the
   AOH form-states and per-question-permission editors; publish creates form → draft
   → version.
3. `/aoh/form/create` → **Create submission** → pick a published form →
   `/aoh/form/edit/[id]` **renders the form**; attach a file (exercises AMM upload
   through the gateway) and submit → lands back on the submissions list. If the form
   has an image element, confirm it **renders** (exercises the image reverse-proxy).
4. Edit an existing submission → `useSubmissionData()` shows committed answers with
   attachments intact.

## Common failure modes

| Symptom | Cause | Fix |
|---|---|---|
| `Cannot find module … 'survey-js-ui'` (svelte-check/build) | transitive dep of form-web-sdk, not hoisted to your app under strict pnpm | add `survey-js-ui` as a **direct** dep (Prerequisites). |
| Build/SSR fails resolving `@mssfoobar/form-*` | the form graph (web-sdk `.svelte` + client/types source) not SSR-bundled | `ssr.noExternal: [/^@mssfoobar\//]` (Step 1). |
| Creator/designed form renders **white in dark mode** | the Creator chrome + form-surface themes weren't set | `creator.applyCreatorTheme(isDark ? customDesignerDarkTheme : customDesignerLightTheme)` + `creator.theme = isDark ? defaultDark : defaultLight` (Step 4). |
| Renderer/creator page throws `document is not defined` / blank on first paint | SurveyJS renders into the DOM (browser-only) | `export const ssr = false` in the page's `+page.ts` (Step 1); mount in `onMount`. |
| The **list** page errors/flashes under SSR (no renderer on it) | it still builds a `FormClient` in `onMount` | `export const ssr = false` on the list page too — every client-in-`onMount` page (Step 1). |
| Form renders but **fields are unstyled / SurveyJS chrome missing** | renderer CSS not imported | `import "@mssfoobar/form-web-sdk/renderer/styles.css"` on the page; `import "survey-js-ui"` on submission pages (Steps 4–5). |
| `theme_json: creator.theme` won't type-check (`ITheme` not assignable) | `creator.theme` is `ITheme`, the API wants `Record<string, unknown>` | `creator.theme as unknown as Record<string, unknown>` (Step 4). |
| Client imports resolve but **types don't** (`Form`, `Submission`, …) | DTOs live in `@mssfoobar/form-types`, not the client/renderer | import DTO types from `@mssfoobar/form-types` (Step 3). |
| Every client call **503s** (`No upstream configured for module …`) | the gateway has no entry for the module | register `form`/`theme`/`amm` (config-driven base) or add the `upstreamFor` cases (hand-rolled) + set `FORM_URL`/`AMM_URL` (Step 2). A genuine **404** instead means the module IS mapped but the upstream endpoint path is wrong. |
| `401` from the gateway | `IAM_URL` set but no/expired session | sign in; the BFF fails closed by design (Step 2 / `references/bff.md`). |
| Form opens but **every field is read-only / hidden** | `userRoles` empty/unsourced → permissions deny all | source roles from your auth layer's active-tenant roles via layout data (Step 5). |
| Code "ignores errors" / `res.data` is `undefined` | the API returns `Result<T>`, not a throw | branch on `if (!res.ok)` before reading `res.data` (Step 3). |
| Existing attachments don't load when editing/viewing | `setupFileHandlersAndData` ran after `model.data` was set | register handlers first — pass `initialData` to `setupFileHandlersAndData` (Step 5). |
| Form **image elements** 404 (`GET /aoh/api/form/images/… 404`) | no host route serving the SurveyJS image URLs (`IMAGE_SERVE_PATH`) | add the image reverse-proxy route (`references/bff.md` → "Serving form images"). |
| Relative URLs fail in a server `load` | client built without `fetch` | `new FormClient({ module: "form", fetch })` (Step 3). |
| `pnpm lint` fails on `goto("…")` on an AOH base | the base enforces `svelte/no-navigation-without-resolve` | wrap navigation: `goto(resolve("/aoh/form"))`. |

## References

- `references/clients.md` — the full method catalogue for `FormClient` (admin +
  read + submission + theme), `FormSubmitterClient` (the read+submit subset), and
  `AmmClient`, all from `@mssfoobar/form-client`; the `FormClientConfig` constructor;
  the `Result<T>` envelope and `FormClientApiError`; the
  `{gatewayURL}/{module}/{apiVersion}/{path}`
  URL model + SSR `fetch`; and where the DTOs live (`@mssfoobar/form-types`).
- `references/renderer.md` — `createFormModel` / `FormModel`; the `newSubmission` /
  `editSubmission` / `viewSubmission` flows and their option shapes;
  `setupFileHandlersAndData` (ordering rule); the host field-population hooks
  (`onAfterRenderQuestion` + `setValue`, ≥ 1.1.0) with a worked button-in-`html`-question
  → host-modal → write-back example; `createSurveyCreator` + the publish/draft/version
  flow; the author-set date/time display formats (`dateFormat` / `timeFormat` /
  `timezone`, ≥ 1.2.0) and the answer shapes a host should expect;
  `applyStatePermissions` / `updateFormContext`; the preview-tab-vs-AMM file
  distinction; the theme exports. All from `@mssfoobar/form-web-sdk/renderer`.
- `references/bff.md` — the shared `/aoh/gateway/[...path]` module-multiplexing
  proxy (binary-safe request + response, `sandbox`/`nosniff` hardening): `upstreamFor`,
  the fail-closed token policy, the `FORM_URL`/`AMM_URL`/`IAM_URL` env contract, the
  config-driven `aoh-web-init` branch, the **image reverse-proxy route** that serves
  a rendered form's `<img>` URLs, and the server-build/container wiring so the SDK
  subpath exports resolve at runtime.
- Forms inside workflows: use the `aoh-wfe-designer` skill (`@mssfoobar/wfe-web-sdk`
  embeds the Form renderer for human-task steps).
