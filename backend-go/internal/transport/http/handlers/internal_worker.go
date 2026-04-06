package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"digidocs-mgt/backend-go/internal/config"
	"digidocs-mgt/backend-go/internal/domain/task"
	"digidocs-mgt/backend-go/internal/queue"
	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type InternalWorkerHandler struct {
	cfg       config.Config
	consumer  queue.Consumer
	assistant service.AssistantService
}

func NewInternalWorkerHandler(
	cfg config.Config,
	consumer queue.Consumer,
	assistant service.AssistantService,
) InternalWorkerHandler {
	return InternalWorkerHandler{cfg: cfg, consumer: consumer, assistant: assistant}
}

func (h InternalWorkerHandler) ReceiveResult(w http.ResponseWriter, r *http.Request) {
	if !workerAuthorized(r, h.cfg) {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid worker callback token")
		return
	}

	var result task.Result
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	if err := h.assistant.ReceiveResult(r.Context(), result); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "not_found", "assistant request not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to persist worker result")
		return
	}

	response.WriteData(w, http.StatusOK, map[string]any{
		"accepted":   true,
		"request_id": result.RequestID,
		"status":     result.Status,
	})
}

// PollTasks returns up to 10 pending task messages from the queue.
func (h InternalWorkerHandler) PollTasks(w http.ResponseWriter, r *http.Request) {
	if !workerAuthorized(r, h.cfg) {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid worker callback token")
		return
	}

	messages := h.consumer.Poll(r.Context(), 10)
	if messages == nil {
		messages = []task.Message{}
	}

	response.WriteData(w, http.StatusOK, messages)
}
