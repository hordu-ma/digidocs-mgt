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

type Provider interface {
	PutObject(ctx context.Context, input PutObjectInput) (PutObjectResult, error)
}
