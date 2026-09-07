package aohhttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"
	"go.uber.org/zap"
)

// NotFoundHandler renders the conformant AOH 404 for a path no route matched.
// Mount it on the router — r.NotFound(aohhttp.NotFoundHandler()) — in place of
// chi's default, which writes the plain-text body "404 page not found".
func NotFoundHandler() http.HandlerFunc {
	return NotFoundHandlerWithLogger(nil)
}

// NotFoundHandlerWithLogger is NotFoundHandler for services holding their own
// logger; nil means the package default, resolved per request.
func NotFoundHandlerWithLogger(logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		aoherr.RenderWithLogger(logger, w, r, aoherr.NotFoundError())
	}
}

// MethodNotAllowedHandler renders the conformant AOH 405 when the path matched a
// route that does not allow the request's method. Mount it as
// r.MethodNotAllowed(aohhttp.MethodNotAllowedHandler()); chi's default writes
// the plain-text body "405 method not allowed".
func MethodNotAllowedHandler() http.HandlerFunc {
	return MethodNotAllowedHandlerWithLogger(nil)
}

// MethodNotAllowedHandlerWithLogger is MethodNotAllowedHandler for services
// holding their own logger; nil means the package default, resolved per request.
func MethodNotAllowedHandlerWithLogger(logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		aoherr.RenderWithLogger(logger, w, r, aoherr.MethodNotAllowedError())
	}
}

// chiRoutePattern is the low-cardinality route the router matched, or "" when
// nothing matched.
func chiRoutePattern(r *http.Request) string {
	rc := chi.RouteContext(r.Context())
	if rc == nil {
		return ""
	}
	return rc.RoutePattern()
}
