package projection_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/projection"
)

// --------------------------------------------------------------------------- //
// Doubles
// --------------------------------------------------------------------------- //

// fakeOutbox records the worker's delivery bookkeeping. Hand-written: the interface is
// two methods, so a mock framework would cost more than it saves.
type fakeOutbox struct {
	mu        sync.Mutex
	delivered []string
	attempts  []string
	lastError string
}

func (f *fakeOutbox) MarkDelivered(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.delivered = append(f.delivered, id)
	return nil
}

func (f *fakeOutbox) RecordAttempt(_ context.Context, id string, lastError string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.attempts = append(f.attempts, id)
	f.lastError = lastError
	return nil
}

func (f *fakeOutbox) snapshot() ([]string, []string, string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.delivered...), append([]string(nil), f.attempts...), f.lastError
}

// stubGIS stands in for gis-service, recording what reached it.
type stubGIS struct {
	mu       sync.Mutex
	requests []string
	tokens   []string
	status   int
	failFor  int // fail this many times before succeeding
}

func (s *stubGIS) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.requests = append(s.requests, r.Method+" "+r.URL.Path)
		s.tokens = append(s.tokens, strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		fail := s.failFor > 0
		if fail {
			s.failFor--
		}
		status := s.status
		s.mu.Unlock()

		if fail {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
	}
}

func (s *stubGIS) seen() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.requests...)
}

func upsertRow(id, code string) projection.Row {
	return projection.Row{ID: id, UnitCode: code, Intent: projection.IntentUpsert, Payload: []byte(`{}`)}
}

func deleteRow(id, code string) projection.Row {
	return projection.Row{ID: id, UnitCode: code, Intent: projection.IntentDelete, Payload: []byte(`{}`)}
}

func liveToken() time.Time { return time.Now().Add(5 * time.Minute) }

func retry() projection.Retry {
	return projection.Retry{Attempts: 3, Backoff: 5 * time.Millisecond, MaxBackoff: 20 * time.Millisecond}
}

// The production default: no attempt cap, bounded only by the token.
func tokenBoundedRetry() projection.Retry {
	return projection.Retry{Attempts: 0, Backoff: 5 * time.Millisecond, MaxBackoff: 20 * time.Millisecond}
}

// --------------------------------------------------------------------------- //
// Tests
// --------------------------------------------------------------------------- //

func TestDeliver_UpsertGoesToGisAsTheWritingOperator(t *testing.T) {
	gis := &stubGIS{}
	srv := httptest.NewServer(gis.handler())
	defer srv.Close()

	store := &fakeOutbox{}
	w := projection.NewWorker(store, srv.URL, retry())

	w.Deliver([]projection.Row{upsertRow("row-1", "FU-101")}, "operator-token", liveToken())
	w.Wait()

	assert.Equal(t, []string{"PUT /geoentity"}, gis.seen())
	gis.mu.Lock()
	assert.Equal(t, []string{"operator-token"}, gis.tokens,
		"the projection carries the writing operator's bearer, never a service account's")
	gis.mu.Unlock()

	delivered, attempts, _ := store.snapshot()
	assert.Equal(t, []string{"row-1"}, delivered)
	assert.Empty(t, attempts)
}

func TestDeliver_DeleteAddressesTheEntityByUnitCode(t *testing.T) {
	gis := &stubGIS{status: http.StatusNoContent}
	srv := httptest.NewServer(gis.handler())
	defer srv.Close()

	store := &fakeOutbox{}
	w := projection.NewWorker(store, srv.URL, retry())

	w.Deliver([]projection.Row{deleteRow("row-2", "FU-900")}, "operator-token", liveToken())
	w.Wait()

	assert.Equal(t, []string{"DELETE /geoentity/entity_id/FU-900"}, gis.seen())
	delivered, _, _ := store.snapshot()
	assert.Equal(t, []string{"row-2"}, delivered)
}

// Delivery is at-least-once, so the second delivery of a delete is the normal case, not a
// failure. A 404 there means the entity is already gone — which is the desired end state.
func TestDeliver_DeletingAnEntityThatIsAlreadyGoneCountsAsDelivered(t *testing.T) {
	gis := &stubGIS{status: http.StatusNotFound}
	srv := httptest.NewServer(gis.handler())
	defer srv.Close()

	store := &fakeOutbox{}
	w := projection.NewWorker(store, srv.URL, retry())

	w.Deliver([]projection.Row{deleteRow("row-3", "FU-204")}, "operator-token", liveToken())
	w.Wait()

	assert.Len(t, gis.seen(), 1, "a 404 on delete must not be retried")
	delivered, attempts, _ := store.snapshot()
	assert.Equal(t, []string{"row-3"}, delivered)
	assert.Empty(t, attempts)
}

// Replaying a row is a no-op in GIS because PUT /geoentity is an upsert keyed on
// entity_id. What this pins is the worker's half: the same row delivered twice marks
// delivered twice and never errors, so an at-least-once redelivery is safe.
func TestDeliver_ReplayingARowIsSafe(t *testing.T) {
	gis := &stubGIS{}
	srv := httptest.NewServer(gis.handler())
	defer srv.Close()

	store := &fakeOutbox{}
	w := projection.NewWorker(store, srv.URL, retry())

	row := upsertRow("row-4", "FU-101")
	w.Deliver([]projection.Row{row}, "operator-token", liveToken())
	w.Wait()
	w.Deliver([]projection.Row{row}, "operator-token", liveToken())
	w.Wait()

	assert.Equal(t, []string{"PUT /geoentity", "PUT /geoentity"}, gis.seen())
	delivered, attempts, _ := store.snapshot()
	assert.Equal(t, []string{"row-4", "row-4"}, delivered)
	assert.Empty(t, attempts, "a replay is not a failure")
}

func TestDeliver_RecoversFromABriefOutageWithNoOperatorAction(t *testing.T) {
	gis := &stubGIS{failFor: 2}
	srv := httptest.NewServer(gis.handler())
	defer srv.Close()

	store := &fakeOutbox{}
	w := projection.NewWorker(store, srv.URL, retry())

	w.Deliver([]projection.Row{upsertRow("row-5", "FU-101")}, "operator-token", liveToken())
	w.Wait()

	assert.Len(t, gis.seen(), 3, "two failures then a success")
	delivered, attempts, _ := store.snapshot()
	assert.Equal(t, []string{"row-5"}, delivered)
	assert.Empty(t, attempts)
}

func TestDeliver_AnOutageLongerThanTheRetryBudgetLeavesTheRowPending(t *testing.T) {
	gis := &stubGIS{failFor: 100}
	srv := httptest.NewServer(gis.handler())
	defer srv.Close()

	store := &fakeOutbox{}
	w := projection.NewWorker(store, srv.URL, retry())

	w.Deliver([]projection.Row{upsertRow("row-6", "FU-101")}, "operator-token", liveToken())
	w.Wait()

	delivered, attempts, lastErr := store.snapshot()
	assert.Empty(t, delivered, "a row that never landed must not be marked delivered")
	assert.Equal(t, []string{"row-6"}, attempts)
	assert.Contains(t, lastErr, "502", "the last error is recorded so the failure is inspectable")
}

// A delivery is bounded by the remaining lifetime of the token it carries. Past that, the
// row stays pending — it is never handed to a later operator's credential.
func TestDeliver_AnExpiredTokenLeavesEveryRowPendingWithoutCallingGis(t *testing.T) {
	gis := &stubGIS{}
	srv := httptest.NewServer(gis.handler())
	defer srv.Close()

	store := &fakeOutbox{}
	w := projection.NewWorker(store, srv.URL, retry())

	w.Deliver(
		[]projection.Row{upsertRow("row-7", "FU-101"), deleteRow("row-8", "FU-204")},
		"expired-token", time.Now().Add(-time.Second),
	)
	w.Wait()

	assert.Empty(t, gis.seen(), "an expired token is not worth a request")
	delivered, attempts, lastErr := store.snapshot()
	assert.Empty(t, delivered)
	assert.ElementsMatch(t, []string{"row-7", "row-8"}, attempts)
	assert.Contains(t, lastErr, "expired")
}

// The bound that matters is the token, not the attempt count. A gis-service outage of a
// few tens of seconds is well inside a 300s token, and stranding the row there would fail
// the "delivery resumes after a brief outage" promise while the credential was still good.
func TestDeliver_RetriesUntilTheTokenExpiresRatherThanAFixedAttemptCount(t *testing.T) {
	// Far more failures than any sane attempt cap; only the deadline stops this.
	gis := &stubGIS{failFor: 10_000}
	srv := httptest.NewServer(gis.handler())
	defer srv.Close()

	store := &fakeOutbox{}
	w := projection.NewWorker(store, srv.URL, tokenBoundedRetry())

	w.Deliver([]projection.Row{upsertRow("row-10", "FU-101")}, "operator-token", time.Now().Add(250*time.Millisecond))
	w.Wait()

	assert.Greater(t, len(gis.seen()), 5,
		"an uncapped worker must keep trying for the token's life, not stop after a handful")
	delivered, attempts, lastErr := store.snapshot()
	assert.Empty(t, delivered)
	assert.Equal(t, []string{"row-10"}, attempts, "the row is left pending exactly once")
	assert.Contains(t, lastErr, "deadline", "the recorded cause names the expiry")
}

// ...and it recovers without operator action when the outage ends inside that window.
func TestDeliver_RecoversFromAnOutageThatOutlastsAnAttemptCapButNotTheToken(t *testing.T) {
	gis := &stubGIS{failFor: 20}
	srv := httptest.NewServer(gis.handler())
	defer srv.Close()

	store := &fakeOutbox{}
	w := projection.NewWorker(store, srv.URL, tokenBoundedRetry())

	w.Deliver([]projection.Row{upsertRow("row-11", "FU-101")}, "operator-token", time.Now().Add(5*time.Second))
	w.Wait()

	delivered, attempts, _ := store.snapshot()
	assert.Equal(t, []string{"row-11"}, delivered, "20 failures is more than any attempt cap, and still recovers")
	assert.Empty(t, attempts)
}

func TestRetry_BackoffDoublesUpToTheCeiling(t *testing.T) {
	// Uncapped doubling would spend a 300s token asleep after a dozen attempts.
	r := projection.Retry{Backoff: 100 * time.Millisecond, MaxBackoff: 400 * time.Millisecond}
	assert.Equal(t, 100*time.Millisecond, r.Delay(1))
	assert.Equal(t, 200*time.Millisecond, r.Delay(2))
	assert.Equal(t, 400*time.Millisecond, r.Delay(3))
	assert.Equal(t, 400*time.Millisecond, r.Delay(9))
}

func TestDeliver_NothingToDoIsNotAnError(t *testing.T) {
	store := &fakeOutbox{}
	w := projection.NewWorker(store, "http://127.0.0.1:1", retry())

	w.Deliver(nil, "operator-token", liveToken())
	w.Wait()

	delivered, attempts, _ := store.snapshot()
	assert.Empty(t, delivered)
	assert.Empty(t, attempts)
}

// The handler must never wait on gis-service: an outage must not fail or slow a unit
// write. Deliver hands off to a goroutine and returns.
func TestDeliver_DoesNotBlockTheCaller(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-release
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	w := projection.NewWorker(&fakeOutbox{}, srv.URL, retry())

	done := make(chan struct{})
	go func() {
		w.Deliver([]projection.Row{upsertRow("row-9", "FU-101")}, "operator-token", liveToken())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Deliver blocked on gis-service")
	}

	close(release)
	w.Wait()
	require.NotEmpty(t, srv.URL)
}
