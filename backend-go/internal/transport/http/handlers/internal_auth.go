package handlers

import (
	"crypto/subtle"
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
	// Constant-time comparison: this token gates the internal worker-results
	// callback and task-poll endpoints, so a timing side channel could in
	// principle help an attacker on the same network guess it byte-by-byte.
	expected := cfg.WorkerCallbackToken
	if len(token) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1
}
