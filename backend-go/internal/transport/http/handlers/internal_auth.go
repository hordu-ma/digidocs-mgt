package handlers

import (
	"net/http"
	"strings"

	"digidocs-mgt/backend-go/internal/config"
)

func workerAuthorized(r *http.Request, cfg config.Config) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer"))
	return token == cfg.WorkerCallbackToken
}
