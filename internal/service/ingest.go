package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"

	"hookrelay/internal/domain"
	"hookrelay/internal/filter"
	"hookrelay/internal/metrics"
	"hookrelay/internal/storage"
	"hookrelay/internal/verifier"
)

type ingestStore interface {
	GetSourceByUID(ctx context.Context, uid string) (*domain.Source, error)
	CreateEvent(ctx context.Context, p storage.CreateEventParams) (*domain.Event, error)
	ListActiveSubscriptionsForSource(ctx context.Context, projectID string, sourceID *string) ([]domain.Subscription, error)
	CreateDelivery(ctx context.Context, p storage.CreateDeliveryParams) (*domain.EventDelivery, error)
}

type IngestService struct {
	store ingestStore
}

func NewIngestService(store ingestStore) *IngestService {
	return &IngestService{store: store}
}

type IngestResult struct {
	EventID    string `json:"event_id"`
	Dispatched int    `json:"dispatched"`
}

func (s *IngestService) Ingest(ctx context.Context, sourceUID string, r *http.Request, body []byte, idempotencyKey *string) (*IngestResult, error) {
	src, err := s.store.GetSourceByUID(ctx, sourceUID)
	if err != nil {
		return nil, err
	}
	if !src.IsActive {
		return nil, domain.ErrNotFound
	}

	v, err := verifier.New(src.VerifierType, src.VerifierCfg)
	if err != nil {
		return nil, err
	}
	if err := v.Verify(r, body); err != nil {
		return nil, domain.ErrUnauthorized
	}

	eventType := r.Header.Get("X-HookRelay-Event")
	if eventType == "" {
		eventType = r.URL.Query().Get("event_type")
	}

	rawHeaders := make(map[string]string, len(r.Header))
	for k := range r.Header {
		rawHeaders[k] = r.Header.Get(k)
	}
	headersJSON, _ := json.Marshal(rawHeaders)

	event, err := s.store.CreateEvent(ctx, storage.CreateEventParams{
		ProjectID:      src.ProjectID,
		SourceID:       &src.ID,
		EventType:      eventType,
		Headers:        headersJSON,
		Payload:        body,
		SenderIP:       senderIP(r),
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return nil, err
	}

	metrics.EventsIngested.WithLabelValues(sourceUID, eventType).Inc()

	if !event.IsNew {
		return &IngestResult{EventID: event.ID, Dispatched: 0}, nil
	}

	subs, err := s.store.ListActiveSubscriptionsForSource(ctx, src.ProjectID, &src.ID)
	if err != nil {
		return nil, err
	}

	dispatched := 0
	for _, sub := range subs {
		var fc *domain.FilterConfig
		if sub.FilterCfg != nil {
			fc = &domain.FilterConfig{}
			if err := json.Unmarshal(sub.FilterCfg, fc); err != nil {
				slog.Error("unmarshal filter_cfg failed", "subscription_id", sub.ID, "error", err)
			}
		}
		if !filter.Match(fc, event.EventType) {
			continue
		}
		subID := sub.ID
		_, createErr := s.store.CreateDelivery(ctx, storage.CreateDeliveryParams{
			ProjectID:      src.ProjectID,
			EventID:        event.ID,
			EndpointID:     sub.EndpointID,
			SubscriptionID: &subID,
			MaxRetries:     sub.MaxRetries,
			RetryInterval:  sub.RetryInterval,
			RetryType:      sub.RetryType,
		})
		if createErr != nil {
			slog.Error("create delivery failed", "subscription_id", sub.ID, "event_id", event.ID, "error", createErr)
			continue
		}
		dispatched++
	}

	return &IngestResult{EventID: event.ID, Dispatched: dispatched}, nil
}

func senderIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
