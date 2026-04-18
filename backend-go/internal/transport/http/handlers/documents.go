package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/domain/task"
	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/shared"
	"digidocs-mgt/backend-go/internal/transport/http/middleware"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type DocumentHandler struct {
	service   service.DocumentService
	assistant service.AssistantService
}

func NewDocumentHandler(svc service.DocumentService, assistant ...service.AssistantService) DocumentHandler {
	h := DocumentHandler{service: svc}
	if len(assistant) > 0 {
		h.assistant = assistant[0]
	}
	return h
}

func (h DocumentHandler) Create(w http.ResponseWriter, r *http.Request) {
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

	actorID := middleware.UserIDFromContext(r.Context())
	actorRole := middleware.UserRoleFromContext(r.Context())
	ownerID := r.FormValue("current_owner_id")
	if ownerID == "" {
		ownerID = actorID
	}

	data, err := h.service.CreateWithFirstVersion(
		r.Context(),
		command.DocumentCreateInput{
			TeamSpaceID:    r.FormValue("team_space_id"),
			ProjectID:      r.FormValue("project_id"),
			FolderID:       r.FormValue("folder_id"),
			Title:          r.FormValue("title"),
			Description:    r.FormValue("description"),
			CurrentOwnerID: ownerID,
			ActorID:        actorID,
			ActorRole:      actorRole,
		},
		header.Filename,
		header.Size,
		r.FormValue("commit_message"),
		file,
	)
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			response.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			response.WriteError(w, http.StatusForbidden, "forbidden", "permission denied")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to create document")
		return
	}

	response.WriteData(w, http.StatusCreated, data)

	// P1: auto-trigger text extraction for new document
	h.queueExtraction(r, data, header.Filename)
}

func (h DocumentHandler) queueExtraction(r *http.Request, data map[string]any, fileName string) {
	if (h.assistant == service.AssistantService{}) {
		return
	}
	documentID, _ := data["id"].(string)
	var versionID string
	if cv, ok := data["current_version"].(map[string]any); ok {
		versionID, _ = cv["id"].(string)
	}
	if documentID == "" || versionID == "" {
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
		log.Printf("[documents] auto-extract queue failed document=%s version=%s err=%v", documentID, versionID, err)
	} else {
		log.Printf("[documents] auto-extract queued document=%s version=%s", documentID, versionID)
	}
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
		PageSize:        parsePageSize(r.URL.Query().Get("page_size")),
	}

	items, total, err := h.service.ListDocuments(r.Context(), filter)
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
	item, err := h.service.GetDocument(r.Context(), r.PathValue("documentID"))
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

func (h DocumentHandler) Update(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		FolderID    string `json:"folder_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}

	actorID := middleware.UserIDFromContext(r.Context())
	actorRole := middleware.UserRoleFromContext(r.Context())
	data, err := h.service.UpdateDocument(r.Context(), command.DocumentUpdateInput{
		DocumentID:  r.PathValue("documentID"),
		Title:       body.Title,
		Description: body.Description,
		FolderID:    body.FolderID,
		ActorID:     actorID,
		ActorRole:   actorRole,
	})
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			response.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		if errors.Is(err, service.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "not_found", "document not found")
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			response.WriteError(w, http.StatusForbidden, "forbidden", "permission denied")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to update document")
		return
	}

	response.WriteData(w, http.StatusOK, data)
}

func (h DocumentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}

	actorID := middleware.UserIDFromContext(r.Context())
	actorRole := middleware.UserRoleFromContext(r.Context())
	documentID := r.PathValue("documentID")
	err := h.service.DeleteDocument(r.Context(), command.DocumentDeleteInput{
		DocumentID: documentID,
		Reason:     body.Reason,
		ActorID:    actorID,
		ActorRole:  actorRole,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "not_found", "document not found")
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			response.WriteError(w, http.StatusForbidden, "forbidden", "permission denied")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to delete document")
		return
	}

	response.WriteData(w, http.StatusOK, map[string]any{"id": documentID, "is_deleted": true})
}

func (h DocumentHandler) Restore(w http.ResponseWriter, r *http.Request) {
	actorID := middleware.UserIDFromContext(r.Context())
	actorRole := middleware.UserRoleFromContext(r.Context())
	documentID := r.PathValue("documentID")
	err := h.service.RestoreDocument(r.Context(), documentID, actorID, actorRole)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "not_found", "document not found")
			return
		}
		if errors.Is(err, service.ErrForbidden) {
			response.WriteError(w, http.StatusForbidden, "forbidden", "permission denied")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to restore document")
		return
	}

	response.WriteData(w, http.StatusOK, map[string]any{"id": documentID, "is_deleted": false})
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

const maxPageSize = 200

func parsePageSize(raw string) int {
	size := parseIntOrDefault(raw, 20)
	if size > maxPageSize {
		size = maxPageSize
	}
	return size
}
