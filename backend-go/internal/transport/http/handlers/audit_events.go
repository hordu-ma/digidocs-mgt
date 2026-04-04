package handlers

import (
	"net/http"
	"strconv"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type AuditEventHandler struct {
	queryService service.AuditQueryService
}

func NewAuditEventHandler(queryService service.AuditQueryService) AuditEventHandler {
	return AuditEventHandler{queryService: queryService}
}

func (h AuditEventHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	items, total, err := h.queryService.List(r.Context(), query.AuditEventFilter{
		ProjectID:  r.URL.Query().Get("project_id"),
		DocumentID: r.URL.Query().Get("document_id"),
		ActionType: r.URL.Query().Get("action_type"),
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to list audit events")
		return
	}

	response.WriteWithMeta(w, http.StatusOK, items, query.PaginationMeta{
		Page:     max(page, 1),
		PageSize: max(pageSize, 20),
		Total:    total,
	})
}

func (h AuditEventHandler) Summary(w http.ResponseWriter, r *http.Request) {
	data, err := h.queryService.Summary(r.Context(), r.URL.Query().Get("project_id"))
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to build audit summary")
		return
	}

	response.WriteData(w, http.StatusOK, data)
}

func max(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
