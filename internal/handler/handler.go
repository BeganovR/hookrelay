package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"hookrelay/internal/middleware"
	"hookrelay/internal/ratelimit"
	"hookrelay/internal/service"
	"hookrelay/internal/storage"
)

func SetupRoutes(db *storage.DB, ingestSvc *service.IngestService, baseURL string, ingestRPS float64, ingestBurst int, workerStuckTimeout time.Duration) http.Handler {
	ph := &projectHandler{db: db}
	kh := &apikeyHandler{db: db}
	sh := &sourceHandler{db: db, baseURL: baseURL}
	eh := &endpointHandler{db: db, maxHTTPTimeout: workerStuckTimeout}
	subh := &subscriptionHandler{db: db}
	evh := &eventHandler{db: db}
	dh := &deliveryHandler{db: db}
	ih := &ingestHandler{svc: ingestSvc, limiter: ratelimit.New(ingestRPS, ingestBurst)}
	hh := &healthHandler{db: db}

	protected := http.NewServeMux()
	protected.HandleFunc("GET /v1/projects/{id}", ph.get)
	protected.HandleFunc("PATCH /v1/projects/{id}", ph.update)
	protected.HandleFunc("DELETE /v1/projects/{id}", ph.delete)

	protected.HandleFunc("POST /v1/api-keys", kh.create)
	protected.HandleFunc("GET /v1/api-keys", kh.list)
	protected.HandleFunc("DELETE /v1/api-keys/{id}", kh.delete)

	protected.HandleFunc("POST /v1/sources", sh.create)
	protected.HandleFunc("GET /v1/sources", sh.list)
	protected.HandleFunc("GET /v1/sources/{id}", sh.get)
	protected.HandleFunc("PATCH /v1/sources/{id}", sh.update)
	protected.HandleFunc("DELETE /v1/sources/{id}", sh.delete)

	protected.HandleFunc("POST /v1/endpoints", eh.create)
	protected.HandleFunc("GET /v1/endpoints", eh.list)
	protected.HandleFunc("GET /v1/endpoints/{id}", eh.get)
	protected.HandleFunc("PATCH /v1/endpoints/{id}", eh.update)
	protected.HandleFunc("DELETE /v1/endpoints/{id}", eh.delete)

	protected.HandleFunc("POST /v1/subscriptions", subh.create)
	protected.HandleFunc("GET /v1/subscriptions", subh.list)
	protected.HandleFunc("GET /v1/subscriptions/{id}", subh.get)
	protected.HandleFunc("PATCH /v1/subscriptions/{id}", subh.update)
	protected.HandleFunc("DELETE /v1/subscriptions/{id}", subh.delete)

	protected.HandleFunc("GET /v1/events", evh.list)
	protected.HandleFunc("GET /v1/events/{id}", evh.get)
	protected.HandleFunc("GET /v1/events/{id}/deliveries", evh.deliveries)

	protected.HandleFunc("GET /v1/deliveries/{id}", dh.get)
	protected.HandleFunc("GET /v1/deliveries/{id}/attempts", dh.attempts)
	protected.HandleFunc("POST /v1/deliveries/{id}/retry", dh.retry)

	authed := middleware.Auth(db)(protected)

	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.Handler())
	mux.HandleFunc("GET /v1/healthz", hh.health)
	mux.HandleFunc("POST /v1/projects", ph.create)
	mux.HandleFunc("POST /ingest/{uid}", ih.ingest)
	mux.Handle("/", authed)

	return middleware.RequestID(middleware.Logger(mux))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode response", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func projectID(r *http.Request) string {
	return middleware.ProjectID(r.Context())
}
