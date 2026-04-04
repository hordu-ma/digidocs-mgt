package service

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/repository"
)

type VersionQueryService struct {
	versions repository.VersionReader
}

func NewVersionQueryService(versions repository.VersionReader) VersionQueryService {
	return VersionQueryService{versions: versions}
}

func (s VersionQueryService) List(ctx context.Context, documentID string) ([]query.VersionItem, error) {
	return s.versions.ListVersions(ctx, documentID)
}

func (s VersionQueryService) Get(ctx context.Context, versionID string) (*query.VersionDetail, error) {
	return s.versions.GetVersion(ctx, versionID)
}
