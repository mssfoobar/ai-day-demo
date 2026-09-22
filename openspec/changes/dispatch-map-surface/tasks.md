## 1. Implement `dispatch-web`

- [ ] 1.1 Consult the `aoh-conventions` skill (`references/web.md`) and the `aoh-design`
      skill before writing any `.svelte` file, and open
      `openspec/changes/dispatch-map-surface/design/map-mock.html` in a browser as visual
      reference. Visual primitives come from `@mssfoobar/ui` subpaths — no raw `<button>` /
      `<div class="rounded border">` equivalents. Read
      `apps/dispatch-web/AGENTS.md` first: this app has no route groups, no gateway and no
      nav, and several `references/web.md` rules therefore do not apply to it.
- [ ] 1.2 The map is built by a platform SDK, so ALSO consult the `aoh-gis-integration`
      skill and compose the surface from `@mssfoobar/gis-web-sdk`. Do NOT hand-roll a map
      renderer, a tile client or a layer panel. (The UI-builder register in the tasks
      template lists `aoh-dashboard` only; a map on `gis-web-sdk` is the same category.)
- [ ] 1.3 Install as runtime `dependencies` — not devDependencies, so a prune-before-build
      cannot drop them: `@mssfoobar/gis-web-sdk@^2.0.0`, `@mssfoobar/auth-sdk`,
      `@cesium/engine@^25`, `@cesium/widgets@^15`. `@mssfoobar/ui`, `@mssfoobar/logger`
      (≥ 1.0.5) and `@mssfoobar/sse-client` are already present at compatible versions. The
      two `@cesium/*` packages are only transitive deps of the SDK, and under pnpm's strict
      layout `require.resolve` cannot find them from the consumer app — which is what breaks
      the asset plugin in 1.4. Note `pnpm-workspace.yaml`'s `minimumReleaseAge: 10080`: a
      version published in the last seven days will not resolve.
- [ ] 1.4 Add the Cesium asset-copy plugin to `apps/dispatch-web/vite.config.ts`: resolve
      `@cesium/engine` and `@cesium/widgets` via `require.resolve`, copy `Build/Workers`,
      `Source/Assets` and `Source/ThirdParty` from the engine and `Source` from the widgets
      into `static/cesium/` (`fs.cpSync` recursive + force + dereference) on **both**
      `buildStart` and `configureServer`, and define
      `CESIUM_BASE_URL` as `"/cesium"`. Add `@mssfoobar/*`, `@lucide/svelte`,
      `@cesium/engine` and `@cesium/widgets` to `ssr.noExternal`. Without this the canvas is
      entirely black and the only evidence is 404s — design.md R1.
- [ ] 1.5 Add `static/cesium/` to `apps/dispatch-web/.gitignore`. `eslint.config.js` already
      ignores it (committed with the baseline), so leave that file alone.
- [ ] 1.6 Mount `<GisProvider>` in `src/routes/+layout.svelte`, fed by the store the app's
      existing `src/lib/aoh/core/provider/theme/ThemeProvider` already owns. Do not create a
      second dark-mode store — design.md D5.
- [ ] 1.7 Add `src/routes/map/+page.ts` exporting `ssr = false`, and
      `src/routes/map/+page.svelte` composing `CesiumMapEngineProvider` → `Map` (with the
      **required** `initial_camera_view`; omit `rtus_seh_url`, `rtus_map_name`, `user_id` and
      `tenant_id` entirely, so the SDK's live feed never attempts to subscribe) →
      `MapBaseLayerProvider` + `MapXyzSourceProvider` pointed at OpenStreetMap →
      `MapLayerManager`. Add **no** `MapEntityLayerProvider` and **no** `MapEntityProvider`:
      entities are the workshop exercise, and an empty provider now is only a stub to delete.
- [ ] 1.8 Add the page's chrome from `@mssfoobar/ui`: a page header, the "No entities are
      shown on this map yet." note naming wiring field units as the next step, and a link
      back to the console. Add the **Map** link to the console page. Introduce no `nav.ts`,
      `Sidebar` or `Headerbar` — design.md D7.
- [ ] 1.9 Update the docs this change makes stale: `apps/dispatch-web/README.md` (the map
      route, and that `/` still redirects to `/units`), `apps/dispatch-web/AGENTS.md` (the
      Cesium asset contract and the `ssr = false` requirement are both load-bearing and
      non-obvious), and `UBIQUITOUS_LANGUAGE.md` — keep the *field unit* ↔ `geo-entity`
      mapping **conditional**, since units still carry no position, but correct its guess:
      `field-unit` would be the `kind`, `track` the `entity_type`. Add no *position* term.
- [ ] 1.10 Write the workshop exercise this change exists to set up. `WORKSHOP.md` gains a
      fourth exercise and `USER_STORIES.md` its paste-ready brief, in the same shape as the
      existing three — user story, acceptance criteria, "what already exists", suggested API,
      out-of-scope list. What already exists is the map surface itself. Note that the
      exercise needs `gis-service` and RTUS, which need IAMS, so it is the larger of the
      workshop's exercises; `openspec/changes/dispatch-iams-and-unit-map/` already works out
      that integration in detail and is the brief's reference. Flag the one thing an attendee
      cannot guess: a client-credentials token carries **no** `active_tenant` claim and
      `gis-service` resolves the tenant from it, so a projection must ride an operator's
      bearer.
- [ ] 1.11 Document the env block for a native run. The map needs **no new variable** —
      `.env.development` is unchanged, `IAM_URL` stays unset (which is what keeps the auth
      hook a passthrough, design.md D3), and `ORIGIN=http://localhost:5173` is as today. Say
      so explicitly in `apps/dispatch-web/README.md` so the next reader does not go looking
      for GIS configuration that is not there. `PUBLIC_CESIUM_TOKEN` stays unset: Ion is
      disabled and no asset quota is consumed.
- [ ] 1.12 Verify: `cd apps/dispatch-web && pnpm build && pnpm check-types && pnpm lint` all
      exit 0, and `git status --porcelain static/cesium` is empty (the copied assets are
      ignored, not staged).

## 2. End-to-end verification

- [ ] 2.1 Native dev smoke, against the unchanged one-container stack: `pnpm start`, then
      open `http://localhost:5173/map` and confirm — base tiles render (not a black canvas);
      the browser's network log shows **no 404 under `/cesium/`**; the layer panel lists and
      toggles the OpenStreetMap layer; the page states that no entities are shown yet; the
      link back to the console works and the console's **Map** link returns; switching the
      app's theme repaints the map; and the network log shows **no** request to
      `dispatch-svc`, `gis-service`, RTUS or any identity provider.
- [ ] 2.2 Confirm the no-auth posture holds with nothing else running: stop the PostgreSQL
      container (`pnpm stop`), reload `/map`, and confirm the map still renders — it depends
      on no backend at all. The console will show its service-unavailable state, which is
      correct and unchanged.
- [ ] 2.3 Confirm the production build serves the assets too, since 1.4's plugin runs on a
      different hook for each: `cd apps/dispatch-web && pnpm build && pnpm preview`, open
      `/map`, and repeat 2.1's first two checks.

> No reproducibility gate: this change composes no infrastructure and seeds nothing, so
> there is no volume to tear and no state to reconverge. The existing PostgreSQL container
> is untouched, and `pnpm reset-db` means exactly what it meant before.
