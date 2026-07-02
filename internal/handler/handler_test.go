package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hookrelay/internal/domain"
	"hookrelay/internal/middleware"
	"hookrelay/internal/service"
)

type mockPinger struct{ err error }

func (m *mockPinger) Ping(_ context.Context) error { return m.err }

type mockProjectStore struct {
	proj   *domain.Project
	key    *domain.APIKey
	err    error
	getErr error
}

func (m *mockProjectStore) CreateProject(_ context.Context, _, _ string) (*domain.Project, error) {
	return m.proj, m.err
}
func (m *mockProjectStore) GetProject(_ context.Context, _ string) (*domain.Project, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.proj, m.err
}
func (m *mockProjectStore) UpdateProject(_ context.Context, _, _, _ string) (*domain.Project, error) {
	return m.proj, m.err
}
func (m *mockProjectStore) DeleteProject(_ context.Context, _ string) error { return m.err }
func (m *mockProjectStore) CreateAPIKey(_ context.Context, _, _, _, _ string) (*domain.APIKey, error) {
	return m.key, m.err
}

type mockIngestSvc struct {
	result *service.IngestResult
	err    error
}

type mockLimiter struct{ allow bool }

func (m *mockLimiter) Allow(_ string) bool { return m.allow }

func (m *mockIngestSvc) Ingest(_ context.Context, _ string, _ *http.Request, _ []byte, _ *string) (*service.IngestResult, error) {
	return m.result, m.err
}

func TestHealth_OK(t *testing.T) {
	h := &healthHandler{db: &mockPinger{}}
	r := httptest.NewRequest(http.MethodGet, "/v1/healthz", nil)
	w := httptest.NewRecorder()
	h.health(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %q", body["status"])
	}
}

func TestHealth_Unhealthy(t *testing.T) {
	h := &healthHandler{db: &mockPinger{err: errors.New("connection refused")}}
	r := httptest.NewRequest(http.MethodGet, "/v1/healthz", nil)
	w := httptest.NewRecorder()
	h.health(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestProject_Create_OK(t *testing.T) {
	now := time.Now()
	proj := &domain.Project{ID: "proj-1", Name: "test-proj", CreatedAt: now, UpdatedAt: now}
	key := &domain.APIKey{ID: "key-1", ProjectID: "proj-1", Prefix: "hr_testpfx"}
	h := &projectHandler{db: &mockProjectStore{proj: proj, key: key}}

	body := `{"name":"test-proj","description":"desc"}`
	r := httptest.NewRequest(http.MethodPost, "/v1/projects", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.create(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp createProjectResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Project == nil || resp.Project.ID != "proj-1" {
		t.Errorf("unexpected project: %+v", resp.Project)
	}
	if resp.APIKey == nil || resp.APIKey.RawKey == "" {
		t.Error("expected non-empty raw_key in response")
	}
}

func TestProject_Create_MissingName(t *testing.T) {
	h := &projectHandler{db: &mockProjectStore{}}
	r := httptest.NewRequest(http.MethodPost, "/v1/projects", strings.NewReader(`{"description":"no name here"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.create(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestProject_Create_InvalidJSON(t *testing.T) {
	h := &projectHandler{db: &mockProjectStore{}}
	r := httptest.NewRequest(http.MethodPost, "/v1/projects", strings.NewReader(`not json at all`))
	w := httptest.NewRecorder()
	h.create(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestProject_Get_OK(t *testing.T) {
	now := time.Now()
	proj := &domain.Project{ID: "proj-1", Name: "test", CreatedAt: now, UpdatedAt: now}
	h := &projectHandler{db: &mockProjectStore{proj: proj}}

	r := httptest.NewRequest(http.MethodGet, "/v1/projects/proj-1", nil)
	r.SetPathValue("id", "proj-1")
	r = r.WithContext(withProjectID(r.Context(), "proj-1"))
	w := httptest.NewRecorder()
	h.get(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestProject_Get_NotFound(t *testing.T) {
	h := &projectHandler{db: &mockProjectStore{getErr: domain.ErrNotFound}}
	r := httptest.NewRequest(http.MethodGet, "/v1/projects/missing", nil)
	r.SetPathValue("id", "proj-1")
	r = r.WithContext(withProjectID(r.Context(), "proj-1"))
	w := httptest.NewRecorder()
	h.get(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestProject_Get_Forbidden(t *testing.T) {
	now := time.Now()
	proj := &domain.Project{ID: "proj-1", Name: "test", CreatedAt: now, UpdatedAt: now}
	h := &projectHandler{db: &mockProjectStore{proj: proj}}

	r := httptest.NewRequest(http.MethodGet, "/v1/projects/proj-other", nil)
	r.SetPathValue("id", "proj-other")
	r = r.WithContext(withProjectID(r.Context(), "proj-1"))
	w := httptest.NewRecorder()
	h.get(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 (cross-project access), got %d", w.Code)
	}
}

func TestIngest_OK(t *testing.T) {
	svc := &mockIngestSvc{result: &service.IngestResult{EventID: "evt-123", Dispatched: 2}}
	h := &ingestHandler{svc: svc, limiter: &mockLimiter{allow: true}}

	r := httptest.NewRequest(http.MethodPost, "/ingest/src-uid", bytes.NewBufferString(`{"order_id":42}`))
	r.SetPathValue("uid", "src-uid")
	w := httptest.NewRecorder()
	h.ingest(w, r)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
	var result service.IngestResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.EventID != "evt-123" {
		t.Errorf("expected event_id=evt-123, got %q", result.EventID)
	}
	if result.Dispatched != 2 {
		t.Errorf("expected dispatched=2, got %d", result.Dispatched)
	}
}

func TestIngest_SourceNotFound(t *testing.T) {
	h := &ingestHandler{svc: &mockIngestSvc{err: domain.ErrNotFound}, limiter: &mockLimiter{allow: true}}
	r := httptest.NewRequest(http.MethodPost, "/ingest/unknown", bytes.NewBufferString(`{}`))
	r.SetPathValue("uid", "unknown")
	w := httptest.NewRecorder()
	h.ingest(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestIngest_SignatureFailed(t *testing.T) {
	h := &ingestHandler{svc: &mockIngestSvc{err: domain.ErrUnauthorized}, limiter: &mockLimiter{allow: true}}
	r := httptest.NewRequest(http.MethodPost, "/ingest/src", bytes.NewBufferString(`{}`))
	r.SetPathValue("uid", "src")
	w := httptest.NewRecorder()
	h.ingest(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestIngest_PayloadTooLarge(t *testing.T) {
	svc := &mockIngestSvc{result: &service.IngestResult{EventID: "evt-1", Dispatched: 0}}
	h := &ingestHandler{svc: svc, limiter: &mockLimiter{allow: true}}

	large := bytes.Repeat([]byte("x"), 2<<20)
	r := httptest.NewRequest(http.MethodPost, "/ingest/src", bytes.NewReader(large))
	r.SetPathValue("uid", "src")
	w := httptest.NewRecorder()
	h.ingest(w, r)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d", w.Code)
	}
}

func TestIngest_RateLimited(t *testing.T) {
	svc := &mockIngestSvc{result: &service.IngestResult{EventID: "evt-1", Dispatched: 0}}
	h := &ingestHandler{svc: svc, limiter: &mockLimiter{allow: false}}

	r := httptest.NewRequest(http.MethodPost, "/ingest/src", bytes.NewBufferString(`{}`))
	r.SetPathValue("uid", "src")
	w := httptest.NewRecorder()
	h.ingest(w, r)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
}

func withProjectID(ctx context.Context, id string) context.Context {
	return middleware.WithProjectID(ctx, id)
}
