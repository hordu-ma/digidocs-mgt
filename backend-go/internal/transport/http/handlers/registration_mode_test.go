package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"digidocs-mgt/backend-go/internal/bootstrap"
	"digidocs-mgt/backend-go/internal/config"
	"digidocs-mgt/backend-go/internal/domain/auth"
	"digidocs-mgt/backend-go/internal/transport/http/router"
)

func registrationServer(t *testing.T, mode string) (http.Handler, string) {
	t.Helper()
	cfg := config.Config{APIV1Prefix: "/api/v1", DataBackend: "memory", JWTSecret: "s", WorkerCallbackToken: "w", CodeRepoRoot: t.TempDir(), SelfRegistration: mode}
	container, err := bootstrap.BuildContainer(cfg)
	if err != nil {
		t.Fatal(err)
	}
	token, _ := container.TokenService.Generate(auth.Claims{UserID: "00000000-0000-0000-0000-000000000001", Role: "admin"})
	return router.New(cfg, container), token
}

func register(handler http.Handler, email string, invite string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("POST", "/api/v1/auth/register", jsonBody(map[string]string{
		"email": email, "password": "password1", "display_name": "新用户", "invite_token": invite,
	})))
	return rec
}

func TestRegistrationModes(t *testing.T) {
	open, _ := registrationServer(t, "")
	if rec := register(open, "a@example.com", ""); rec.Code != http.StatusOK {
		t.Fatalf("open mode: %d %s", rec.Code, rec.Body.String())
	}

	closed, _ := registrationServer(t, "closed")
	if rec := register(closed, "b@example.com", ""); rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "registration_closed") {
		t.Fatalf("closed mode: %d %s", rec.Code, rec.Body.String())
	}

	invite, admin := registrationServer(t, "invite_only")
	if rec := register(invite, "c@example.com", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("invite_only without token: %d", rec.Code)
	}
	if rec := register(invite, "c@example.com", "not-a-token"); rec.Code != http.StatusForbidden {
		t.Fatalf("invite_only bad token: %d", rec.Code)
	}
	created := httptest.NewRecorder()
	invite.ServeHTTP(created, authedRequest("POST", "/api/v1/projects/00000000-0000-0000-0000-000000000020/invites", strings.NewReader("{}"), admin))
	data, _ := parseResponse(t, created)["data"].(map[string]any)
	tok, _ := data["token"].(string)
	if rec := register(invite, "c@example.com", tok); rec.Code != http.StatusOK {
		t.Fatalf("invite_only valid token: %d %s", rec.Code, rec.Body.String())
	}

	info := httptest.NewRecorder()
	invite.ServeHTTP(info, httptest.NewRequest("GET", "/api/v1/system/info", nil))
	if !strings.Contains(info.Body.String(), `"self_registration":"invite_only"`) {
		t.Fatalf("system info: %s", info.Body.String())
	}
}
