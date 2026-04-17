package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/shared"
	"digidocs-mgt/backend-go/internal/transport/http/middleware"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type DataAssetHandler struct {
	service service.DataAssetService
}

func NewDataAssetHandler(svc service.DataAssetService) DataAssetHandler {
	return DataAssetHandler{service: svc}
}

// ─────────────────────────── folders ────────────────────────────

// ListFolders  GET /api/v1/projects/{id}/data-folders
func (h DataAssetHandler) ListFolders(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("id")
	items, err := h.service.ListDataFolders(r.Context(), projectID)
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			response.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to list folders")
		return
	}
	response.WriteData(w, http.StatusOK, items)
}

// CreateFolder  POST /api/v1/data-folders
func (h DataAssetHandler) CreateFolder(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProjectID string `json:"project_id"`
		ParentID  string `json:"parent_id"`
		Name      string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid JSON")
		return
	}

	actorID := middleware.UserIDFromContext(r.Context())
	actorRole := middleware.UserRoleFromContext(r.Context())

	folder, err := h.service.CreateDataFolder(r.Context(), command.DataFolderCreateInput{
		ProjectID: body.ProjectID,
		ParentID:  body.ParentID,
		Name:      body.Name,
		ActorID:   actorID,
		ActorRole: actorRole,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			response.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		case errors.Is(err, service.ErrForbidden):
			response.WriteError(w, http.StatusForbidden, "forbidden", "permission denied")
		case errors.Is(err, service.ErrConflict):
			response.WriteError(w, http.StatusConflict, "conflict", err.Error())
		default:
			response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to create folder")
		}
		return
	}
	response.WriteData(w, http.StatusCreated, folder)
}

// DeleteFolder  DELETE /api/v1/data-folders/{id}
func (h DataAssetHandler) DeleteFolder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	actorID := middleware.UserIDFromContext(r.Context())
	actorRole := middleware.UserRoleFromContext(r.Context())

	if err := h.service.DeleteDataFolder(r.Context(), id, actorID, actorRole); err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			response.WriteError(w, http.StatusNotFound, "not_found", "folder not found")
		case errors.Is(err, service.ErrValidation):
			response.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		case errors.Is(err, service.ErrForbidden):
			response.WriteError(w, http.StatusForbidden, "forbidden", "permission denied")
		default:
			response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to delete folder")
		}
		return
	}
	response.WriteData(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ─────────────────────────── assets ─────────────────────────────

// List  GET /api/v1/data-assets
func (h DataAssetHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	items, total, err := h.service.ListDataAssets(r.Context(), query.DataAssetListFilter{
		ProjectID: r.URL.Query().Get("project_id"),
		FolderID:  r.URL.Query().Get("folder_id"),
		Keyword:   r.URL.Query().Get("keyword"),
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to list data assets")
		return
	}
	response.WriteData(w, http.StatusOK, map[string]any{
		"items": items,
		"total": total,
		"page":  page,
	})
}

// Get  GET /api/v1/data-assets/{id}
func (h DataAssetHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	asset, err := h.service.GetDataAsset(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "not_found", "data asset not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to get data asset")
		return
	}
	response.WriteData(w, http.StatusOK, asset)
}

// Upload  POST /api/v1/data-assets  (multipart/form-data)
func (h DataAssetHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(shared.MaxDataAssetMemoryBuffer); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "file is required")
		return
	}
	defer file.Close()

	if !shared.ValidateDataAssetFileName(header.Filename) {
		response.WriteError(w, http.StatusBadRequest, "validation_error", "file name is empty")
		return
	}

	actorID := middleware.UserIDFromContext(r.Context())
	actorRole := middleware.UserRoleFromContext(r.Context())

	result, err := h.service.UploadDataAsset(
		r.Context(),
		command.DataAssetCreateInput{
			TeamSpaceID: r.FormValue("team_space_id"),
			ProjectID:   r.FormValue("project_id"),
			FolderID:    r.FormValue("folder_id"),
			DisplayName: r.FormValue("display_name"),
			Description: r.FormValue("description"),
			FileSize:    header.Size,
			ActorID:     actorID,
			ActorRole:   actorRole,
		},
		file,
		header.Filename,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			response.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		case errors.Is(err, service.ErrForbidden):
			response.WriteError(w, http.StatusForbidden, "forbidden", "permission denied")
		default:
			response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to upload data asset")
		}
		return
	}
	response.WriteData(w, http.StatusCreated, result)
}

// Update  PUT /api/v1/data-assets/{id}
func (h DataAssetHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		DisplayName string `json:"display_name"`
		Description string `json:"description"`
		FolderID    string `json:"folder_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid JSON")
		return
	}

	actorID := middleware.UserIDFromContext(r.Context())
	actorRole := middleware.UserRoleFromContext(r.Context())

	if err := h.service.UpdateDataAsset(r.Context(), command.DataAssetUpdateInput{
		DataAssetID: id,
		DisplayName: body.DisplayName,
		Description: body.Description,
		FolderID:    body.FolderID,
		ActorID:     actorID,
		ActorRole:   actorRole,
	}); err != nil {
		switch {
		case errors.Is(err, service.ErrValidation):
			response.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
		case errors.Is(err, service.ErrForbidden):
			response.WriteError(w, http.StatusForbidden, "forbidden", "permission denied")
		case errors.Is(err, service.ErrNotFound):
			response.WriteError(w, http.StatusNotFound, "not_found", "data asset not found")
		default:
			response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to update data asset")
		}
		return
	}
	response.WriteData(w, http.StatusOK, map[string]string{"status": "updated"})
}

// Delete  DELETE /api/v1/data-assets/{id}
func (h DataAssetHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	actorID := middleware.UserIDFromContext(r.Context())
	actorRole := middleware.UserRoleFromContext(r.Context())

	if err := h.service.DeleteDataAsset(r.Context(), command.DataAssetDeleteInput{
		DataAssetID: id,
		ActorID:     actorID,
		ActorRole:   actorRole,
	}); err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			response.WriteError(w, http.StatusNotFound, "not_found", "data asset not found")
		case errors.Is(err, service.ErrForbidden):
			response.WriteError(w, http.StatusForbidden, "forbidden", "permission denied")
		default:
			response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to delete data asset")
		}
		return
	}
	response.WriteData(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Download  GET /api/v1/data-assets/{id}/download
func (h DataAssetHandler) Download(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	out, asset, err := h.service.DownloadDataAsset(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "not_found", "data asset not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to download data asset")
		return
	}
	defer out.Reader.Close()

	contentType := asset.MimeType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+asset.FileName+"\"")
	if out.Size > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(out.Size, 10))
	}
	w.WriteHeader(http.StatusOK)
	//nolint:errcheck
	io.Copy(w, out.Reader)
}

// ─────────────────────── handover data items ─────────────────────

// ListHandoverDataItems  GET /api/v1/handovers/{id}/data-items
func (h DataAssetHandler) ListHandoverDataItems(w http.ResponseWriter, r *http.Request) {
	handoverID := r.PathValue("id")
	items, err := h.service.ListHandoverDataItems(r.Context(), handoverID)
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			response.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to list handover data items")
		return
	}
	response.WriteData(w, http.StatusOK, items)
}

// UpdateHandoverDataItems  PUT /api/v1/handovers/{id}/data-items
func (h DataAssetHandler) UpdateHandoverDataItems(w http.ResponseWriter, r *http.Request) {
	handoverID := r.PathValue("id")
	var body struct {
		Items []command.HandoverDataItemInput `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid JSON")
		return
	}

	result, err := h.service.UpdateHandoverDataItems(r.Context(), command.HandoverDataItemUpdateInput{
		HandoverID: handoverID,
		Items:      body.Items,
	})
	if err != nil {
		if errors.Is(err, service.ErrValidation) {
			response.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to update handover data items")
		return
	}
	response.WriteData(w, http.StatusOK, result)
}
