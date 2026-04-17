package handlers

import (
	"errors"
	"log"
	"net/http"

	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/shared"
	"digidocs-mgt/backend-go/internal/transport/http/middleware"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type VersionHandler struct {
	service service.VersionService
}

func NewVersionHandler(svc service.VersionService) VersionHandler {
	return VersionHandler{service: svc}
}

func (h VersionHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(shared.MaxUploadSize); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "file is required")
		return
	}
	defer file.Close()

	if !shared.ValidateFileName(header.Filename) {
		response.WriteError(w, http.StatusBadRequest, "validation_error", "file type not allowed")
		return
	}

	data, err := h.service.UploadAndCreateVersion(
		r.Context(),
		r.PathValue("documentID"),
		header.Filename,
		header.Size,
		r.FormValue("commit_message"),
		file,
		middleware.UserIDFromContext(r.Context()),
		middleware.UserRoleFromContext(r.Context()),
	)
	if err != nil {
		log.Printf("[versions] upload failed document=%s actor=%s err=%v", r.PathValue("documentID"), middleware.UserIDFromContext(r.Context()), err)
		if errors.Is(err, service.ErrValidation) {
			response.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			response.WriteError(w, http.StatusForbidden, "forbidden", "permission denied")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to upload version")
		return
	}

	response.WriteData(w, http.StatusOK, data)
}
