// Package handler is the HTTP layer: routing, and rendering the AOH wire contract.
package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"
	aohhttp "github.com/mssfoobar/ops-hub/packages/aoh-golib/http"

	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/domain"
)

// UnitLister is the service surface the handler needs.
type UnitLister interface {
	List(ctx context.Context) ([]domain.Unit, error)
	Get(ctx context.Context, unitCode string) (domain.Unit, error)
}

type UnitHandler struct {
	units UnitLister
}

func NewUnitHandler(units UnitLister) *UnitHandler {
	return &UnitHandler{units: units}
}

// Routes mounts the unit resource. Paths are /v1/<resource> with no /api prefix, per
// aoh-conventions.
func (h *UnitHandler) Routes(r chi.Router) {
	r.Get("/", h.list)
	r.Get("/{unit_code}", h.get)
}

func (h *UnitHandler) list(w http.ResponseWriter, r *http.Request) {
	units, err := h.units.List(r.Context())
	if err != nil {
		aoherr.Render(w, r, err)
		return
	}
	// Always the AOH envelope — never a bare array.
	render.Render(w, r, aohhttp.Response(http.StatusOK, "", units)) //nolint:errcheck
}

func (h *UnitHandler) get(w http.ResponseWriter, r *http.Request) {
	unit, err := h.units.Get(r.Context(), chi.URLParam(r, "unit_code"))
	if err != nil {
		aoherr.Render(w, r, err)
		return
	}
	render.Render(w, r, aohhttp.Response(http.StatusOK, "", unit)) //nolint:errcheck
}
