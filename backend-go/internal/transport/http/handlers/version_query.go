package handlers

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"digidocs-mgt/backend-go/internal/transport/http/response"
)

func (h VersionHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.List(r.Context(), r.PathValue("documentID"))
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to list versions")
		return
	}

	response.WriteWithMeta(w, http.StatusOK, items, map[string]string{
		"document_id": r.PathValue("documentID"),
	})
}

func (h VersionHandler) Get(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Get(r.Context(), r.PathValue("versionID"))
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to get version")
		return
	}

	response.WriteData(w, http.StatusOK, item)
}

func (h VersionHandler) Download(w http.ResponseWriter, r *http.Request) {
	ver, obj, err := h.service.GetFile(r.Context(), r.PathValue("versionID"))
	if err != nil {
		if ver == nil {
			response.WriteError(w, http.StatusNotFound, "not_found", "version not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to retrieve file")
		return
	}
	defer obj.Reader.Close()

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, ver.FileName))
	w.Header().Set("Content-Type", contentTypeFromName(ver.FileName, obj.ContentType))
	if obj.Size > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", obj.Size))
	}
	w.WriteHeader(http.StatusOK)
	io.Copy(w, obj.Reader)
}

func (h VersionHandler) Preview(w http.ResponseWriter, r *http.Request) {
	ver, obj, err := h.service.GetFile(r.Context(), r.PathValue("versionID"))
	if err != nil {
		if ver == nil {
			response.WriteError(w, http.StatusNotFound, "not_found", "version not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to retrieve file")
		return
	}
	defer obj.Reader.Close()

	ct := contentTypeFromName(ver.FileName, obj.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, ver.FileName))
	w.Header().Set("Content-Type", ct)
	if obj.Size > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", obj.Size))
	}
	w.WriteHeader(http.StatusOK)
	io.Copy(w, obj.Reader)
}

// contentTypeFromName infers MIME type from file extension, falling back to provided default.
func contentTypeFromName(fileName, fallback string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".doc":
		return "application/msword"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".ppt":
		return "application/vnd.ms-powerpoint"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".txt":
		return "text/plain; charset=utf-8"
	default:
		if fallback != "" {
			return fallback
		}
		return "application/octet-stream"
	}
}
