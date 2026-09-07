package aohotel

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

// HTTPMiddleware returns a chi-compatible middleware (func(http.Handler)
// http.Handler) that opens a server span and records HTTP server metrics for
// each request via otelhttp. chi handlers are stdlib http.Handlers, so the
// official otelhttp instruments them directly — no third-party otelchi needed.
//
// Mount it as the outermost middleware (r.Use(aohotel.HTTPMiddleware("svc"))).
// Pair with ChiRouteSpanName, mounted after the router has matched, to give
// spans low-cardinality, route-pattern names.
func HTTPMiddleware(service string, opts ...otelhttp.Option) func(http.Handler) http.Handler {
	return otelhttp.NewMiddleware(service, opts...)
}

// ChiRouteSpanName renames the active server span to "<METHOD> <route pattern>"
// (e.g. "GET /widgets/{id}") once chi has resolved the route, keeping span
// names low-cardinality instead of embedding path parameters. Mount it inside
// the chi router (after HTTPMiddleware) so the route context is populated.
func ChiRouteSpanName(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		rc := chi.RouteContext(r.Context())
		if rc == nil {
			return
		}
		if pattern := rc.RoutePattern(); pattern != "" {
			trace.SpanFromContext(r.Context()).SetName(r.Method + " " + pattern)
		}
	})
}
