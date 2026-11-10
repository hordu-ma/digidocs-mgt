package handlers

import (
	"fmt"
	"io"
	"net/http"

	"digidocs-mgt/backend-go/internal/config"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

// workerScope is the scope used by the worker-facing internal endpoints. These
// are gated by the shared-secret callback token above, not by a per-user JWT, so
// there is no acting user to narrow against and they run unscoped — matching the
// existing ("", "admin") convention used for the per-resource reads here.
var workerScope = query.ActorScope{IsAdmin: true}

type InternalAssistantContextHandler struct {
	cfg       config.Config
	assistant service.AssistantService
	documents service.DocumentService
	versions  service.VersionService
	flows     service.FlowService
	handovers service.HandoverService
	dashboard service.DashboardQueryService
	search    service.SearchService
}

// WithSearch enables the retrieval endpoint used by assistant.ask.
func (h InternalAssistantContextHandler) WithSearch(search service.SearchService) InternalAssistantContextHandler {
	h.search = search
	return h
}

// SearchProjectText  GET /internal/assistant-context/projects/{projectID}/search?q=&document_id=&limit=
// Retrieval for assistant.ask: the most relevant chunks of the project's (or
// one document's) current-version text. Pinned to the path project; the user
// was already authorized for that scope when the question was queued.
func (h InternalAssistantContextHandler) SearchProjectText(w http.ResponseWriter, r *http.Request) {
	if !workerAuthorized(r, h.cfg) {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid worker callback token")
		return
	}
	projectID := r.PathValue("projectID")
	if projectID == "" {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "project_id is required")
		return
	}
	hits, err := h.search.SearchChunks(r.Context(), query.TextSearchFilter{
		Query:      r.URL.Query().Get("q"),
		ProjectID:  projectID,
		DocumentID: r.URL.Query().Get("document_id"),
		Scope:      workerScope,
		Limit:      min(parseIntOrDefault(r.URL.Query().Get("limit"), 8), 50),
	})
	if err != nil {
		writeServiceError(w, err, "project not found", "failed to search project text")
		return
	}
	response.WriteData(w, http.StatusOK, hits)
}

func NewInternalAssistantContextHandler(
	cfg config.Config,
	assistant service.AssistantService,
	documents service.DocumentService,
	versions service.VersionService,
	flows service.FlowService,
	handovers service.HandoverService,
	dashboard service.DashboardQueryService,
) InternalAssistantContextHandler {
	return InternalAssistantContextHandler{
		cfg:       cfg,
		assistant: assistant,
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
	// This endpoint is gated by the worker shared-secret token above, not a
	// per-user JWT, so there is no "acting user" to scope project membership
	// against; pass the admin role to bypass the view-permission check.
	document, err := h.documents.GetDocument(r.Context(), documentID, "", "admin")
	if err != nil {
		writeServiceError(w, err, "document not found", "failed to load document context")
		return
	}

	versions, err := h.versions.List(r.Context(), documentID, "", "admin")
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load version context")
		return
	}

	flows, err := h.flows.ListFlows(r.Context(), documentID, "", "admin")
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load flow context")
		return
	}

	versionID := r.URL.Query().Get("version_id")
	if versionID == "" {
		versionID = document.CurrentVersionID
	}
	if versionID != "" {
		version, err := h.versions.Get(r.Context(), versionID, "", "admin")
		if err != nil {
			writeServiceError(w, err, "version not found", "failed to load version")
			return
		}
		if version.DocumentID != documentID {
			response.WriteError(w, http.StatusBadRequest, "validation_error", "version does not belong to document")
			return
		}
	}
	extractedText, err := h.assistant.GetDocumentVersionExtractedText(r.Context(), documentID, versionID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load extracted text context")
		return
	}

	response.WriteData(w, http.StatusOK, map[string]any{
		"scope": map[string]any{
			"document_id": documentID,
			"project_id":  document.ProjectID,
		},
		"document":                  document,
		"versions":                  versions,
		"flows":                     flows,
		"extracted_text":            extractedText,
		"version_id":                versionID,
		"extracted_text_version_id": versionID,
	})
}

func (h InternalAssistantContextHandler) GetProjectContext(w http.ResponseWriter, r *http.Request) {
	if !workerAuthorized(r, h.cfg) {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid worker callback token")
		return
	}

	projectID := r.PathValue("projectID")
	overview, err := h.dashboard.Overview(r.Context(), projectID, workerScope)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load project overview")
		return
	}

	recentFlows, err := h.dashboard.RecentFlows(r.Context(), projectID, workerScope)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load recent flow context")
		return
	}

	riskDocuments, err := h.dashboard.RiskDocuments(r.Context(), projectID, workerScope)
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
	handover, err := h.handovers.Get(r.Context(), handoverID, "", "admin")
	if err != nil {
		writeServiceError(w, err, "handover not found", "failed to load handover context")
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

func (h InternalAssistantContextHandler) DownloadVersionFile(w http.ResponseWriter, r *http.Request) {
	if !workerAuthorized(r, h.cfg) {
		response.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid worker callback token")
		return
	}

	ver, obj, err := h.versions.GetFile(r.Context(), r.PathValue("versionID"), "", "admin")
	if err != nil {
		if ver == nil {
			response.WriteError(w, http.StatusNotFound, "not_found", "version not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to retrieve version file")
		return
	}
	defer obj.Reader.Close()

	w.Header().Set("Content-Type", contentTypeFromName(ver.FileName, obj.ContentType))
	w.Header().Set("Content-Disposition", `attachment; filename="`+ver.FileName+`"`)
	if obj.Size > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", obj.Size))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, obj.Reader)
}
