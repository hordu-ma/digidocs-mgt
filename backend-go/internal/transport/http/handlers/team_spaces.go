package handlers

import (
	"net/http"

	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type TeamSpaceHandler struct {
	queryService service.QueryService
}

func NewTeamSpaceHandler(queryService service.QueryService) TeamSpaceHandler {
	return TeamSpaceHandler{queryService: queryService}
}

func (h TeamSpaceHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.queryService.ListTeamSpaces(r.Context())
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to list team spaces")
		return
	}

	response.WriteData(w, http.StatusOK, items)
}
