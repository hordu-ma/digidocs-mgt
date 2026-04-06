package storage

import (
	"context"
	"io"
	"time"
)

type PutObjectInput struct {
	ObjectKey   string
	Reader      io.Reader
	Overwrite   bool // true = overwrite existing file; maps to Synology create_parents + overwrite
	CreatePaths bool // true = auto-create parent directories
}

type PutObjectResult struct {
	ObjectKey string `json:"object_key"`
	Provider  string `json:"provider"`
}

type GetObjectOutput struct {
	Reader      io.ReadCloser
	ContentType string
	Size        int64
}

// FileInfo describes a file or directory entry.
type FileInfo struct {
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	IsDir      bool      `json:"is_dir"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modified_at"`
}

// ShareLinkResult contains information about a created share link.
type ShareLinkResult struct {
	URL       string     `json:"url"`
	ID        string     `json:"id"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// Provider defines the storage abstraction.
// Implementations map to actual storage backends (memory, Synology File Station, etc.).
type Provider interface {
	// PutObject uploads a file.
	PutObject(ctx context.Context, input PutObjectInput) (PutObjectResult, error)
	// GetObject downloads a file by key.
	GetObject(ctx context.Context, objectKey string) (*GetObjectOutput, error)
	// DeleteObject removes a file or directory.
	DeleteObject(ctx context.Context, objectKey string) error
	// Stat returns metadata for a file or directory without downloading content.
	Stat(ctx context.Context, objectKey string) (*FileInfo, error)
	// ListDir lists entries under a folder path.
	ListDir(ctx context.Context, folderPath string) ([]FileInfo, error)
	// CreateFolder creates a directory (including parents if needed).
	CreateFolder(ctx context.Context, folderPath string) error
	// CreateShareLink creates a time-limited share link for a file.
	CreateShareLink(ctx context.Context, objectKey string, expireDays int) (*ShareLinkResult, error)
}
