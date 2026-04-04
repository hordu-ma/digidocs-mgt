package service

import (
	"context"
	"io"

	"digidocs-mgt/backend-go/internal/storage"
)

type UploadService struct {
	storage storage.Provider
}

func NewUploadService(storage storage.Provider) UploadService {
	return UploadService{storage: storage}
}

func (s UploadService) Upload(
	ctx context.Context,
	objectKey string,
	reader io.Reader,
) (storage.PutObjectResult, error) {
	return s.storage.PutObject(ctx, storage.PutObjectInput{
		ObjectKey: objectKey,
		Reader:    reader,
	})
}
