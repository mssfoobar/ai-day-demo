// Command server runs the dispatch field-unit service.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/config"
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/db"
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/handler"
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/projection"
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/repo"
	"github.com/mssfoobar/fleet-dispatch-console/apps/dispatch-svc/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("dispatch-svc: %v", err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	pool, err := db.Open(ctx, cfg.DSN())
	if err != nil {
		// The most common local failure by far: Postgres is not up yet.
		return fmt.Errorf("%w (is the compose postgres running?)", err)
	}
	defer func() { _ = pool.Close() }()

	// Migrations run on start so the workshop needs no separate migration tool.
	if err := db.Migrate(ctx, pool, cfg.SQLSchemaName); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	log.Printf("dispatch-svc: schema %q ready", cfg.SQLSchemaName)

	logger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("build logger: %w", err)
	}
	defer func() { _ = logger.Sync() }()

	unitRepo := repo.NewUnitRepo(pool, cfg.SQLSchemaName)

	// The projection worker drains the outbox as the operator who caused each row. It
	// is started by a write, not by a timer: a timer waking with no operator token
	// could deliver nothing (design.md D2a).
	worker := projection.NewWorker(unitRepo, cfg.GISURL, projection.Retry{
		Attempts:   cfg.ProjectionAttempts,
		Backoff:    cfg.ProjectionBackoff,
		MaxBackoff: cfg.ProjectionMaxBackoff,
	})
	// In-flight deliveries finish before the process exits; anything still pending is
	// left in gis_outbox, visible, for an operator to recover by re-saving the unit.
	defer worker.Wait()

	units := service.NewUnitService(unitRepo, worker)
	router := handler.Router(handler.NewUnitHandler(units), func(ctx context.Context) error {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return pool.PingContext(pingCtx)
	}, cfg.IssuerURL(), logger)
	log.Printf("dispatch-svc: bearer tokens validated against %s", cfg.IssuerURL())

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Printf("dispatch-svc: listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	select {
	case err := <-serveErr:
		return fmt.Errorf("serve: %w", err)
	case <-ctx.Done():
		log.Print("dispatch-svc: shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}
