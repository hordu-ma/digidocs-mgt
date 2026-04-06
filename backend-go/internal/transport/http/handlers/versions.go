package handlers

import (
	"errors"
	"net/http"

	"digidocs-mgt/backend-go/internal/service"
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
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "file is required")
		return
	}
	defer file.Close()

	data, err := h.service.UploadAndCreateVersion(
		r.Context(),
		r.PathValue("documentID"),
		header.Filename,
		header.Size,
		r.FormValue("commit_message"),
		file,
		middleware.UserIDFromContext(r.Context()),
	)
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			response.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to upload version")
		return
	}

	response.WriteData(w, http.StatusOK, data)
}
