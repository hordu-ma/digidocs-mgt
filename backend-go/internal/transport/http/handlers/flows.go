package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/middleware"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type FlowHandler struct {
	service service.FlowService
}

func NewFlowHandler(svc service.FlowService) FlowHandler {
	return FlowHandler{service: svc}
}

func (h FlowHandler) MarkInProgress(w http.ResponseWriter, r *http.Request) {
	h.writeAction(w, r, "mark_in_progress")
}

func (h FlowHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	h.writeAction(w, r, "transfer")
}

func (h FlowHandler) AcceptTransfer(w http.ResponseWriter, r *http.Request) {
	h.writeAction(w, r, "accept_transfer")
}

func (h FlowHandler) Finalize(w http.ResponseWriter, r *http.Request) {
	h.writeAction(w, r, "finalize")
}

func (h FlowHandler) Archive(w http.ResponseWriter, r *http.Request) {
	h.writeAction(w, r, "archive")
}

func (h FlowHandler) Unarchive(w http.ResponseWriter, r *http.Request) {
	h.writeAction(w, r, "unarchive")
}

func (h FlowHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListFlows(r.Context(), r.PathValue("documentID"))
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to list flows")
		return
	}

	response.WriteWithMeta(w, http.StatusOK, items, map[string]string{
		"document_id": r.PathValue("documentID"),
	})
}

func (h FlowHandler) writeAction(w http.ResponseWriter, r *http.Request, action string) {
	payload := map[string]any{}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&payload)
	}

	data, err := h.service.ApplyAction(r.Context(), command.FlowActionInput{
		DocumentID: r.PathValue("documentID"),
		Action:     action,
		Note:       stringValue(payload["note"]),
		ToUserID:   stringValue(payload["to_user_id"]),
		ActorID:    middleware.UserIDFromContext(r.Context()),
	})
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			response.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		if errors.Is(err, service.ErrInvalidTransition) {
			response.WriteError(w, http.StatusBadRequest, "invalid_status_transition", "invalid flow transition")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to apply flow action")
		return
	}

	response.WriteData(w, http.StatusOK, data)
}

func stringValue(value any) string {
	raw, ok := value.(string)
	if !ok {
		return ""
	}

	return raw
}
