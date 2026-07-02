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

type projectStore interface {
	CreateProject(ctx context.Context, name, description string) (*domain.Project, error)
	GetProject(ctx context.Context, id string) (*domain.Project, error)
	UpdateProject(ctx context.Context, id, name, description string) (*domain.Project, error)
	DeleteProject(ctx context.Context, id string) error
	CreateAPIKey(ctx context.Context, projectID, name, keyHash, prefix string) (*domain.APIKey, error)
}

type projectHandler struct {
	db projectStore
}

type createProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type createProjectResponse struct {
	Project *domain.Project `json:"project"`
	APIKey  *domain.APIKey  `json:"api_key"`
}

func (h *projectHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	proj, err := h.db.CreateProject(r.Context(), req.Name, req.Description)
	if err != nil {
		slog.Error("create project failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create project")
		return
	}

	rawKey, keyHash, prefix, err := auth.Generate()
	if err != nil {
		slog.Error("generate api key failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to generate api key")
		return
	}

	key, err := h.db.CreateAPIKey(r.Context(), proj.ID, "default", keyHash, prefix)
	if err != nil {
		slog.Error("create api key failed", "project_id", proj.ID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create api key")
		return
	}
	key.RawKey = rawKey

	writeJSON(w, http.StatusCreated, createProjectResponse{Project: proj, APIKey: key})
}

func (h *projectHandler) get(w http.ResponseWriter, r *http.Request) {
	pid := projectID(r)
	id := r.PathValue("id")
	if id != pid {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	proj, err := h.db.GetProject(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		slog.Error("get project failed", "id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get project")
		return
	}
	writeJSON(w, http.StatusOK, proj)
}

type updateProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (h *projectHandler) update(w http.ResponseWriter, r *http.Request) {
	pid := projectID(r)
	id := r.PathValue("id")
	if id != pid {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	var req updateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	proj, err := h.db.UpdateProject(r.Context(), id, req.Name, req.Description)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		slog.Error("update project failed", "id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update project")
		return
	}
	writeJSON(w, http.StatusOK, proj)
}

func (h *projectHandler) delete(w http.ResponseWriter, r *http.Request) {
	pid := projectID(r)
	id := r.PathValue("id")
	if id != pid {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if err := h.db.DeleteProject(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		slog.Error("delete project failed", "id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete project")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
