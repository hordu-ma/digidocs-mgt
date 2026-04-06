package storage

import (
	"context"
	"io"
)

type PutObjectInput struct {
	ObjectKey string
	Reader    io.Reader
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

type Provider interface {
	PutObject(ctx context.Context, input PutObjectInput) (PutObjectResult, error)
	GetObject(ctx context.Context, objectKey string) (*GetObjectOutput, error)
}
