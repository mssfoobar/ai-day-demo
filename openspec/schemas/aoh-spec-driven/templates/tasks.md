## 1. Scaffold

- [ ] 1.1 Consult the `aoh-conventions` skill for AOH coding conventions before implementing.
- [ ] 1.2 Scaffold ... using the `<aoh-go-init | aoh-web-init>` skill.
- [ ] 1.3 Verify: <concrete command, e.g. `go build ./...` or `pnpm build`>

## 2. Compose runtime dependencies

- [ ] 2.1 Compose ... using the `aoh-compose` skill.
- [ ] 2.2 Verify: `podman compose up -d <svc>` (or `docker compose up -d <svc>`) followed by `podman compose ps` (or `docker compose ps`) shows `<svc>` healthy.

## 3. Seed / resource creation

- [ ] 3.1 Consult the `aoh-knowledge` skill (`references/services/<svc>.md`) before editing seed artifacts for that service.
- [ ] 3.2 Edit `<path/to/seed/artifact>` to add `<role | bucket | topic | channel | template | row>`. The artifact MUST be idempotent (re-applied on every stack-up without divergence).
- [ ] 3.3 Verify (state present) with a concrete command — e.g. token decode shows the role in `active_tenant.roles`; `mc ls` shows the bucket; `curl /admin/...` shows the template; `psql -c 'SELECT count(*) FROM <table>'` shows the seeded row count.

## 4. Implement `<backend-app-name>`

- [ ] 4.1 Consult the `aoh-conventions` skill (Go conventions, API conventions, DB schema conventions) before implementing.
- [ ] 4.2 <DB migration / repo / service / handler / middleware task…>
- [ ] 4.3 <Tests…>
- [ ] 4.4 Verify: `cd apps/<backend-app> && go build ./... && golangci-lint run ./... && go test ./... -count=1 -race` all exit 0.

## 5. Implement `<frontend-app-name>`

<!-- Duplicate this section for each additional app the change touches, in
     dependency order. Remove entirely if the change touches only one app. -->

- [ ] 5.1 Consult the `aoh-conventions` skill (web conventions) and the `aoh-design` skill before writing the page. Open `openspec/changes/<change>/design/<surface>-mock.html` in a browser as visual reference.
- [ ] 5.2 If the surface is built by an AOH **UI-builder skill** — a platform SDK for a specific surface type — ALSO consult that skill and build on its SDK; do NOT hand-roll the surface from raw `@mssfoobar/ui` primitives.
  Current UI-builder skills:
  - `aoh-dashboard` — dashboards: a `@mssfoobar/dash-web-sdk` `<Grid>` of widgets registered via `defineWidget`.
  <!-- Register new UI-builder skills here as the platform ships them (e.g. a
       form-builder skill on a form SDK). Delete this task if the surface
       matches no UI-builder skill. -->
- [ ] 5.3 <Page / route / gateway proxy / `+page.server.ts` / form action task using `@mssfoobar/ui` primitives by subpath…>
- [ ] 5.4 <Tests…>
- [ ] 5.5 Verify: `cd apps/<frontend-app> && pnpm build && pnpm check && pnpm test` all exit 0.

## 6. End-to-end verification + reproducibility gate

- [ ] 6.1 Native dev smoke / E2E: <concrete command, e.g. `pnpm -F <web-app> test:e2e tests/...`>
- [ ] 6.2 **Reproducibility gate** — tear and rebuild infra, restart native dev processes for every app the change touches, re-run the SAME E2E command from 6.1:
  ```bash
  podman compose down -v                       # or: docker compose down -v
  podman compose up -d                         # or: docker compose up -d
  # wait for healthy: podman compose ps (or docker compose ps)
  cd apps/<backend-app>  && go run ./cmd/server &   # native, one per backend
  cd apps/<frontend-app> && pnpm dev &              # native, one per frontend
  cd apps/<frontend-app> && pnpm test:e2e <same spec as 6.1>
  ```
