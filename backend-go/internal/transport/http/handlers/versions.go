package handlers

import (
	"errors"
	"log"
	"net/http"

	"digidocs-mgt/backend-go/internal/domain/task"
	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/shared"
	"digidocs-mgt/backend-go/internal/transport/http/middleware"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type VersionHandler struct {
	service   service.VersionService
	assistant service.AssistantService
}

func NewVersionHandler(svc service.VersionService, assistant ...service.AssistantService) VersionHandler {
	h := VersionHandler{service: svc}
	if len(assistant) > 0 {
		h.assistant = assistant[0]
	}
	return h
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

	// P1: auto-trigger text extraction in background
	h.queueExtraction(r, r.PathValue("documentID"), data)
}

func (h VersionHandler) queueExtraction(r *http.Request, documentID string, versionData map[string]any) {
	if (h.assistant == service.AssistantService{}) {
		return
	}
	versionID, _ := versionData["id"].(string)
	fileName, _ := versionData["file_name"].(string)
	if versionID == "" {
		return
	}
	actorID := middleware.UserIDFromContext(r.Context())
	_, err := h.assistant.QueueTask(
		r.Context(),
		task.TaskTypeDocumentExtractText,
		"document", documentID,
		map[string]any{
			"version_id": versionID,
			"file_name":  fileName,
		},
		actorID,
	)
	if err != nil {
		log.Printf("[versions] auto-extract queue failed document=%s version=%s err=%v", documentID, versionID, err)
	} else {
		log.Printf("[versions] auto-extract queued document=%s version=%s", documentID, versionID)
	}
}
