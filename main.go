package main

import (
	"fmt"
	"hookrelay/internal/config"
	"hookrelay/internal/handler"
	"hookrelay/internal/service"
	"hookrelay/internal/storage"
	"log/slog"
	"net/http"
	"os"
	"time"
)

const version = "1.0.0"

func main() {
	// Logger settings
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Config
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Unable to load config. See your .env file or system environment", "error", err)
		os.Exit(1)
	}

	// Database pool
	dbPool, err := storage.ConnectDB(cfg.DB.PostgresURL)
	if err != nil {
		slog.Error("Unable to connect to Database", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	// Structs
	myStorage := storage.NewStorage(dbPool)
	myService := service.NewWebhookService(myStorage)
	myHandler := handler.NewWebhookHandler(myService)

	// Server settings
	router := initializeRoutes(myHandler)
	server := &http.Server{
		Addr:              fmt.Sprintf("localhost:%s", cfg.App.Port), //TODO: delete "localhost"
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
	}
	err = server.ListenAndServe()
	if err != nil {
		logger.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}

func initializeRoutes(h *handler.WebhookHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/ingest/{id}", h.ReceiverHandler)
	//mux.HandleFunc("GET /v1/healthcheck", handler.HealthHandler) //TODO: uncomment, do healthcheck for DB, version and port
	return mux
}
