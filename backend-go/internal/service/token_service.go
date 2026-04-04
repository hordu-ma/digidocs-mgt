package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"digidocs-mgt/backend-go/internal/domain/auth"
)

// jwtHeader is the base64url encoding of {"alg":"HS256","typ":"JWT"}.
const jwtHeader = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"

type TokenService struct {
	secret string
}

func NewTokenService(secret string) TokenService {
	return TokenService{secret: secret}
}

// Generate creates a signed HS256 JWT valid for 2 hours.
func (s TokenService) Generate(claims auth.Claims) (string, error) {
	now := time.Now().Unix()
	payload := map[string]any{
		"user_id":      claims.UserID,
		"username":     claims.Username,
		"display_name": claims.DisplayName,
		"role":         claims.Role,
		"iat":          now,
		"exp":          now + 7200,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	payloadEncoded := base64.RawURLEncoding.EncodeToString(raw)
	unsigned := jwtHeader + "." + payloadEncoded
	sig := s.sign(unsigned)

	return unsigned + "." + sig, nil
}

// Parse validates the HS256 signature and expiry, then returns the embedded claims.
func (s TokenService) Parse(token string) (auth.Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return auth.Claims{}, errors.New("invalid token format")
	}

	unsigned := parts[0] + "." + parts[1]
	expected := s.sign(unsigned)
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return auth.Claims{}, errors.New("invalid token signature")
	}

	rawPayload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return auth.Claims{}, fmt.Errorf("invalid token payload: %w", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(rawPayload, &payload); err != nil {
		return auth.Claims{}, fmt.Errorf("invalid token payload: %w", err)
	}

	if exp, ok := payload["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return auth.Claims{}, errors.New("token expired")
		}
	}

	return auth.Claims{
		UserID:      stringClaim(payload, "user_id"),
		Username:    stringClaim(payload, "username"),
		DisplayName: stringClaim(payload, "display_name"),
		Role:        stringClaim(payload, "role"),
	}, nil
}

func (s TokenService) sign(input string) string {
	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write([]byte(input))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func stringClaim(payload map[string]any, key string) string {
	val, _ := payload[key].(string)
	return val
}

// ExtractBearerToken extracts the token from an "Authorization: Bearer <token>" header.
func ExtractBearerToken(authHeader string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return ""
	}
	return authHeader[len(prefix):]
}
