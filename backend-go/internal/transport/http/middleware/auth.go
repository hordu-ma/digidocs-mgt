package middleware

import (
	"net/http"
	"strings"

	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

// Auth returns a middleware that validates HS256 JWT in the Authorization header.
func Auth(tokenService service.TokenService) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				response.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid authorization header")
				return
			}

			token := authHeader[len("Bearer "):]
			if _, err := tokenService.Parse(token); err != nil {
				response.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
