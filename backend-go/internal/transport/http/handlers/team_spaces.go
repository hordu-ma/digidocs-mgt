package handlers

import (
	"net/http"

	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/middleware"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type TeamSpaceHandler struct {
	queryService service.QueryService
}

func NewTeamSpaceHandler(queryService service.QueryService) TeamSpaceHandler {
	return TeamSpaceHandler{queryService: queryService}
}

func (h TeamSpaceHandler) List(w http.ResponseWriter, r *http.Request) {
	actorID := middleware.UserIDFromContext(r.Context())
	actorRole := middleware.UserRoleFromContext(r.Context())
	items, err := h.queryService.ListTeamSpaces(r.Context(), actorID, actorRole)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to list team spaces")
		return
	}

	response.WriteData(w, http.StatusOK, items)
}
