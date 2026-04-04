package service

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/repository"
)

type VersionCommandService struct {
	writer repository.VersionWriter
}

func NewVersionCommandService(writer repository.VersionWriter) VersionCommandService {
	return VersionCommandService{writer: writer}
}

func (s VersionCommandService) Create(ctx context.Context, input command.VersionCreateInput) (map[string]any, error) {
	return s.writer.CreateVersion(ctx, input)
}
