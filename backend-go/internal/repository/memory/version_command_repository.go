package memory

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/command"
)

func (r VersionRepository) CreateVersion(ctx context.Context, input command.VersionCreateInput) (map[string]any, error) {
	_ = ctx

	return map[string]any{
		"id":             "00000000-0000-0000-0000-000000000200",
		"document_id":    input.DocumentID,
		"version_no":     1,
		"commit_message": input.CommitMessage,
		"file_name":      input.FileName,
	}, nil
}
