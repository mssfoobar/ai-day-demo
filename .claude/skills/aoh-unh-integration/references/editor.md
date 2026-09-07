# The email body editor & its override seam

`<NotificationTemplateForm>` needs a rich editor for the email `body`. The SDK
ships one built in, but exposes a snippet seam so you can swap in your own. Read
this before embedding the form — especially before building a visual email
builder to replace the default.

## The built-in editor

`<EmailBodyEditor>` is a Svelte + TipTap-v3 rich-text editor (based on the
MIT-licensed Edra), trimmed to an **email-safe command set** and skinned on
`@mssfoobar/ui`. `<NotificationTemplateForm>` uses it as the **default** editor;
you interact with it only through the form, or replace it via the `bodyEditor`
seam below.

The form wires both directions of the round-trip:

```svelte
{#if bodyEditor}
  {@render bodyEditor({
    body: email.body,
    editorJson: email.editor_json,
    variables,
    onChange: (next) => { email.body = next.body; email.editor_json = next.editorJson; },
  })}
{:else}
  <!-- Default: the SDK's built-in rich-text editor. -->
  <EmailBodyEditor
    body={email.body}
    editorJson={email.editor_json}
    {variables}
    onChange={(next) => { email.body = next.body; email.editor_json = next.editorJson; }}
  />
{/if}
```

### The `body` / `editor_json` round-trip

Two artefacts travel together for an email payload:

- **`body`** — rendered HTML. This is what `unh-service` **sends** and what the
  `{{var}}` binder substitutes into. It is the source of truth for delivery.
- **`editor_json`** — opaque TipTap-document JSON (`{ type: 'doc', … }`),
  persisted alongside the template. It exists only so the editor can
  **re-hydrate** the exact authoring state next time the template is opened.

On every (debounced) edit the built-in editor reports both via `onChange` — and
it passes its HTML through an **email-safe serialization step first**, so the
stored `body` is already inbox-ready (see *Email-safety* below). The form
persists `editor_json` only when it's present.

**Content guard.** When seeding, the editor prefers the saved TipTap doc but
**falls back to the HTML `body`** when `editor_json` isn't a real TipTap document
(it checks `doc.type === "doc"`). So a template authored before `editor_json`
existed — or by a different `bodyEditor` override — still opens cleanly: it
renders from `body` rather than choking on a foreign source. **`body` is the
durable contract; `editor_json` is best-effort hydration.**

## Email-safety

A rich-text editor emits HTML for a *browser*; email clients (Gmail especially)
strip `<style>` blocks and classes and reject much modern CSS, so raw editor HTML
renders broken in the inbox. The built-in editor + `unh-service` handle this in
three layers — using the built-in editor, you get all three for free:

1. **Command curation** — the editor only exposes commands whose output email
   can carry (the kept/dropped set below).
2. **Serialization** — on save, the built-in editor **inlines** the styling that
   otherwise lived in stripped classes (lists, blockquote, tables + cell borders,
   inline code, `hr`), resolves the image's percentage `width` to pixels +
   `max-width:100%`, swaps the deprecated `align` attribute for block margins,
   and converts `rem` font sizes to `px`. **This is what makes the kept-but-rich
   elements (tables, lists, images, font size) actually render — command curation
   alone does not.**
3. **Envelope** — `unh-service` sends the stored `body` as `multipart/alternative`
   (a generated `text/plain` part for deliverability + text-only clients) with the
   HTML wrapped in a minimal email document (charset + viewport meta, a centred
   600px container).

Layer 2 runs **only inside the built-in `<EmailBodyEditor>`**. A custom
`bodyEditor` override does **not** get it (see the seam below).

## The email-safe command set (kept vs dropped)

Layer 1 of the pipeline — the editor is trimmed to what mail clients render
reliably (with layer 2 inlining the styling):

**Kept** (layer 2 inlines their styling): headings (H1–H4), paragraph, bold /
italic / underline / strikethrough, super/subscript, blockquote, text colour +
highlight, text alignment, bullet + ordered lists, horizontal rule, links (URL
prompt; `target=_blank rel=noopener`), **tables**, and **URL-only images**.

**Dropped** and **why**:

- **Code blocks** — email has no syntax highlighting; monospaced code blocks are
  dev-tool chrome, not message content.
- **Video / audio / iframe / embeds** — `<iframe>` / `<video>` are stripped by
  every mail client; they never render in an inbox.
- **Math, task-lists** — niche editor features with no email rendering story.
- **File-upload / paste-to-embed images** — the image insert path is
  **URL-only** (a `type="url"` input). A hosted `https://` image renders in mail
  clients, whereas an uploaded/pasted file becomes a base64 data-URI that
  **Gmail and Outlook strip** — so the editor refuses the base64 path entirely.
  Use publicly-reachable `https://` image URLs.

## The override seam — `bodyEditor`

`<NotificationTemplateForm>` accepts an optional `bodyEditor` snippet prop:

```ts
bodyEditor?: Snippet<[EmailBodyEditorArgs]>;
```

When supplied it **replaces** the built-in editor; when omitted the form uses
`<EmailBodyEditor>`. The snippet receives `EmailBodyEditorArgs` (a barrel export
of `@mssfoobar/unh-web-sdk`):

```ts
export interface EmailBodyEditorArgs {
  /** Current rendered HTML body — the initial value to seed the editor with. */
  body: string;
  /** Current visual-editor source, if any — initial value to re-hydrate. */
  editorJson?: Record<string, unknown>;
  /** Known `{{placeholder}}` names to offer as merge tags / quick-picks. */
  variables: string[];
  /** Report an edit: new rendered HTML body and (optional) editor source. */
  onChange: (next: { body: string; editorJson?: Record<string, unknown> }) => void;
}
```

The contract for an override: seed from `body` (or re-hydrate from `editorJson`
if you understand it), and on every edit call `onChange` with the **rendered
HTML** `body` (the binder needs HTML) plus, optionally, your own opaque source.
If you don't produce a source, omit `editorJson` — the content guard means a
later open of your template just renders from `body`.

### When you'd override

The built-in editor is a *rich-text* editor (a document with formatting). Reach
for an override when you need a **visual email builder** — drag-and-drop blocks,
columns, branded templates, MJML-style layouts — a fundamentally different
authoring model than free-form rich text.

```svelte
<script lang="ts">
  import NotificationTemplateForm from "@mssfoobar/unh-web-sdk/notification-template-form";
  import MyVisualEmailBuilder from "$lib/MyVisualEmailBuilder.svelte";
</script>

<NotificationTemplateForm variables={["recipient.name", "incident.id"]}>
  {#snippet bodyEditor({ body, editorJson, variables, onChange })}
    <MyVisualEmailBuilder
      initialHtml={body}
      initialSource={editorJson}
      mergeTags={variables}
      onChange={(html, source) => onChange({ body: html, editorJson: source })}
    />
  {/snippet}
</NotificationTemplateForm>
```

`onChange` must report the **rendered HTML** as `body` regardless of your
builder's internal representation — that HTML is what sends and what binds
`{{vars}}`.

**Your override owns email-safety.** The email-safe serialization (layer 2 above)
runs only inside the built-in `<EmailBodyEditor>`; HTML you report from a custom
`bodyEditor` is stored verbatim. `unh-service` still wraps it in the email
document + `text/plain` part (layer 3), but it does **not** inline element styles
for you — so emit inbox-safe HTML directly (inline styles, pixel image widths,
table `border` / `cellpadding` attributes), not class- or `<style>`-dependent
markup.

## Rendering a saved template (read-only preview)

To display a stored `body` (e.g. a review or preview screen), render it in a
**sandboxed iframe**, never with `{@html}`:

```svelte
<iframe title="Email body preview" sandbox="" srcdoc={template.email_notification.body}></iframe>
```

`sandbox=""` (maximal restriction) blocks scripts, so stored template HTML can't
execute in your admin UI. `{{placeholder}}` tokens render **literally** here —
they're bound at send, not at preview.

## Gotchas

- **The `@source` Tailwind import is REQUIRED or the toolbar breaks.** The
  editor's rarer utility classes (dropdown-chevron sizing, the colour-swatch
  grid) only exist if the SDK's compiled output is in Tailwind's content scan.
  Tailwind v4 excludes `node_modules`, so without the SDK's `@source` bridge those
  classes are never generated and the toolbar fails *partially* — chevrons balloon
  to the icon default, the colour grid collapses to one column. Chain
  `@import "@mssfoobar/unh-web-sdk/styles/app.css"` into your single Tailwind
  entry (**SKILL Step 2**).
- **`ssr.noExternal` must include `@tiptap`.** The built-in editor pulls
  `@tiptap/*` + `svelte-tiptap`; if `vite.config.ts`'s `ssr.noExternal` doesn't
  bundle them the editor's SSR/bundle step fails. Use
  `[/^@mssfoobar\//, /^@tiptap\//, "svelte-tiptap"]` (**SKILL Step 1**).
- **SSR-safe by design.** The editor is dynamically `import()`-ed inside
  `onMount`, so it never evaluates during SSR (its dep graph doesn't resolve under
  pnpm-strict Node SSR). Your page still SSRs normally — you do **not** set
  `ssr = false`. Until the dynamic import resolves, the component shows a
  "Loading editor…" placeholder.
- **Insecure dev origins.** `crypto.randomUUID` is secure-context-only; an
  insecure dev origin (e.g. plain `http://`) makes TipTap's id minting throw.
  `<EmailBodyEditor>` polyfills `randomUUID` from `getRandomValues` in that case —
  a no-op under HTTPS in production.
