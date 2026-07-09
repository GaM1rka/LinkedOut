package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "linkedout/docs"
	"linkedout/internal/config"
	"linkedout/internal/logger"
	"linkedout/internal/metricsapi"
	"linkedout/internal/storage"

	"github.com/jackc/pgx/v5/pgxpool"
)

// @title LinkedOut Metrics API
// @version 1.0
// @description REST API for LinkedOut Telegram bot MVP metrics.
// @BasePath /
func main() {
	cfg := config.Load()
	log := logger.New("metrics-service")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := connectPostgres(ctx, cfg.PostgresDSN, log)
	if err != nil {
		log.Error("connect postgres failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := storage.RunMigrations(ctx, db, cfg.MigrationsDir); err != nil {
		log.Error("run migrations failed", "error", err)
		os.Exit(1)
	}

	store := storage.NewStore(db)
	server := &http.Server{
		Addr:              ":" + cfg.MetricsHTTPPort,
		Handler:           metricsapi.NewRouter(store, log),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("metrics-service listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("metrics-service failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown failed", "error", err)
	}
}

func connectPostgres(ctx context.Context, dsn string, log *slog.Logger) (*pgxpool.Pool, error) {
	var lastErr error
	for attempt := 1; attempt <= 30; attempt++ {
		db, err := pgxpool.New(ctx, dsn)
		if err == nil {
			pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			err = db.Ping(pingCtx)
			cancel()
			if err == nil {
				return db, nil
			}
			db.Close()
		}

		lastErr = err
		log.Info("waiting for postgres", "attempt", attempt, "error", err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return nil, lastErr
}
