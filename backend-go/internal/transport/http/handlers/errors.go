package handlers

import (
	"errors"
	"net/http"

	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

// writeServiceError maps a service-layer sentinel error to an HTTP error
// response, collapsing the error-handling boilerplate shared by most handlers.
//
// notFoundMsg is used for service.ErrNotFound and internalMsg for any
// unrecognized error. Validation, forbidden and conflict errors carry their
// own message. Domain-specific cases (e.g. invalid state transitions with
// bespoke error codes, or auth-specific messages) should be handled by the
// caller before delegating here.
func writeServiceError(w http.ResponseWriter, err error, notFoundMsg, internalMsg string) {
	switch {
	case errors.Is(err, service.ErrValidation):
		response.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, service.ErrNotFound):
		response.WriteError(w, http.StatusNotFound, "not_found", notFoundMsg)
	case errors.Is(err, service.ErrForbidden):
		response.WriteError(w, http.StatusForbidden, "forbidden", "permission denied")
	case errors.Is(err, service.ErrConflict):
		response.WriteError(w, http.StatusConflict, "conflict", err.Error())
	default:
		response.WriteError(w, http.StatusInternalServerError, "internal_error", internalMsg)
	}
}
