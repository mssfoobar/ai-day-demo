package aohhttp

import (
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/mssfoobar/ops-hub/packages/aoh-golib/aoherr"
)

// checkResultOK is the sentinel a passing check records.
const checkResultOK = "OK"

type Check func() error

type HealthCheck interface {
	AddLivenessCheck(name string, check Check)
	AddReadinessCheck(name string, check Check)
	LiveEndpoint(w http.ResponseWriter, r *http.Request)
	ReadyEndpoint(w http.ResponseWriter, r *http.Request)
}

// handler is a basic Handler implementation.
type handler struct {
	http.ServeMux
	checksMutex     sync.RWMutex
	livenessChecks  map[string]Check
	readinessChecks map[string]Check
}

// NewHealthCheck create liveness and readiness probe check http handler for kubernetes.
// Their endpoints are at "/livez" and "/readyz" respectively.
//
// To add healthcheck handlers, use [HealthCheck.AddLivenessCheck] & [HealthCheck.AddReadinessCheck]
//
// Example:
//
//	 health := NewHealthCheck(router)
//	 health.AddReadinessCheck("readiness check", func() error {
//	 	// do readiness checking here
//			return nil
//	 }
func NewHealthCheck(router *chi.Mux) HealthCheck {
	h := &handler{
		livenessChecks:  make(map[string]Check),
		readinessChecks: make(map[string]Check),
	}
	router.Get("/livez", h.LiveEndpoint)
	router.Get("/readyz", h.ReadyEndpoint)
	return h
}

func (s *handler) LiveEndpoint(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, s.livenessChecks)
}

func (s *handler) ReadyEndpoint(w http.ResponseWriter, r *http.Request) {
	s.handle(w, r, s.readinessChecks)
}

func (s *handler) AddLivenessCheck(name string, check Check) {
	s.checksMutex.Lock()
	defer s.checksMutex.Unlock()
	s.livenessChecks[name] = check
}

func (s *handler) AddReadinessCheck(name string, check Check) {
	s.checksMutex.Lock()
	defer s.checksMutex.Unlock()
	s.readinessChecks[name] = check
}

func (s *handler) collectChecks(checks map[string]Check, resultsOut map[string]string, statusOut *int) {
	s.checksMutex.RLock()
	defer s.checksMutex.RUnlock()
	for name, check := range checks {
		if err := check(); err != nil {
			*statusOut = http.StatusServiceUnavailable
			resultsOut[name] = err.Error()
		} else {
			resultsOut[name] = checkResultOK
		}
	}
}

func (s *handler) handle(w http.ResponseWriter, r *http.Request, checks ...map[string]Check) {
	checkResults := make(map[string]string)
	status := http.StatusOK
	for _, checks := range checks {
		s.collectChecks(checks, checkResults, &status)
	}

	// A 200 keeps its historical empty body — probes match on the status, and
	// giving them a body to parse would be a gratuitous change.
	if status == http.StatusOK {
		w.WriteHeader(status)
		return
	}

	// A failing probe used to be a bare 503: no body for a caller, and the
	// per-check causes collected just above were discarded, so nothing recorded
	// *which* dependency was down. Render the conformant envelope and put the
	// causes in the log record, where internal detail belongs — a check error is
	// typically "dial tcp 10.0.3.14:5432: connect: connection refused", i.e. an
	// internal host and port that must not reach the response body. The renderer
	// suppresses it for us at >= 500, and also emits the one ERROR record.
	failed := make([]aoherr.Detail, 0, len(checkResults))
	for name, result := range checkResults {
		if result != checkResultOK {
			failed = append(failed, aoherr.FieldDetail(name, result))
		}
	}
	sort.Slice(failed, func(i, j int) bool { return failed[i].Field < failed[j].Field })

	err := aoherr.New(aoherr.ClassUpstreamUnavailable, aoherr.CodeUpstreamUnavailable,
		"health check failed: "+describeFailures(failed)).WithDetails(failed...)
	aoherr.Render(w, r, err)
}

// describeFailures builds the developer-facing message. It is safe to include the
// causes here: this string goes to the log record, and the renderer replaces it
// with the class's generic message before the body is written.
func describeFailures(failed []aoherr.Detail) string {
	parts := make([]string, 0, len(failed))
	for _, d := range failed {
		parts = append(parts, d.Field+"="+d.Message)
	}
	return strings.Join(parts, ", ")
}
