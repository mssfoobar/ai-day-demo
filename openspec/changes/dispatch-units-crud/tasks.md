## 1. Implement `dispatch-svc`

- [x] 1.1 Consult the `aoh-conventions` skill (Go, API and DB conventions) and `aoh-error-handling` before implementing writes.
- [x] 1.2 `internal/domain`: add `OccLock int` to `Unit` (`json:"occ_lock"`) and a `UnitInput` struct for the writable fields.
- [x] 1.3 `internal/repo`: `Create`, `Update(code, occLock, input)`, `Delete(code, occLock)`. Map `23505` → `ErrConflict`, `23514` → `ErrInvalid`, `0 rows` on guarded update/delete → `ErrStale` (design.md D5). Bump `last_contact` and `occ_lock` on update; set `created_by`/`updated_by` to the placeholder.
- [x] 1.4 `internal/service`: validate required fields and status vocabulary into `aoherr` validation errors with `FieldDetail`s; classify repo errors into `DISPATCH_UNIT_INVALID` / `DISPATCH_UNIT_CODE_TAKEN` / `DISPATCH_UNIT_STALE` / `DISPATCH_UNIT_WRITE_FAILED`; declare the codes in `errors.go`.
- [x] 1.5 `internal/handler`: `POST /v1/units` (201), `PUT /v1/units/{unit_code}` (200), `DELETE /v1/units/{unit_code}?occ_lock` (204). Decode JSON strictly; render errors with `aoherr.Render`.
- [x] 1.6 Tests: service validation (blank field, bad status), conflict / stale classification; handler status codes for 201/200/204/400/404/409, 405 for `PATCH` and collection `PUT`/`DELETE`, `occ_lock` present on read.
- [x] 1.7 Verify: `cd apps/dispatch-svc && go build ./... && go vet ./... && go test ./... -count=1` exit 0.

## 2. Implement `dispatch-web`

- [x] 2.1 Consult the `aoh-conventions` skill (web) and `aoh-design` before building the forms. Use `Sheet`, `Label`, `Input`, `Select`, `AlertDialog`, `toast` from `@mssfoobar/ui` by subpath; field errors as a sibling `<p class="text-destructive text-xs">` (no `FormMessage` exists).
- [x] 2.2 `units.server.ts`: add `createUnit`, `updateUnit`, `deleteUnit`; decode the AOH error payload into a `DispatchServiceError` carrying `status`, `errorCode` and field details; map `occ_lock` on read.
- [x] 2.3 `types.ts`: add `occLock` to `FieldUnit`; add `UnitInput`.
- [x] 2.4 A pure `parseUnitForm(FormData)` in `src/lib/aoh/dispatch/forms.ts` returning `{ input } | { errors }`, unit-tested (blank required, bad status, capabilities split on commas).
- [x] 2.5 `+page.server.ts`: `create` / `update` / `delete` actions using `parseUnitForm` and the client; `fail(400/409, …)` with field errors or a conflict message; never forward raw service text.
- [x] 2.6 `components/UnitForm.svelte`: the `Sheet` form used for add and edit, showing field errors from the action result.
- [x] 2.7 Wire the page: *Add unit* button in the header; *Edit* and *Delete* in the detail header; `AlertDialog` for delete; `use:enhance` with toasts; preserve selection across reload, clear on delete. Mount `<Toaster />` in `src/routes/+layout.svelte`.
- [x] 2.8 Verify: `cd apps/dispatch-web && pnpm build && pnpm check && pnpm lint && pnpm test:unit` exit 0.

## 3. End-to-end verification

- [x] 3.1 Persistence smoke against the running stack (`pnpm start`):
  ```bash
  curl -s -X POST localhost:8081/v1/units -H 'content-type: application/json' -d '{...FU-401...}'   # 201
  curl -s -X PUT  localhost:8081/v1/units/FU-401 -d '{...,"occ_lock":0}'                            # 200, occ_lock 1
  curl -s -X PUT  localhost:8081/v1/units/FU-401 -d '{...,"occ_lock":0}'                            # 409 stale
  # restart the service, then:
  curl -s localhost:8081/v1/units/FU-401                                                          # still there
  curl -s -X DELETE 'localhost:8081/v1/units/FU-401?occ_lock=1' -o /dev/null -w '%{http_code}'      # 204
  ```
- [ ] 3.2 Console check: add a unit via the form, edit its status, delete it — each reflected in the list and confirmed by a toast; a blank call sign shows a field error. (Manual — no E2E suite by decision.)
- [ ] 3.3 **Reproducibility gate**: `podman compose down -v` (or `docker compose down -v`), `up -d postgres`, restart both apps, re-run 3.1. Seed converges to 5 units; the created unit is gone with the volume, as it should be.
