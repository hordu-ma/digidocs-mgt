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

func TestUsers_List(t *testing.T) {
	handler, token := testServer(t)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, authedRequest("GET", "/api/v1/users", nil, token))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	result := parseResponse(t, rec)
	items, ok := result["data"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected non-empty user list, got %#v", result["data"])
	}
	first, _ := items[0].(map[string]any)
	if first["display_name"] == "" {
		t.Fatalf("expected display_name in first user item, got %#v", first)
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

func TestVersions_UploadThenDownloadAndPreview(t *testing.T) {
	handler, token := testServer(t)

	var uploadBuf bytes.Buffer
	uploadWriter := multipart.NewWriter(&uploadBuf)
	uploadWriter.WriteField("commit_message", "smoke upload")
	part, _ := uploadWriter.CreateFormFile("file", "smoke.txt")
	_, _ = part.Write([]byte("hello version smoke"))
	uploadWriter.Close()

	uploadReq := httptest.NewRequest("POST", "/api/v1/documents/00000000-0000-0000-0000-000000000100/versions", &uploadBuf)
	uploadReq.Header.Set("Authorization", "Bearer "+token)
	uploadReq.Header.Set("Content-Type", uploadWriter.FormDataContentType())

	uploadRec := httptest.NewRecorder()
	handler.ServeHTTP(uploadRec, uploadReq)
	if uploadRec.Code != http.StatusOK {
		t.Fatalf("upload status = %d, want 200; body = %s", uploadRec.Code, uploadRec.Body.String())
	}
	uploadResult := parseResponse(t, uploadRec)
	uploadData, _ := uploadResult["data"].(map[string]any)
	versionID, _ := uploadData["id"].(string)
	if versionID == "" {
		t.Fatal("expected version id")
	}

	downloadRec := httptest.NewRecorder()
	handler.ServeHTTP(downloadRec, authedRequest("GET", "/api/v1/versions/"+versionID+"/download", nil, token))
	if downloadRec.Code != http.StatusOK {
		t.Fatalf("download status = %d, want 200; body = %s", downloadRec.Code, downloadRec.Body.String())
	}
	if body := downloadRec.Body.String(); body != "hello version smoke" {
		t.Fatalf("download body = %q, want hello version smoke", body)
	}
	if got := downloadRec.Header().Get("Content-Disposition"); !strings.Contains(got, "attachment") {
		t.Fatalf("download content-disposition = %s, want attachment", got)
	}

	previewRec := httptest.NewRecorder()
	handler.ServeHTTP(previewRec, authedRequest("GET", "/api/v1/versions/"+versionID+"/preview", nil, token))
	if previewRec.Code != http.StatusOK {
		t.Fatalf("preview status = %d, want 200; body = %s", previewRec.Code, previewRec.Body.String())
	}
	if body := previewRec.Body.String(); body != "hello version smoke" {
		t.Fatalf("preview body = %q, want hello version smoke", body)
	}
	if got := previewRec.Header().Get("Content-Disposition"); !strings.Contains(got, "inline") {
		t.Fatalf("preview content-disposition = %s, want inline", got)
	}
}

func TestInternalAssistantContext_DownloadVersionFile(t *testing.T) {
	handler, token := testServer(t)

	var uploadBuf bytes.Buffer
	uploadWriter := multipart.NewWriter(&uploadBuf)
	part, _ := uploadWriter.CreateFormFile("file", "assistant.txt")
	_, _ = part.Write([]byte("assistant asset"))
	uploadWriter.Close()

	uploadReq := httptest.NewRequest("POST", "/api/v1/documents/00000000-0000-0000-0000-000000000100/versions", &uploadBuf)
	uploadReq.Header.Set("Authorization", "Bearer "+token)
	uploadReq.Header.Set("Content-Type", uploadWriter.FormDataContentType())
	uploadRec := httptest.NewRecorder()
	handler.ServeHTTP(uploadRec, uploadReq)
	uploadResult := parseResponse(t, uploadRec)
	uploadData, _ := uploadResult["data"].(map[string]any)
	versionID, _ := uploadData["id"].(string)
	if versionID == "" {
		t.Fatal("expected version id")
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, workerRequest("GET", "/api/v1/internal/assistant-assets/versions/"+versionID+"/download", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); body != "assistant asset" {
		t.Fatalf("body = %q, want assistant asset", body)
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

	pendingRec := httptest.NewRecorder()
	handler.ServeHTTP(pendingRec, authedRequest("GET", "/api/v1/assistant/suggestions?related_type=document&related_id=doc-1&status=pending", nil, token))
	if pendingRec.Code != http.StatusOK {
		t.Fatalf("pending list status = %d, want 200; body = %s", pendingRec.Code, pendingRec.Body.String())
	}

	pendingResult := parseResponse(t, pendingRec)
	pendingItems, _ := pendingResult["data"].([]any)
	if len(pendingItems) != 0 {
		t.Fatalf("expected no pending suggestions after confirm, got %#v", pendingResult["data"])
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

func TestAssistant_ListRequests_WithFilters(t *testing.T) {
	handler, token := testServer(t)

	queueRec := httptest.NewRecorder()
	handler.ServeHTTP(queueRec, authedRequest("POST", "/api/v1/assistant/ask", jsonBody(map[string]any{
		"question": "请总结项目进度",
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
			"answer":     "这是 AI 回答",
			"model":      "openclaw/default",
			"request_id": "chatcmpl_test_1",
		},
	})))
	if callbackRec.Code != http.StatusOK {
		t.Fatalf("callback status = %d, want 200; body = %s", callbackRec.Code, callbackRec.Body.String())
	}

	listRec := httptest.NewRecorder()
	handler.ServeHTTP(listRec, authedRequest("GET", "/api/v1/assistant/requests?request_type=assistant.ask&status=completed&keyword=项目&page=1&page_size=10", nil, token))
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200; body = %s", listRec.Code, listRec.Body.String())
	}
	result := parseResponse(t, listRec)
	items, _ := result["data"].([]any)
	if len(items) != 1 {
		t.Fatalf("items len = %d, want 1", len(items))
	}
	first, _ := items[0].(map[string]any)
	if first["question"] != "请总结项目进度" {
		t.Fatalf("question = %v, want 请总结项目进度", first["question"])
	}
	if first["model"] != "openclaw/default" {
		t.Fatalf("model = %v, want openclaw/default", first["model"])
	}
	if first["upstream_request_id"] != "chatcmpl_test_1" {
		t.Fatalf("upstream_request_id = %v, want chatcmpl_test_1", first["upstream_request_id"])
	}
}

func TestAssistant_AskConversationFlow_PersistsMessages(t *testing.T) {
	handler, token := testServer(t)

	queueRec := httptest.NewRecorder()
	handler.ServeHTTP(queueRec, authedRequest("POST", "/api/v1/assistant/ask", jsonBody(map[string]any{
		"question": "请总结课题状态",
		"scope": map[string]any{
			"project_id": "project-1",
		},
	}), token))
	if queueRec.Code != http.StatusOK {
		t.Fatalf("queue status = %d, want 200; body = %s", queueRec.Code, queueRec.Body.String())
	}

	queueResult := parseResponse(t, queueRec)
	queueData, _ := queueResult["data"].(map[string]any)
	requestID, _ := queueData["request_id"].(string)
	conversationID, _ := queueData["conversation_id"].(string)
	if requestID == "" || conversationID == "" {
		t.Fatalf("expected request_id and conversation_id, got %#v", queueData)
	}

	callbackRec := httptest.NewRecorder()
	handler.ServeHTTP(callbackRec, workerRequest("POST", "/api/v1/internal/worker-results", jsonBody(map[string]any{
		"request_id": requestID,
		"status":     "completed",
		"output": map[string]any{
			"answer":     "这是第一轮回答",
			"model":      "openclaw/default",
			"request_id": "chatcmpl_conv_1",
		},
	})))
	if callbackRec.Code != http.StatusOK {
		t.Fatalf("callback status = %d, want 200; body = %s", callbackRec.Code, callbackRec.Body.String())
	}

	convRec := httptest.NewRecorder()
	handler.ServeHTTP(convRec, authedRequest("GET", "/api/v1/assistant/conversations?project_id=project-1", nil, token))
	if convRec.Code != http.StatusOK {
		t.Fatalf("conversation list status = %d, want 200; body = %s", convRec.Code, convRec.Body.String())
	}
	convResult := parseResponse(t, convRec)
	conversations, _ := convResult["data"].([]any)
	if len(conversations) != 1 {
		t.Fatalf("expected 1 conversation, got %#v", convResult["data"])
	}

	msgRec := httptest.NewRecorder()
	handler.ServeHTTP(msgRec, authedRequest("GET", "/api/v1/assistant/conversations/"+conversationID+"/messages", nil, token))
	if msgRec.Code != http.StatusOK {
		t.Fatalf("message list status = %d, want 200; body = %s", msgRec.Code, msgRec.Body.String())
	}
	msgResult := parseResponse(t, msgRec)
	items, _ := msgResult["data"].([]any)
	if len(items) != 2 {
		t.Fatalf("expected 2 messages, got %#v", msgResult["data"])
	}
	first, _ := items[0].(map[string]any)
	second, _ := items[1].(map[string]any)
	if first["role"] != "user" || second["role"] != "assistant" {
		t.Fatalf("unexpected roles: %#v", msgResult["data"])
	}
}

func TestAssistant_AskConversationFollowup_EmbedsMemory(t *testing.T) {
	handler, token := testServer(t)

	firstRec := httptest.NewRecorder()
	handler.ServeHTTP(firstRec, authedRequest("POST", "/api/v1/assistant/ask", jsonBody(map[string]any{
		"question": "请先总结一次",
		"scope": map[string]any{
			"project_id": "project-1",
		},
	}), token))
	firstResult := parseResponse(t, firstRec)
	firstData, _ := firstResult["data"].(map[string]any)
	firstRequestID, _ := firstData["request_id"].(string)
	conversationID, _ := firstData["conversation_id"].(string)

	callbackRec := httptest.NewRecorder()
	handler.ServeHTTP(callbackRec, workerRequest("POST", "/api/v1/internal/worker-results", jsonBody(map[string]any{
		"request_id": firstRequestID,
		"status":     "completed",
		"output": map[string]any{
			"answer": "这是上一轮回答",
		},
	})))

	secondRec := httptest.NewRecorder()
	handler.ServeHTTP(secondRec, authedRequest("POST", "/api/v1/assistant/ask", jsonBody(map[string]any{
		"conversation_id": conversationID,
		"question":        "继续追问上一轮内容",
	}), token))
	if secondRec.Code != http.StatusOK {
		t.Fatalf("second ask status = %d, want 200; body = %s", secondRec.Code, secondRec.Body.String())
	}
	secondResult := parseResponse(t, secondRec)
	secondData, _ := secondResult["data"].(map[string]any)
	secondRequestID, _ := secondData["request_id"].(string)

	pollRec := httptest.NewRecorder()
	handler.ServeHTTP(pollRec, workerRequest("GET", "/api/v1/internal/poll-tasks", nil))
	if pollRec.Code != http.StatusOK {
		t.Fatalf("poll status = %d, want 200; body = %s", pollRec.Code, pollRec.Body.String())
	}
	pollResult := parseResponse(t, pollRec)
	items, _ := pollResult["data"].([]any)
	if len(items) == 0 {
		t.Fatal("expected pending task")
	}
	var taskPayload map[string]any
	for _, raw := range items {
		item, _ := raw.(map[string]any)
		if item["request_id"] == secondRequestID {
			taskPayload = item
			break
		}
	}
	if taskPayload == nil {
		t.Fatalf("expected task for request_id=%s, got %#v", secondRequestID, items)
	}
	payload, _ := taskPayload["payload"].(map[string]any)
	memory, _ := payload["memory"].(map[string]any)
	recentMessages, _ := memory["recent_messages"].([]any)
	memorySources, _ := payload["memory_sources"].([]any)
	if len(recentMessages) == 0 {
		t.Fatalf("expected recent_messages in memory payload, got %#v", payload["memory"])
	}
	if len(memorySources) == 0 {
		t.Fatalf("expected memory_sources in payload, got %#v", payload)
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
	result := parseResponse(t, rec)
	data, _ := result["data"].(map[string]any)
	if data["email"] == "" || data["phone"] == "" || data["wechat"] == "" {
		t.Fatalf("expected contact fields in current user profile, got %#v", data)
	}
}

func TestAuth_UpdateMe(t *testing.T) {
	handler, token := testServer(t)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, authedRequest("PATCH", "/api/v1/auth/me", jsonBody(map[string]string{
		"display_name": "测试管理员",
		"email":        "admin@example.com",
		"phone":        "13900000000",
		"wechat":       "admin_new",
	}), token))
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	result := parseResponse(t, rec)
	data, _ := result["data"].(map[string]any)
	if data["display_name"] != "测试管理员" || data["wechat"] != "admin_new" {
		t.Fatalf("expected updated profile data, got %#v", data)
	}
}
