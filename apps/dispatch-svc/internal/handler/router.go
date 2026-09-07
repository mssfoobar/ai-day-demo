package handler

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	aohhttp "github.com/mssfoobar/ops-hub/packages/aoh-golib/http"
)

// Router builds the service's HTTP routes.
//
// readiness is probed by /readyz; it should return an error when the database is
// unreachable, so a broken dependency shows up there rather than as a 500 on a data
// route.
func Router(units *UnitHandler, readiness func(ctx context.Context) error) *chi.Mux {
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

	r.Route("/v1/units", units.Routes)

	return r
}
