# `@mssfoobar/unh-web-sdk` component & export catalog

Every public export, what it does, and the subpath to import it from. The
barrel (`@mssfoobar/unh-web-sdk`) re-exports everything; the subpaths
exist for granular imports / smaller dependency graphs. Both forms work.

Companion packages: wire shapes (DTOs) originate in `@mssfoobar/unh-types`;
the typed REST client is `@mssfoobar/unh-client`. This SDK bundles both and
re-exports the pieces you need through its own subpaths — so **import DTO types
from `@mssfoobar/unh-web-sdk/types`**, not the bare `@mssfoobar/unh-types`
(transitive-only, unresolvable by name from a consumer under pnpm).

**Unlike some AOH module SDKs, this one ships NO list/table components and no
simple channel/distribution create-edit forms.** The surface is the editors,
managers, IAMS picker, template form, and provider — the pieces that carry real
module logic. **Your app composes the lists AND the plain channel/distribution
CRUD forms itself** from `@mssfoobar/unh-client` (`.list()` / `.create()` /
`.update()` / `.remove()`) + the SDK's exported `validate*` / `initial*Values`
helpers + a `@mssfoobar/ui` `DataTable` / `Input` / `Sheet` (see
[Host-composed lists](#host-composed-lists-no-list-components)).

## Published subpath exports

| Subpath | What it is |
|---|---|
| `.` (barrel) | Everything below, re-exported. |
| `./provider` | `<UnhProvider>` (default) + the context accessors (`GetUnhClient`, `GetUnhProviderContext`, `GetDarkModeStore`, `resolveBrowserBase`, `UNH_PROVIDER_CONTEXT`, type `UnhProviderValue`). |
| `./nav` | `unhNav` + types `UnhNav`, `UnhNavSidebarEntry`. |
| `./custom-channel-parameters` | `CustomChannelParameters` (default). |
| `./distribution-members` | `DistributionMembers` (default export). Its types `MemberOption` / `DirectoryLoad` / `DirectoryType` are **not** exported from this subpath — import them from the barrel `@mssfoobar/unh-web-sdk`. |
| `./notification-template-form` | `NotificationTemplateForm` (default) + type `EmailBodyEditorArgs`. |
| `./send-template-dialog` | `SendTemplateDialog` (default). |
| `./email-body-editor` | `EmailBodyEditor` (default) — the template form's rich-text email-body editor (Edra/TipTap, `{{variable}}` binding). Also usable standalone when you compose your own template UI. Props: `EmailBodyEditorArgs` (from the barrel) + `fill?: boolean` — `fill` makes it fill its parent height with a pinned toolbar + internal body scroll (for a full-height host like a dialog); the default grows with content and never scrolls internally (for an inline host relying on a single outer page scroll). |
| `./server` | `createUnhProxy(cfg)` — the transparent BFF proxy factory (server-only). |
| `./types` | Wire DTOs + client surface (re-exported from `@mssfoobar/unh-types`). |
| `./styles/app.css` | The SDK's Tailwind v4 stylesheet. |

The barrel also re-exports: `createUnhClient`, `UnhHttpError` (from
`@mssfoobar/unh-client`); `DEFAULT_UNH_BFF_BASE`, `ROUTES`;
`getUnhLogLevel` / `setUnhLogLevel` / type `UnhLogLevel`; `renderTemplate`,
`extractPlaceholders` (binding helpers); and the validation helpers.
`EmailBodyEditor` is exported at `./email-body-editor` — the form's built-in
default body editor, also usable standalone when you compose your own template UI
(see below).

## Provider & context

| Export | Subpath | Role |
|---|---|---|
| `UnhProvider` (default) | `/provider` | App-root context wrapper. Mount once, high in the tree. Owns the browser `@mssfoobar/unh-client`, the BFF base, dark-mode store, feature flags, log level. See props below. |
| `GetUnhClient()` | `/provider`, barrel | Returns the active `UnhClient`. **Throws** if no `<UnhProvider>` ancestor — a missing client is a wiring bug, not a standalone path. |
| `GetUnhProviderContext()` | `/provider`, barrel | Returns the active `UnhProviderValue` or `undefined` (no provider mounted). |
| `GetDarkModeStore()` | `/provider`, barrel | The provider's `dark_mode_store`, or a fresh `writable(false)` if none. |
| `resolveBrowserBase(bffBase)` | `/provider`, barrel | Prefixes a relative BFF base with `location.origin` (absolute URLs pass through) — `@mssfoobar/unh-client` needs an absolute base. |
| `UNH_PROVIDER_CONTEXT` | `/provider`, barrel | The context `Symbol`. |

### `<UnhProvider>` props (`Props` / `UnhProviderValue`)

| Prop | Type | Required | Notes |
|---|---|---|---|
| `client` | `UnhClient` | no | Supply a client directly (tests / SSR). Otherwise one is built lazily from `bff_base`. |
| `bff_base` | `string` | no | Base path of your app's UNH BFF (transparent-proxy) routes. Default `DEFAULT_UNH_BFF_BASE` = `"/aoh/unh/api"`. |
| `dark_mode_store` | `Writable<boolean>` | no | Your app's theme store; the SDK only reacts to it. Falls back to a stable internal `writable(false)`. |
| `feature_flags` | `Record<string, boolean>` | no | Open-ended SDK-side toggle bag. |
| `log_level` | `UnhLogLevel` | no | UNH SDK log verbosity. Default `"warn"`. Reactive via `$effect` → `setUnhLogLevel`. |
| `children` | `Snippet` | **yes** | App subtree. |

> The SDK does **not** mount `Toaster`. Components emit toasts via
> `svelte-sonner`; your app mounts `<Toaster />` (from
> `@mssfoobar/ui/toast`) once at its app root.

## Channel-settings components

The plain create/edit **channel forms are host-composed** — see
[Host-composed lists](#host-composed-lists-no-list-components). Build them from
the exported validators (`validateEmailChannel` / `validatePushChannel` /
`validateCustomChannel`, `initialEmailValues` / `initialPushValues`,
`parseServiceAccountKey`) + `client.{email,push,custom}Channels.create/update` +
`@mssfoobar/ui`. Channel secrets (email `password`, push `service_account_key`)
are **write-only** — the `initial*Values` helpers seed them blank; send a secret
only when the operator re-enters it, and map a 409 onto the `name` field.

The SDK ships one channel-settings component — the custom-channel **parameters
manager**, which carries real CRUD + validation logic:

| Component | Subpath | Purpose | Key props |
|---|---|---|---|
| `CustomChannelParameters` | `/custom-channel-parameters` | Manage a custom channel's parameters: list + add/edit/remove (own `DataTable` + dialogs) driven through the client. Per-param: name, description, `regexp_validation`, `is_multi_value`. | `channelId: string` (**required**) |

### Host-composed forms — field shapes + worked example

The create/edit forms are yours to build. Each **seeds** from an exported helper
(or a literal), **validates** with an exported `validate*`, and calls the client.
Two rules to honor: **secrets are write-only** (seed blank/null, send only when
re-entered), and the email **`port` must be coerced to a number** for the create
body (the seed types it `number | string`).

| Form | Seed | Validate | Create body | Update body |
|---|---|---|---|---|
| Email | `initialEmailValues(channel?)` → `{ name, username, password: "", send_from, sender_name, host, port, encryption, auth_method, ca_cert }` (`port`: `number \| string`; a new channel starts on `tls` + `plain` + port `465`) | `validateEmailChannel(v)` | `{ name, send_from, host, port: number, username?, password?, sender_name?, encryption?, auth_method?, ca_cert? }` | create body **+ `occ_lock`** |
| Push | `initialPushValues(channel?)` → `{ name, service_account_key: null }`; parse the upload with `parseServiceAccountKey(text)` → `{ value?, error? }` | `validatePushChannel(v)` | `{ name, service_account_key: Record<string, unknown> }` | create body **+ `occ_lock`** |
| Custom | literal `{ name: channel?.name ?? "", endpoint: channel?.endpoint ?? "" }` (no seed helper) | `validateCustomChannel(v)` | `{ name, endpoint }` | create body **+ `occ_lock`** |
| Distribution list | literal `{ name: list?.name ?? "" }` (no seed helper) | `validateDistributionList(v)` | `{ name }` | `{ name, occ_lock }` |

`validate*` returns `FieldErrors` — an object keyed by field name (`name`,
`username`, `port`, `send_from`, …); render each message beside its input. On
**edit**, fetch the current DTO (`client.<resource>.get(id)`) and pass its
**`occ_lock`** to `.update` — every read DTO carries `occ_lock`. A `409` is a
duplicate-name / OCC conflict — detect with `isConflict(err)` and show
`errorMessageOf(err)`. (All of `initial*Values`, `validate*`,
`parseServiceAccountKey`, `isConflict`, `errorMessageOf`, `hasErrors` are
exported from the barrel.)

```svelte
<script lang="ts">
  import { GetUnhClient, initialEmailValues, validateEmailChannel, hasErrors, isConflict, errorMessageOf } from "@mssfoobar/unh-web-sdk";
  import type { EmailChannel } from "@mssfoobar/unh-web-sdk/types";
  let { channel, onSaved }: { channel?: EmailChannel; onSaved?: () => void } = $props();
  const client = GetUnhClient();
  let v = $state(initialEmailValues(channel));                 // password seeded "" (write-only)
  let errors = $state<Record<string, string | undefined>>({});
  async function save() {
    errors = validateEmailChannel(v);
    if (hasErrors(errors)) return;
    const body = { ...v, port: Number(v.port) };               // create wants port: number
    try {
      channel
        ? await client.emailChannels.update(channel.id, { ...body, occ_lock: channel.occ_lock })
        : await client.emailChannels.create(body);
      onSaved?.();
    } catch (e) {
      errors = { ...errors, name: errorMessageOf(e, isConflict(e) ? "Name already exists" : "Failed to save the channel") };
    }
  }
</script>
<!-- render <Input bind:value={v.name}/>, <Input type="password" bind:value={v.password}/>,
     <Input type="number" bind:value={v.port}/>, a Select for v.encryption and
     one for v.auth_method, a Textarea for v.ca_cert, … -->
```

Push/custom/distribution-list forms follow the same shape — swap the seed,
`validate*`, and client resource per the table above.

## Distribution-list components

The name-only distribution-list **create/edit form is host-composed**
(`validateDistributionList` + `client.distributionLists`). The SDK ships the
member picker — the hard part:

| Component | Subpath | Purpose | Key props |
|---|---|---|---|
| `DistributionMembers` | `/distribution-members` | Manage a list's members live (per-member add/remove, no save step). Internal IAMS members (user/role/group) via a browse dialog; external email/phone added inline; To/Cc/Bcc delivery mode on email + IAMS members (not phone). Own member `DataTable` + bulk "Remove selected". | `listId: string` (**required**), `loadDirectory?: DirectoryLoad` |

### The IAMS picker — `loadDirectory`

`DistributionMembers`'s `loadDirectory` is your directory loader:

```ts
type DirectoryType = "user" | "role" | "group";
type DirectoryLoad = (type: DirectoryType) => Promise<MemberOption[]>;
interface MemberOption { value: string; label: string; hint?: string }
```

Each `value` must be the id/name the resolver looks the member up by, **keyed by
`type`**: `user` → id, `role` → **name** (AAS addresses roles by name), `group` →
id. A `group` keyed by name (like a `role`) silently drops at send time — AAS 404s,
the member lands in the send's `unresolved[]`, and the send still returns 200.

Fetch the tenant directory **server-side** (so the bearer never reaches
the browser). IAMS has no usable substring search, so return a capped list
and let the picker filter it client-side as the operator types. **When
`loadDirectory` is omitted, the picker degrades to free-text id entry** —
internal members still work, just without browse.

Wire it through your own BFF route (e.g.
`src/routes/aoh/unh/iams/[type]/+server.ts`) and pass the **client**
`loadDirectory={iamsEnabled ? iamsDirectory : undefined}` (see `bff.md` for the
client `iamsDirectory` glue vs the server `loadIamsDirectory` loader).

## Notification-template components

| Component | Subpath | Purpose | Key props |
|---|---|---|---|
| `NotificationTemplateForm` | `/notification-template-form` | Create / edit a template with per-channel payloads (email / push / one-or-more custom), a distribution-list link with a resolved-recipient To/Cc/Bcc preview, and `{{key}}` binding editors. At least one channel required; a **distribution list is required by default**. The email body edits in a **popup dialog** — the form shows an auto-sizing sandboxed `<iframe>` preview of the rendered HTML (Edit button beside the label). The host gives the form a **bounded height** (it fills its container and pins Save/Cancel at the bottom). | `template?: NotificationTemplate`, `onSaved?`, `onCancel?`, `loadDirectory?: DirectoryLoad` (resolves internal recipient names in the preview), `variables?: string[]`, `bodyEditor?: Snippet<[EmailBodyEditorArgs]>`, `requireDistributionList?: boolean` (default `true`; `false` restores the "None" option for ad-hoc-recipient hosts) |
| `SendTemplateDialog` | `/send-template-dialog` | Trigger a templated send; surfaces per-channel result + unresolved members. Attempt-and-surface: **not** gated on a role claim — a 403 is surfaced like any other failure. | `templateId: string` (**required**), `templateName?: string`, `open?: boolean` (`$bindable`) |

### The `bodyEditor` override (`EmailBodyEditorArgs`)

The editor (built-in or your override) opens in a **popup dialog**; the form
itself shows a read-only auto-sizing sandboxed `<iframe>` preview of the rendered
`body`. When omitted, the dialog hosts the **built-in** rich-text editor
(`EmailBodyEditor` with `fill`, based on Edra / TipTap v3, client-side HTML —
imported dynamically in `onMount` so it never evaluates during SSR). You can
replace it by passing a `bodyEditor` snippet receiving `EmailBodyEditorArgs`
(rendered inside the same dialog):

```ts
interface EmailBodyEditorArgs {
  body: string;                              // current rendered HTML, seed value
  editorJson?: Record<string, unknown>;      // opaque visual-editor source, for round-trip
  variables: string[];                       // known {{placeholder}} names → merge-tag quick-picks
  onChange: (next: { body: string; editorJson?: Record<string, unknown> }) => void;
}
```

The form stores the HTML on `email_notification.body` and the source on
`email_notification.editor_json`.

> If you use the built-in editor, SSR-bundle `@tiptap/*` + `svelte-tiptap`
> (vite `ssr.noExternal`) so the dynamic import resolves.

## Nav (`unhNav`)

```ts
import { unhNav } from "@mssfoobar/unh-web-sdk/nav";
```

`unhNav` is a plain nav descriptor you feed to your own sidebar/router
explicitly — the SDK does not register routes for you. Shape:

```ts
const unhNav: UnhNav = {
  code: "UNH",
  header: { name: "UNH", url: ROUTES.BASE },          // "/aoh/unh"
  sidebar: [
    { name: "Distribution Lists", icon, url: ROUTES.DISTRIBUTIONS },         // "/aoh/unh/distributions"
    { name: "Message Templates",  icon, url: ROUTES.TEMPLATES },             // "/aoh/unh/notification-templates"
    { name: "Channel Settings",   icon, url: ROUTES.SETTINGS },              // "/aoh/unh/settings"
  ],
};
```

Each `sidebar` entry is a `UnhNavSidebarEntry { name: string; url: string;
icon: Component }` — interpret `icon` however your sidebar renders it.
`ROUTES` exposes: `BASE`, `DISTRIBUTIONS`, `TEMPLATES`, `SETTINGS`.

## Compose your own edit pages

Most UNH edit surfaces offer two options: drop in a **whole component** (the
template form, the member picker, the params manager — fast), or **compose your
own layout** from the SDK's primitives + `@mssfoobar/ui`, taking the SDK only for
what's hard to rebuild or contract-bound. The plain channel and distribution-list
create-edit forms have **no whole-component option** — compose them from the
exported `validate*` / `initial*Values` helpers + the client (as the table below
shows). (The list/table pages are always host-composed — see the **Host-composed
lists** section below.) Everything in the "reuse" column is already a public export:

| Page | Reuse from the SDK | Build yourself (`@mssfoobar/ui` + `@mssfoobar/unh-client`) | Gotcha to honor |
|---|---|---|---|
| **Template** | `EmailBodyEditor` (`./email-body-editor`), `validateTemplate` | name, channel toggles, distribution/channel pickers (`Combobox`), layout | custom-channel save is a **flat** list (below); `distribution_list_id: "" ⇒ null` (the service allows no list; the built-in form requires one by default — `requireDistributionList`) |
| **Distribution list** | `DistributionMembers` (`./distribution-members` — the IAMS picker), `validateDistributionList`, `validateMember` | name field, layout, buttons | member `value` keyed per type (`user→id`, `role→name`, `group→id`) — the picker handles it; honor it if you wire `loadDirectory` |
| **Channel config** (email/push/custom) | `CustomChannelParameters` (`./custom-channel-parameters`), `validateEmailChannel`/`validatePushChannel`/`validateCustomChannel`, `parseServiceAccountKey`, `initialEmailValues`/`initialPushValues` | SMTP/name/endpoint fields, toggles | **secrets are write-only on edit** — the `initial*Values` helpers seed them blank; send a secret only when re-entered |

Reuse the `validate*` helpers so your pages enforce the same rules as the service
instead of re-deriving them.

### Template page

The custom-channel save shape is the one non-obvious bit — the read model groups
parameters *by channel*, but the create/update body wants a **flat** list across
*all* channels:

```ts
// create/update body — flatten every channel's params into one array
body.custom_notification = blocks.flatMap((b) =>
  paramsByChannel[b.channel_id].map((p) => ({
    parameter_id: p.id,
    parameter_value: b.values[p.id] ?? "",
  })),
);
body.distribution_list_id = distributionListId || null; // "" ⇒ null (no list)
```

Minimal wiring of the email body + shared validation:

```svelte
<script lang="ts">
  import { EmailBodyEditor } from "@mssfoobar/unh-web-sdk/email-body-editor";
  import { validateTemplate, hasErrors, GetUnhClient } from "@mssfoobar/unh-web-sdk";

  const client = GetUnhClient();
  let email = $state({ enabled: true, channel_id: "", subject: "", body: "", editor_json: undefined });

  async function save(name: string, distributionListId: string) {
    const errors = validateTemplate({ name, email, push: { enabled: false }, custom: { enabled: false } });
    if (hasErrors(errors)) return errors;
    await client.templates.create({
      name,
      distribution_list_id: distributionListId || null,
      email_notification: email.enabled
        ? { channel_id: email.channel_id, subject: email.subject, body: email.body }
        : undefined,
    });
  }
</script>

<EmailBodyEditor
  body={email.body}
  editorJson={email.editor_json}
  variables={[]}
  onChange={(n) => { email.body = n.body; email.editor_json = n.editorJson; }}
/>
```

`EmailBodyEditor` standalone carries the same tiptap `ssr.noExternal` requirement
noted above.

### Distribution-list page

The IAMS picker is the hard part — **reuse `DistributionMembers`**, don't rebuild
it (server directory + client filter + To/Cc/Bcc + the value-per-type contract).
It takes `listId` + your `loadDirectory` (see **The IAMS picker** above) and
manages members live (no save step). Build the name field + layout yourself;
validate with `validateDistributionList` (name) and `validateMember` (inline
external email/phone).

```svelte
<script lang="ts">
  import { DistributionMembers } from "@mssfoobar/unh-web-sdk/distribution-members";
  import { validateDistributionList, GetUnhClient } from "@mssfoobar/unh-web-sdk";
  import { iamsDirectory } from "$lib/unh/iams-directory"; // CLIENT DirectoryLoad — never import $lib/server into a component

  const client = GetUnhClient();
  // create/rename the list (name only) via client.distributionLists.*, then
  // manage its members live with the picker:
</script>

<DistributionMembers listId={list.id} loadDirectory={iamsDirectory} />
```

### Channel-config page

The channel forms are mostly typed inputs, so you can own them entirely. Reuse the
`validate*Channel` helpers, `parseServiceAccountKey` (validate the FCM
service-account JSON), and `CustomChannelParameters` for a custom channel's dynamic
param editor. **Secrets are write-only on edit** — seed the form with
`initialEmailValues` / `initialPushValues` (they blank the password /
service-account-key by design), and send a secret only when the user re-enters it.

```svelte
<script lang="ts">
  import { validateEmailChannel, initialEmailValues, hasErrors, GetUnhClient } from "@mssfoobar/unh-web-sdk";

  const client = GetUnhClient();
  let form = $state(initialEmailValues(channel)); // channel? ⇒ edit; password left blank
  async function save() {
    const errors = validateEmailChannel(form);
    if (hasErrors(errors)) return errors;
    // POST create / PUT update via client.emailChannels.*; on edit, omit password if blank
  }
</script>
```

## Host-composed lists (no list components)

The SDK ships **no** list/table component. Build each list page yourself
from the client's `.list()` + a `@mssfoobar/ui` `DataTable`, and delete via
`.remove()`.

The client exposes one `ResourceClient` per resource: `client.emailChannels`,
`client.pushChannels`, `client.customChannels` (+ `.params(channelId)`),
`client.distributionLists` (+ `.members(listId)`), and **`client.templates`** —
the notification-template resource (the accessor is named **`templates`**, NOT
`notificationTemplates`). Each has `.list(params)` / `.get(id)` / `.create(body)`
/ `.update(id, body)` / `.remove(id)`. **Only `.list()` is enveloped** — it
resolves to the aohhttp pagination envelope, so read `.data` for the rows. Every
other accessor resolves to the **DTO directly**: `.get(id)`, `.create()`, and
`.update()` return the resource (no `.data`), and `.remove()` returns `void`
(don't write `.get(id).data`). The row DTO types come from
`@mssfoobar/unh-web-sdk/types` — `EmailChannel`, `PushChannel`, `CustomChannel`,
`DistributionList`, `NotificationTemplate`, etc. (Import DTO types from the SDK's
`./types` subpath, **not** the bare `@mssfoobar/unh-types` — that's a transitive
dep and won't resolve by name from a consumer under pnpm.)

A distribution-lists page, for example — the same shape backs every list:

```svelte
<script lang="ts">
  import { onMount } from "svelte";
  import { DataTable, createCheckboxColumn, createActionsColumn,
           type AppColumnDef, type DataTableBulkAction } from "@mssfoobar/ui/table";
  import { toast } from "@mssfoobar/ui/toast";
  import { GetUnhClient } from "@mssfoobar/unh-web-sdk/provider";
  import type { DistributionList } from "@mssfoobar/unh-web-sdk/types";

  const client = GetUnhClient();
  let rows = $state<DistributionList[]>([]);
  let loading = $state(true);

  async function load() {
    loading = true;
    try {
      rows = (await client.distributionLists.list({ size: 100 })).data;   // envelope → .data
    } catch (e) {
      toast.error(e instanceof Error ? e.message : "Failed to load");
    } finally {
      loading = false;
    }
  }
  onMount(load);

  async function remove(l: DistributionList) {
    await client.distributionLists.remove(l.id);   // delete via the client
    load();
  }

  const columns: AppColumnDef<DistributionList>[] = [
    createCheckboxColumn<DistributionList>(),
    { accessorKey: "name", header: "Name", sortable: true },
    createActionsColumn<DistributionList>({
      onClickEdit: (row) => goto(resolve(`/aoh/unh/distributions/${row.id}`)), // resolve() from $app/paths — the base's no-navigation-without-resolve lint requires it
      onClickDelete: (row) => remove(row),
    }),
  ];
</script>

<DataTable {columns} data={rows} {loading}
  emptyState={{ icon: ContactIcon, title: "No Distribution Lists", content: "…" }} />
```

The **channel-settings** list is the same shape over `client.emailChannels.list()`
(etc.), and the **templates** list over `client.templates.list()`. The SDK's
*forms* fill the create/edit routes; your app owns the list + detail routing
around them.

## Server (BFF proxy)

| Export | Subpath | Role |
|---|---|---|
| `createUnhProxy(cfg)` | `/server` | Factory returning SvelteKit-shaped `{ GET, POST, PUT, PATCH, DELETE }` handlers. ONE transparent catch-all proxy: strips the BFF base prefix, forwards path + query + caller bearer + a correlation id to `unh-service`, passes the response (aohhttp success/pagination AND error envelopes) back verbatim with its status. |

`UnhProxyConfig`: `baseUrl: string` (e.g. `env.UNH_URL`), `getToken:
(event) => string | undefined | Promise<…>` (typically
`event.locals.authResult?.access_token`), `bffBase?: string` (default
`"/aoh/unh/api"`), `fetch?` (test override). Mount it once:

```ts
// src/routes/aoh/unh/api/[...path]/+server.ts
import { createUnhProxy } from "@mssfoobar/unh-web-sdk/server";
import { env } from "$env/dynamic/private";

export const { GET, POST, PUT, PATCH, DELETE } = createUnhProxy({
  baseUrl: env.UNH_URL ?? "",   // $env/dynamic/private is string|undefined
  getToken: (event) => event.locals.authResult?.access_token,
});
```

The same `@mssfoobar/unh-client` serves the browser (pointed at the BFF
base), so the proxy needs no second client — it is a generic pass-through.
Authn/authz gating stays your app's concern.

## Import-style cheat sheet

```ts
// Barrel (everything):
import { UnhProvider, GetUnhClient, unhNav, createUnhClient } from "@mssfoobar/unh-web-sdk";

// Subpaths (granular):
import UnhProvider, { GetUnhClient } from "@mssfoobar/unh-web-sdk/provider";
import { unhNav } from "@mssfoobar/unh-web-sdk/nav";
import DistributionMembers from "@mssfoobar/unh-web-sdk/distribution-members";
import NotificationTemplateForm from "@mssfoobar/unh-web-sdk/notification-template-form";
import SendTemplateDialog from "@mssfoobar/unh-web-sdk/send-template-dialog";
import { createUnhProxy } from "@mssfoobar/unh-web-sdk/server";   // server-only
import type { DistributionList, EmailBodyEditorArgs } from "@mssfoobar/unh-web-sdk/types";
```
