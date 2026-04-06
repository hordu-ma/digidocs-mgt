package handlers_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"digidocs-mgt/backend-go/internal/bootstrap"
	"digidocs-mgt/backend-go/internal/config"
	"digidocs-mgt/backend-go/internal/domain/auth"
	"digidocs-mgt/backend-go/internal/transport/http/router"
)

// testServer builds a full HTTP handler backed by memory repositories.
func testServer(t *testing.T) (http.Handler, string) {
	t.Helper()

	cfg := config.Config{
		APIV1Prefix: "/api/v1",
		DataBackend: "memory",
		JWTSecret:   "test-secret-key-for-handler-tests",
	}

	container, err := bootstrap.BuildContainer(cfg)
	if err != nil {
		t.Fatalf("failed to build container: %v", err)
	}

	handler := router.New(cfg, container)

	// Generate a valid JWT for test requests.
	token, err := container.TokenService.Generate(auth.Claims{
		UserID:      "00000000-0000-0000-0000-000000000001",
		Username:    "testuser",
		DisplayName: "Test User",
		Role:        "admin",
	})
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	return handler, token
}

func authedRequest(method, path string, body io.Reader, token string) *http.Request {
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func jsonBody(v any) io.Reader {
	data, _ := json.Marshal(v)
	return bytes.NewReader(data)
}

func parseResponse(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response body: %v (body: %s)", err, rec.Body.String())
	}
	return result
}

// --- Healthz (smoke) ---

func TestHealthz(t *testing.T) {
	handler, _ := testServer(t)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

// --- Document CRUD ---

func TestDocuments_List(t *testing.T) {
	handler, token := testServer(t)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, authedRequest("GET", "/api/v1/documents?page=1&page_size=10", nil, token))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	result := parseResponse(t, rec)
	if result["data"] == nil {
		t.Error("expected data in response")
	}
}

func TestDocuments_GetByID(t *testing.T) {
	handler, token := testServer(t)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, authedRequest("GET", "/api/v1/documents/some-id", nil, token))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
}

func TestDocuments_Update(t *testing.T) {
	handler, token := testServer(t)
	rec := httptest.NewRecorder()
	body := jsonBody(map[string]string{"title": "Updated Title"})
	handler.ServeHTTP(rec, authedRequest("PATCH", "/api/v1/documents/some-id", body, token))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	result := parseResponse(t, rec)
	data, _ := result["data"].(map[string]any)
	if data["title"] != "Updated Title" {
		t.Errorf("title = %v, want Updated Title", data["title"])
	}
}

func TestDocuments_Update_NoFields(t *testing.T) {
	handler, token := testServer(t)
	rec := httptest.NewRecorder()
	body := jsonBody(map[string]string{})
	handler.ServeHTTP(rec, authedRequest("PATCH", "/api/v1/documents/some-id", body, token))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
}

func TestDocuments_Delete(t *testing.T) {
	handler, token := testServer(t)
	rec := httptest.NewRecorder()
	body := jsonBody(map[string]string{"reason": "test"})
	handler.ServeHTTP(rec, authedRequest("POST", "/api/v1/documents/some-id/delete", body, token))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	result := parseResponse(t, rec)
	data, _ := result["data"].(map[string]any)
	if data["is_deleted"] != true {
		t.Errorf("is_deleted = %v, want true", data["is_deleted"])
	}
}

func TestDocuments_Restore(t *testing.T) {
	handler, token := testServer(t)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, authedRequest("POST", "/api/v1/documents/some-id/restore", strings.NewReader("{}"), token))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
}

func TestDocuments_Create_Multipart(t *testing.T) {
	handler, token := testServer(t)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.WriteField("team_space_id", "ts-1")
	writer.WriteField("project_id", "p-1")
	writer.WriteField("title", "New Doc")
	part, _ := writer.CreateFormFile("file", "test.pdf")
	part.Write([]byte("fake-pdf-content"))
	writer.Close()

	req := httptest.NewRequest("POST", "/api/v1/documents", &buf)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201; body = %s", rec.Code, rec.Body.String())
	}
}

// --- Unauthenticated Access ---

func TestDocuments_Unauthenticated(t *testing.T) {
	handler, _ := testServer(t)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/documents", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

// --- Dashboard ---

func TestDashboard_Overview(t *testing.T) {
	handler, token := testServer(t)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, authedRequest("GET", "/api/v1/dashboard/overview", nil, token))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
}

func TestDashboard_RecentFlows(t *testing.T) {
	handler, token := testServer(t)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, authedRequest("GET", "/api/v1/dashboard/recent-flows", nil, token))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
}

func TestDashboard_RiskDocuments(t *testing.T) {
	handler, token := testServer(t)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, authedRequest("GET", "/api/v1/dashboard/risk-documents", nil, token))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
}

// --- Handovers ---

func TestHandovers_List(t *testing.T) {
	handler, token := testServer(t)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, authedRequest("GET", "/api/v1/handovers", nil, token))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
}

func TestHandovers_Create(t *testing.T) {
	handler, token := testServer(t)
	rec := httptest.NewRecorder()
	body := jsonBody(map[string]string{
		"target_user_id":   "u-1",
		"receiver_user_id": "u-2",
	})
	handler.ServeHTTP(rec, authedRequest("POST", "/api/v1/handovers", body, token))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
}

func TestHandovers_Create_MissingFields(t *testing.T) {
	handler, token := testServer(t)
	rec := httptest.NewRecorder()
	body := jsonBody(map[string]string{})
	handler.ServeHTTP(rec, authedRequest("POST", "/api/v1/handovers", body, token))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
}

// --- Audit Events ---

func TestAuditEvents_List(t *testing.T) {
	handler, token := testServer(t)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, authedRequest("GET", "/api/v1/audit-events?page=1", nil, token))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
}

func TestAuditEvents_Summary(t *testing.T) {
	handler, token := testServer(t)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, authedRequest("GET", "/api/v1/audit-events/summary", nil, token))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
}

// --- Auth ---

func TestAuth_Me(t *testing.T) {
	handler, token := testServer(t)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, authedRequest("GET", "/api/v1/auth/me", nil, token))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
}
