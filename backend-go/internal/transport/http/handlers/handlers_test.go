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
		APIV1Prefix:         "/api/v1",
		DataBackend:         "memory",
		JWTSecret:           "test-secret-key-for-handler-tests",
		WorkerCallbackToken: "worker-test-token",
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

func workerRequest(method, path string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Authorization", "Bearer worker-test-token")
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

func TestInternalAssistantContext_Document(t *testing.T) {
	handler, _ := testServer(t)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, workerRequest("GET", "/api/v1/internal/assistant-context/documents/some-id", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	result := parseResponse(t, rec)
	data, _ := result["data"].(map[string]any)
	if data["document"] == nil {
		t.Error("expected document context")
	}
	if data["versions"] == nil {
		t.Error("expected version context")
	}
	if data["flows"] == nil {
		t.Error("expected flow context")
	}
}

func TestInternalAssistantContext_Project_Unauthorized(t *testing.T) {
	handler, _ := testServer(t)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/api/v1/internal/assistant-context/projects/p-1", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body = %s", rec.Code, rec.Body.String())
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

// --- Assistant ---

func TestAssistant_DocumentSummarize_PersistsSuggestionAfterWorkerCallback(t *testing.T) {
	handler, token := testServer(t)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, authedRequest("POST", "/api/v1/assistant/documents/doc-1/summarize", jsonBody(map[string]string{
		"version_id": "ver-1",
	}), token))
	if rec.Code != http.StatusOK {
		t.Fatalf("queue status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}

	queueResult := parseResponse(t, rec)
	queueData, _ := queueResult["data"].(map[string]any)
	requestID, _ := queueData["request_id"].(string)
	if requestID == "" {
		t.Fatal("expected request_id in summarize response")
	}

	pollRec := httptest.NewRecorder()
	handler.ServeHTTP(pollRec, workerRequest("GET", "/api/v1/internal/poll-tasks", nil))
	if pollRec.Code != http.StatusOK {
		t.Fatalf("poll status = %d, want 200; body = %s", pollRec.Code, pollRec.Body.String())
	}

	callbackRec := httptest.NewRecorder()
	handler.ServeHTTP(callbackRec, workerRequest("POST", "/api/v1/internal/worker-results", jsonBody(map[string]any{
		"request_id": requestID,
		"status":     "completed",
		"output": map[string]any{
			"summary_text": "这是测试摘要",
			"suggestions": []map[string]any{
				{
					"title":           "测试摘要",
					"content":         "这是测试摘要",
					"suggestion_type": "document_summary",
				},
			},
		},
	})))
	if callbackRec.Code != http.StatusOK {
		t.Fatalf("callback status = %d, want 200; body = %s", callbackRec.Code, callbackRec.Body.String())
	}

	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, authedRequest("GET", "/api/v1/assistant/suggestions?related_type=document&related_id=doc-1", nil, token))
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200; body = %s", listRec.Code, listRec.Body.String())
	}

	listResult := parseResponse(t, listRec)
	items, ok := listResult["data"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected persisted suggestions, got %#v", listResult["data"])
	}
}

func TestAssistant_ConfirmSuggestion(t *testing.T) {
	handler, token := testServer(t)

	queueRec := httptest.NewRecorder()
	handler.ServeHTTP(queueRec, authedRequest("POST", "/api/v1/assistant/documents/doc-1/summarize", jsonBody(map[string]string{}), token))
	queueResult := parseResponse(t, queueRec)
	queueData, _ := queueResult["data"].(map[string]any)
	requestID, _ := queueData["request_id"].(string)

	callbackRec := httptest.NewRecorder()
	handler.ServeHTTP(callbackRec, workerRequest("POST", "/api/v1/internal/worker-results", jsonBody(map[string]any{
		"request_id": requestID,
		"status":     "completed",
		"output": map[string]any{
			"summary_text": "这是测试摘要",
		},
	})))

	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, authedRequest("GET", "/api/v1/assistant/suggestions?related_type=document&related_id=doc-1", nil, token))
	listResult := parseResponse(t, listRec)
	items, _ := listResult["data"].([]any)
	first, _ := items[0].(map[string]any)
	suggestionID, _ := first["id"].(string)
	if suggestionID == "" {
		t.Fatal("expected suggestion id")
	}

	confirmRec := httptest.NewRecorder()
	handler.ServeHTTP(confirmRec, authedRequest("POST", "/api/v1/assistant/suggestions/"+suggestionID+"/confirm", jsonBody(map[string]string{
		"note": "采纳",
	}), token))
	if confirmRec.Code != http.StatusOK {
		t.Fatalf("confirm status = %d, want 200; body = %s", confirmRec.Code, confirmRec.Body.String())
	}
}

func TestAssistant_GetRequest_ReturnsWorkerOutput(t *testing.T) {
	handler, token := testServer(t)

	queueRec := httptest.NewRecorder()
	handler.ServeHTTP(queueRec, authedRequest("POST", "/api/v1/assistant/ask", jsonBody(map[string]any{
		"question": "请总结当前状态",
		"scope": map[string]any{
			"project_id": "project-1",
		},
	}), token))
	queueResult := parseResponse(t, queueRec)
	queueData, _ := queueResult["data"].(map[string]any)
	requestID, _ := queueData["request_id"].(string)
	if requestID == "" {
		t.Fatal("expected request_id")
	}

	callbackRec := httptest.NewRecorder()
	handler.ServeHTTP(callbackRec, workerRequest("POST", "/api/v1/internal/worker-results", jsonBody(map[string]any{
		"request_id": requestID,
		"status":     "completed",
		"output": map[string]any{
			"answer": "这是 AI 回答",
		},
	})))
	if callbackRec.Code != http.StatusOK {
		t.Fatalf("callback status = %d, want 200; body = %s", callbackRec.Code, callbackRec.Body.String())
	}

	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, authedRequest("GET", "/api/v1/assistant/requests/"+requestID, nil, token))
	if getRec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200; body = %s", getRec.Code, getRec.Body.String())
	}
	result := parseResponse(t, getRec)
	data, _ := result["data"].(map[string]any)
	output, _ := data["output"].(map[string]any)
	if output["answer"] != "这是 AI 回答" {
		t.Fatalf("answer = %v, want 这是 AI 回答", output["answer"])
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
