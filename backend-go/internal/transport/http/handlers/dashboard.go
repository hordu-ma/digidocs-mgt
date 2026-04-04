package handlers

import (
	"net/http"

	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type DashboardHandler struct{}

func NewDashboardHandler() DashboardHandler {
	return DashboardHandler{}
}

func (h DashboardHandler) Overview(w http.ResponseWriter, r *http.Request) {
	response.WriteData(w, http.StatusOK, map[string]any{
		"project_id":             r.URL.Query().Get("project_id"),
		"document_total":         0,
		"status_counts":          map[string]int{},
		"handover_pending_count": 0,
		"risk_document_count":    0,
	})
}

func (h DashboardHandler) RecentFlows(w http.ResponseWriter, r *http.Request) {
	response.WriteWithMeta(w, http.StatusOK, []map[string]any{}, map[string]string{
		"project_id": r.URL.Query().Get("project_id"),
	})
}

func (h DashboardHandler) RiskDocuments(w http.ResponseWriter, r *http.Request) {
	response.WriteWithMeta(w, http.StatusOK, []map[string]any{}, map[string]string{
		"project_id": r.URL.Query().Get("project_id"),
	})
}
