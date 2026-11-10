package handlers

import (
	"net/http"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type SearchHandler struct {
	svc service.SearchService
}

func NewSearchHandler(svc service.SearchService) SearchHandler {
	return SearchHandler{svc: svc}
}

// Search  GET /search?q=&project_id=&limit=  full-text search over the current
// version of documents in projects the caller belongs to.
func (h SearchHandler) Search(w http.ResponseWriter, r *http.Request) {
	hits, err := h.svc.SearchDocuments(r.Context(), query.TextSearchFilter{
		Query:     r.URL.Query().Get("q"),
		ProjectID: r.URL.Query().Get("project_id"),
		Scope:     scopeFrom(r),
		Limit:     parseIntOrDefault(r.URL.Query().Get("limit"), 20),
	})
	if err != nil {
		writeServiceError(w, err, "not found", "failed to search documents")
		return
	}
	response.WriteData(w, http.StatusOK, hits)
}
