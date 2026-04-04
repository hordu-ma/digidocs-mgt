package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type DocumentHandler struct {
	queryService service.QueryService
}

func NewDocumentHandler(queryService service.QueryService) DocumentHandler {
	return DocumentHandler{queryService: queryService}
}

func (h DocumentHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := query.DocumentListFilter{
		TeamSpaceID:     r.URL.Query().Get("team_space_id"),
		ProjectID:       r.URL.Query().Get("project_id"),
		FolderID:        r.URL.Query().Get("folder_id"),
		OwnerID:         r.URL.Query().Get("owner_id"),
		Status:          r.URL.Query().Get("status"),
		Keyword:         r.URL.Query().Get("keyword"),
		IncludeArchived: r.URL.Query().Get("include_archived") == "true",
		Page:            parseIntOrDefault(r.URL.Query().Get("page"), 1),
		PageSize:        parseIntOrDefault(r.URL.Query().Get("page_size"), 20),
	}

	items, total, err := h.queryService.ListDocuments(r.Context(), filter)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to list documents")
		return
	}

	response.WriteWithMeta(w, http.StatusOK, items, query.PaginationMeta{
		Page:     filter.Page,
		PageSize: filter.PageSize,
		Total:    total,
	})
}

func (h DocumentHandler) Get(w http.ResponseWriter, r *http.Request) {
	item, err := h.queryService.GetDocument(r.Context(), r.PathValue("documentID"))
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "not_found", "document not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to get document")
		return
	}

	response.WriteData(w, http.StatusOK, item)
}

func parseIntOrDefault(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}

	return value
}
