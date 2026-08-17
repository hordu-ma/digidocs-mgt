package shared

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"
)

// AllowedFileExtensions is the whitelist of permitted file extensions for upload.
var AllowedFileExtensions = map[string]bool{
	".pdf":  true,
	".doc":  true,
	".docx": true,
	".xls":  true,
	".xlsx": true,
	".ppt":  true,
	".pptx": true,
	".txt":  true,
	".md":   true,
	".csv":  true,
	".png":  true,
	".jpg":  true,
	".jpeg": true,
}

// MaxUploadSize is the maximum allowed request size for a document version
// upload (32 MB). It is enforced with http.MaxBytesReader, NOT by passing it to
// ParseMultipartForm — that argument only sizes the in-memory buffer and lets
// anything larger spill to disk, so on its own it caps nothing.
const MaxUploadSize = 32 << 20

// MaxDataAssetUploadSize is the maximum allowed request size for a data asset
// upload. Data assets legitimately carry media and model files, so the cap is
// higher than for document versions, but it is a real cap: without one, any
// project member could fill the NAS.
//
// Keep this at or below the reverse proxy's client_max_body_size
// (frontend/nginx/default.conf) so oversized uploads are rejected consistently
// whichever entry point they arrive through.
const MaxDataAssetUploadSize = 512 << 20 // 512 MB

// MaxMultipartMemory bounds how much of a multipart body is buffered in memory
// before ParseMultipartForm spools the remainder to a temp file.
const MaxMultipartMemory = 32 << 20 // 32 MB

// ValidateFileName checks that a filename has an allowed extension for document uploads.
// Returns true if allowed, false otherwise.
func ValidateFileName(fileName string) bool {
	ext := strings.ToLower(filepath.Ext(fileName))
	return AllowedFileExtensions[ext]
}

// ValidateDataAssetFileName returns true for any non-empty filename.
// Data assets accept all file types; naming convention is enforced on the frontend only.
func ValidateDataAssetFileName(fileName string) bool {
	return strings.TrimSpace(fileName) != ""
}

// SanitizeStoredFileName strips any directory components and path traversal
// sequences from a client-supplied filename before it is used to build a
// storage object key. Callers must never interpolate the raw upload filename
// into an object key: multipart headers are fully attacker-controlled and a
// name like "../../../etc/passwd" would otherwise escape the intended
// storage prefix (documents/<id>/..., data/<project>/...).
func SanitizeStoredFileName(fileName string) string {
	name := strings.ReplaceAll(fileName, "\\", "/")
	name = filepath.Base(name)
	if name == "" || name == "." || name == ".." || name == string(filepath.Separator) {
		return "file"
	}
	return name
}

// LimitUploadBody caps a multipart request body at maxBytes. It must be called
// before ParseMultipartForm; once the limit is exceeded the parse fails with an
// *http.MaxBytesError instead of the oversized body being spooled to disk.
func LimitUploadBody(w http.ResponseWriter, r *http.Request, maxBytes int64) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
}

// IsBodyTooLarge reports whether err came from exceeding a MaxBytesReader limit,
// so handlers can answer 413 rather than a generic 400.
func IsBodyTooLarge(err error) bool {
	var maxErr *http.MaxBytesError
	return errors.As(err, &maxErr)
}
