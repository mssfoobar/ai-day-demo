---
name: aoh-error-handling
description: >
  AOH error and logging conventions across Go microservices, BFF, and
  TS/Svelte frontend. Owns the error wire contract: the five-field
  envelope (timestamp, trace_id, errorCode in UPPER_SNAKE_CASE,
  errorMessage, details), the class → HTTP-status map, the errorCode
  format and namespacing, trace_id (OTEL) propagation, the BFF
  redaction (drops errorMessage/details, keeps errorCode), the frontend
  error surface, log levels (4xx WARN / 5xx
  ERROR, logged once by the renderer), and the shipped API
  (`aoherr`, `aohhttp` middlewares, `@mssfoobar/errors`,
  `@mssfoobar/ui/error-surface`). **Always consult before designing an
  error response, choosing a log level, propagating errors across
  service boundaries, or implementing frontend error UI — getting it
  wrong leaks raw errors and breaks cross-service debugging.** Trigger
  on "how do I format this error response", "what errorCode for X",
  "what log level", "WARN or ERROR", "trace_id propagation",
  "correlationId", "BFF error translation", "retry logging".
---

# AOH error handling and logging

The AOH error story is a deliberate end-to-end contract: a single user action
can traverse multiple services, and every error must be traceable,
machine-readable, and presented safely to the operator. Get the contract right
and a support ticket that arrives with one `trace_id` lights up the entire
request trail. Get it wrong and you spend hours grepping logs across services.

This skill is the **single source of truth for error responses**. `aoh-conventions`
› `api.md` owns the *success* envelope and points here for failures; it no longer
restates an error shape of its own.

## Core principles

- **Clarity & consistency** — errors are handled and presented uniformly, regardless of where they originate.
- **Actionability** — error messages should empower users and developers to take the next appropriate step.
- **No sensitive data leakage** — raw system errors, stack traces, internal config must NEVER reach end-users.
- **Traceability** — every error must be traceable across the entire request lifecycle via `trace_id` (the OpenTelemetry trace id, shared by the error, its logs, and its distributed trace).
- **Graceful degradation** — services should fail predictably and not cause cascading failures.
- **Enforced, not requested** — suppression, status derivation, log level and correlation are properties of the shared renderer, so a call site cannot get them wrong.

## The error envelope

Every error response a service renders through the contract is this JSON object,
on any 4xx or 5xx:

```json
{
  "timestamp": "2026-07-29T03:15:08.123Z",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "errorCode": "UNH_TEMPLATE_NOT_FOUND",
  "errorMessage": "No notification template exists for the supplied id.",
  "details": [{ "field": "template_id", "message": "must reference an existing template" }]
}
```

| Field | Required | Format | Purpose |
|---|---|---|---|
| `timestamp` | ✓ | RFC 3339 UTC, millisecond precision | When the error occurred |
| `trace_id` | when a span is active | 32-hex OpenTelemetry trace id | Correlation key — joins the error to its logs and distributed trace |
| `errorCode` | ✓ | `UPPER_SNAKE_CASE` (see format below) | Machine-readable; the key clients branch on and the catalogue is keyed by |
| `errorMessage` | ✓ | Free text, non-empty | Developer-focused; goes to logs, NOT to end-users |
| `details` | optional | array of objects | Contextual info — which field, which identifier, which constraint |

**These five keys are the whole envelope.** `trace_id` and `details` are
*omitted* when absent — never `null`, `""` or `[]`. A service never emits
`userMessage` or `isRetryable`: those are presentation fields, added only by the
BFF, so "the envelope" and "the client payload" stay two separately-testable
shapes.

**`details` is an array of objects, and carries no control fields.** One entry per
thing you are describing (per-field validation failures being the common case) —
a map cannot express a list without inventing index keys. Entries are *descriptive
only*: `isRetryable` and `action` do not belong inside them, and the Go renderer
strips those keys however they are spelled.

The `errorMessage` is for developers reading logs. The end-user sees a separate
`userMessage` produced by the BFF — never `errorMessage` directly, not even as a
fallback for an unmapped code.

> **`correlationId` is retired.** It was a BFF-generated UUID-v4 propagated via an
> `x-correlation-id` header. The field is now `trace_id` — the OpenTelemetry trace
> id, established at the edge and propagated via W3C `traceparent` — and services
> neither mint nor forward an id of their own. Consumers reading `correlationId`
> switch to `trace_id`.

## HTTP status mapping by error class

The status is derived from the **class**, by the renderer. It is never chosen per
call site.

| Class | Examples | Status |
|---|---|---|
| Validation | malformed body, bad date format, missing required field | 400 |
| Business rule | a well-formed request that violates a domain rule | 400 |
| Authentication | missing, malformed or expired credential | 401 |
| Authorization | authenticated but not permitted | 403 |
| Not found | resource absent, or invisible to the caller's tenant | 404 |
| Conflict | stale `occ_lock`, duplicate key | 409 |
| System / internal | DB failure, bug, recovered panic | 500 |
| Upstream unavailable | dependency unreachable or timed out | 503 |

Two traps this table closes: an internal failure is **500, never 503** (503 means
*someone else* is down), and a failure that must present as 409 is a **conflict**,
not a business rule with an overridden status.

## errorCode format, namespacing and stability

`errorCode` MUST match:

```text
^[A-Z][A-Z0-9]*(_[A-Z0-9]+)*$
```

So `CONFLICT`, `UNH_TEMPLATE_NOT_FOUND` and `DB_TIMEOUT_2` are valid;
`unh_not_found`, `1_BAD_CODE`, `TRAILING_`, `DOUBLE__UNDERSCORE`, `HAS-HYPHEN`
and lowercase prose like `database error` are not. Validation happens at
construction, so a bad code fails in your own tests instead of on a client's
screen.

- **Namespace by module** — `UNH_TEMPLATE_NOT_FOUND`, not `NOT_FOUND`, so a
  client can tell one module's 404 from another's.
- **A small shared set is platform-owned** for genuinely cross-cutting failures —
  authentication, authorization, malformed request, unhandled internal failure,
  unmatched route, disallowed method — plus a per-class fallback the renderer uses
  when a call site supplied no code. Don't reach for these for a domain failure.
- **A published code is part of the API.** Once a client branches on it, renaming
  or removing it is a **breaking change** (a major bump in the changeset), and a
  retired value is never reassigned to a different failure.
- Be specific. `INSUFFICIENT_PERMISSIONS` beats `FORBIDDEN`; `DATABASE_TIMEOUT`
  beats `INTERNAL_ERROR`.

## trace_id propagation

The `trace_id` is the single most important piece of operational data in the system — the one id that joins an error to its logs and its distributed trace. Without it, support tickets are nearly impossible to debug.

It comes from OpenTelemetry, so you do **not** generate or forward it by hand:

- **Establishment**: the OTEL HTTP instrumentation opens a server span at the system edge and assigns the `trace_id` (adopting an inbound `traceparent` if one exists). No UUID generation.
- **Propagation**: context flows to every downstream service automatically via the W3C `traceparent` header (the OTEL HTTP/fetch instrumentation injects it) — there is no `x-correlation-id` header to set. The same `trace_id` therefore appears in every service's spans.
- **Logs**: correlation is the job of the *emitter*, not the call site. Node/Svelte auto-injects `trace_id`/`span_id` via the `@mssfoobar/logger` OTEL mixin registered by `startObservability`. In Go the `otelzap` bridge **exports** records but does not correlate them — it adopts a span only when a `context.Context` field is passed (corrected AOH-7540; this file previously claimed Go auto-injected). Satisfy that with the context-aware `aohlog` helpers — `aohlog.Ctx(ctx)`, `aohlog.ErrorCtx(ctx, …)`, `aohlog.WithCtx(ctx, logger)` — which read the active span and attach the pair for you (AOH-3985). `aohotel.TraceContextFields(ctx)` remains for a caller assembling a field slice by hand. Either way, don't rely on remembering: that is why no Go error record carried a trace id.
- **Error responses**: each carries the active span's `trace_id` (stamped at render time on the service; surfaced/forwarded by the BFF).
- **Frontend display**: expose it to the user in an "Error Details" / "Advanced" section as the support id:

**When there is no active span, `trace_id` is omitted — and nothing is minted in
its place.** No fallback UUID, no request id, no literal `"unknown"`. An
identifier that correlates to nothing is worse than none, because it looks
actionable. The UI degrades accordingly: it hides the support-reference block
entirely and presents `errorCode` + `timestamp` as the reportable tuple, which is
enough to find the log record. Environments with OTLP export switched off
therefore have no support id, visibly — that is honest signal, not a bug to paper
over.

See `aoh-conventions` › observability for the instrumentation wiring.

## Layer-specific guidance

### Backend microservices

Per error condition:

1. **Classify** — pick the class from the table above and a specific,
   module-namespaced `errorCode`. That is the whole decision; the status follows.
2. **Attach the cause and any descriptive details** and hand the error to the
   shared renderer.
3. **Do not log it, and do not sanitize it.** The renderer is the single logging
   point and owns suppression: it emits exactly one record at the level the status
   implies, with the cause and stack on a 5xx, and it guarantees no internal
   detail reaches the body even if you passed the raw error in.
4. **Don't double-log up the call stack.** Repositories, services and wrappers
   propagate; only the rendering layer reports. Add context by wrapping the
   error, not by logging it on the way up.

### Backend for Frontend (BFF)

The BFF is the translation layer between technical microservice errors and
user-facing messages. **It redacts and translates.**

**`trace_id` is automatic**: the OTEL instrumentation establishes the trace and propagates `traceparent` to downstream calls — you don't pull, generate, or forward a correlation id.

1. **Record the original failure** — the unmodified upstream status and payload
   (truncated, not discarded), *before* redacting, at the level the status you
   will return implies (4xx → `WARN`, 5xx and no-response → `ERROR`). This is
   what makes cross-service debugging possible: the record shares the trace with
   the upstream service's own record, and it is the *only* place the developer
   detail now survives.
2. **Resolve the presentation fields** from the catalogue, keyed by `errorCode`
   (exact code, then module prefix, then a generic fallback).
3. **Return the redacted payload** — the three machine identifiers, plus the two
   presentation fields. `errorMessage` and `details` are dropped and go no
   further than the BFF:

```json
{
  "timestamp": "2026-07-29T03:15:08.123Z",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "errorCode": "UNH_TEMPLATE_NOT_FOUND",
  "userMessage": "That notification template no longer exists. Pick another one.",
  "isRetryable": false
}
```

**Why the developer fields are dropped rather than merely not-rendered.** A
browser cannot leak what it never received. An earlier draft of this contract
forwarded `errorMessage`/`details` and relied on every author knowing which of
two similarly-named fields was safe to show; review of AOH-3985 found that
failing in three separate places, one of which fed 205 `toast.error()` call
sites. Redaction makes the mistake unrepresentable.

**But `errorCode` and `timestamp` MUST survive the BFF.** This document used to
specify a payload of `{userMessage, trace_id, isRetryable}`, dropping both.
That is a defect, for two reasons. A frontend that receives no code cannot branch
on the failure — which is why the one service to adopt the shape discarded the
BFF step and reimplemented translation as a hard-coded if/else chain in the
browser. And with no active span, `errorCode` + `timestamp` is the *only*
reportable tuple an operator has; dropping them leaves an untraced failure with
nothing to quote, in exactly the environments where OTLP export is not yet wired.

**Known gap.** Per-field inline form errors need `details` in the browser, so
they are not currently possible. A `400` presents as its `userMessage` alone
until a deliberate channel for the validation subset is designed.

`isRetryable` is a single **top-level** boolean, resolved once, in a fixed order:
an explicit value already on the payload received, then the catalogue entry for
the exact code, then the entry for its module prefix, then the default derived
from the failure class (transport, timeout, upstream-unavailable and internal
failures retryable; validation, authentication, authorization, not-found and
conflict not). It is never read from inside `details`. The common case therefore
needs no decision, and an override is a catalogue entry rather than an argument
at every call site.

**BFF-native failures** — connection refused, DNS failure, timeout, aborted
transport — get a conformant error of the BFF's own, with a module-namespaced
code and `isRetryable: true`. The transport exception never reaches the client.

**Anything that is not an envelope** — an HTML error page, a plain-text body,
invalid JSON, a zero-length body, JSON of the wrong shape — is normalized into
the same shape. No raw body, parser message or exception text reaches the user.

### Frontend (Web UI)

**Display**: use the shared error surface, not a per-page error UI. It renders
`userMessage` as the primary text, offers a Retry control only when
`isRetryable` is `true` *and* a handler was supplied, and exposes `trace_id` as a
copyable support reference inside a collapsed disclosure block. Keep inline
form-validation errors next to their field.

**Never render `errorMessage`, `details`, exception text or a stack trace.**
`userMessage` is the only failure string a user sees.

**No trace, no support block** — the surface renders `errorCode` + `timestamp`
instead. It never renders an empty reference or the literal `unknown`.

**Unhandled exceptions** reaching the framework error hook are converted to the
same conformant shape before they reach the error page, so the page can show a
support reference and never the exception message.

**There is no client-side exception sink yet.** Sentry (or an equivalent) needs a
vendor decision, and `@mssfoobar/logger` goes silent in the browser in
production — so a browser-only `ERROR` currently reaches nothing. Don't wire one
ad-hoc per app; the platform decision is tracked separately.

## Log levels

Production runs at `INFO`. For error responses the level is **derived from the
status**, by the renderer: `400–499` → `WARN`, `500–599` → `ERROR`. Never per
call site, and never a 4xx at `ERROR` — a rejected credential or a failed
validation is not an alert.

| Level | When | Example |
|---|---|---|
| `DEBUG` | Diagnose program state during dev | "request received, show all inputs" |
| `INFO` | Program is functioning normally | "server ready to listen to requests" |
| `WARN` | Potential issues; may need investigation. Includes every 4xx response and every retried-and-recovered attempt | "no host name supplied, using default" |
| `ERROR` | Unexpected errors; investigate ASAP. Includes every 5xx response | "failed to connect to database" |

`TRACE` and `FATAL` are framework-dependent. `TRACE` is the lowest level (most
verbose); `FATAL` is highest (always printed regardless of configured level).

An access-log middleware may still record every response, but only as **bounded
metadata** — status, method, route, duration, size. It must not carry the
`errorCode`, the cause, or the response body, or it becomes a second report of a
failure the renderer already logged.

## Language-specific

### Go (backend microservices)

`packages/aoh-golib/aoherr` is the vocabulary and the renderer. Use it; don't
hand-assemble a near-conformant body.

```go
import (
    "github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"
    aohhttp "github.com/mssfoobar/ops-hub/packages/aoh-golib/http"
    aohlog "github.com/mssfoobar/ops-hub/packages/aoh-golib/logger"
)

// Package-level codes: MustCode panics at init on a non-conforming value.
var codeTemplateNotFound = aoherr.MustCode("UNH_TEMPLATE_NOT_FOUND")

// In the service layer — classify, wrap the cause, don't log.
if errors.Is(err, repo.ErrNotFound) {
    return aoherr.Wrap(aoherr.ClassNotFound, codeTemplateNotFound,
        "no template for the supplied id", err)
}

// In the handler — one call renders the response and emits the one log record.
if err != nil {
    aoherr.Render(w, r, err)   // status from the class, trace_id from the span
    return
}
```

- `aoherr.New` / `Wrap` construct; `WithDetails(aoherr.FieldDetail("name", "must not be empty"))`
  and `WithCause` decorate a copy (so a package-level sentinel is safe to reuse);
  `From` classifies an unclassified error as a suppressed 500 rather than leaking it.
- `aoherr.Render(w, r, err)` writes the body and logs; `RenderWithLogger` is the
  same for a middleware holding its own `*zap.Logger`.
- `aoherr.TraceID(ctx)` returns the active trace id, or `""` for absent.

Close the empty-body holes centrally rather than per handler:

- `r.Use(aohhttp.Recoverer)` — a drop-in for chi's `middleware.Recoverer`, which
  turns a panic into a bare 500 with a zero-length body. The AOH one records the
  panic on the span, logs it once with the stack, and renders a conformant 500
  carrying the `trace_id`.
- `r.NotFound(aohhttp.NotFoundHandler())` and
  `r.MethodNotAllowed(aohhttp.MethodNotAllowedHandler())` — chi's defaults write
  plain text.
- `aohhttp.BearerAuth` already renders a conformant 401 for a rejected token.

Logging uses `aohlog` — **never the standard `log` package**:

```go
aohlog.Info("server starting", zap.String("port", "8080"))
aohlog.ErrorCtx(ctx, "cache rebuild failed", zap.Error(err)) // trace-correlated
```

Inside a request use the context-aware forms (`Ctx`, `WithCtx`, `DebugCtx`,
`InfoCtx`, `WarnCtx`, `ErrorCtx`) so the record carries `trace_id`/`span_id`
without any field-splatting. `aohlog.SetDevelopment()` gives human-readable
output; `SetProduction()` gives JSON.

**`FATAL` only at `main()`.** It calls `os.Exit(1)`, which skips defers, doesn't
flush buffers, and skips temp-file cleanup. Anywhere else, return the error.

### TypeScript / Svelte

`@mssfoobar/errors` is the TS half of the contract:

- `parseErrorEnvelope(body)` — tolerant parse; keeps only fields that were
  present *and* conformant.
- `normalizeError({ status, body, contentType, cause }, opts)` → `AohError` —
  handles every non-envelope failure, including no response at all.
- `AohError` — carries the envelope fields plus the resolved `userMessage` and
  `isRetryable`; `toEnvelope()` / `toPayload()` emit the two shapes.
- `toClientError(opts)` → `{ status, payload }` — the BFF seam; redacts. Pass a
  `catalogue`, your `modulePrefix`, and a `logger` (injected, so the package
  takes no logging dependency).
- `resolveUserMessage(errorCode, catalogue)` / `resolveIsRetryable(opts)`, the
  `ErrorCatalogue` type (`Record<string, {userMessage, isRetryable?}>`, keyed by
  code or module prefix), and `GENERIC_USER_MESSAGE`. The catalogue **ships
  empty**; resolution still always yields a non-empty string.
- `ERROR_CODE_PATTERN`, `isErrorCode`, `isTraceId`, `modulePrefixOf`,
  `failureClassForStatus`, `isRetryableByClass` for the checks.

`@mssfoobar/ui/error-surface` is the presentation half:

```svelte
<script lang="ts">
  import { ErrorSurface, type ErrorSurfaceFailure } from "@mssfoobar/ui/error-surface";

  let { failure, retry }: { failure: ErrorSurfaceFailure; retry: () => void } = $props();
</script>

<ErrorSurface {failure} variant="page" onRetry={retry} />
```

`failure` needs `userMessage`; `errorCode`, `timestamp`, `trace_id` and
`isRetryable` are optional and drive the support reference and the Retry control.
`variant` is `"page"` or `"inline"`. The type deliberately has no `errorMessage`
or `details` field — this surface must never render them. `onRetry` re-runs the
failed operation; it must not navigate or reload the page.

Use the project's `Logger` (`@mssfoobar/logger`, pino — fields object first,
message second). **Never `console.log`.** Nothing in the repo's lint config
currently blocks it, so this one is on you. Records are JSON with at minimum a
`level` field (`DEBUG`/`INFO`/`WARN`/`ERROR`).

**Retry semantics**: attempts that get retried log at `WARN`; only the terminal
failure escalates to `ERROR`. That keeps `ERROR` meaningful — alerts shouldn't
fire for transient issues that recovered.

```ts
try {
  result = await callWithRetry(api.fetch, { retries: 3 });
} catch (err) {
  log.warn({ attempt, err }, "fetch retry failed, will retry");
  // ... eventually
  log.error({ totalAttempts, err }, "fetch failed permanently");
}
```

## Anti-patterns

- **Double-logging the same error** as it bubbles up the call stack. The layer that renders is the layer that logs.
- **Logging an error *and* rendering it** in a handler. The renderer already logged it.
- **Pre-sanitizing before rendering**, or picking the HTTP status by hand. Both belong to the renderer.
- **Returning raw stack traces or DB errors in the API response.** Suppression is automatic — don't defeat it by writing the cause into `details`.
- **Dropping `errorCode` at the BFF.** A client that can't read the code can't branch on the failure.
- **Smuggling `isRetryable` or `action` into `details`.** Both are top-level presentation concerns.
- **Manually generating or forwarding a correlation id** (an `x-correlation-id` header, a UUID) — or rendering `"unknown"` when there is no trace. Omit the field and degrade the UI.
- **Putting `errorMessage` in front of users.** That's the developer-facing string; users see `userMessage`.
- **`console.log` in TypeScript.** Nothing lints it out — use the Logger anyway.
- **`ERROR` for retried-and-recovered failures.** That's a `WARN`. Likewise never a 4xx at `ERROR`.
- **Logging tokens, passwords, or full request bodies with PII.** Records carry field names, ids and counts — not submitted values.

## Pre-merge checklist

- [ ] Every new or migrated error response goes through the shared renderer (`aoherr.Render` / the BFF `toClientError` helper) — no hand-assembled body
- [ ] `errorCode` matches `^[A-Z][A-Z0-9]*(_[A-Z0-9]+)*$`, is module-namespaced, and is constructed through the validating constructor
- [ ] Renaming or removing a published `errorCode` carries a major-bump changeset
- [ ] HTTP status comes from the error class, not from the call site
- [ ] `details` is an array of descriptive objects, with no `isRetryable` / `action` / `userMessage` inside
- [ ] The service emits neither `userMessage` nor `isRetryable`; the BFF adds both and forwards `errorCode`
- [ ] `trace_id` is stamped from the active span, omitted when there is none, and never substituted
- [ ] Exactly one record reports the failure, at `WARN` for 4xx and `ERROR` for 5xx, and it carries the trace context automatically
- [ ] No 5xx body contains the cause, a stack frame, a SQL or driver message, a path or a host; no record contains a token, password or whole request body
- [ ] The user-facing surface renders only `userMessage`, offers Retry only when `isRetryable`, and hides the support reference when there is no `trace_id`
- [ ] Panic recovery, unmatched routes and disallowed methods are mounted on the AOH middlewares, so no 4xx/5xx has a zero-length or plain-text body
