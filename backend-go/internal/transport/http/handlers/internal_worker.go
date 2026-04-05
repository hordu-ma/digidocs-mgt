package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"digidocs-mgt/backend-go/internal/config"
	"digidocs-mgt/backend-go/internal/domain/task"
	"digidocs-mgt/backend-go/internal/queue"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type InternalWorkerHandler struct {
	cfg      config.Config
	consumer queue.Consumer
}

func NewInternalWorkerHandler(cfg config.Config, consumer queue.Consumer) InternalWorkerHandler {
	return InternalWorkerHandler{cfg: cfg, consumer: consumer}
}

func (h InternalWorkerHandler) ReceiveResult(w http.ResponseWriter, r *http.Request) {
	if !h.authorized(r) {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid worker callback token")
		return
	}

	var result task.Result
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
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
	if !h.authorized(r) {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid worker callback token")
		return
	}

	messages := h.consumer.Poll(r.Context(), 10)
	if messages == nil {
		messages = []task.Message{}
	}

	response.WriteData(w, http.StatusOK, messages)
}

func (h InternalWorkerHandler) authorized(r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer"))
	return token == h.cfg.WorkerCallbackToken
}
