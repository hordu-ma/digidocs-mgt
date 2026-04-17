package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"digidocs-mgt/backend-go/internal/domain/auth"
	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/request"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type AuthHandler struct {
	authService  service.AuthService
	tokenService service.TokenService
}

func NewAuthHandler(authService service.AuthService, tokenService service.TokenService) AuthHandler {
	return AuthHandler{authService: authService, tokenService: tokenService}
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

	token, claims, err := h.authService.Login(r.Context(), payload.Username, payload.Password)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			response.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid username or password")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "login failed")
		return
	}

	response.WriteData(w, http.StatusOK, map[string]any{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   7200,
		"user": map[string]string{
			"id":           claims.UserID,
			"username":     claims.Username,
			"display_name": claims.DisplayName,
			"role":         claims.Role,
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

	profile, err := h.authService.GetProfile(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			response.WriteError(w, http.StatusUnauthorized, "unauthorized", "user not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load current user")
		return
	}

	response.WriteData(w, http.StatusOK, profile)
}

func (h AuthHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
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

	var payload request.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	profile, err := h.authService.UpdateProfile(r.Context(), claims.UserID, auth.ProfileUpdateInput{
		DisplayName: payload.DisplayName,
		Email:       payload.Email,
		Phone:       payload.Phone,
		Wechat:      payload.Wechat,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid profile data")
		case errors.Is(err, service.ErrNotFound):
			response.WriteError(w, http.StatusUnauthorized, "unauthorized", "user not found")
		default:
			response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to update current user")
		}
		return
	}

	response.WriteData(w, http.StatusOK, profile)
}

func (h AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	_ = r

	response.WriteData(w, http.StatusOK, map[string]bool{
		"success": true,
	})
}
