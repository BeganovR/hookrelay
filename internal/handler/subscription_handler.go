package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"hookrelay/internal/auth"
	"hookrelay/internal/domain"
	"hookrelay/internal/storage"
)

type subscriptionStore interface {
	CreateSubscription(ctx context.Context, p storage.CreateSubscriptionParams) (*domain.Subscription, error)
	GetSubscription(ctx context.Context, projectID, id string) (*domain.Subscription, error)
	ListSubscriptions(ctx context.Context, projectID string) ([]domain.Subscription, error)
	UpdateSubscription(ctx context.Context, projectID, id string, isActive bool) (*domain.Subscription, error)
	DeleteSubscription(ctx context.Context, projectID, id string) error
}

type subscriptionHandler struct {
	db subscriptionStore
}

type createSubscriptionRequest struct {
	Name          string          `json:"name"`
	SourceID      *string         `json:"source_id,omitempty"`
	EndpointID    string          `json:"endpoint_id"`
	MaxRetries    int             `json:"max_retries"`
	RetryInterval int             `json:"retry_interval"`
	RetryType     string          `json:"retry_type"`
	FilterCfg     json.RawMessage `json:"filter_cfg,omitempty"`
}

func (h *subscriptionHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.EndpointID == "" {
		writeError(w, http.StatusBadRequest, "name and endpoint_id are required")
		return
	}
	signingSecret, err := auth.GenerateSigningSecret()
	if err != nil {
		slog.Error("generate signing secret failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to generate signing secret")
		return
	}
	sub, err := h.db.CreateSubscription(r.Context(), storage.CreateSubscriptionParams{
		ProjectID:     projectID(r),
		Name:          req.Name,
		SourceID:      req.SourceID,
		EndpointID:    req.EndpointID,
		MaxRetries:    req.MaxRetries,
		RetryInterval: req.RetryInterval,
		RetryType:     req.RetryType,
		FilterCfg:     req.FilterCfg,
		SigningSecret: signingSecret,
	})
	if err != nil {
		slog.Error("create subscription failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create subscription")
		return
	}
	writeJSON(w, http.StatusCreated, sub)
}

func (h *subscriptionHandler) get(w http.ResponseWriter, r *http.Request) {
	sub, err := h.db.GetSubscription(r.Context(), projectID(r), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "subscription not found")
			return
		}
		slog.Error("get subscription failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get subscription")
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

func (h *subscriptionHandler) list(w http.ResponseWriter, r *http.Request) {
	subs, err := h.db.ListSubscriptions(r.Context(), projectID(r))
	if err != nil {
		slog.Error("list subscriptions failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list subscriptions")
		return
	}
	if subs == nil {
		subs = []domain.Subscription{}
	}
	writeJSON(w, http.StatusOK, subs)
}

type updateSubscriptionRequest struct {
	IsActive *bool `json:"is_active"`
}

func (h *subscriptionHandler) update(w http.ResponseWriter, r *http.Request) {
	var req updateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.IsActive == nil {
		writeError(w, http.StatusBadRequest, "is_active is required")
		return
	}
	sub, err := h.db.UpdateSubscription(r.Context(), projectID(r), r.PathValue("id"), *req.IsActive)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "subscription not found")
			return
		}
		slog.Error("update subscription failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update subscription")
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

func (h *subscriptionHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.db.DeleteSubscription(r.Context(), projectID(r), r.PathValue("id")); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "subscription not found")
			return
		}
		slog.Error("delete subscription failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to delete subscription")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
