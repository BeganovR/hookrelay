package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"hookrelay/internal/auth"
	"hookrelay/internal/domain"
)

type apikeyStore interface {
	CreateAPIKey(ctx context.Context, projectID, name, keyHash, prefix string) (*domain.APIKey, error)
	ListAPIKeys(ctx context.Context, projectID string) ([]domain.APIKey, error)
	DeleteAPIKey(ctx context.Context, projectID, keyID string) error
}

type apikeyHandler struct {
	db apikeyStore
}

type createKeyRequest struct {
	Name string `json:"name"`
}

func (h *apikeyHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		req.Name = "default"
	}

	rawKey, keyHash, prefix, err := auth.Generate()
	if err != nil {
		slog.Error("generate api key failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to generate key")
		return
	}

	key, err := h.db.CreateAPIKey(r.Context(), projectID(r), req.Name, keyHash, prefix)
	if err != nil {
		slog.Error("create api key failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create api key")
		return
	}
	key.RawKey = rawKey
	writeJSON(w, http.StatusCreated, key)
}

func (h *apikeyHandler) list(w http.ResponseWriter, r *http.Request) {
	keys, err := h.db.ListAPIKeys(r.Context(), projectID(r))
	if err != nil {
		slog.Error("list api keys failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list api keys")
		return
	}
	if keys == nil {
		keys = []domain.APIKey{}
	}
	writeJSON(w, http.StatusOK, keys)
}

func (h *apikeyHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.db.DeleteAPIKey(r.Context(), projectID(r), r.PathValue("id")); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "api key not found")
			return
		}
		slog.Error("delete api key failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete api key")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
