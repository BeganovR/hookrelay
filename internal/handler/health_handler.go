package handler

import (
	"context"
	"net/http"
)

type pinger interface {
	Ping(ctx context.Context) error
}

type healthHandler struct {
	db pinger
}

func (h *healthHandler) health(w http.ResponseWriter, r *http.Request) {
	if err := h.db.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unhealthy", "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
