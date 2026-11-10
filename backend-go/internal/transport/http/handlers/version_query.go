package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

func (h VersionHandler) List(w http.ResponseWriter, r *http.Request) {
	actorID, actorRole := actorFrom(r)
	items, err := h.service.List(r.Context(), r.PathValue("documentID"), actorID, actorRole)
	if err != nil {
		writeServiceError(w, err, "document not found", "failed to list versions")
		return
	}

	response.WriteWithMeta(w, http.StatusOK, items, map[string]string{
		"document_id": r.PathValue("documentID"),
	})
}

func (h VersionHandler) Get(w http.ResponseWriter, r *http.Request) {
	actorID, actorRole := actorFrom(r)
	item, err := h.service.Get(r.Context(), r.PathValue("versionID"), actorID, actorRole)
	if err != nil {
		writeServiceError(w, err, "version not found", "failed to get version")
		return
	}

	response.WriteData(w, http.StatusOK, item)
}

func (h VersionHandler) Download(w http.ResponseWriter, r *http.Request) {
	actorID, actorRole := actorFrom(r)
	ver, obj, err := h.service.GetFile(r.Context(), r.PathValue("versionID"), actorID, actorRole)
	if err != nil {
		writeServiceError(w, err, "version not found", "failed to retrieve file")
		return
	}
	defer obj.Reader.Close()

	if err := h.audit.Record(r.Context(), accessEvent(r, service.AuditActionDownload, ver.DocumentID, ver.ID, map[string]any{
		"file_name": ver.FileName, "version_no": ver.VersionNo,
	})); err != nil {
		log.Printf("[audit] download not recorded, refusing to serve version=%s: %v", ver.ID, err)
		response.WriteError(w, http.StatusInternalServerError, "audit_failed", "下载记录写入失败，请稍后重试")
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, ver.FileName))
	w.Header().Set("Content-Type", contentTypeFromName(ver.FileName, obj.ContentType))
	if obj.Size > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", obj.Size))
	}
	w.WriteHeader(http.StatusOK)
	io.Copy(w, obj.Reader)
}

func (h VersionHandler) Preview(w http.ResponseWriter, r *http.Request) {
	actorID, actorRole := actorFrom(r)
	ver, obj, err := h.service.GetFile(r.Context(), r.PathValue("versionID"), actorID, actorRole)
	if err != nil {
		writeServiceError(w, err, "version not found", "failed to retrieve file")
		return
	}
	defer obj.Reader.Close()

	if err := h.audit.Record(r.Context(), accessEvent(r, service.AuditActionPreview, ver.DocumentID, ver.ID, map[string]any{
		"file_name": ver.FileName, "version_no": ver.VersionNo,
	})); err != nil {
		log.Printf("[audit] preview not recorded, refusing to serve version=%s: %v", ver.ID, err)
		response.WriteError(w, http.StatusInternalServerError, "audit_failed", "预览记录写入失败，请稍后重试")
		return
	}

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
