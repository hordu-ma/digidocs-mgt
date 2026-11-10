package handlers

import (
	"net/http"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type SearchHandler struct {
	svc       service.SearchService
	documents service.DocumentService
	audit     service.AuditService
}

func NewSearchHandler(svc service.SearchService, documents service.DocumentService, audit service.AuditService) SearchHandler {
	return SearchHandler{svc: svc, documents: documents, audit: audit}
}

// DocumentText  GET /documents/{documentID}/text — read the current version's
// extracted text in the browser. This is the preview for Office files, which
// browsers cannot render inline. It hands out content, so like preview it is
// audited fail-closed.
func (h SearchHandler) DocumentText(w http.ResponseWriter, r *http.Request) {
	actorID, actorRole := actorFrom(r)
	documentID := r.PathValue("documentID")
	doc, err := h.documents.GetDocument(r.Context(), documentID, actorID, actorRole)
	if err != nil {
		writeServiceError(w, err, "document not found", "failed to get document")
		return
	}
	text, err := h.svc.DocumentText(r.Context(), doc.ID)
	if err != nil {
		writeServiceError(w, err, "document not found", "failed to load document text")
		return
	}
	if text.Available {
		if err := h.audit.Record(r.Context(), accessEvent(r, service.AuditActionPreview, doc.ID, text.VersionID, map[string]any{
			"project_id": doc.ProjectID, "mode": "text",
		})); err != nil {
			response.WriteError(w, http.StatusInternalServerError, "audit_failed", "预览记录写入失败，请稍后重试")
			return
		}
	}
	response.WriteData(w, http.StatusOK, text)
}

// Search  GET /search?q=&project_id=&limit=  full-text search over the current
// version of documents in projects the caller belongs to.
func (h SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	hits, err := h.svc.SearchDocuments(r.Context(), query.TextSearchFilter{
		Query:     r.URL.Query().Get("q"),
		ProjectID: r.URL.Query().Get("project_id"),
		Scope:     scopeFrom(r),
		Limit:     min(parseIntOrDefault(r.URL.Query().Get("limit"), 20), 50),
	})
	if err != nil {
		writeServiceError(w, err, "not found", "failed to search documents")
		return
	}
	response.WriteData(w, http.StatusOK, hits)
}
