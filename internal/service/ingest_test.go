package service_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hookrelay/internal/domain"
	"hookrelay/internal/service"
	"hookrelay/internal/storage"
)

type mockIngestStore struct {
	getSource      func(ctx context.Context, uid string) (*domain.Source, error)
	createEvent    func(ctx context.Context, p storage.CreateEventParams) (*domain.Event, error)
	listSubs       func(ctx context.Context, projectID string, sourceID *string) ([]domain.Subscription, error)
	createDelivery func(ctx context.Context, p storage.CreateDeliveryParams) (*domain.EventDelivery, error)
}

func (m *mockIngestStore) GetSourceByUID(ctx context.Context, uid string) (*domain.Source, error) {
	return m.getSource(ctx, uid)
}
func (m *mockIngestStore) CreateEvent(ctx context.Context, p storage.CreateEventParams) (*domain.Event, error) {
	return m.createEvent(ctx, p)
}
func (m *mockIngestStore) ListActiveSubscriptionsForSource(ctx context.Context, projectID string, sourceID *string) ([]domain.Subscription, error) {
	return m.listSubs(ctx, projectID, sourceID)
}
func (m *mockIngestStore) CreateDelivery(ctx context.Context, p storage.CreateDeliveryParams) (*domain.EventDelivery, error) {
	return m.createDelivery(ctx, p)
}

func noopSource() *domain.Source {
	return &domain.Source{ID: "src-1", ProjectID: "proj-1", IsActive: true, VerifierType: ""}
}

func stubEvent(eventType string) *domain.Event {
	return &domain.Event{ID: "evt-1", EventType: eventType, IsNew: true}
}

func newIngestRequest(body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, "/ingest/uid", strings.NewReader(body))
}

func TestIngest_SourceNotFound(t *testing.T) {
	svc := service.NewIngestService(&mockIngestStore{
		getSource: func(_ context.Context, _ string) (*domain.Source, error) {
			return nil, domain.ErrNotFound
		},
	})
	_, err := svc.Ingest(context.Background(), "uid", newIngestRequest("{}"), []byte("{}"), nil)
	if err != domain.ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestIngest_SourceInactive(t *testing.T) {
	svc := service.NewIngestService(&mockIngestStore{
		getSource: func(_ context.Context, _ string) (*domain.Source, error) {
			return &domain.Source{IsActive: false}, nil
		},
	})
	_, err := svc.Ingest(context.Background(), "uid", newIngestRequest("{}"), []byte("{}"), nil)
	if err != domain.ErrNotFound {
		t.Fatalf("want ErrNotFound for inactive source, got %v", err)
	}
}

func TestIngest_HMACVerificationFailed(t *testing.T) {
	cfgJSON, _ := json.Marshal(domain.HMACVerifierConfig{Header: "X-Sig", Secret: "secret", Encoding: "hex"})
	svc := service.NewIngestService(&mockIngestStore{
		getSource: func(_ context.Context, _ string) (*domain.Source, error) {
			return &domain.Source{
				ID: "src-1", ProjectID: "proj-1", IsActive: true,
				VerifierType: domain.VerifierHMAC, VerifierCfg: cfgJSON,
			}, nil
		},
	})
	_, err := svc.Ingest(context.Background(), "uid", newIngestRequest(`{"x":1}`), []byte(`{"x":1}`), nil)
	if err != domain.ErrUnauthorized {
		t.Fatalf("want ErrUnauthorized, got %v", err)
	}
}

func TestIngest_Success_NoSubscriptions(t *testing.T) {
	svc := service.NewIngestService(&mockIngestStore{
		getSource:   func(_ context.Context, _ string) (*domain.Source, error) { return noopSource(), nil },
		createEvent: func(_ context.Context, _ storage.CreateEventParams) (*domain.Event, error) { return stubEvent(""), nil },
		listSubs:    func(_ context.Context, _ string, _ *string) ([]domain.Subscription, error) { return nil, nil },
	})
	res, err := svc.Ingest(context.Background(), "uid", newIngestRequest("{}"), []byte("{}"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.EventID != "evt-1" || res.Dispatched != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestIngest_DuplicateIdempotencyKey_SkipsDispatch(t *testing.T) {
	dispatched := 0
	svc := service.NewIngestService(&mockIngestStore{
		getSource: func(_ context.Context, _ string) (*domain.Source, error) { return noopSource(), nil },
		createEvent: func(_ context.Context, _ storage.CreateEventParams) (*domain.Event, error) {
			return &domain.Event{ID: "evt-1", IsNew: false}, nil
		},
		listSubs: func(_ context.Context, _ string, _ *string) ([]domain.Subscription, error) {
			return []domain.Subscription{
				{ID: "sub-1", EndpointID: "ep-1", MaxRetries: 3, RetryInterval: 5, RetryType: domain.RetryExponential},
			}, nil
		},
		createDelivery: func(_ context.Context, _ storage.CreateDeliveryParams) (*domain.EventDelivery, error) {
			dispatched++
			return &domain.EventDelivery{}, nil
		},
	})
	key := "evt-live-001"
	res, err := svc.Ingest(context.Background(), "uid", newIngestRequest("{}"), []byte("{}"), &key)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.EventID != "evt-1" || res.Dispatched != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
	if dispatched != 0 {
		t.Fatalf("expected no deliveries created for duplicate event, got %d", dispatched)
	}
}

func TestIngest_Success_TwoSubscriptions(t *testing.T) {
	dispatched := 0
	svc := service.NewIngestService(&mockIngestStore{
		getSource:   func(_ context.Context, _ string) (*domain.Source, error) { return noopSource(), nil },
		createEvent: func(_ context.Context, _ storage.CreateEventParams) (*domain.Event, error) { return stubEvent(""), nil },
		listSubs: func(_ context.Context, _ string, _ *string) ([]domain.Subscription, error) {
			return []domain.Subscription{
				{ID: "sub-1", EndpointID: "ep-1", MaxRetries: 3, RetryInterval: 5, RetryType: domain.RetryExponential},
				{ID: "sub-2", EndpointID: "ep-2", MaxRetries: 3, RetryInterval: 5, RetryType: domain.RetryExponential},
			}, nil
		},
		createDelivery: func(_ context.Context, _ storage.CreateDeliveryParams) (*domain.EventDelivery, error) {
			dispatched++
			return &domain.EventDelivery{}, nil
		},
	})
	res, err := svc.Ingest(context.Background(), "uid", newIngestRequest("{}"), []byte("{}"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Dispatched != 2 || dispatched != 2 {
		t.Fatalf("want 2 dispatched, got result=%+v actual=%d", res, dispatched)
	}
}

func TestIngest_FilteredSubscription(t *testing.T) {
	dispatched := 0
	allowFilter, _ := json.Marshal(domain.FilterConfig{EventTypes: []string{"order.created"}})
	denyFilter, _ := json.Marshal(domain.FilterConfig{EventTypes: []string{"order.updated"}})
	svc := service.NewIngestService(&mockIngestStore{
		getSource: func(_ context.Context, _ string) (*domain.Source, error) { return noopSource(), nil },
		createEvent: func(_ context.Context, p storage.CreateEventParams) (*domain.Event, error) {
			return stubEvent(p.EventType), nil
		},
		listSubs: func(_ context.Context, _ string, _ *string) ([]domain.Subscription, error) {
			return []domain.Subscription{
				{ID: "sub-1", EndpointID: "ep-1", FilterCfg: allowFilter},
				{ID: "sub-2", EndpointID: "ep-2", FilterCfg: denyFilter},
			}, nil
		},
		createDelivery: func(_ context.Context, _ storage.CreateDeliveryParams) (*domain.EventDelivery, error) {
			dispatched++
			return &domain.EventDelivery{}, nil
		},
	})
	req := newIngestRequest("{}")
	req.Header.Set("X-HookRelay-Event", "order.created")
	res, err := svc.Ingest(context.Background(), "uid", req, []byte("{}"), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Dispatched != 1 || dispatched != 1 {
		t.Fatalf("want 1 dispatched, got result=%+v actual=%d", res, dispatched)
	}
}
