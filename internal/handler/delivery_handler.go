package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"hookrelay/internal/domain"
)

type deliveryStore interface {
	GetDelivery(ctx context.Context, projectID, id string) (*domain.EventDelivery, error)
	ListDeliveryAttempts(ctx context.Context, deliveryID string) ([]domain.DeliveryAttempt, error)
	ResetDeliveryForRetry(ctx context.Context, projectID, id string) error
}

type deliveryHandler struct {
	db deliveryStore
}

func (h *deliveryHandler) get(w http.ResponseWriter, r *http.Request) {
	d, err := h.db.GetDelivery(r.Context(), projectID(r), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "delivery not found")
			return
		}
		slog.Error("get delivery failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get delivery")
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (h *deliveryHandler) attempts(w http.ResponseWriter, r *http.Request) {
	deliveryID := r.PathValue("id")

	if _, err := h.db.GetDelivery(r.Context(), projectID(r), deliveryID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "delivery not found")
			return
		}
		slog.Error("get delivery failed", "delivery_id", deliveryID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get delivery")
		return
	}

	attempts, err := h.db.ListDeliveryAttempts(r.Context(), deliveryID)
	if err != nil {
		slog.Error("list delivery attempts failed", "delivery_id", deliveryID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list attempts")
		return
	}
	if attempts == nil {
		attempts = []domain.DeliveryAttempt{}
	}
	writeJSON(w, http.StatusOK, attempts)
}

func (h *deliveryHandler) retry(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.db.ResetDeliveryForRetry(r.Context(), projectID(r), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "delivery not found or not retryable")
			return
		}
		slog.Error("reset delivery for retry failed", "delivery_id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to retry delivery")
		return
	}
	d, err := h.db.GetDelivery(r.Context(), projectID(r), id)
	if err != nil {
		slog.Error("get delivery failed", "delivery_id", id, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get delivery")
		return
	}
	writeJSON(w, http.StatusAccepted, d)
}
