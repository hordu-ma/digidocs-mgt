package memory

import (
	"context"
	"io"

	"digidocs-mgt/backend-go/internal/storage"
)

type Provider struct{}

func NewProvider() Provider {
	return Provider{}
}

func (p Provider) PutObject(ctx context.Context, input storage.PutObjectInput) (storage.PutObjectResult, error) {
	_ = ctx
	if input.Reader != nil {
		_, _ = io.Copy(io.Discard, input.Reader)
	}

	return storage.PutObjectResult{
		ObjectKey: input.ObjectKey,
		Provider:  "memory",
	}, nil
}
