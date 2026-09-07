# @mssfoobar/aoh-golib

## 0.3.0

### Minor Changes

- 3bf66f4: feat(aoh-golib): actually export logs over OTLP (AOH-7540)

  `signoz_logs.distributed_logs_v2` held **0 rows all-time** — not "recently", not
  "since a regression". Traces and metrics flowed normally the whole time.

  Root cause was a documented-but-unbuilt design. The repo described the log path as
  "structured stdout JSON scraped by the Collector's `filelog` receiver" in 18 lines
  across 15 files (Go lib, TS package, docs site, openspec specs, six service
  middlewares). **No `filelog` receiver was ever deployed** — the AOH gateway
  collector has only an `otlp` receiver. Meanwhile the OTLP-native alternative,
  `aohotel.ZapCore`, had **zero callers repo-wide**. `Init` dutifully built a full
  OTLP `LoggerProvider` that nothing ever wrote to.

  **The fix is one seam, not six edits.** All six `aohotel.Init` callers log through
  the same package-level `aohlog` logger, so:

  - `logger`: new `SetExtraCoreFactory` registers a factory whose core `newZapLogger`
    tees alongside stdout. `logger` gains **no** OpenTelemetry dependency — the
    factory is just a func, so the package stays zap-only.
  - `otel`: `Init` registers `ZapCore(cfg.ServiceName)` through that seam, and
    detaches it on shutdown so late log calls hit stdout rather than a dead exporter.

  Honours the standard **`OTEL_LOGS_EXPORTER=none`** to disable log export (provider
  and bridge both skipped) rather than inventing an AOH-specific flag. Still a total
  no-op when `OTEL_EXPORTER_OTLP_ENDPOINT` is unset.

  Two silent-failure traps found by inspection and each pinned by a mutation-verified
  test:

  1. **Rebuild trap.** `SetDevelopment`/`SetProduction` discard and rebuild
     `defaultLogger`, and `wfe-engine` calls `SetDevelopment` _after_ `Init`. A
     one-shot tee is silently dropped there — that service's logs would simply be
     absent from the backend, with no error anywhere. Hence a factory re-consulted on
     every build, not a core applied once.
  2. **Level trap.** `otelzap`'s core reports `Enabled()==true` at every level, so an
     ungated tee exports DEBUG while stdout sits at Info — silently multiplying
     backend volume and cost. The tee is wrapped in `NewIncreaseLevelCore` at the
     logger's own level, so both sinks always agree.

  Verification: 11 tests added. Mutation-proven, not assumed — reverting to a
  one-shot tee fails 6 of them; gating at Debug instead of the logger's level fails
  exactly the level guard; removing the bridge registration fails exactly the
  installation guard. All 8 Go modules in `go.work` build against the change.

  Also corrects `otel/log.go`'s own doc comments, which recommended the filelog path
  that never existed and steered readers away from `ZapCore` — the advice that left
  logs dead.

  **Scope limit — logs are exported, but NOT natively trace-correlated.** Measured,
  not assumed (`otel/logcorrelation_test.go` pins all three cases):

  | how services log today                                             | native OTLP `TraceID` on the record              |
  | ------------------------------------------------------------------ | ------------------------------------------------ |
  | `aohlog.Info(msg)`                                                 | zero                                             |
  | `aohlog.Info(msg, TraceContextFields(ctx)...)` ← every AOH service | **zero** (ids present only as string attributes) |
  | `aohlog.Info(msg, zap.Any("ctx", ctx))`                            | correct trace id                                 |

  `otelzap` defaults its emit context to `context.Background()` and only adopts a span
  when a field of type `context.Context` is present. `TraceContextFields` produces
  plain zap fields, so records arrive with `trace_id`/`span_id` as ordinary attributes
  while the **native** field — the one backends join logs to traces on — stays zero.

  That is a real limitation of this change, and it is a limitation of `aohlog`'s API:
  `Info(msg, fields...)` takes no `context.Context`, so the seam has no context to
  pass. Closing it needs ctx-aware helpers (`aohlog.InfoCtx(ctx, …)`) threaded through
  call sites — a separate change, filed as a follow-up rather than bolted onto an
  already wide-blast-radius PR.

  Note this also contradicts several docs asserting "the `otelzap` bridge injects
  `trace_id`/`span_id` into every `aohlog` record — do not hand-roll log
  correlation". Wrong twice over: the bridge does not auto-correlate, and every
  service does hand-roll. Corrected in a companion docs-truth change.

  **Second scope limit — `aohlog.Fatal`/`Fatalf` records do not reach the backend.**
  Verified in the dependency source, not inferred: zap's default terminal hook is
  `WriteThenFatal`, which calls `os.Exit(1)` immediately after the core write, so the
  deferred `Init` shutdown never runs and the `BatchProcessor` never flushes. Syncing
  would not save it either — `otelzap`'s `Core.Sync()` is `return nil`, a no-op
  despite its "flushes buffered logs" doc comment. The record is handed to the bridge
  and then dies in the queue at exit.

  Fatal is exactly the record you most want in the backend, so this is a real gap.
  Closing it needs the process-exit path to flush the `LoggerProvider`, which `logger`
  cannot reach — the factory seam passes a core, not a provider — so it needs a second
  seam (a flush hook `otel.Init` registers, honoured by a `zap.WithFatalHook`).
  Follow-up, not bolted onto this PR. Until then, treat stdout as the source of truth
  for fatal exits; `Error` immediately before a deliberate exit does export.

  **Not yet proven end-to-end:** rows landing in ClickHouse needs this merged, images
  rebuilt, and the deploy plane rolled out. The remaining ~16 stale `filelog` claims
  elsewhere in the repo (service middleware comments, docs site, openspec specs) are
  that same follow-up docs-truth change, not this one.

- 1018447: feat: AOH error contract — the `aoh-golib` and `ui` halves (AOH-3985)

  Split out of `aoh-error-contract` so the UNH 2.0.0 release could ship without
  also releasing `@mssfoobar/ui` and `@mssfoobar/aoh-golib`. Same change, same
  body — only the packages differ. `@mssfoobar/errors` had to ship with UNH (it is
  a runtime dependency of `unh-client`/`unh-web-sdk` and had never been released);
  these two did not, because `ui` is a peer that resolves to the already-published
  1.0.0 and no UNH package imports `ui/error-surface`.

  Ships the platform error contract on top of the OTEL foundation, as a shared substrate plus one
  end-to-end reference implementation on `unh`. Entirely additive: the legacy `aoh-golib` error
  envelope and its 363 call sites keep their exact behaviour and are deprecated-but-supported.

  **`@mssfoobar/aoh-golib`** — new `aoherr` package: an error-class enum with the canonical
  class→status map (system is **500**), a validated `errorCode` type
  (`^[A-Z][A-Z0-9]*(_[A-Z0-9]+)*$`), a `TraceID(ctx)` helper, and a renderer emitting
  `{timestamp, trace_id, errorCode, errorMessage, details}` that owns the single log record
  (4xx→`WARN`, 5xx→`ERROR`) and enforces internal-detail suppression so no call site can leak a
  cause. Closes the empty-body hole centrally: `aohhttp.Recoverer` (drop-in for chi's
  `middleware.Recoverer`), conformant `NotFoundHandler`/`MethodNotAllowedHandler`, and `BearerAuth`
  now renders a conformant 401 instead of a bare status. `aohlog` gains context-aware emitters that
  inject `trace_id`/`span_id` from the active span automatically, and the legacy envelope gains an
  additive optional `trace_id` so correlation does not wait for the per-service migration.

  **`@mssfoobar/errors`** (new) — the service envelope type, a tolerant `parseErrorEnvelope`, a
  typed `AohError`, `normalizeError` for non-envelope failures, and the
  `errorCode → { userMessage, isRetryable }` catalogue seam with exact-code → module-prefix →
  generic resolution. The catalogue ships empty by design; population is tracked separately.

  The BFF seam is `toClientError`, which **redacts and translates**: it forwards `timestamp`,
  `trace_id` and `errorCode`, adds `userMessage` and `isRetryable`, and drops `errorMessage` and
  `details` so neither can travel past the BFF. A browser therefore cannot render developer detail
  because it never receives it, rather than because a convention says it must not — review of this
  change found that convention failing in three separate places, one feeding 205 `toast.error()`
  call sites. `errorCode` and `timestamp` are deliberately kept: the code lets a client branch on
  the failure, and together they are the reportable tuple an operator quotes when no span is
  active. Known gap, tracked as AOH-8334: per-field inline form errors need `details` in the
  browser, so a 400 now presents as its `userMessage` alone.

  **`@mssfoobar/ui`** — an `ErrorSurface` component presenting the user-facing message, a Retry
  action only when the failure is retryable, and `trace_id` as a copyable support reference inside a
  progressive-disclosure block. With no active trace the disclosure is hidden entirely rather than
  showing a placeholder.

  **`@mssfoobar/unh-types` / `unh-client` / `unh-web-sdk`** — the reference consumer half. The
  envelope type is shared rather than redeclared, and `UnhHttpError` exposes
  `errorCode`/`details`/`trace_id` plus the presentation fields. The BFF proxy drops its hand-minted
  `x-correlation-id` in favour of W3C `traceparent` propagation and enriches downstream errors.

  **BREAKING (`@mssfoobar/unh-web-sdk`)** — the proxy's error response body no longer contains
  `errorMessage` or `details`. Any browser code reading either from a proxied failure now gets
  `undefined`. That is the intended effect, but it is an observable change to what the package
  emits, so it is a major rather than a minor.

  **BREAKING (`@mssfoobar/unh-client`)** — via the BFF proxy, `UnhHttpError` no longer reports an
  `errorMessage` sourced from the response or any `details`, because the proxy no longer sends
  them. Both remain populated when the client is pointed straight at a service (a server-side
  use). Additionally, `UnhHttpError.message` is now the **user-facing** string.
  It previously derived from the response body (`body.message ?? body.errors[0].message`), i.e. the
  developer-facing text. Any consumer doing `toast.error(err.message)` — the dominant pattern, at 205
  sites — will now show operator-appropriate copy instead of a raw service message, which is the
  point, but it _is_ a silent change in what an end user sees and so is a major. Read `errorMessage`
  if you specifically want the developer-facing string. Additionally, every rejection from the client
  is now a `UnhHttpError`: a transport failure or a non-JSON 2xx body previously escaped as a bare
  `TypeError`/`SyntaxError`, so code that pattern-matched on those will no longer match.

  **`@mssfoobar/agent-skills`** — reconciles the two contradictory in-repo error documents that were
  the root cause of the divergence, and updates the Go and web scaffolds so newly generated services
  and apps are born conformant.

### Patch Changes

- 4d28f83: test(aoh-golib): pin the legacy error envelope with a golden fixture (AOH-3985)

  Adds `packages/aoh-golib/http/testdata/legacy-envelope.golden.json` and the byte
  comparison against it, which the archived `error-envelope` spec mandated but the
  implementation never shipped.

  This is the only thing in the repo that pins the promise the whole error-contract change
  rests on: that the legacy `{data, message, sent_at, errors}` envelope — emitted at ~356
  call sites across five unmigrated services and parsed by four published `*HttpError`
  classes — did not change shape when it gained its additive optional `trace_id`.

  A byte comparison rather than field assertions is deliberate: those call sites have no
  other test coverage, so the failure mode to catch is a _silent_ alteration (a renamed
  json tag, a dropped `omitempty`, a reordered field). Any of those is a breaking change
  for every SDK consumer and none would fail a test that merely checked a field was
  present. Mutation-verified — dropping `omitempty` from the added `trace_id` fails both
  new tests.

- 1ef0f18: Two fixes on the authentication path every Go service already uses. No API
  change, so this is a patch — but both are reachable in production today.

  **A panic on any bearer token without a `sub` claim.** `isValidJWT` compared the
  userinfo `sub` through an unchecked type assertion (`parsed["sub"].(string)`), so
  a token missing that claim panicked rather than being rejected. `aohhttp.Recoverer`
  turned it into a 500, which means a malformed credential was reported as a server
  fault instead of a rejected one — on attacker-controlled input, on every service,
  at the cost of a panic and recover per request. Now checked, with a regression
  test that fails (on the panic) if the assertion is restored.

  **`OpenIDCache` ignored its `issuerUrl` argument.** It held a single discovery
  document for the whole process, so the first discovery won and a second issuer
  would silently receive the first one's endpoints. Entries are now keyed by
  issuer. Nothing in AOH runs two issuers in one process today, but the package's
  own tests had to document and work around this ("one issuer, one discovery"),
  which is a fair sign it would eventually bite something real. No exported API
  changed — every field involved is unexported and both method signatures are
  unchanged.

  Two known defects on this path are deliberately NOT addressed here and are
  tracked instead, because both predate this change and neither has a small fix:
  discovery has no single-flight and no failure backoff, so a cold or failing
  issuer costs one HTTP GET per request (AOH-8738); and `DiscoverWellKnownConfig`
  still holds the cache mutex across that GET, so one slow issuer stalls every
  authentication in the process (same issue).
