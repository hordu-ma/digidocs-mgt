package handlers

import (
	"net/http"

	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/middleware"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type ProjectHandler struct {
	queryService service.QueryService
}

func NewProjectHandler(queryService service.QueryService) ProjectHandler {
	return ProjectHandler{queryService: queryService}
}

func (h ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	actorID := middleware.UserIDFromContext(r.Context())
	actorRole := middleware.UserRoleFromContext(r.Context())
	items, err := h.queryService.ListProjects(r.Context(), r.URL.Query().Get("team_space_id"), actorID, actorRole)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to list projects")
		return
	}

	response.WriteData(w, http.StatusOK, items)
}

func (h ProjectHandler) FolderTree(w http.ResponseWriter, r *http.Request) {
	items, err := h.queryService.GetFolderTree(r.Context(), r.PathValue("projectID"))
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to get folder tree")
		return
	}

	response.WriteData(w, http.StatusOK, items)
}
