package projection

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Row is one pending projection, as the repository hands it over.
//
// It carries no credential, by design: the token comes from the caller whose write
// produced the row and is passed alongside, never stored (design.md D2a).
type Row struct {
	ID       string
	UnitCode string
	Intent   Intent
	Payload  []byte
}

// OutboxStore is the persistence the worker needs. Declared here, at the consumer, so a
// test can substitute a fake with no database.
type OutboxStore interface {
	MarkDelivered(ctx context.Context, id string) error
	RecordAttempt(ctx context.Context, id string, lastError string) error
}

// Retry shapes how often a delivery is retried. It does NOT bound how long: the token's
// remaining lifetime does that, because past it no attempt can succeed and the row must
// not be handed to anyone else's credential.
//
// Attempts is a safety cap, not the budget. Zero — the default — means "as many as fit
// before the token expires", which is what makes a gis-service outage of tens of seconds
// recover on its own rather than stranding the row while the token is still perfectly
// good.
type Retry struct {
	Attempts   int
	Backoff    time.Duration
	MaxBackoff time.Duration
}

// Delay is the wait before attempt n (1-based), doubling up to the ceiling. Uncapped
// doubling would overshoot the token's whole lifetime in a dozen attempts and spend the
// window asleep.
func (r Retry) Delay(attempt int) time.Duration {
	d := r.Backoff
	for i := 1; i < attempt && d < r.MaxBackoff; i++ {
		d *= 2
	}
	if r.MaxBackoff > 0 && d > r.MaxBackoff {
		d = r.MaxBackoff
	}
	return d
}

// exhausted reports whether the attempt cap has been reached. A zero cap never is — the
// deadline stops the loop instead.
func (r Retry) exhausted(attempts int) bool {
	return r.Attempts > 0 && attempts >= r.Attempts
}

// Worker delivers outbox rows to gis-service as the operator who caused them.
//
// There is deliberately no timer and no sweeper. A projection is triggered by the write
// that produced it and runs on the token that write carried, so a worker waking on a
// schedule would have no credential to deliver with. A row that cannot be delivered
// before its token expires stays pending and visible in `gis_outbox`; recovery is an
// operator re-saving that unit, which enqueues a fresh row on a fresh token. Draining a
// backlog on some later request is rejected in design.md D2a: it would attribute one
// operator's write to another, and could make a `dispatch-viewer`'s read perform a GIS
// write.
type Worker struct {
	store  OutboxStore
	client *http.Client
	gisURL string
	retry  Retry

	// wg lets tests (and shutdown) wait for in-flight deliveries. Production never
	// blocks on it: Deliver returns as soon as the goroutine is started.
	wg sync.WaitGroup
}

func NewWorker(store OutboxStore, gisURL string, retry Retry) *Worker {
	return &Worker{
		store:  store,
		client: &http.Client{Timeout: 10 * time.Second},
		gisURL: strings.TrimSuffix(gisURL, "/"),
		retry:  retry,
	}
}

// Deliver hands rows to a detached goroutine and returns immediately, so a gis-service
// outage never slows or fails a unit write.
//
// token and deadline are taken by value: the request context is cancelled the moment the
// response is written, so anything read from it here would already be dead.
func (w *Worker) Deliver(rows []Row, token string, tokenExpiry time.Time) {
	if len(rows) == 0 {
		return
	}
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.deliverAll(rows, token, tokenExpiry)
	}()
}

// Wait blocks until every in-flight delivery has finished. For shutdown and for tests.
func (w *Worker) Wait() { w.wg.Wait() }

func (w *Worker) deliverAll(rows []Row, token string, tokenExpiry time.Time) {
	// The whole batch is bounded by the token, not by each row in turn: once it expires
	// no remaining row can be delivered with it.
	budget := time.Until(tokenExpiry)
	if budget <= 0 {
		for _, row := range rows {
			w.record(row, fmt.Errorf("token already expired; row left pending"))
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()

	for _, row := range rows {
		w.deliverOne(ctx, row, token)
	}
}

func (w *Worker) deliverOne(ctx context.Context, row Row, token string) {
	var lastErr error
	for attempt := 0; !w.retry.exhausted(attempt); attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				// The token's lifetime ran out mid-retry. This is the long-outage case:
				// the row stays pending and visible, and recovery is an operator
				// re-saving the unit on a fresh token.
				w.record(row, fmt.Errorf("%w (last error: %v)", ctx.Err(), lastErr))
				return
			case <-time.After(w.retry.Delay(attempt)):
			}
		}

		if err := w.send(ctx, row, token); err != nil {
			lastErr = err
			if ctx.Err() != nil {
				w.record(row, fmt.Errorf("%w (last error: %v)", ctx.Err(), lastErr))
				return
			}
			continue
		}

		if err := w.store.MarkDelivered(context.WithoutCancel(ctx), row.ID); err != nil {
			log.Printf("dispatch-svc: projection %s delivered but not marked: %v", row.UnitCode, err)
		}
		return
	}
	w.record(row, lastErr)
}

// record leaves the row pending with its attempt count and last error, which is what
// makes a stranded projection a visible, recoverable condition rather than a silent one.
func (w *Worker) record(row Row, cause error) {
	msg := "unknown error"
	if cause != nil {
		msg = cause.Error()
	}
	log.Printf("dispatch-svc: projection for %s left pending: %s", row.UnitCode, msg)
	if err := w.store.RecordAttempt(context.WithoutCancel(context.Background()), row.ID, msg); err != nil {
		log.Printf("dispatch-svc: could not record projection failure for %s: %v", row.UnitCode, err)
	}
}

func (w *Worker) send(ctx context.Context, row Row, token string) error {
	var (
		req *http.Request
		err error
	)
	switch row.Intent {
	case IntentUpsert:
		req, err = http.NewRequestWithContext(ctx, http.MethodPut,
			w.gisURL+"/geoentity", bytes.NewReader(row.Payload))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
		}
	case IntentDelete:
		req, err = http.NewRequestWithContext(ctx, http.MethodDelete,
			w.gisURL+"/geoentity/entity_id/"+row.UnitCode, nil)
	default:
		// Unreachable: the outbox CHECK constrains the vocabulary. Retrying would not
		// help, so it is recorded like any other permanent failure.
		return fmt.Errorf("unknown projection intent %q", row.Intent)
	}
	if err != nil {
		return fmt.Errorf("build %s request for %s: %w", row.Intent, row.UnitCode, err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", row.Intent, row.UnitCode, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	// A delete for an entity that is already gone is done, not failed — delivery is
	// at-least-once, so the second delivery of a delete is the normal case.
	if row.Intent == IntentDelete && resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("%s %s: gis-service answered %d", row.Intent, row.UnitCode, resp.StatusCode)
	}
	return nil
}
