package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"hookrelay/internal/auth"
	"hookrelay/internal/domain"
)

type ctxKey string

const projectIDKey ctxKey = "project_id"

type keyLookup interface {
	GetAPIKeyByHash(ctx context.Context, hash string) (*domain.APIKey, error)
}

func Auth(store keyLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}
			rawKey := strings.TrimPrefix(header, "Bearer ")
			key, err := store.GetAPIKeyByHash(r.Context(), auth.Hash(rawKey))
			if err != nil || (key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now())) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}
			ctx := context.WithValue(r.Context(), projectIDKey, key.ProjectID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ProjectID(ctx context.Context) string {
	v, _ := ctx.Value(projectIDKey).(string)
	return v
}

func WithProjectID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, projectIDKey, id)
}
