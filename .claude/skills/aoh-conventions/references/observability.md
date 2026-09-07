# Observability conventions (OpenTelemetry)

AOH instruments every service with OpenTelemetry against a single, vendor-neutral
seam: each service exports OTLP to the per-spoke **gateway Collector**, which
forwards to the backend (SigNoz by default). Services never know the backend —
swapping it is a Collector change, not a code change.

**Everything is a no-op until `OTEL_EXPORTER_OTLP_ENDPOINT` is set.** Importing
and initialising the SDK is safe in every environment (local, test, CI); telemetry
only flows when the endpoint points at a Collector (e.g.
`http://otel-collector:4317`). So always wire instrumentation in — never gate it
behind a feature flag.

New services are **born instrumented** by the scaffolders (`aoh-go-init`,
`aoh-web-init`, `aoh-module-init`). The rules below are what those templates
emit; follow them when hand-writing or reviewing instrumentation.

## Resource attributes (every service)

| Attribute | Value | Source |
|-----------|-------|--------|
| `service.name` | `<module>-service` (Go) · `web-<app>` (SvelteKit BFF) | set at init |
| `service.namespace` | `aoh` (fixed, every service) | set by the distribution |
| `service.version` | the service's version | set at init |
| `deployment.environment.name` | `dev` / `qa` / `prod` | `OTEL_RESOURCE_ATTRIBUTES` |
| `service.instance.id` | a UUID per process | set by the distribution |

Propagation is **W3C `tracecontext` + `baggage`** (the SDK default) — so a
`trace_id` is continuous across service hops (a BFF `fetch` to a Go service joins
one trace).

## Go services — `aoh-golib/otel`

In `cmd/server/main.go`: `aohotel.Init(ctx, aohotel.Config{ServiceName: "<module>-service", ServiceVersion: ...})` and `defer` its returned shutdown (flushes on exit). In the chi middleware chain (outermost first): `aohotel.HTTPMiddleware("<module>-service")` (server span + RED/HTTP + Go-runtime metrics) → `aohotel.ChiRouteSpanName` (renames the span to the matched route pattern — low cardinality) → a trace-correlated access logger that appends `aohotel.TraceContextFields(ctx)`.

- Logs: two separate concerns, both needed.
  - **Export** — `aohlog` records ship over OTLP via the `otelzap` bridge, registered
    by `aohotel.Init` since AOH-7540 (disable with `OTEL_LOGS_EXPORTER=none`).
  - **Correlation** — the bridge does *not* do it: `otelzap` adopts a span only when a
    field of type `context.Context` is passed, so a record logged the normal way carries
    a zero native `TraceID`. The fields must therefore be attached explicitly, but as of
    AOH-3985 use the context-aware emitters rather than splatting at the call site —
    `aohlog.Ctx(ctx)`, `aohlog.ErrorCtx(ctx, …)`, `aohlog.WithCtx(ctx, logger)` — which
    read the active span and attach `trace_id`/`span_id` themselves.
    `aohotel.TraceContextFields(ctx)` produces the same pair for a caller assembling a
    field slice by hand. (This bullet once said "do not hand-roll log correlation" while
    nothing correlated at all; the point is not that it happens for free, but that the
    emitter does it rather than every author remembering to.)
- Health probes (`/livez`, `/readyz`) log at `debug` (their spans + metrics still flow) to avoid steady-state log spam.

## SvelteKit / Node — `@mssfoobar/observability`

- `src/instrumentation.server.ts`: `startObservability({ serviceName: "web-<app>" })`. SvelteKit loads it before any other server module (gated on `experimental.instrumentation.server: true` in `svelte.config.js`, which needs `@sveltejs/adapter-node` ≥ 5.3.0).
- `src/hooks.server.ts`: mount `createObservabilityHandle()` **outermost** in `sequence(...)` — it **creates** the route-named `SERVER` span (extracts any inbound `traceparent`, records status; 5xx → ERROR), the Node analog of Go's `otelhttp` middleware. Pass `teamID` to stamp the tenant once auth has populated `locals`.
- **Bundle-safe by design — NOT auto-instrumentation.** A SvelteKit `adapter-node` build is a bundled ESM server, which erases the module boundaries OTEL's auto-instrumentation monkey-patches; `getNodeAutoInstrumentations()` therefore can't create HTTP server spans, and forcing the ESM loader hook over a bundle crashes. `startObservability` uses explicit, bundle-safe instrumentation instead: server spans from the handle hook (above); outbound `fetch` client spans + W3C propagation from `@opentelemetry/instrumentation-undici` (`diagnostics_channel`, no patch); runtime metrics from `@opentelemetry/instrumentation-runtime-node` (`perf_hooks`); RED metrics derived from spans downstream.
- Logs: `@mssfoobar/observability` registers a `@mssfoobar/logger` mixin from `startObservability` that injects `trace_id`/`span_id` into every pino record — works even when the logger is bundled (same reason). Do **not** hand-roll correlation; do **not** rely on `@opentelemetry/instrumentation-pino`.
- In-browser tracing is intentionally deferred — this instruments the server + BFF only.

## `team_id`

Carry the tenant on **baggage**, stamped on spans and logs via the distribution's
`team_id` helper. **Never put `team_id` (or any unbounded id) on a metric** — it
explodes cardinality.

## `service.name` registry

`<module>-service` for Go backends (`msr-service`, `gis-service`, `dash-service`,
`form-service`, `unh-service`, …); `web-<app>` for SvelteKit BFFs
(`web-reference-host`, …). One name **per deployable** — a module with more than
one Go binary names each separately (WFE ships two: `wfe-manager` + `wfe-engine`,
not a single `wfe-service`). Keep names stable — dashboards and alerts key on them.

## Sampling

- **Head sampling (SDK)** defaults to `parentbased_always_on` — services keep every span and defer the decision downstream. Leave it: don't drop traces at the service, where you lack the cross-service view.
- **Tail sampling (gateway, prod)** is where the real policy lives: the gateway Collector keeps full traces for **errors + slow requests** and down-samples the rest. A commented `tail_sampling` stub ships in the `otel` `aoh-compose` fragment — uncomment it for prod. It buffers spans per trace, so an HA gateway tier must be fronted by a trace-id load-balancing exporter (two-tier collector layout).
- Override head sampling only for special cases via `OTEL_TRACES_SAMPLER` / `OTEL_TRACES_SAMPLER_ARG`; prefer tail sampling at the gateway.

## The env contract

| Var | Effect |
|-----|--------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | the gateway Collector; **unset = no-op** (nothing exported) |
| `OTEL_RESOURCE_ATTRIBUTES` | extra resource attrs, e.g. `deployment.environment.name=dev` |
| `OTEL_TRACES_SAMPLER` / `OTEL_TRACES_SAMPLER_ARG` | head-sampler override; default `parentbased_always_on` (keep all — sample at the gateway instead) |
| other `OTEL_*` | honored by the SDKs (headers, timeouts, …) |

## Pointers

- Go distribution: `packages/aoh-golib/otel`. Node distribution: `packages/observability`.
- Gateway Collector + SigNoz backend: the `otel` / `signoz` `aoh-compose` fragments (`aoh-compose` skill).
- A bundled SvelteKit app that force-bundles its deps (`ssr.noExternal`, like `reference-host`'s self-contained image) needs `NODE_OPTIONS=--max-old-space-size=4096` on its `vite build` — bundling the OTEL tree otherwise OOMs CI.

## Deployment: BYO backend, air-gap, SigNoz UI auth

- **Bring-your-own backend.** The gateway Collector is the vendor-neutral seam — services only target `OTEL_EXPORTER_OTLP_ENDPOINT` and never know the backend. SigNoz is the default bundled backend (the `signoz` fragment); to use your own, repoint the Collector's exporter `endpoint` in `otel-collector-config.yaml` — no service change.
- **Air-gap.** Mirror the SigNoz / ClickHouse / signoz-otel-collector images into your registry (Nexus/Harbor), and mirror the ClickHouse histogram-quantile binary (point `HISTOGRAM_QUANTILE_BASE_URL` at the mirror). Outbound usage telemetry (phone-home) is **off by default** (`SIGNOZ_ANALYTICS_ENABLED` / `TELEMETRY_ENABLED = false`). See the `signoz` fragment header for the exact knobs.
- **SigNoz UI auth.** Ships with SigNoz's **built-in accounts + basic RBAC** by default — deliberately **not** gated behind Keycloak, so operators don't hit a double login. First run requires creating the initial org/account at `signoz.${DEV_DOMAIN}` before telemetry is ingested (the OpAMP collector gate). To put the UI behind platform SSO, add a Traefik + Keycloak `forwardAuth` middleware on the SigNoz route — an **opt-in**, not the default.
