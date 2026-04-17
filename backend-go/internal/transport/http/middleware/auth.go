package middleware

import (
	"context"
	"net/http"
	"strings"

	"digidocs-mgt/backend-go/internal/domain/auth"
	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type contextKey string

const claimsKey contextKey = "auth_claims"

// Auth returns a middleware that validates HS256 JWT in the Authorization header
// and injects the parsed claims into the request context.
func Auth(tokenService service.TokenService) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				response.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid authorization header")
				return
			}

			token := authHeader[len("Bearer "):]
			claims, err := tokenService.Parse(token)
			if err != nil {
				response.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext extracts auth claims from the request context.
func ClaimsFromContext(ctx context.Context) (auth.Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(auth.Claims)
	return claims, ok
}

// UserIDFromContext extracts the user ID from the request context.
// Returns the system user ID if no claims are found.
func UserIDFromContext(ctx context.Context) string {
	claims, ok := ClaimsFromContext(ctx)
	if !ok || claims.UserID == "" {
		return "00000000-0000-0000-0000-000000000001"
	}
	return claims.UserID
}

func UserRoleFromContext(ctx context.Context) string {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return ""
	}
	return claims.Role
}
