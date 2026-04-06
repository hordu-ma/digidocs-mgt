package memory

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/command"
)

type VersionWorkflow struct {
	repo *VersionRepository
}

func NewVersionWorkflow(repo *VersionRepository) VersionWorkflow {
	return VersionWorkflow{repo: repo}
}

func (w VersionWorkflow) CreateUploadedVersion(ctx context.Context, input command.VersionCreateInput) (map[string]any, error) {
	return w.repo.createUploadedVersion(ctx, input)
}
