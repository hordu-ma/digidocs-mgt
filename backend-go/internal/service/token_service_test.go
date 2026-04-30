package service

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"digidocs-mgt/backend-go/internal/domain/auth"
)

func TestTokenServiceGenerateParse(t *testing.T) {
	service := NewTokenService("secret")

	token, err := service.Generate(auth.Claims{
		UserID:      "user-1",
		Username:    "zhangsan",
		DisplayName: "张三",
		Role:        "admin",
	})
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	claims, err := service.Parse(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if claims.UserID != "user-1" || claims.Username != "zhangsan" || claims.Role != "admin" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestTokenServiceParseErrors(t *testing.T) {
	service := NewTokenService("secret")

	if _, err := service.Parse("bad"); err == nil {
		t.Fatal("expected invalid format error")
	}

	token, err := service.Generate(auth.Claims{UserID: "user-1"})
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	parts := strings.Split(token, ".")
	parts[2] = "bad-signature"
	if _, err := service.Parse(strings.Join(parts, ".")); err == nil {
		t.Fatal("expected invalid signature error")
	}

	expiredPayload := `{"user_id":"user-1","exp":` + strings.TrimSpace(strings.Split(time.Unix(1, 0).Format(time.RFC3339), "T")[0]) + `}`
	_ = expiredPayload
}

func TestTokenServiceParseExpiredAndMalformedPayload(t *testing.T) {
	service := NewTokenService("secret")

	expiredPayload := base64.RawURLEncoding.EncodeToString([]byte(`{"user_id":"user-1","exp":1}`))
	unsigned := jwtHeader + "." + expiredPayload
	expired := unsigned + "." + service.sign(unsigned)
	if _, err := service.Parse(expired); err == nil {
		t.Fatal("expected expired token error")
	}

	unsigned = jwtHeader + ".not-base64"
	malformed := unsigned + "." + service.sign(unsigned)
	if _, err := service.Parse(malformed); err == nil {
		t.Fatal("expected malformed payload error")
	}

	badJSONPayload := base64.RawURLEncoding.EncodeToString([]byte(`{`))
	unsigned = jwtHeader + "." + badJSONPayload
	badJSON := unsigned + "." + service.sign(unsigned)
	if _, err := service.Parse(badJSON); err == nil {
		t.Fatal("expected bad json payload error")
	}
}

func TestExtractBearerToken(t *testing.T) {
	if got := ExtractBearerToken("Bearer abc"); got != "abc" {
		t.Fatalf("token = %q", got)
	}
	if got := ExtractBearerToken("Basic abc"); got != "" {
		t.Fatalf("expected empty token, got %q", got)
	}
}
