package handlers

import (
	"fmt"
	"net/http"

	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type VersionHandler struct {
	uploadService service.UploadService
	auditService  service.AuditService
	queryService  service.VersionQueryService
}

func NewVersionHandler(
	uploadService service.UploadService,
	auditService service.AuditService,
	queryService service.VersionQueryService,
) VersionHandler {
	return VersionHandler{
		uploadService: uploadService,
		auditService:  auditService,
		queryService:  queryService,
	}
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

	documentID := r.PathValue("documentID")
	objectKey := fmt.Sprintf("documents/%s/%s", documentID, header.Filename)

	result, err := h.uploadService.Upload(r.Context(), objectKey, file)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to upload file")
		return
	}

	_ = h.auditService.Record(r.Context(), "replace_version", "", documentID, map[string]any{
		"file_name":  header.Filename,
		"object_key": result.ObjectKey,
		"provider":   result.Provider,
	})

	response.WriteData(w, http.StatusOK, map[string]any{
		"document_id": documentID,
		"file_name":   header.Filename,
		"storage":     result,
		"status":      "uploaded",
	})
}
