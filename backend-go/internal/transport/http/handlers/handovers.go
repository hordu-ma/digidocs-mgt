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

type HandoverHandler struct {
	queryService  service.HandoverQueryService
	actionService service.ActionService
}

func NewHandoverHandler(
	queryService service.HandoverQueryService,
	actionService service.ActionService,
) HandoverHandler {
	return HandoverHandler{
		queryService:  queryService,
		actionService: actionService,
	}
}

func (h HandoverHandler) Create(w http.ResponseWriter, r *http.Request) {
	payload := map[string]any{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	data, err := h.actionService.CreateHandover(r.Context(), command.HandoverCreateInput{
		TargetUserID:   stringValue(payload["target_user_id"]),
		ReceiverUserID: stringValue(payload["receiver_user_id"]),
		ProjectID:      stringValue(payload["project_id"]),
		Remark:         stringValue(payload["remark"]),
		ActorID:        middleware.UserIDFromContext(r.Context()),
	})
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to create handover")
		return
	}

	response.WriteData(w, http.StatusOK, data)
}

func (h HandoverHandler) List(w http.ResponseWriter, r *http.Request) {
	_ = r

	items, err := h.queryService.List(r.Context())
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to list handovers")
		return
	}

	response.WriteData(w, http.StatusOK, items)
}

func (h HandoverHandler) Get(w http.ResponseWriter, r *http.Request) {
	item, err := h.queryService.Get(r.Context(), r.PathValue("handoverID"))
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "not_found", "handover not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to get handover")
		return
	}

	response.WriteData(w, http.StatusOK, item)
}

func (h HandoverHandler) UpdateItems(w http.ResponseWriter, r *http.Request) {
	payload := map[string]any{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	items := make([]command.HandoverItemInput, 0)
	if rawItems, ok := payload["items"].([]any); ok {
		for _, raw := range rawItems {
			itemMap, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			items = append(items, command.HandoverItemInput{
				DocumentID: stringValue(itemMap["document_id"]),
				Selected:   boolValue(itemMap["selected"]),
				Note:       stringValue(itemMap["note"]),
			})
		}
	}

	data, err := h.actionService.UpdateHandoverItems(r.Context(), command.HandoverItemUpdateInput{
		HandoverID: r.PathValue("handoverID"),
		Items:      items,
		ActorID:    middleware.UserIDFromContext(r.Context()),
	})
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to update handover items")
		return
	}

	response.WriteData(w, http.StatusOK, data)
}

func (h HandoverHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	h.writeAction(w, r, "confirm")
}

func (h HandoverHandler) Complete(w http.ResponseWriter, r *http.Request) {
	h.writeAction(w, r, "complete")
}

func (h HandoverHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	h.writeAction(w, r, "cancel")
}

func (h HandoverHandler) writeAction(w http.ResponseWriter, r *http.Request, action string) {
	payload := map[string]any{}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&payload)
	}

	data, err := h.actionService.ApplyHandover(r.Context(), command.HandoverActionInput{
		HandoverID: r.PathValue("handoverID"),
		Action:     action,
		Note:       stringValue(payload["note"]),
		Reason:     stringValue(payload["reason"]),
		ActorID:    middleware.UserIDFromContext(r.Context()),
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidTransition) {
			response.WriteError(w, http.StatusBadRequest, "invalid_transition", "invalid handover transition")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to apply handover action")
		return
	}

	response.WriteData(w, http.StatusOK, data)
}

func boolValue(value any) bool {
	raw, ok := value.(bool)
	if !ok {
		return false
	}

	return raw
}
