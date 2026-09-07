// Package handler is the HTTP layer: routing, and rendering the AOH wire contract.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"
	aohhttp "github.com/mssfoobar/ops-hub/packages/aoh-golib/http"

	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/domain"
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/service"
)

// UnitService is the service surface the handler needs.
type UnitService interface {
	List(ctx context.Context) ([]domain.Unit, error)
	Get(ctx context.Context, unitCode string) (domain.Unit, error)
	Create(ctx context.Context, in domain.UnitInput) (domain.Unit, error)
	Update(ctx context.Context, unitCode string, occLock int, in domain.UnitInput) (domain.Unit, error)
	Delete(ctx context.Context, unitCode string, occLock int) error
}

type UnitHandler struct {
	units UnitService
}

func NewUnitHandler(units UnitService) *UnitHandler {
	return &UnitHandler{units: units}
}

// Bodies larger than this are rejected before decoding — a unit is a few hundred bytes.
const maxBody = 64 << 10

// Routes mounts the unit resource. Paths are /v1/<resource> with no /api prefix, per
// aoh-conventions. PATCH and collection-level PUT/DELETE are deliberately unmounted and
// fall through to the router's 405.
func (h *UnitHandler) Routes(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Get("/{unit_code}", h.get)
	r.Put("/{unit_code}", h.update)
	r.Delete("/{unit_code}", h.delete)
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

func (h *UnitHandler) create(w http.ResponseWriter, r *http.Request) {
	var in domain.UnitInput
	if err := decode(r, &in); err != nil {
		aoherr.Render(w, r, err)
		return
	}
	unit, err := h.units.Create(r.Context(), in)
	if err != nil {
		aoherr.Render(w, r, err)
		return
	}
	render.Render(w, r, aohhttp.Response(http.StatusCreated, "", unit)) //nolint:errcheck
}

// replaceBody is a UnitInput plus the caller's occ_lock. A pointer distinguishes "sent 0"
// from "not sent" — a missing lock is a 400, not a silent match on version 0.
type replaceBody struct {
	domain.UnitInput
	OccLock *int `json:"occ_lock"`
}

func (h *UnitHandler) update(w http.ResponseWriter, r *http.Request) {
	var body replaceBody
	if err := decode(r, &body); err != nil {
		aoherr.Render(w, r, err)
		return
	}
	if body.OccLock == nil {
		aoherr.Render(w, r, aoherr.New(aoherr.ClassValidation, service.CodeUnitInvalid,
			"occ_lock is required").WithDetails(aoherr.FieldDetail("occ_lock", "must be provided")))
		return
	}
	unit, err := h.units.Update(r.Context(), chi.URLParam(r, "unit_code"), *body.OccLock, body.UnitInput)
	if err != nil {
		aoherr.Render(w, r, err)
		return
	}
	render.Render(w, r, aohhttp.Response(http.StatusOK, "", unit)) //nolint:errcheck
}

func (h *UnitHandler) delete(w http.ResponseWriter, r *http.Request) {
	raw := r.URL.Query().Get("occ_lock")
	occLock, err := strconv.Atoi(raw)
	if raw == "" || err != nil {
		aoherr.Render(w, r, aoherr.New(aoherr.ClassValidation, service.CodeUnitInvalid,
			"occ_lock query parameter is required").
			WithDetails(aoherr.FieldDetail("occ_lock", "must be an integer query parameter")))
		return
	}
	if err := h.units.Delete(r.Context(), chi.URLParam(r, "unit_code"), occLock); err != nil {
		aoherr.Render(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// decode reads a JSON body strictly: unknown fields and trailing data are rejected, so a
// typo'd field name fails loudly instead of being silently ignored.
func decode(r *http.Request, into any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, maxBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(into); err != nil {
		return aoherr.Wrap(aoherr.ClassValidation, service.CodeUnitInvalid,
			"request body is not valid JSON for a unit", err).
			WithDetails(aoherr.FieldDetail("body", err.Error()))
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return aoherr.New(aoherr.ClassValidation, service.CodeUnitInvalid,
			"request body must contain exactly one JSON object")
	}
	return nil
}
