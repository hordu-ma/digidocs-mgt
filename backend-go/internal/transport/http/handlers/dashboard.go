package handlers

import (
	"errors"
	"net/http"

	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type DashboardHandler struct {
	queryService service.DashboardQueryService
}

func NewDashboardHandler(queryService service.DashboardQueryService) DashboardHandler {
	return DashboardHandler{queryService: queryService}
}

func (h DashboardHandler) Overview(w http.ResponseWriter, r *http.Request) {
	data, err := h.queryService.Overview(r.Context(), r.URL.Query().Get("project_id"))
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to build dashboard overview")
		return
	}

	response.WriteData(w, http.StatusOK, data)
}

func (h DashboardHandler) RecentFlows(w http.ResponseWriter, r *http.Request) {
	items, err := h.queryService.RecentFlows(r.Context(), r.URL.Query().Get("project_id"))
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load recent flows")
		return
	}

	response.WriteWithMeta(w, http.StatusOK, items, map[string]string{
		"project_id": r.URL.Query().Get("project_id"),
	})
}

func (h DashboardHandler) RiskDocuments(w http.ResponseWriter, r *http.Request) {
	items, err := h.queryService.RiskDocuments(r.Context(), r.URL.Query().Get("project_id"))
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "not_found", "project not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load risk documents")
		return
	}

	response.WriteWithMeta(w, http.StatusOK, items, map[string]string{
		"project_id": r.URL.Query().Get("project_id"),
	})
}
