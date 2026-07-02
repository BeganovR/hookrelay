package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"hookrelay/internal/config"
	"hookrelay/internal/handler"
	"hookrelay/internal/service"
	"hookrelay/internal/storage"
)

const version = "0.1.0"

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}

	pool, err := connectWithRetry(cfg.DB.PostgresURL, 5, 2*time.Second)
	if err != nil {
		slog.Error("database connect failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := storage.RunMigrations(cfg.DB.PostgresURL); err != nil {
		slog.Error("migrations failed", "error", err)
		os.Exit(1)
	}

	db := storage.New(pool)
	ingestSvc := service.NewIngestService(db)
	worker := service.NewWorker(db, cfg.Worker)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.App.Port),
		Handler:           handler.SetupRoutes(db, ingestSvc, cfg.App.BaseURL, cfg.App.IngestRatePerSec, cfg.App.IngestRateBurst, cfg.Worker.StuckTimeout),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	workerCtx, workerCancel := context.WithCancel(context.Background())
	go worker.Run(workerCtx)

	go func() {
		slog.Info("server started", "addr", srv.Addr, "version", version)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down gracefully")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("http shutdown failed", "error", err)
	}

	workerCancel()

	slog.Info("server stopped")
}

func connectWithRetry(dbURL string, attempts int, delay time.Duration) (*pgxpool.Pool, error) {
	var pool *pgxpool.Pool
	var err error
	for i := 1; i <= attempts; i++ {
		pool, err = storage.Connect(dbURL)
		if err == nil {
			return pool, nil
		}
		if i < attempts {
			slog.Warn("database connect attempt failed, retrying", "attempt", i, "error", err)
			time.Sleep(delay)
		}
	}
	return nil, err
}
