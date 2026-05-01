package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"digidocs-mgt/backend-go/internal/bootstrap"
	"digidocs-mgt/backend-go/internal/config"
)

func TestRouterPublicSystemRoutes(t *testing.T) {
	cfg := config.Config{
		AppName:          "DigiDocs Test",
		AppEnv:           "test",
		APIV1Prefix:      "/api/v1",
		CORSAllowOrigins: "*",
	}
	handler := New(cfg, bootstrap.Container{})

	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d, want 200", health.Code)
	}
	var healthBody struct {
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal(health.Body.Bytes(), &healthBody); err != nil {
		t.Fatalf("decode health body: %v", err)
	}
	if healthBody.Data["status"] != "ok" {
		t.Fatalf("health body = %#v, want status ok", healthBody)
	}

	info := httptest.NewRecorder()
	infoReq := httptest.NewRequest(http.MethodGet, "/api/v1/system/info", nil)
	infoReq.Header.Set("Origin", "http://localhost:5173")
	handler.ServeHTTP(info, infoReq)
	if info.Code != http.StatusOK {
		t.Fatalf("info status = %d, want 200", info.Code)
	}
	var infoBody struct {
		Data map[string]string `json:"data"`
	}
	if err := json.Unmarshal(info.Body.Bytes(), &infoBody); err != nil {
		t.Fatalf("decode info body: %v", err)
	}
	if infoBody.Data["app_name"] != "DigiDocs Test" || infoBody.Data["env"] != "test" {
		t.Fatalf("info body = %#v, want configured app info", infoBody)
	}
	if got := info.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("CORS header = %q, want *", got)
	}
	if got := info.Header().Get("Content-Type"); got != "application/json" && got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
}

func TestRouterProtectedRouteRequiresToken(t *testing.T) {
	cfg := config.Config{APIV1Prefix: "/api/v1", CORSAllowOrigins: "*"}
	handler := New(cfg, bootstrap.Container{})

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/users", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}
