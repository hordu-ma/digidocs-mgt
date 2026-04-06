package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"digidocs-mgt/backend-go/internal/domain/task"
	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type AssistantHandler struct {
	taskService service.TaskService
}

func NewAssistantHandler(taskService service.TaskService) AssistantHandler {
	return AssistantHandler{taskService: taskService}
}

func (h AssistantHandler) Ask(w http.ResponseWriter, r *http.Request) {
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	scope, _ := payload["scope"].(map[string]any)
	projectID := stringValue(payload["project_id"])
	documentID := stringValue(payload["document_id"])
	if scope != nil {
		if projectID == "" {
			projectID = stringValue(scope["project_id"])
		}
		if documentID == "" {
			documentID = stringValue(scope["document_id"])
		}
	}

	message, err := h.taskService.Publish(
		r.Context(),
		task.TaskTypeAssistantAsk,
		"",
		"",
		payload,
	)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to queue assistant ask")
		return
	}

	response.WriteData(w, http.StatusOK, map[string]any{
		"request_id":   message.RequestID,
		"question":     payload["question"],
		"status":       "queued",
		"answer":       "",
		"generated_at": time.Now().UTC().Format(time.RFC3339),
		"source_scope": map[string]any{
			"project_id":  projectID,
			"document_id": documentID,
		},
	})
}

func (h AssistantHandler) SummarizeDocument(w http.ResponseWriter, r *http.Request) {
	var payload map[string]any
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&payload)
	}

	documentID := r.PathValue("documentID")
	message, err := h.taskService.Publish(
		r.Context(),
		task.TaskTypeDocumentSummarize,
		"document",
		documentID,
		payload,
	)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to queue document summary")
		return
	}

	response.WriteData(w, http.StatusOK, map[string]any{
		"document_id": documentID,
		"status":      "queued",
		"request_id":  message.RequestID,
		"payload":     payload,
	})
}

func (h AssistantHandler) SummarizeHandover(w http.ResponseWriter, r *http.Request) {
	handoverID := r.PathValue("handoverID")
	message, err := h.taskService.Publish(
		r.Context(),
		task.TaskTypeHandoverSummarize,
		"handover",
		handoverID,
		nil,
	)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to queue handover summary")
		return
	}

	response.WriteData(w, http.StatusOK, map[string]any{
		"handover_id": handoverID,
		"status":      "queued",
		"request_id":  message.RequestID,
	})
}

func (h AssistantHandler) ListSuggestions(w http.ResponseWriter, r *http.Request) {
	_ = r
	response.WriteData(w, http.StatusOK, []map[string]any{})
}

func (h AssistantHandler) ConfirmSuggestion(w http.ResponseWriter, r *http.Request) {
	var payload map[string]any
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&payload)
	}

	response.WriteData(w, http.StatusOK, map[string]any{
		"id":      r.PathValue("suggestionID"),
		"action":  "confirm",
		"payload": payload,
	})
}

func (h AssistantHandler) DismissSuggestion(w http.ResponseWriter, r *http.Request) {
	var payload map[string]any
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&payload)
	}

	response.WriteData(w, http.StatusOK, map[string]any{
		"id":      r.PathValue("suggestionID"),
		"action":  "dismiss",
		"payload": payload,
	})
}
