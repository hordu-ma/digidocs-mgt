package memory

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	"digidocs-mgt/backend-go/internal/storage"
)

type Provider struct {
	mu    sync.RWMutex
	store map[string][]byte
}

func NewProvider() *Provider {
	return &Provider{store: make(map[string][]byte)}
}

func (p *Provider) PutObject(ctx context.Context, input storage.PutObjectInput) (storage.PutObjectResult, error) {
	_ = ctx
	var data []byte
	if input.Reader != nil {
		var err error
		data, err = io.ReadAll(input.Reader)
		if err != nil {
			return storage.PutObjectResult{}, err
		}
	}

	p.mu.Lock()
	p.store[input.ObjectKey] = data
	p.mu.Unlock()

	return storage.PutObjectResult{
		ObjectKey: input.ObjectKey,
		Provider:  "memory",
	}, nil
}

func (p *Provider) GetObject(_ context.Context, objectKey string) (*storage.GetObjectOutput, error) {
	p.mu.RLock()
	data, ok := p.store[objectKey]
	p.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("object not found: %s", objectKey)
	}

	return &storage.GetObjectOutput{
		Reader:      io.NopCloser(bytes.NewReader(data)),
		ContentType: "application/octet-stream",
		Size:        int64(len(data)),
	}, nil
}
