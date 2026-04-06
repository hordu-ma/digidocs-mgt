package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/domain/task"
	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/middleware"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type AssistantHandler struct {
	service service.AssistantService
}

func NewAssistantHandler(service service.AssistantService) AssistantHandler {
	return AssistantHandler{service: service}
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

	message, err := h.service.QueueTask(
		r.Context(),
		task.TaskTypeAssistantAsk,
		"",
		"",
		payload,
		middleware.UserIDFromContext(r.Context()),
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
	message, err := h.service.QueueTask(
		r.Context(),
		task.TaskTypeDocumentSummarize,
		"document",
		documentID,
		payload,
		middleware.UserIDFromContext(r.Context()),
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
	message, err := h.service.QueueTask(
		r.Context(),
		task.TaskTypeHandoverSummarize,
		"handover",
		handoverID,
		nil,
		middleware.UserIDFromContext(r.Context()),
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

func (h AssistantHandler) GetRequest(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.GetRequest(r.Context(), r.PathValue("requestID"))
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "not_found", "assistant request not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to get assistant request")
		return
	}
	response.WriteData(w, http.StatusOK, item)
}

func (h AssistantHandler) ListRequests(w http.ResponseWriter, r *http.Request) {
	page := parseIntOrDefault(r.URL.Query().Get("page"), 1)
	pageSize := parseIntOrDefault(r.URL.Query().Get("page_size"), 20)

	items, total, err := h.service.ListRequests(r.Context(), query.AssistantRequestFilter{
		RequestType: r.URL.Query().Get("request_type"),
		RelatedType: r.URL.Query().Get("related_type"),
		RelatedID:   r.URL.Query().Get("related_id"),
		Status:      r.URL.Query().Get("status"),
		Keyword:     r.URL.Query().Get("keyword"),
		Page:        page,
		PageSize:    pageSize,
	})
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to list assistant requests")
		return
	}

	response.WriteWithMeta(w, http.StatusOK, items, query.PaginationMeta{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	})
}

func (h AssistantHandler) ListSuggestions(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListSuggestions(r.Context(), query.AssistantSuggestionFilter{
		RelatedType:    r.URL.Query().Get("related_type"),
		RelatedID:      r.URL.Query().Get("related_id"),
		Status:         r.URL.Query().Get("status"),
		SuggestionType: r.URL.Query().Get("suggestion_type"),
	})
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to list suggestions")
		return
	}
	response.WriteData(w, http.StatusOK, items)
}

func (h AssistantHandler) ConfirmSuggestion(w http.ResponseWriter, r *http.Request) {
	var payload map[string]any
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&payload)
	}

	data, err := h.service.ConfirmSuggestion(
		r.Context(),
		r.PathValue("suggestionID"),
		middleware.UserIDFromContext(r.Context()),
		stringValue(payload["note"]),
	)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "not_found", "suggestion not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to confirm suggestion")
		return
	}
	response.WriteData(w, http.StatusOK, data)
}

func (h AssistantHandler) DismissSuggestion(w http.ResponseWriter, r *http.Request) {
	var payload map[string]any
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&payload)
	}

	data, err := h.service.DismissSuggestion(
		r.Context(),
		r.PathValue("suggestionID"),
		middleware.UserIDFromContext(r.Context()),
		stringValue(payload["reason"]),
	)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "not_found", "suggestion not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to dismiss suggestion")
		return
	}
	response.WriteData(w, http.StatusOK, data)
}
