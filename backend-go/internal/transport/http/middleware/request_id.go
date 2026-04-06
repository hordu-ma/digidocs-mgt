package middleware

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"digidocs-mgt/backend-go/internal/shared"
)

var requestCounter uint64

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-Id")
		if requestID == "" {
			requestID = newRequestID()
		}

		w.Header().Set("X-Request-Id", requestID)
		ctx := shared.WithRequestID(r.Context(), requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func newRequestID() string {
	seq := atomic.AddUint64(&requestCounter, 1)
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), seq)
}
