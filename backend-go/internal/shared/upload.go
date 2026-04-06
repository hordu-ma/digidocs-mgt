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

// MaxUploadSize is the maximum allowed upload size (32 MB).
const MaxUploadSize = 32 << 20

// ValidateFileName checks that a filename has an allowed extension.
// Returns true if allowed, false otherwise.
func ValidateFileName(fileName string) bool {
	ext := strings.ToLower(filepath.Ext(fileName))
	return AllowedFileExtensions[ext]
}
