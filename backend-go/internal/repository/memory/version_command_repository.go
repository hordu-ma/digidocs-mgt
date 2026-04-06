package memory

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/command"
)

func (r *VersionRepository) CreateVersion(ctx context.Context, input command.VersionCreateInput) (map[string]any, error) {
	return r.createUploadedVersion(ctx, input)
}
