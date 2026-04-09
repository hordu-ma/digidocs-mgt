package handlers

import (
	"net/http"

	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type UserHandler struct {
	queryService service.QueryService
}

func NewUserHandler(queryService service.QueryService) UserHandler {
	return UserHandler{queryService: queryService}
}

func (h UserHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.queryService.ListUsers(r.Context())
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to list users")
		return
	}

	response.WriteData(w, http.StatusOK, items)
}
