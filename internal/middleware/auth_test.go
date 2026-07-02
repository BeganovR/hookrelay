package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hookrelay/internal/auth"
	"hookrelay/internal/domain"
	"hookrelay/internal/middleware"
)

type fakeKeyStore struct {
	keys map[string]*domain.APIKey
}

func (f *fakeKeyStore) GetAPIKeyByHash(_ context.Context, hash string) (*domain.APIKey, error) {
	k, ok := f.keys[hash]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return k, nil
}

func TestAuth_ValidKey(t *testing.T) {
	rawKey, keyHash, _, _ := auth.Generate()
	store := &fakeKeyStore{
		keys: map[string]*domain.APIKey{
			keyHash: {ID: "key-1", ProjectID: "proj-1", KeyHash: keyHash},
		},
	}
	var capturedProjectID string
	h := middleware.Auth(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedProjectID = middleware.ProjectID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
	r := httptest.NewRequest(http.MethodGet, "/v1/projects/proj-1", nil)
	r.Header.Set("Authorization", "Bearer "+rawKey)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if capturedProjectID != "proj-1" {
		t.Errorf("expected project_id=proj-1, got %q", capturedProjectID)
	}
}

func TestAuth_MissingHeader(t *testing.T) {
	store := &fakeKeyStore{keys: map[string]*domain.APIKey{}}
	h := middleware.Auth(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	r := httptest.NewRequest(http.MethodGet, "/v1/projects/x", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuth_WrongKey(t *testing.T) {
	store := &fakeKeyStore{keys: map[string]*domain.APIKey{}}
	h := middleware.Auth(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	r := httptest.NewRequest(http.MethodGet, "/v1/projects/x", nil)
	r.Header.Set("Authorization", "Bearer hr_wrongkeyvalue")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuth_ExpiredKey(t *testing.T) {
	rawKey, keyHash, _, _ := auth.Generate()
	past := time.Now().Add(-time.Hour)
	store := &fakeKeyStore{
		keys: map[string]*domain.APIKey{
			keyHash: {ID: "key-1", ProjectID: "proj-1", KeyHash: keyHash, ExpiresAt: &past},
		},
	}
	h := middleware.Auth(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	r := httptest.NewRequest(http.MethodGet, "/v1/projects/proj-1", nil)
	r.Header.Set("Authorization", "Bearer "+rawKey)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for expired key, got %d", w.Code)
	}
}
