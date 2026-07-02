package handler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"hookrelay/internal/domain"
	"hookrelay/internal/service"
)

type ingestServicer interface {
	Ingest(ctx context.Context, sourceUID string, r *http.Request, body []byte, idempotencyKey *string) (*service.IngestResult, error)
}

type ingestLimiter interface {
	Allow(key string) bool
}

type ingestHandler struct {
	svc     ingestServicer
	limiter ingestLimiter
}

func (h *ingestHandler) ingest(w http.ResponseWriter, r *http.Request) {
	uid := r.PathValue("uid")

	if !h.limiter.Allow(uid) {
		writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "payload too large")
		return
	}

	var idempotencyKey *string
	if k := r.Header.Get("Idempotency-Key"); k != "" {
		idempotencyKey = &k
	}

	result, err := h.svc.Ingest(r.Context(), uid, r, body, idempotencyKey)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			writeError(w, http.StatusNotFound, "source not found")
		case errors.Is(err, domain.ErrUnauthorized):
			writeError(w, http.StatusUnauthorized, "signature verification failed")
		default:
			slog.Error("ingest failed", "uid", uid, "error", err)
			writeError(w, http.StatusInternalServerError, "ingest failed")
		}
		return
	}

	writeJSON(w, http.StatusAccepted, result)
}
