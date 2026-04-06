package handlers

import (
	"errors"
	"net/http"

	"digidocs-mgt/backend-go/internal/config"
	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type InternalAssistantContextHandler struct {
	cfg       config.Config
	documents service.DocumentService
	versions  service.VersionService
	flows     service.FlowService
	handovers service.HandoverService
	dashboard service.DashboardQueryService
}

func NewInternalAssistantContextHandler(
	cfg config.Config,
	documents service.DocumentService,
	versions service.VersionService,
	flows service.FlowService,
	handovers service.HandoverService,
	dashboard service.DashboardQueryService,
) InternalAssistantContextHandler {
	return InternalAssistantContextHandler{
		cfg:       cfg,
		documents: documents,
		versions:  versions,
		flows:     flows,
		handovers: handovers,
		dashboard: dashboard,
	}
}

func (h InternalAssistantContextHandler) GetDocumentContext(w http.ResponseWriter, r *http.Request) {
	if !workerAuthorized(r, h.cfg) {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid worker callback token")
		return
	}

	documentID := r.PathValue("documentID")
	document, err := h.documents.GetDocument(r.Context(), documentID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "not_found", "document not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load document context")
		return
	}

	versions, err := h.versions.List(r.Context(), documentID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load version context")
		return
	}

	flows, err := h.flows.ListFlows(r.Context(), documentID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load flow context")
		return
	}

	response.WriteData(w, http.StatusOK, map[string]any{
		"scope": map[string]any{
			"document_id": documentID,
		},
		"document": document,
		"versions": versions,
		"flows":    flows,
	})
}

func (h InternalAssistantContextHandler) GetProjectContext(w http.ResponseWriter, r *http.Request) {
	if !workerAuthorized(r, h.cfg) {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid worker callback token")
		return
	}

	projectID := r.PathValue("projectID")
	overview, err := h.dashboard.Overview(r.Context(), projectID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load project overview")
		return
	}

	recentFlows, err := h.dashboard.RecentFlows(r.Context(), projectID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load recent flow context")
		return
	}

	riskDocuments, err := h.dashboard.RiskDocuments(r.Context(), projectID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load risk document context")
		return
	}

	response.WriteData(w, http.StatusOK, map[string]any{
		"scope": map[string]any{
			"project_id": projectID,
		},
		"overview":       overview,
		"recent_flows":   recentFlows,
		"risk_documents": riskDocuments,
	})
}

func (h InternalAssistantContextHandler) GetHandoverContext(w http.ResponseWriter, r *http.Request) {
	if !workerAuthorized(r, h.cfg) {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid worker callback token")
		return
	}

	handoverID := r.PathValue("handoverID")
	handover, err := h.handovers.Get(r.Context(), handoverID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "not_found", "handover not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load handover context")
		return
	}

	response.WriteData(w, http.StatusOK, map[string]any{
		"scope": map[string]any{
			"handover_id": handoverID,
			"project_id":  handover.ProjectID,
		},
		"handover": handover,
	})
}
