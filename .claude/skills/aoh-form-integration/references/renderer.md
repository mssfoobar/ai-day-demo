# Form renderer & SurveyJS Creator

Everything under `@mssfoobar/form-web-sdk/renderer` — framework-agnostic TS plus a
compiled-Svelte authoring UI. Always `import
"@mssfoobar/form-web-sdk/renderer/styles.css"` on pages that mount any of this, and
`import "survey-js-ui"` on pages that render (not author) a form.

> **Browser-only.** Everything here renders into a DOM node (`.render(el)`).
> Pages that mount it set `export const ssr = false` and mount in `onMount`. To keep
> a host SSR-able instead (e.g. when embedding the renderer inside another SSR page),
> dynamic-`import()` the renderer inside `onMount`.

## High-level submission flows (preferred)

These fetch the published version, build the model, register AMM file handlers,
and apply state-based permissions for you. All take a `formClient`
(`FormSubmitterClient` or `FormClient`, from `@mssfoobar/form-client`) + an
`ammClient` (`@mssfoobar/form-client/amm`) — the **same** `AmmClient` type these
flows type their param from, so it passes straight through with **no cast**.

### `newSubmission(options): Promise<NewSubmissionResult>`

Creates a submission on the backend, returns `{ model, save }`.

```ts
const ns = await newSubmission({
  formId, userRoles,           // userRoles drives state-permission evaluation
  formClient, ammClient,
  onComplete: (data) => { /* survey "Complete" pressed */ },
  onSave?: () => void,
});
ns.model.render(el);           // FormModel
// ns.save()  → save a draft (wire to a "Save Draft" button)
```

### `editSubmission(options): Promise<EditSubmissionResult>`

Loads an existing submission + any draft, returns
`{ model, save, useSubmissionData, useDraftData }`. Pick which dataset to seed
**before** `render`:

```ts
const r = await editSubmission({ submissionId, userRoles, formClient, ammClient, onComplete });
r.useSubmissionData();   // committed answers; protects already-committed files from deletion
// or r.useDraftData();  // resume the unsaved draft
r.model.render(el);
```

### `viewSubmission(options): Promise<FormModel>`

Read-only; returns the `FormModel` directly. `{ submissionId, userRoles, formClient, ammClient }`.

## Low-level: `createFormModel` + `setupFileHandlersAndData`

When you already have the form JSON (e.g. a WFE human-task step fetched the version
itself) and want the model without the flow plumbing:

```ts
import { createFormModel, setupFileHandlersAndData } from "@mssfoobar/form-web-sdk/renderer";

const model = createFormModel({
  formJson,                    // SurveyJS definition (Record<string, unknown>)
  themeJson,                   // SurveyJS theme
  context: { currentState: "Draft", userRoles: ["submitter"] },  // RenderContext → permissions
  initialData,                 // pre-fill (optional)
  editable: true,              // default false
  onComplete: (data) => {},
  onValueChanged, onValidationFailed,   // optional
});

// REGISTER HANDLERS BEFORE SETTING DATA — else initial attachments don't load:
setupFileHandlersAndData(model, { ammClient, initialData });   // (+ submissionData? to protect committed files)
model.render(el);
```

`FormModel`: `.data` (get/set the answers), `.render(container)`,
`.setValue(name, value)` (write one field into form state — validation, `visibleIf`/
`enableIf`, `.data`, and `onValueChanged` all update, so it persists on submit),
`.onAfterRenderQuestion((name, htmlElement) => …)` (run a callback after each question
renders — the seam for host-driven controls, see below), and `_getSurveyModel()`
(`@internal`). To re-apply permissions after a state change without rebuilding, call
`updateFormContext(model, newContext)`. (`.setValue` / `.onAfterRenderQuestion` require
`@mssfoobar/form-web-sdk` ≥ 1.1.0.)

## Permissions

```ts
import { applyStatePermissions, resetQuestionPermissions,
         evaluateQuestionPermission, getDefaultState } from "@mssfoobar/form-web-sdk/renderer";
```

Permissions are evaluated **once at model build** from `context` (current state +
user roles). The flows above apply them for you; call `updateFormContext` when the
workflow transitions the submission to a new state. `getDefaultState` picks the
initial state for a fresh submission.

## Host-driven field population (button in an `html` question → host modal → `setValue`)

The renderer only draws standard question types. Rich, app-specific interactions —
a button that opens a **host** modal / data-picker and writes the chosen value back
into a field — are the host's job. The form just declares intent via an `html`
question; the host wires the DOM event with `onAfterRenderQuestion` and writes the
result back with `setValue`. This is how reference-host's submission fill page does
it (`apps/reference-host/src/routes/aoh/form/edit/[id]/+page.svelte`).

**1. Author declares intent in the form JSON.** An `html` question renders the
trigger (label left, button right via flexbox); the value lands in a sibling
question whose title is hidden:

```json
{
  "type": "panel",
  "name": "panel1",
  "elements": [
    {
      "type": "html",
      "name": "runway_picker",
      "html": "<div style=\"display:flex;justify-content:space-between;align-items:center;\"><p style=\"margin:0\"><b>Runway affected</b></p><button type=\"button\" class=\"survey-question\" data-question-id=\"runway_affected\" data-picker=\"runway\">+ Pick</button></div>"
    },
    { "type": "comment", "name": "runway_affected", "titleLocation": "hidden" }
  ]
}
```

The `class` / `data-*` are your own convention (SurveyJS's sanitizer keeps
`<button>`, `class`, and `data-*`, but **strips inline `on*` handlers** — never wire
via `onclick`). `data-question-id` = the question to populate; `data-picker` = which
host dialog to open.

**2. Host wires the buttons once, before `render()`.** One generic handler serves
every button on every form; a registry maps `data-picker` → your dialog config:

```ts
const PICKERS: Record<string, { title: string; options: string[] }> = {
  runway:    { title: "Runway / location", options: ["Runway 02L", "Runway 20R", "Taxiway B"] },
  personnel: { title: "Duty manager",      options: ["Alex Tan", "Priya Menon"] },
};
let pickerOpen = $state(false), target = $state(""), kind = $state("");

const { model } = await editSubmission({ submissionId, userRoles, formClient, ammClient });

// Register BEFORE render() so the first paint is covered.
model.onAfterRenderQuestion((_name, el) => {
  el.querySelectorAll<HTMLElement>(".survey-question[data-question-id]").forEach((btn) => {
    if (btn.dataset.wired) return;        // handler can re-fire → guard against stacking listeners
    btn.dataset.wired = "1";
    btn.addEventListener("click", () => {
      target = btn.dataset.questionId ?? "";
      kind = btn.dataset.picker ?? "";
      pickerOpen = true;                  // open YOUR modal component (e.g. @mssfoobar/ui Dialog)
    });
  });
});
model.render(container);
```

**3. On the host modal's confirm, write back** with `setValue` — SurveyJS state
updates, so the value shows in the (hidden-title) field and persists on submit:

```ts
function choose(value: string) {
  model.setValue(target, value);
  pickerOpen = false;
}
```

Notes:

- Register `onAfterRenderQuestion` **before** `model.render()`.
- The handler re-fires as questions (re)render — the `data-wired` guard prevents
  duplicate listeners and also auto-wires buttons in newly-added dynamic-panel rows.
- The host owns the modal entirely; the SDK only surfaces the click
  (`onAfterRenderQuestion`) and the write-back (`setValue`) — it never learns about
  "buttons" or "modals". The element isn't limited to a button; any element works.
- ⚠️ Dynamic panels: a static `data-question-id` is identical across repeated rows,
  so per-row targeting needs the question instance (which this hook doesn't pass).
  Flat forms are fully covered.

## Authoring — `createSurveyCreator`

```ts
import {
  createSurveyCreator,
  customDesignerLightTheme, customDesignerDarkTheme,   // Creator CHROME themes
  defaultLight, defaultDark,                           // designed-FORM surface themes
} from "@mssfoobar/form-web-sdk/renderer";

// Browser-only: build + render in onMount, theme BOTH layers, dispose on teardown.
onMount(() => {
  const creator = createSurveyCreator({ isEditMode: true });   // + customThemes?, tableSchema?

  const isDark = document.documentElement.classList.contains("dark");
  creator.applyCreatorTheme(isDark ? customDesignerDarkTheme : customDesignerLightTheme);
  creator.theme = isDark ? defaultDark : defaultLight;   // else the designed form renders white in dark mode

  creator.JSON = existingFormJson;   // load a draft/version
  creator.render(el);                // `el` needs a sized, bounded container, e.g. class="min-h-0 flex-1"
  return () => creator.dispose();    // REQUIRED — leaks the Creator + its mounted editors on navigation otherwise
});
// read back creator.JSON / creator.theme to persist
```

`createSurveyCreator` configures the AOH customizations — the form-states property
editor, per-question permissions editor, a preview tab with a state/role selector,
a theme save dialog, and an image-upload handler. Those editor components are
mounted/unmounted **internally** (Svelte `mount`/`unmount`); you only touch the
returned creator + `.JSON` / `.theme`. Two kinds of theme export: the
`customDesigner{Light,Dark}Theme` presets style the **Creator chrome** (passed to
`applyCreatorTheme`), while `default{Light,Dark}` (+ the `…Panelless` variants) are
the designed-**form surface** presets (assigned to `creator.theme`, or offered in
`customThemes`).

### Date/time display formats (author-set, not locale-driven)

`date`, `time` and `datetime-local` questions do **not** render the native browser
input — the SDK substitutes its own field and picker so the displayed format comes
from the schema rather than the operator's OS locale. Three properties, registered
by the SDK, control it:

| Property | Choices | Default |
|---|---|---|
| `dateFormat` | `dd/mm/yyyy`, `dd-mm-yyyy`, `mm/dd/yyyy`, `mm-dd-yyyy`, `yyyy-mm-dd`, `yyyy/mm/dd` | `dd/mm/yyyy` |
| `timeFormat` | `24h`, `24h:ss`, `12h`, `12h:ss` (`:ss` adds a seconds column) | `24h` |
| `timezone` | `local`, `utc` — offered on `datetime-local` only | `local` |

Each is settable form-wide (Form properties → General), per question, and per
matrix column, where the default *"Use form setting"* inherits the form's choice.
Nothing to wire in the host: `createSurveyCreator` and the renderer register and
honour them. They carry defaults, so every date/time question — including one in a
matrix cell — renders through the picker.

`timezone` is datetime-only because a zone can only mean something for a value
that is an instant. The picker performs no validation of its own: a question's
`min`/`max` grey out unreachable days in the calendar, but times and typing stay
unrestricted, leaving the form's `checkErrorsMode` to decide when errors surface.

**Answer shapes a host reading `model.data` should expect:**

| Question | Value |
|---|---|
| `date` | `2026-07-01` |
| `time` | `18:30:00` — seconds always present, shown only when `timeFormat` ends `:ss` |
| `datetime-local` | `2026-07-01T10:30:00Z` — an instant **with an offset** |

The datetime shape changed in `@mssfoobar/form-web-sdk` 1.2.0 / form-service 1.1.0:
it was previously an offset-less `2026-07-01T18:30`. A host that parses these
values itself must treat them as instants, not wall clocks.

### Publish flow (form → draft → version)

```ts
const f = await client.createForm({ name });                       // FormClient (admin); check each .ok
const d = await client.createFormDraft({
  form_id: f.data.id, name,
  form_json: creator.JSON,
  theme_json: creator.theme as unknown as Record<string, unknown>, // creator.theme is ITheme
});
const v = await client.createFormVersion({ form_id: f.data.id, draft_id: d.data.id, name: "v1" });
// createFormVersion also takes an optional `description?: string`.
```

Editing an existing draft: server-`load` `getForm`, `getFormDraft`/`getFormVersion`,
`listThemes` and `getFormTable`; set `creator.JSON = resource.form_json` on mount,
and autosave via `updateFormDraft({ form_id, draft_id, form_json: creator.JSON, theme_json: creator.theme })`.

> **Preview tab uploads are NOT AMM.** In the Creator's preview tab, file
> questions use `storeDataAsText = true` (base64 data URLs) — they do **not** hit
> AMM. AMM only handles uploads in real fill pages (the renderer flows). So a file
> "working" in preview doesn't prove the AMM/gateway wiring; test on a real fill/edit
> page (`/aoh/form/edit/[id]`).
