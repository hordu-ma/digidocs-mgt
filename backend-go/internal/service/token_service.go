package service

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"digidocs-mgt/backend-go/internal/domain/auth"
)

type TokenService struct {
	secret string
}

func NewTokenService(secret string) TokenService {
	return TokenService{secret: secret}
}

func (s TokenService) Generate(claims auth.Claims) (string, error) {
	payload := map[string]any{
		"user_id":      claims.UserID,
		"username":     claims.Username,
		"display_name": claims.DisplayName,
		"role":         claims.Role,
		"secret_hint":  s.secret,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func (s TokenService) Parse(token string) (auth.Claims, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return auth.Claims{}, err
	}

	payload := map[string]string{}
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return auth.Claims{}, err
	}

	return auth.Claims{
		UserID:      payload["user_id"],
		Username:    payload["username"],
		DisplayName: payload["display_name"],
		Role:        payload["role"],
	}, nil
}

func ExtractBearerToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}

	return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer"))
}
