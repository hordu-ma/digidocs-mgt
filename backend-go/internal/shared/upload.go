package shared

import (
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

// MaxUploadSize is the maximum allowed upload size for document versions (32 MB).
const MaxUploadSize = 32 << 20

// MaxDataAssetMemoryBuffer is the maximum in-memory buffer when parsing a data asset upload.
// Files larger than this are automatically spooled to OS temp files; there is no total size cap.
const MaxDataAssetMemoryBuffer = 64 << 20 // 64 MB

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
