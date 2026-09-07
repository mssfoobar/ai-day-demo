# AOH platform glossary (published language)

The canonical vocabulary of the **AGIL Ops Hub (AOH)** platform — the terms a
consumer application inherits when it builds on the AOH SDKs and services. One
opinionated word per concept; everything else is pushed to "Aliases to avoid".

**For consumer projects — reference, don't redefine.** These are *platform*
terms. In your own project's `UBIQUITOUS_LANGUAGE.md`, define only *your* domain
vocabulary plus the **boundary mappings** to these (e.g. "our *Vessel* ↔ a GIS
`geo-entity` with `entity_type=vessel`"; "our *Operator* ↔ AOH `active_tenant`").
Do not copy these definitions into your project — link here instead, and record
the `aoh-knowledge` skill version you built against so a skill bump surfaces
vocabulary changes for review. (ops-hub's own build/release/monorepo-internal
vocabulary lives in the ops-hub repo's root `UBIQUITOUS_LANGUAGE.md`, not here.)

## Platform & identity

| Term | Definition | Aliases to avoid |
| ---- | ---------- | ---------------- |
| **AOH** | ST Engineering's modular platform that supplies shared cross-cutting services (auth, real-time, geospatial, storage) so teams build C2 apps without re-implementing them. | Agil Ops Hub, AGIL Ops Hub, Ops Hub |
| **C2 application** | The class of operator-facing system AOH exists to build — real-time monitoring, resource coordination, event response (ops centres, dispatch, incident management). | Command and Control system |
| **ops-hub** | The platform monorepo hosting every shared package, every `modules/<name>/` module, the `aia` CLI, and the agent-skills subtree. | the repo, platform-monorepo |
| **aia CLI** | The `@mssfoobar/ai-accelerator-cli` tool (binary `aia`) that bootstraps local AOH dev environments — checks prerequisites, scaffolds projects, manages spokes and module SDKs. | aa-cli, AI Accelerator CLI |
| **web-base** | The standard scaffolded AOH SvelteKit frontend (OIDC PKCE auth, `(private)`/`(public)` route groups, gateway proxy, `@mssfoobar/ui` primitives) consumer frontends start from. | aoh-web-base, Web App Backend |

## Topology (hub / spoke / infra)

| Term | Definition | Aliases to avoid |
| ---- | ---------- | ---------------- |
| **Hub** | The platform-team-operated, Kubernetes/Helm-deployed shared dev-tooling plane (cross-project Nexus, future Forgejo/Harbor/Backstage) serving all projects. | hub services |
| **Spoke** | A per-project, single-server Docker-Compose workspace hosting that project's Forgejo, Nexus, Harbor, Backstage, shared Postgres, and Traefik. | project spoke |
| **Profile** | A deployment flavour: `laptop` (compose, plaintext HTTP, local admin), `team` (Traefik/TLS on a project server), or `prod` (K8s/Helm, not yet built). | laptop/team/prod profile |
| **Compose fragment** | One AOH service packaged as a self-contained dir under `deploy/spoke-services/` (manifest + compose base + per-profile overlays) that `aia spoke` assembles. | service fragment, spoke-service fragment |
| **Service manifest** | The `manifest.yaml` inside a fragment declaring metadata, depends_on, profiles, env, ports, healthcheck, and Traefik rule that `aia spoke` reads. | manifest.yaml |
| **Traefik** | The reverse-proxy / edge ingress that host-routes all HTTP traffic to services by `<service>.${DEV_DOMAIN}` (TLS on 443 in `team`; bare localhost ports in `laptop`). | infra, API gateway, ingress |
| **DEV_DOMAIN** | The base domain (default `127.0.0.1.nip.io`) under which Traefik exposes each service; internal container-to-container calls use bare compose service names instead. | nip.io domain, host-based routing |
| **Nexus** | The Sonatype Nexus OSS registry that pull-through-proxies public registries (Maven/npm/PyPI/Go/APT/YUM/Docker Hub) and hosts internal cross-project packages. | registry proxy |
| **Group repository** | The Nexus repo that aggregates a format's proxy + hosted repos behind one URL — what developers actually point build tools at (npm-group, maven-group). | proxy/hosted repository |
| **Forgejo** | The per-project SCM + CI + built-in package registry — "the identity plane for a project". | spoke SCM |
| **Harbor** | The spoke's OCI container + Helm registry for internal images (separate from Nexus, which only caches Docker Hub). | — |
| **Runner** | The CI job executor registered to the spoke's SCM (act-runner for Forgejo, gitlab-runner for the GitLab alternative). | act-runner, gitlab-runner |

## Published packages

How consumers consume the platform: install from the `@mssfoobar` scope on GitHub
Packages, then depend on the SDK/library packages directly.

| Term | Definition | Aliases to avoid |
| ---- | ---------- | ---------------- |
| **@mssfoobar scope** | The npm / GitHub-Packages scope namespacing every shared AOH package; consumers point `@mssfoobar:registry` at `npm.pkg.github.com`. | mssfoobar scope |
| **GitHub Packages** | The restricted-access npm registry (`npm.pkg.github.com`, scope `@mssfoobar`) hosting every published AOH package, alpha and stable. | the registry, GHCR npm |
| **@mssfoobar/ui** | The shared AOH design system: shadcn-svelte primitives (Bits UI + Tailwind v4) extended with AOH tokens, shipped as subpath-exported Svelte components. | AOH design system, ui library |
| **@mssfoobar/auth-sdk** | Framework-agnostic OIDC/Keycloak auth SDK with multi-tenant claims and optional SDS-backed sessions, plus a SvelteKit adapter. | auth-sdk |
| **@mssfoobar/logger** | Structured pino-based, ESM-only logging library for AOH web apps, exposed via the bare `logger` specifier. | logger |
| **@mssfoobar/sse-client** | SSE client library that consumes the RTUS-SEH realtime feed with reconnect/heartbeat and tenant/map-scoped subscriptions. | sse-client, SSEClient |
| **@mssfoobar/graphql** | The urql-based GraphQL client for web-base (subscriptions + token refresh); currently `private` pending a publish flow. | graphql library |

## aia CLI surface

| Term | Definition | Aliases to avoid |
| ---- | ---------- | ---------------- |
| **aia check** | Verifies the external prerequisites (Node, Git, pnpm, OpenSpec, Docker) `init` and spokes depend on are installed. | — |
| **aia init** | Scaffolds a new project from the AOH turborepo template. | — |
| **aia config** | Shows or modifies the per-user config at `~/.config/aia/config.json`. | — |
| **aia spoke** | Manages a project spoke (compose-based platform stack): `init`/`up`/`down`/`status`/`logs`/`upgrade`/`describe`. | spoke CLI |
| **aia module** | Installs, lists, and describes AOH module SDKs inside a consumer app (`add`/`list`/`describe`). | — |

## Module anatomy

| Term | Definition | Aliases to avoid |
| ---- | ---------- | ---------------- |
| **Module** | A vertically-sliced AOH feature under `modules/<name>/` bundling every layer to ship one capability — a Go service, a Svelte frontend SDK, a deploy snippet (plus client + types for gis/msr). | vertical slice, feature module |
| **Service** | The Go backend package of a module (chi + Postgres HTTP server), `"private"`, wired into `go.work` rather than registry-published. NOT the whole module. | Go service, backend |
| **web-sdk** | A gis/msr/unh/wfe/dash/form module's packaged Svelte-5 frontend (`@mssfoobar/<name>-web-sdk`), living under `web/` and built with `svelte-package` (source under a `sdk/` subdir) — the replacement for the legacy modlet. Form's is the SurveyJS renderer/Creator; its typed clients + DTOs live in the sibling `form-client` + `form-types` packages (AOH-7762), so Form now follows the full `types ← client ← web` trio like the others. | web SDK |
| **client** | A module's typed fetch-based REST client (`@mssfoobar/<name>-client`) wrapping the service's HTTP API with error typing, envelope unwrapping, pagination. | fetch client |
| **types** | A module's shared TS package (`@mssfoobar/<name>-types`) of hand-authored mirrors of the Go service's wire DTOs, edited in lockstep with `dto/*.go`. | wire types |
| **deploy** | A module's `deploy/` dir holding the compose snippet and bring-up/cutover docs (k8s/ArgoCD manifests live in the separate dev-infra repo). | deploy assets |
| **compose snippet** | The composable `deploy/compose.snippet.yml` standing up a module's service + profile-gated bundled DBs, designed to be included into a larger hub stack. | compose.snippet.yml |
| **go.work** | The repo-root Go workspace tying every module's private service together for builds — the Go counterpart to `pnpm-workspace.yaml`. | Go workspace |

## web-sdk integration surface

| Term | Definition | Aliases to avoid |
| ---- | ---------- | ---------------- |
| **Host** | The SvelteKit app consuming a module: installs the web-sdk as a normal dep, adds `@mssfoobar/` to vite `ssr.noExternal`, mounts the Provider, owns the BFF routes, places the nav. | consumer app, host app |
| **reference-host** | The executable consumer example at `apps/reference-host/` wiring a module's web-sdk + BFF end-to-end — the canonical integration reference and home of the module E2E tests. | integration sandbox, harness |
| **Provider** | The application-root Svelte context component a web-sdk exports (`GisProvider`, `MsrProvider`) supplying dark-mode, BFF base path, and auth/theme to every module component. | context provider, wrapper |
| **module nav** | The nav object a web-sdk exports from `./nav` (`gisNav`/`msrNav`, shape `{ code, header, sidebar[] }`) that the host imports and places explicitly into its chrome. | nav.ts |
| **BFF** | The host-owned server layer (SvelteKit `+server.ts` routes) a web-sdk talks to exclusively — holds the upstream service URL + bearer token so the browser never calls the service directly. | backend-for-frontend |
| **BFF handlers** | Server-side handler factories a web-sdk exports from `./server` (e.g. `msrBffHandlers(cfg)`) that the host mounts one-per-route to proxy authenticated requests upstream. | handler factories |
| **subpath export** | Each component/entry of a web-sdk exposed as its own package export path (e.g. `@mssfoobar/msr-web-sdk/multi-session-replay`) — the SDK's compose-via-props customization seam. | package export |
| **styles/app.css bridge** | The `@mssfoobar/<name>-web-sdk/styles/app.css` import: a Tailwind-v4 `@source` registration putting the SDK's `dist/` under the host's content scan (replaces modlet tailwind.config mutation). | @source bridge |
| **map engine** | A pluggable map-rendering backend of the GIS web-sdk (`./engines/cesium`, `./engines/maplibre`) — GIS-specific; the MSR web-sdk has none. | engine |

## Cross-cutting (auth, wire contract, errors)

| Term | Definition | Aliases to avoid |
| ---- | ---------- | ---------------- |
| **IAMS** | AOH's Identity & Access Management System — Keycloak (authentication) + AAS (authorization) + iams-web/iams-init — that nearly every service depends on. | Identity & Access Management System |
| **Keycloak** | The OIDC provider and JWT issuer inside IAMS, hosting the single `aoh` realm and minting tokens via Authorization Code Flow with PKCE. | iams-keycloak |
| **AAS** | IAMS's tenant-scoped authorization control plane owning all application-level authz (roles, groups, resources, scopes, permissions) via its `/admin/` API. Expands to "Authorization and Admin Service". | iams-aas, Attribute-based access |
| **realm (aoh)** | The single Keycloak realm `aoh` holding OIDC clients, claim mappers, and platform-bootstrap realm roles; apps only add OIDC clients, never realm roles. | aoh realm |
| **tenant** | An organization within the `aoh` realm and AAS's unit of multi-tenancy; every service filters data by tenant. `development` is the bootstrapped dev tenant. | — |
| **active_tenant** | The JWT claim carrying the user's current tenant context (`tenant_id`, `tenant_name`, `roles`) — the source of truth for request-time role checks. | active_tenant claim |
| **AAS tenant role** | An application-defined role created per tenant in AAS (e.g. `field-reporter`), surfaced into the JWT as `active_tenant.roles` — the canonical home for app-specific roles. | application role, realm role (a realm role is different) |
| **realm role** | A platform-bootstrap Keycloak role (`system-admin`, `tenant-admin`) owned by the platform team; applications MUST NOT add to these. | — |
| **gateway proxy** | The web-base route (`(private)/aoh/gateway/[...path]`) that proxies authenticated browser requests to backend modules and attaches the access token; browser-only, never from `load()`. | gateway, API gateway |
| **aoh-golib** | The shared Go library (`github.com/mssfoobar/ops-hub/packages/aoh-golib`) providing `aohhttp` (envelopes + bearer auth), `aohlog` (logging), and `temporal` helpers. | — |
| **response envelope** | The uniform success wrapper every service returns — `{ data, message, sent_at, errors? }` (+ a `page` sibling for lists) — produced by `aohhttp`, typed as `HTTPResponseBody<T>`. | HTTPResponseBody, ResponsePayload |
| **occ_lock** | The integer optimistic-concurrency version every mutating request must echo; a stale value is rejected with HTTP 409 Conflict. | optimistic concurrency lock |
| **trace_id** | The OpenTelemetry trace id — the single correlation key joining an error to its logs and its distributed trace. Established automatically at the system edge by the OTEL HTTP instrumentation, propagated across services via the W3C `traceparent` header, auto-injected into logs (Go: the otelzap bridge; Node/Svelte: the `@mssfoobar/logger` OTEL mixin registered by `startObservability`), surfaced in the error envelope as `trace_id` (omitted when no trace is active), and shown to users as the "Support ID". | correlationId, x-correlation-id |
| **errorCode** | The machine-readable UPPER_SNAKE_CASE failure identifier (e.g. `INSUFFICIENT_PERMISSIONS`) the BFF translates to a user message and that drives HTTP-status mapping. | — |
| **standardized error response** | The mandatory 4xx/5xx body `{ timestamp, trace_id?, errorCode, errorMessage, details? }` (`trace_id` omitted when no trace is active); `errorMessage` is developer-facing (logs only), never shown to users. | standardized error payload |

## GIS module

| Term | Definition | Aliases to avoid |
| ---- | ---------- | ---------------- |
| **GIS map** | The interactive map surface rendering live geospatial entities, base tiles, and operator panels, wrapped by exactly one engine. | Map, GisMap |
| **Cesium engine** | The 3D map rendering engine (needs `PUBLIC_CESIUM_TOKEN`; falls back to OSM 2D without it). | — |
| **MapLibre engine** | The pure-2D map rendering engine, the alternative to Cesium. | — |
| **geo-entity** | A geospatial feature (aircraft, vehicle, incident, boundary, responder) stored as a GeoJSON feature with `entity_id` + `entity_type`, rendered as a marker/shape. | entity, GeoEntity, map entity |
| **kind** | The app-level classifier on a geo-entity (`geojson.properties.kind`) that buckets entities for per-layer rendering — distinct from the `entity_type` DB column. | — |
| **entity layer** | A named, toggleable grouping of map entities (renderer/filter over the RTUS-fed store, no data prop) shown with a visibility switch in the Layer Manager. | layer, MapEntityLayerProvider |
| **base layer** | A selectable background tile source (e.g. an OSM XYZ URL); the first declared is the default. | layer, MapBaseLayerProvider |
| **bookmark** | A saved map camera position + orientation (lon/lat/alt/zoom/pitch/yaw/roll) to jump back to a known view. | — |
| **RTUS** | The platform Real-Time Update Service the map subscribes to over SSE for live entity create/update/remove — the only source that populates the map's entity store. | live feed, Real-Time Update Service |
| **RTUS json-map** | A per-tenant keyed KV map inside RTUS (named by `rtus_map_name`, e.g. `"gis"`) holding the live entity set; the backend upserts/deletes keys. | json-map, map |
| **transactional outbox** | The gis-service write pattern committing a repo row + an outbox row atomically, then dispatching to RTUS via a background worker (replaces inline RTUS calls). | outbox |
| **GIS sync** | Admin reconciliation of the GIS DB against RTUS (diff/full/partial) that deliberately bypasses the outbox because it IS the reconciliation. | reconciliation, /sync |

## MSR / replay module

| Term | Definition | Aliases to avoid |
| ---- | ---------- | ---------------- |
| **MSR** | Multi-session Replay — records DB changes via CDC into TimescaleDB and replays them, reconstructing entity state at a chosen timestamp for multiple concurrent users. | replay service |
| **replay session** | A per-user record of an in-progress replay, rate-limited to at most one ACTIVE per user and `MAX_ACTIVE_SESSIONS` concurrent overall. | session |
| **lobby** | The initial MSR replay UI state (a dialog inviting the operator to start) shown before timestamp selection (`ReplayStatus "idle"`). | — |
| **replay controller** | The playback control UI (play/pause/speed/timeline) driving an active replay. | ReplayController |
| **Change Data Capture (CDC)** | The ingestion pipeline (Kafka/Debezium → TimescaleDB) capturing source DB row changes as a stream of `cdc_event` rows feeding MSR. | — |
| **cdc_event** | A single change record — `entity_id`, `entity_state` (JSONB, NULL on delete), `op` (`c`/`u`/`d`), `event_timestamp`, `table_name` — stored raw in the hypertable. | change event |
| **hypertable** | The TimescaleDB time-partitioned table (`msr.cdc_event`) storing cdc_events, split by `event_timestamp` into chunks. | partitioned table |
| **chunk** | One time-partition of a hypertable (1-hour interval); retention drops whole chunks, and each carries its own lock. | partition, shard |
| **earliest snapshot** | A periodically-rebuilt point-in-time materialization of every entity's last state at the playback cutoff, kept in rotating A/B tables — the base tier of state reconstruction. | snapshot |
| **continuous aggregate** | The TimescaleDB materialized view (`msr.entity_last_states`) of daily last-state per entity — the middle tier minimizing chunk decompression. | CAGG |
| **max playback range** | `MAX_PLAYBACK_RANGE` (days, default 7) — the max historical span a user may replay; sets the snapshot cutoff below which raw chunks are dropped and drives CAGG retention. | playback range, replay window |
| **replay frame** | A single playback tick (default 30 FPS) in which the worker applies all buffered change events due by the current playback time to the entities store. | frame, FRAME_UPDATE |

## Dashboard module

| Term | Definition | Aliases to avoid |
| ---- | ---------- | ---------------- |
| **Dashboard** | A named, tenant-scoped container laying out a collection of widget instances on a grid; favouritable and taggable. | board |
| **Widget** | A single placed instance of a widget type on one dashboard, carrying grid position + a persisted config blob. | widget instance |
| **WidgetType** | The catalog template a widget instances — declares name, icon, sizing constraints, category, instance limit; its stable id becomes the backend `widget_type_id`. | WidgetMeta, widget type template |
| **defineWidget** | The widget-authoring API building a reactive widget class from a declarative spec (type id, meta, config schema, derived, actions). | — |
| **widget config schema** | The `ConfigSchema` that is the single source of truth for a widget's persisted state — each field becomes a reactive property, a picker control, and a config-blob key. | config, ConfigSchema |
| **WidgetFactory** | The singleton catalog registering each widget class with its View component and applying backend overrides/categories onto metadata. | WidgetRegistry, factory |
| **DashboardCanvas** | The lean SvelteKit renderer drawing a dashboard's widgets onto the GridStack grid and hosting the WidgetPicker overlay. | Grid |
| **WidgetPicker** | The right-side overlay palette to browse widget types by category, drag onto the canvas, and edit the selected widget's config. | widget palette, picker |
| **createDashboardEditor** | The composable bundling all dashboard-editing state (widgets, models, dirty flag, mode, picker, save flows) so a page hands one `editor` to DashboardCanvas. | DashboardEditor |
| **42-column layout** | The dash GridStack convention (42 columns, square cells, 18 min rows, 0.5rem margin) the backend and renderer CSS both assume. | DEFAULT_GRID_CONFIG, 42-column grid |
| **DashboardClient** | The pure-HTTP SDK client (no UI state) doing all dashboard/widget/tag CRUD against the dash service through the AOH gateway. | — |
| **dash service** | The Go (chi + sqlx + Postgres) backend owning dashboards, widgets, widget types, categories, favourites, and shared configs. | dash-service, dashboard service |
| **shared_config** | A reusable named widget-config blob in the dash backend keyed by `widget_type_id`, sharable across dashboards/tenants. | shared config |
| **dash-init seeding** | Provisioning a dashboard before any user opens the UI — via the aoh-compose `dash-init` one-shot from a declarative `dashboard.yaml`, or ad-hoc via `DashboardClient`. | dashboard seeding, dash-init |

## Form module

| Term | Definition | Aliases to avoid |
| ---- | ---------- | ---------------- |
| **Form** | A named, tenant-scoped form definition owned by an admin, pointing to its currently published version. | — |
| **FormVersion** | An immutable published snapshot of a form carrying `form_json`, `theme_json`, and a changelog; submissions are made against a version. | version, published version |
| **FormDraft** | An editable, unpublished work-in-progress of a form an admin authors before publishing — distinct from a submission draft. | draft |
| **Submission** | An end-user's response record against a published FormVersion, holding `data` and a `current_state`. | form response |
| **form_json** | The SurveyJS schema JSON defining a form's pages, elements, choices, workflow states, and per-question state permissions. | SurveyJS schema, form definition |
| **createSurveyCreator** | The Svelte form-authoring UI built on SurveyJS Creator, with AOH editors for workflow states, question permissions, themes, and preview state. | SurveyJS Creator, form designer |
| **form state** | A workflow stage of a submission (Draft, Pending Review, Approved) that gates question permissions; a submission moves between states via a transition. | state, workflow state, current_state |
| **state_permissions** | Question-level rules embedded in `form_json` granting each role read/write access within a specific form state (write > read > hidden). | StatePermission, question permissions |
| **AMM** | The Attachment Management Module — the AOH file-attachment service (and SDK client) to upload/download/scan submission attachments. | Attachment Management Module |
| **surveysql** | The form-service package compiling a FormVersion's `form_json` into a per-form relational SQL table, mapping elements to typed columns and child tables. | internal table, form table |

## Retired terminology

The modlet→SDK migration is the dominant retirement. A legacy **modlet** copied
source into the host and mutated host files at install time; the published
**web-sdk** replaces all of that with a normal versioned dependency.

| Retired term | Replaced by | Why |
| ------------ | ----------- | --- |
| **modlet** | **web-sdk** (or **sdk** for form) | Source-copying installer that copied a package's Svelte source into the host's `src/` and owned/built it there → versioned, published, read-only `dist`. |
| **.mod/ source-subset distribution** | `svelte-package` build (published SDK `dist`) | A `.mod/src/` of raw source copied into the host tree → a built dist consumed read-only. |
| **modlet.config.ts auto-discovery** | exported **module nav** (`gisNav`/`msrNav`) | Host walked `import.meta.glob('/src/lib/aoh/**/modlet.config.ts')` to auto-collect nav → host imports and places an explicit nav object. |
| **modlet.setup.ts env injection** | **Provider** runtime props | Install-time host mutation appending build-time env (`MSR_URL`, `IAMS_AAS_URL`) to `.env.template` → runtime props supplied by the host. |
| **ts-morph post-install host mutation** | **styles/app.css `@source` bridge** | `modlet.setup.ts` rewrote the host `tailwind.config.ts` via ts-morph → a Tailwind `@source` import shipped inside the SDK, zero host mutation. |
| **@mssfoobar/msr-web** | **@mssfoobar/msr-web-sdk** | Legacy MSR modlet package (a source-copying installer, not a runtime dep) → a normal npm dependency. |
| **editing copied source for customization** | **subpath exports** + props/snippets/wrapper composition | Consumers edited copied-in source (owner had no control of what shipped) → compose via package exports. |
| **web-base modlet installer** | normal `pnpm add @mssfoobar/<module>-web-sdk` | The web-base CLI that copied `.mod/` source and ran `modlet.setup.ts` → plain package-manager install. |
| **aoh.config.ts marker** | **aoh-modules.json** | Install marker the modlet installer wrote into consumer source → an agent-readable JSON marker the `aia` CLI writes. |
| **gis-app** (Java/Spring) | **gis-service** (Go) | Legacy Java GIS backend → wire-compatible Go service. Old name is stale. |
| **msr-app** | **msr-service** (Go) | Legacy MSR backend → config-compatible Go service (identical env var names). |
| **IAMS_AAS_URL** | **IAM_URL** (BFF auth toggle) | Legacy modlet frontend env var → renamed, with changed meaning (a BFF toggle). |

**Nuance — modlet auto-discovery is not fully dead.** The `aoh-web-init` host
scaffold still ships a `menuStore.ts` that globs `modlet.config.ts` for nav,
while the same scaffold's `nav.ts` and `AGENTS.md` declare "No modlet
auto-discovery" and use a hardcoded nav list. Treat the hardcoded `nav.ts` as
canonical; the leftover glob should be deleted so the scaffold stops re-teaching
the retired convention.

**`lift-and-shift`** (current) is the related migration *strategy*: the legacy
Go backend is ported into the monorepo unchanged (pgx, chi, layered arch
intact); only the tooling wrapper aligns with `gis-service`. The per-module
**CUTOVER runbook** (`modules/<name>/CUTOVER.md`) is the reversible playbook for
moving a deployed consumer onto the migrated SDK + Go service.

## Relationships

- A **Module** contains exactly one **Service**, exactly one frontend SDK (a **web-sdk** for gis/msr, or a single **sdk** for dash/form), and one **deploy** snippet; gis/msr additionally ship a **client** and **types** package, which dash/form fold into the single **sdk**.
- A **web-sdk** exports exactly one **Provider**, exactly one **module nav**, zero-or-more **BFF handlers**, and many **subpath exports**; it talks only to the host's **BFF**, never to the **Service** directly — which lets frontend and backend cut over independently.
- A **Host** mounts each module's **Provider**, owns the **BFF** routes, and places the **module nav**; **reference-host** is the canonical Host.
- The GIS **web-sdk** has zero-or-more **map engines** (Cesium, MapLibre); the MSR **web-sdk** has none.
- gis-service writes a **geo-entity** row + an **outbox** row atomically; the worker publishes to one **RTUS json-map** per tenant; the **GIS map** subscribes to that json-map via RTUS for the user's tenant. **GIS sync** reconciles the DB against RTUS and bypasses the outbox.
- **MSR** reconstructs state at a timestamp via a 3-tier query (recent **cdc_event** rows + **continuous aggregate** + **earliest snapshot**), then streams later cdc_events the worker applies as **replay frames**. A **hypertable** is partitioned into **chunks**; retention drops chunks older than the snapshot cutoff derived from **max playback range**.
- A **Dashboard** has zero-or-more **Widgets**; each **Widget** references one **WidgetType** and stores a config matching that widget's **widget config schema**. **createDashboardEditor** owns the editing state; **DashboardCanvas** renders it and embeds the **WidgetPicker**.
- A **Form** has many **FormDrafts** and **FormVersions** (one published); a **Submission** is made against one FormVersion, holds a `current_state` that is one of the form's **form states**, and **state_permissions** in **form_json** gate each role's access per state.
- Every mutable GIS/MSR/dash row carries one **occ_lock**; a stale value yields HTTP 409. Every Go **Service** depends on **aoh-golib** (`aohhttp` envelope + bearer auth, `aohlog` logging) and reads **active_tenant.roles** (the **AAS tenant roles**) for authorization.

## Example dialogue

> **Dev:** "I'm wiring MSR into our app. Do I `pnpm add` the **service** or the **web-sdk**?"

> **Domain expert:** "The **web-sdk** — `@mssfoobar/msr-web-sdk`. The **service** is the private Go backend; you never depend on it directly. Your app is the **Host**: it installs the web-sdk, mounts `<MsrProvider>`, and owns the **BFF** routes the SDK talks to."

> **Dev:** "Right, but the old `@mssfoobar/msr-web` modlet copied source into `src/` and I edited it there. Where do I edit now?"

> **Domain expert:** "You don't. The **modlet** is retired — that whole copy-and-mutate model is gone. You customize by composition: import from the **subpath exports** and pass props/snippets/wrappers. Nav comes from the exported **module nav** (`msrNav`), not a `modlet.config.ts` glob, and styles come from the `styles/app.css` **`@source` bridge**, not a tailwind.config mutation."

> **Dev:** "And the env vars the modlet injected — `MSR_URL`?"

> **Domain expert:** "`MSR_URL` is now consumed server-side by your **BFF**, not the browser. You hand config to `<MsrProvider>` as runtime props. `IAMS_AAS_URL` was renamed to `IAM_URL` as a BFF auth toggle — same-name traps are why we flag it."

## Flagged ambiguities

- **"SDK" means three package kinds.** A **web-sdk** (gis/msr Svelte frontend in `web/`), a dash/form **sdk** (in `sdk/`, folds client+types), and a **client** (typed REST wrapper) are all "SDKs" loosely. Reserve "web-sdk" / "sdk" / "client" precisely; never say bare "the SDK" when the layout matters. Internally, a web-sdk's *source* lives in a `sdk/` subdir — call that "the web-sdk source dir", not "the sdk".
- **"service" = the Go backend only**, never the whole module. A **Module** (`modules/<name>/`) is the full vertical slice; the **Service** is `.../service/`.
- **"module" is overloaded** across: an ops-hub vertical slice, the `<module>` segment in `ghcr.io/mssfoobar/<module>/<component>` image tags, the frontend `src/lib/aoh/{module}` grouping, and the AMM upload `module` field. Qualify which sense.
- **"Provider" is the canonical** root context component — avoid "context provider"/"wrapper", which blur it with generic Svelte context.
- **"host" vs "consumer app" vs "hosted repository".** Reserve **Host** for the consumer app mounting a Provider; always say "hosted repository" in full for the Nexus concept; **reference-host** is the specific app.
- **"entity" is overloaded:** GIS **geo-entities** (GeoJSON markers), MSR CDC entities (changed DB rows), and the web-sdk runes-store `Entity`. Reserve "geo-entity" for GIS; qualify MSR as "CDC entity"/"replay entity".
- **"session":** MSR **replay session** (rate-limited record) vs generic auth/HTTP session. Say "replay session" in MSR contexts.
- **"snapshot" means several things:** an alpha-release npm publish, a docs snapshot (frozen VitePress), and MSR's **earliest snapshot** (TimescaleDB A/B materialization) — itself distinct from the GIS RTUS reconnect "init" snapshot. Always qualify.
- **"layer":** **base layer** (background tiles) vs **entity layer** (kind-grouped markers) — both are toggles in the Layer Manager. Never say bare "layer".
- **"config":** grid config (`DEFAULT_GRID_CONFIG`), widget config schema (persisted widget state), dash `shared_config` (reusable blob), and SDK client config are four things. Qualify "grid config" / "widget config" / "shared config".
- **"state":** a **form state** (submission workflow) vs Svelte `$state` vs a widget's reactive model vs the dash service `internal/model`. Reserve "form state" for the workflow concept.
- **"draft":** a **FormDraft** (admin's unpublished form) vs a submission draft (end-user's auto-saved data). Never bare "draft".
- **Two error contracts coexist:** aoh-error-handling's `{ timestamp, trace_id?, errorCode, errorMessage, details }` vs api.md's envelope `{ data, message, sent_at, errors }`. Treat the envelope as the wire shape and the error-handling fields as the logical superset the BFF layers on; document which a given Go service actually emits. Never collapse `errorMessage` (developer-facing) and `userMessage` (user-facing).
- **"AAS tenant role" ≠ "realm role".** Application roles are AAS tenant roles surfaced into `active_tenant.roles`; never label an app role a "realm role".
