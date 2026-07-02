package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"hookrelay/internal/domain"
)

type eventStore interface {
	GetEvent(ctx context.Context, projectID, id string) (*domain.Event, error)
	ListEvents(ctx context.Context, projectID string, limit int) ([]domain.Event, error)
	ListDeliveriesForEvent(ctx context.Context, projectID, eventID string) ([]domain.EventDelivery, error)
}

type eventHandler struct {
	db eventStore
}

func (h *eventHandler) get(w http.ResponseWriter, r *http.Request) {
	ev, err := h.db.GetEvent(r.Context(), projectID(r), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "event not found")
			return
		}
		slog.Error("get event failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get event")
		return
	}
	writeJSON(w, http.StatusOK, ev)
}

func (h *eventHandler) list(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	evs, err := h.db.ListEvents(r.Context(), projectID(r), limit)
	if err != nil {
		slog.Error("list events failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list events")
		return
	}
	if evs == nil {
		evs = []domain.Event{}
	}
	writeJSON(w, http.StatusOK, evs)
}

func (h *eventHandler) deliveries(w http.ResponseWriter, r *http.Request) {
	pid := projectID(r)
	eventID := r.PathValue("id")

	if _, err := h.db.GetEvent(r.Context(), pid, eventID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			writeError(w, http.StatusNotFound, "event not found")
			return
		}
		slog.Error("get event failed", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to get event")
		return
	}

	deliveries, err := h.db.ListDeliveriesForEvent(r.Context(), pid, eventID)
	if err != nil {
		slog.Error("list deliveries for event failed", "event_id", eventID, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list deliveries")
		return
	}
	if deliveries == nil {
		deliveries = []domain.EventDelivery{}
	}
	writeJSON(w, http.StatusOK, deliveries)
}
