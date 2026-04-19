package main

import (
	"fmt"
	"hookrelay/internal/handler"
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

	// Database pool
	dbPool, err := storage.ConnectDB(os.Getenv("DATABASE_URL"))
	if err != nil {
		slog.Error("Unable to connect to Database", "Error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	// Server settings
	router := initializeRoutes()
	server := &http.Server{
		Addr:              fmt.Sprintf("localhost:%s", os.Getenv("PORT")), //TODO: delete "localhost"
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

func initializeRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/ingest/{id}", handler.ReceiverHandler)
	//mux.HandleFunc("GET /v1/healthcheck", handler.HealthHandler) //TODO: uncomment, do healthcheck for DB, version and port
	return mux
}
