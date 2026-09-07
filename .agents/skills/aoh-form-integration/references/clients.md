# Form SDK clients

Three client classes, one constructor, one return envelope. Package:
**`@mssfoobar/form-client`** — `.` = `FormClient`, `./submitter` =
`FormSubmitterClient`, `./amm` = `AmmClient`. The wire DTOs live in
**`@mssfoobar/form-types`** (and are re-exported from `form-client` for convenience).

## Key wire DTOs (`@mssfoobar/form-types`)

The load-bearing shapes a list/detail page renders:

```ts
type Form = {
  id: string; name: string;
  created_at: string; created_by: string; updated_at: string; updated_by: string;
  published_version_id: string | null; tenant_id: string;
};
type Submission = {
  id: string; form_name: string; form_version_name: string;
  form_id: string; form_version_id: string;
  current_state: string; state_id: string;
  created_at: string; created_by: string; updated_at: string; updated_by: string;
  data: SubmissionData; deleted_at?: string | null; deleted_by?: string | null;
};
// plus FormVersion, FormDraft, FormTheme, FormTable, Changelog, Attachment, and the *Params/*Response types.
```

Timestamps are ISO strings. So an admin list can show `name` + `created_at` +
`updated_at`; a submissions table can show `form_name` + `current_state` + `updated_at`.

## Constructor — `FormClientConfig`

All three clients (`FormClient`, `FormSubmitterClient`, `AmmClient`) extend the
same core and take the same config (DTOs in `@mssfoobar/form-types`):

```ts
interface FormClientConfig {
  module: string;          // gateway module code, e.g. "form" or "amm"
  gatewayURL?: string;     // default "/aoh/gateway"
  timeout?: number;        // default 30_000 ms
  fetch?: typeof fetch;    // pass event.fetch in SSR load(); defaults to global fetch
}
```

**URL model.** Every request is `{gatewayURL}/{module}/{apiVersion}/{path}`. So
`new FormClient({ module: "form" })` → `GET /aoh/gateway/form/v1/forms`, and
`new AmmClient({ module: "amm" })` → `/aoh/gateway/amm/…`. `theme` requests also
go through `module: "form"` (the gateway routes `theme` to `FORM_URL` too). The
`gatewayURL` is the host's **own** BFF route, not the backend (see `bff.md`).

The version segment is chosen per module — `v1` for `form` and `theme`, none for
`amm`, which is a different service and serves no version prefix. form-service
still answers the unversioned paths but marks them `Deprecation: true` and removes
them in v2, so the client addresses `/v1` and a host needs to do nothing. Override
with `apiVersion` to pin a version, or `apiVersion: ""` to address the deprecated
paths deliberately.

**SSR.** In a server `load`, pass `fetch`: `new FormClient({ module: "form", fetch })`.
Without it, relative URLs don't resolve and cookies aren't forwarded.

## The `Result<T>` envelope — errors are values, not throws

```ts
export class FormClientApiError extends Error {
  status: number;          // HTTP status
  code: string;            // machine code
  displayMessage: string;  // user-friendly, status-derived
}
export type Result<T, E = FormClientApiError> =
  | ({ ok: true } & T)     // success: spread payload (e.g. res.data) alongside ok
  | { ok: false; error: E };
```

Always branch on `.ok` before reading the payload:

```ts
const res = await client.listForms({ page_size: 10 });
if (!res.ok) { toast.error(res.error.displayMessage); return; }
forms = res.data;
```

## `FormClient` (`.`) — full admin surface

`FormClient` has **every** method below (admin + read + submission + theme). Use it
for admin/authoring pages.

**Forms / drafts / versions**:

```ts
listForms(params?): Result<ListFormsResponse>
createForm(params): Result<CreateFormResponse>
updateForm(params): Result<UpdateFormResponse>
deleteForm(params): Result<void>
getFormTable(params): Result<GetFormTableResponse>           // table schema for validation
getFormDrafts(params): Result<GetFormDraftsResponse>
getFormDraft(params): Result<GetFormDraftByIdResponse>
createFormDraft(params): Result<CreateFormDraftResponse>     // { form_id, name, form_json, theme_json }
updateFormDraft(params): Result<void>                        // autosave target
deleteFormDraft(params) / adminDeleteFormDraft(params): Result<void>
duplicateFormDraft(params) / duplicateFormVersion(params): Result<…Response>
getFormVersions(params): Result<GetFormVersionsResponse>
createFormVersion(params): Result<CreateFormVersionResponse> // { form_id, draft_id, name } — publish
updateFormVersion(params): Result<void>
unpublishFormVersion(params): Result<void>
```

**Themes** (FormClient only):

```ts
listThemes(params?) / getTheme(params) / createTheme(params): Result<…>
updateTheme(params) / deleteTheme(params): Result<void>
```

**Read + submission** (shared with `FormSubmitterClient`, see below), plus the
**admin-only** submission reads:

```ts
adminListSubmissions(params?): Result<AdminListSubmissionsResponse>
adminGetSubmission(params): Result<GetSubmissionResponse>
```

## `FormSubmitterClient` (`./submitter`) — browser-safe read + submit

The read + submission subset — **no** admin form/theme methods, so it's safe to
construct in end-user fill pages. Methods:

**Read**:

```ts
listPublishedForms(params?): Result<ListPublishedFormsResponse>
getForm(params): Result<GetFormByIdResponse>
getFormVersion(params): Result<GetFormVersionByIdResponse>
```

**Submission**:

```ts
createSubmission(params): Result<CreateSubmissionResponse>
getSubmission(params) / listSubmissions(params?): Result<…>
deleteSubmission(params): Result<DeleteSubmissionResponse>
upsertSubmissionData(params): Result<UpsertSubmissionDataResponse>   // commit answers (wfe uses this)
saveSubmissionDraft(params) / getSubmissionDraft(params) / deleteSubmissionDraft(params): Result<…>
transitionSubmissionState(params): Result<TransitionSubmissionStateResponse>  // state-machine move
listDistinctSubmissionUsers(): Result<ListDistinctSubmissionUsersResponse>
```

> You rarely call the submission methods directly on a fill page — the renderer
> flows (`newSubmission`/`editSubmission`, `renderer.md`) call them for you. Call
> them directly for custom flows (e.g. wfe's `upsertSubmissionData`).

## `AmmClient` (`./amm`) — attachments

`module: "amm"`:

```ts
uploadFiles(formData: FormData): Result<UploadFilesResponse>
generateDownloadId(params): Result<GenerateDownloadIdResponse>
downloadFile(params) / downloadPreview(params) / downloadFileWithId(params): Result<{ response: Response }>
deleteAttachment(params): Result<void>
```

Use a generous `timeout` for uploads (`new AmmClient({ module: "amm", timeout: 300_000 })`).
You usually hand the `AmmClient` to a renderer flow rather than calling it directly;
the renderer registers its upload/download/delete handlers on the form
(`setupFileHandlersAndData`, see `renderer.md`).

> **Per-request instance on `event.locals`.** A common host pattern is to construct
> one `AmmClient` per request in `hooks.server.ts`
> (`event.locals.ammClient = new AmmClient({ module: "amm", fetch: event.fetch, timeout: 300_000 })`)
> and declare `ammClient?: AmmClient` on `App.Locals` (import the type from
> `@mssfoobar/form-client/amm`). Binding to `event.fetch` keeps gateway routing
> SSR-correct.
