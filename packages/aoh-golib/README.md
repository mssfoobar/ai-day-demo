# aoh-golib

Shared Go foundation library for AOH services (errors, http middleware, logger,
postgres, nats, temporal, graphql, otel).

```go
import "github.com/mssfoobar/ops-hub/packages/aoh-golib/logger"
```

## Errors

`aoherr` owns the AOH error contract: an error class, a validated `errorCode`,
and one renderer that derives the HTTP status from the class, stamps the active
span's `trace_id`, suppresses internal detail on 5xx, and emits exactly one log
record at the level the status implies (4xx `WARN`, 5xx `ERROR`).

```go
import "github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"

var ErrTemplateNotFound = aoherr.New(
	aoherr.ClassNotFound, aoherr.MustCode("UNH_TEMPLATE_NOT_FOUND"), "template not found")

// in a handler, on any error:
aoherr.Render(w, r, err) // status, trace_id, suppression and the one log record
```

Layers below the handler must not log the failure they return — the renderer is
the single logging point. Use `aohlog.Ctx(ctx)` (or `aohlog.ErrorCtx`) for every
other record, so `trace_id`/`span_id` are injected from the active span instead
of splatted by hand.

The legacy `aohhttp.ErrResponse` envelope keeps working unchanged — it gains only
an additive `trace_id` — and is deprecated for new code.

Mount the shared middleware so no failure escapes with an empty or plain-text
body:

```go
r.Use(aohhttp.Recoverer)                              // conformant 500 on panic
r.NotFound(aohhttp.NotFoundHandler())                 // conformant JSON 404
r.MethodNotAllowed(aohhttp.MethodNotAllowedHandler()) // conformant JSON 405
```

## Versioning

Monorepo services resolve it locally via `go.work` and a `go.mod` `replace`
directive. External consumers fetch it from this repository — versions are the
`packages/aoh-golib/vX.Y.Z` git tags:

```bash
GOPRIVATE=github.com/mssfoobar/* go get github.com/mssfoobar/ops-hub/packages/aoh-golib@v0.2.0
```

Formerly the standalone `github.com/mssfoobar/aoh-golib` repository (archived);
its old tags (≤ v0.1.x) remain fetchable from there under the old import path.
