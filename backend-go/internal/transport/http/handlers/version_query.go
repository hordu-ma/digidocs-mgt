package handlers

import (
	"net/http"

	"digidocs-mgt/backend-go/internal/transport/http/response"
)

func (h VersionHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.queryService.List(r.Context(), r.PathValue("documentID"))
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to list versions")
		return
	}

	response.WriteWithMeta(w, http.StatusOK, items, map[string]string{
		"document_id": r.PathValue("documentID"),
	})
}

func (h VersionHandler) Get(w http.ResponseWriter, r *http.Request) {
	item, err := h.queryService.Get(r.Context(), r.PathValue("versionID"))
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to get version")
		return
	}

	response.WriteData(w, http.StatusOK, item)
}

func (h VersionHandler) Download(w http.ResponseWriter, r *http.Request) {
	item, err := h.queryService.Get(r.Context(), r.PathValue("versionID"))
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to get version download")
		return
	}

	item.Download = "not-implemented"
	response.WriteData(w, http.StatusOK, item)
}

func (h VersionHandler) Preview(w http.ResponseWriter, r *http.Request) {
	item, err := h.queryService.Get(r.Context(), r.PathValue("versionID"))
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to get version preview")
		return
	}

	item.PreviewType = "pdf"
	item.WatermarkEnabled = true
	response.WriteData(w, http.StatusOK, item)
}
