package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"hookrelay/internal/domain"
	"hookrelay/internal/storage"
)

type endpointStore interface {
	CreateEndpoint(ctx context.Context, p storage.CreateEndpointParams) (*domain.Endpoint, error)
	GetEndpoint(ctx context.Context, projectID, id string) (*domain.Endpoint, error)
	ListEndpoints(ctx context.Context, projectID string) ([]domain.Endpoint, error)
	UpdateEndpoint(ctx context.Context, projectID, id string, p storage.UpdateEndpointParams) (*domain.Endpoint, error)
	DeleteEndpoint(ctx context.Context, projectID, id string) error
}

type endpointHandler struct {
	db             endpointStore
	maxHTTPTimeout time.Duration
}

type createEndpointRequest struct {
	Name        string          `json:"name"`
	URL         string          `json:"url"`
	Description string          `json:"description"`
	HTTPTimeout int             `json:"http_timeout"`
	AuthType    string          `json:"auth_type"`
	AuthCfg     json.RawMessage `json:"auth_cfg,omitempty"`
}

func (h *endpointHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.URL == "" {
		writeError(w, http.StatusBadRequest, "name and url are required")
		return
	}
	if !validEndpointURL(req.URL) {
		writeError(w, http.StatusBadRequest, "url must be a valid absolute http(s) url")
		return
	}
	if req.AuthType == "" {
		req.AuthType = domain.AuthNone
	}
	if msg, ok := h.validateHTTPTimeout(req.HTTPTimeout); !ok {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	ep, err := h.db.CreateEndpoint(r.Context(), storage.CreateEndpointParams{
		ProjectID:   projectID(r),
		Name:        req.Name,
		URL:         req.URL,
		Description: req.Description,
		HTTPTimeout: req.HTTPTimeout,
		AuthType:    req.AuthType,
		AuthCfg:     req.AuthCfg,
	})
	if err != nil {
		slog.Error("create endpoint failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create endpoint")
		return
	}
	writeJSON(w, http.StatusCreated, ep)
}

func (h *endpointHandler) get(w http.ResponseWriter, r *http.Request) {
	ep, err := h.db.GetEndpoint(r.Context(), projectID(r), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "endpoint not found")
			return
		}
		slog.Error("get endpoint failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get endpoint")
		return
	}
	writeJSON(w, http.StatusOK, ep)
}

func (h *endpointHandler) list(w http.ResponseWriter, r *http.Request) {
	eps, err := h.db.ListEndpoints(r.Context(), projectID(r))
	if err != nil {
		slog.Error("list endpoints failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list endpoints")
		return
	}
	if eps == nil {
		eps = []domain.Endpoint{}
	}
	writeJSON(w, http.StatusOK, eps)
}

type updateEndpointRequest struct {
	Name        *string         `json:"name"`
	URL         *string         `json:"url"`
	Description *string         `json:"description"`
	HTTPTimeout *int            `json:"http_timeout"`
	AuthType    *string         `json:"auth_type"`
	AuthCfg     json.RawMessage `json:"auth_cfg,omitempty"`
	IsActive    *bool           `json:"is_active"`
}

func (h *endpointHandler) update(w http.ResponseWriter, r *http.Request) {
	var req updateEndpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.URL != nil && !validEndpointURL(*req.URL) {
		writeError(w, http.StatusBadRequest, "url must be a valid absolute http(s) url")
		return
	}
	if req.HTTPTimeout != nil {
		if msg, ok := h.validateHTTPTimeout(*req.HTTPTimeout); !ok {
			writeError(w, http.StatusBadRequest, msg)
			return
		}
	}
	ep, err := h.db.UpdateEndpoint(r.Context(), projectID(r), r.PathValue("id"), storage.UpdateEndpointParams{
		Name:        req.Name,
		URL:         req.URL,
		Description: req.Description,
		HTTPTimeout: req.HTTPTimeout,
		AuthType:    req.AuthType,
		AuthCfg:     req.AuthCfg,
		IsActive:    req.IsActive,
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "endpoint not found")
			return
		}
		slog.Error("update endpoint failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update endpoint")
		return
	}
	writeJSON(w, http.StatusOK, ep)
}

func (h *endpointHandler) validateHTTPTimeout(seconds int) (string, bool) {
	if seconds <= 0 {
		return "", true
	}
	if time.Duration(seconds)*time.Second >= h.maxHTTPTimeout {
		return fmt.Sprintf("http_timeout must be less than the worker stuck timeout (%s)", h.maxHTTPTimeout), false
	}
	return "", true
}

func validEndpointURL(raw string) bool {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func (h *endpointHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.db.DeleteEndpoint(r.Context(), projectID(r), r.PathValue("id")); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "endpoint not found")
			return
		}
		slog.Error("delete endpoint failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete endpoint")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
