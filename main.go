package main

import (
	"flag"
	"fmt"
	"hookrelay/internal/handler"
	"log/slog"
	"net/http"
	"os"
	"time"
)

const version = "1.0.0"

type config struct {
	port int
}

func main() {
	var conf config
	flag.IntVar(&conf.port, "port", 8080, "Server port")
	flag.Parse()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	router := initializeRoutes()
	server := &http.Server{
		Addr:              fmt.Sprintf("localhost:%d", conf.port), //TODO: delete "localhost"
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
	}
	err := server.ListenAndServe()
	if err != nil {
		logger.Error("server failed to start", "error", err)
	}
}

func initializeRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/ingest/{id}", handler.ReceiverHandler)
	//mux.HandleFunc("GET /v1/healthcheck", handler.HealthHandler) //TODO: uncomment, do healthcheck for DB, version and port
	return mux
}
