package handlers

import (
	"encoding/json"
	"net/http"

	"digidocs-mgt/backend-go/internal/domain/auth"
	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/request"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type AuthHandler struct {
	tokenService service.TokenService
}

func NewAuthHandler(tokenService service.TokenService) AuthHandler {
	return AuthHandler{tokenService: tokenService}
}

func (h AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var payload request.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	if payload.Username == "" || payload.Password == "" {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "username and password are required")
		return
	}

	token, err := h.tokenService.Generate(auth.Claims{
		UserID:      "00000000-0000-0000-0000-000000000001",
		Username:    payload.Username,
		DisplayName: "开发用户",
		Role:        "admin",
	})
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to generate token")
		return
	}

	response.WriteData(w, http.StatusOK, map[string]any{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   7200,
		"user": map[string]string{
			"id":           "00000000-0000-0000-0000-000000000001",
			"username":     payload.Username,
			"display_name": "开发用户",
			"role":         "admin",
		},
	})
}

func (h AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	token := service.ExtractBearerToken(r.Header.Get("Authorization"))
	if token == "" {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized", "missing bearer token")
		return
	}

	claims, err := h.tokenService.Parse(token)
	if err != nil {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid bearer token")
		return
	}

	response.WriteData(w, http.StatusOK, map[string]any{
		"id":            claims.UserID,
		"username":      claims.Username,
		"display_name":  claims.DisplayName,
		"role":          claims.Role,
		"last_login_at": nil,
	})
}

func (h AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	_ = r

	response.WriteData(w, http.StatusOK, map[string]bool{
		"success": true,
	})
}
