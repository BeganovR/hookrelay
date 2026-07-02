package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type ctxKeyReqID struct{}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), ctxKeyReqID{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ReqID(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyReqID{}).(string)
	return v
}
