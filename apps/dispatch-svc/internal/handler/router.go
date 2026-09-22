package handler

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	aohhttp "github.com/mssfoobar/ops-hub/packages/aoh-golib/http"
	"go.uber.org/zap"

	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/auth"
)

// Router builds the service's HTTP routes.
//
// readiness is probed by /readyz; it should return an error when the database is
// unreachable, so a broken dependency shows up there rather than as a 500 on a data
// route.
//
// issuerURL is the Keycloak realm every bearer token must come from. Authentication is
// mounted on /v1/units only: /livez and /readyz stay open, because Kubernetes probes them
// without credentials.
func Router(units *UnitHandler, readiness func(ctx context.Context) error, issuerURL string, logger *zap.Logger) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)

	// aoh-golib's fallbacks: a panic, an unmatched route and a disallowed method all
	// render the AOH error contract instead of a bare 404/405 with an empty body.
	r.Use(aohhttp.Recoverer)
	r.NotFound(aohhttp.NotFoundHandler())
	r.MethodNotAllowed(aohhttp.MethodNotAllowedHandler())

	// Health probes live at the root — Kubernetes expects them there, and they are
	// unauthenticated by design.
	health := aohhttp.NewHealthCheck(r)
	health.AddLivenessCheck("process", func() error { return nil })
	health.AddReadinessCheck("database", func() error {
		return readiness(context.Background())
	})

	// Authentication, mounted inside the resource rather than globally so the health
	// probes above stay reachable. BearerAuth is aoh-golib's shipped middleware — it
	// validates against Keycloak's userinfo endpoint, which is also why a token issued
	// without `scope=openid` is rejected. auth.Identify then puts the caller's subject,
	// tenant and roles on the request context.
	//
	// The four workshop stub routes are inside this group too: they keep answering 501,
	// but only to an authorised caller.
	r.Route("/v1/units", func(r chi.Router) {
		r.Use(aohhttp.BearerAuth(issuerURL, logger))
		r.Use(auth.Identify)
		units.Routes(r)
	})

	return r
}
