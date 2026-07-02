package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"hookrelay/internal/domain"
	"hookrelay/internal/storage"
)

type sourceStore interface {
	CreateSource(ctx context.Context, p storage.CreateSourceParams) (*domain.Source, error)
	GetSource(ctx context.Context, projectID, id string) (*domain.Source, error)
	ListSources(ctx context.Context, projectID string) ([]domain.Source, error)
	UpdateSource(ctx context.Context, projectID, id string, p storage.UpdateSourceParams) (*domain.Source, error)
	DeleteSource(ctx context.Context, projectID, id string) error
}

type sourceHandler struct {
	db      sourceStore
	baseURL string
}

type createSourceRequest struct {
	Name         string          `json:"name"`
	UID          string          `json:"uid"`
	VerifierType string          `json:"verifier_type"`
	VerifierCfg  json.RawMessage `json:"verifier_cfg,omitempty"`
}

func (h *sourceHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.UID == "" {
		writeError(w, http.StatusBadRequest, "uid is required")
		return
	}
	if req.VerifierType == "" {
		req.VerifierType = domain.VerifierNoop
	}
	if req.VerifierType != domain.VerifierNoop && req.VerifierType != domain.VerifierHMAC {
		writeError(w, http.StatusBadRequest, "invalid verifier_type")
		return
	}

	src, err := h.db.CreateSource(r.Context(), storage.CreateSourceParams{
		ProjectID:    projectID(r),
		Name:         req.Name,
		UID:          req.UID,
		VerifierType: req.VerifierType,
		VerifierCfg:  req.VerifierCfg,
	})
	if err != nil {
		slog.Error("create source failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create source")
		return
	}
	src.IngestURL = h.ingestURL(req.UID)
	writeJSON(w, http.StatusCreated, src)
}

func (h *sourceHandler) get(w http.ResponseWriter, r *http.Request) {
	src, err := h.db.GetSource(r.Context(), projectID(r), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "source not found")
			return
		}
		slog.Error("get source failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get source")
		return
	}
	src.IngestURL = h.ingestURL(src.UID)
	writeJSON(w, http.StatusOK, src)
}

func (h *sourceHandler) list(w http.ResponseWriter, r *http.Request) {
	srcs, err := h.db.ListSources(r.Context(), projectID(r))
	if err != nil {
		slog.Error("list sources failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list sources")
		return
	}
	if srcs == nil {
		srcs = []domain.Source{}
	}
	for i := range srcs {
		srcs[i].IngestURL = h.ingestURL(srcs[i].UID)
	}
	writeJSON(w, http.StatusOK, srcs)
}

type updateSourceRequest struct {
	Name         *string         `json:"name"`
	VerifierType *string         `json:"verifier_type"`
	VerifierCfg  json.RawMessage `json:"verifier_cfg,omitempty"`
	IsActive     *bool           `json:"is_active"`
}

func (h *sourceHandler) update(w http.ResponseWriter, r *http.Request) {
	var req updateSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.VerifierType != nil && *req.VerifierType != domain.VerifierNoop && *req.VerifierType != domain.VerifierHMAC {
		writeError(w, http.StatusBadRequest, "invalid verifier_type")
		return
	}
	src, err := h.db.UpdateSource(r.Context(), projectID(r), r.PathValue("id"), storage.UpdateSourceParams{
		Name:         req.Name,
		VerifierType: req.VerifierType,
		VerifierCfg:  req.VerifierCfg,
		IsActive:     req.IsActive,
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "source not found")
			return
		}
		slog.Error("update source failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update source")
		return
	}
	src.IngestURL = h.ingestURL(src.UID)
	writeJSON(w, http.StatusOK, src)
}

func (h *sourceHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.db.DeleteSource(r.Context(), projectID(r), r.PathValue("id")); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "source not found")
			return
		}
		slog.Error("delete source failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete source")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *sourceHandler) ingestURL(uid string) string {
	return fmt.Sprintf("%s/ingest/%s", strings.TrimRight(h.baseURL, "/"), uid)
}
