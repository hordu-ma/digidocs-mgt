package memory

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/command"
)

type VersionWorkflow struct{}

func NewVersionWorkflow() VersionWorkflow {
	return VersionWorkflow{}
}

func (w VersionWorkflow) CreateUploadedVersion(ctx context.Context, input command.VersionCreateInput) (map[string]any, error) {
	_ = ctx

	return map[string]any{
		"id":             "00000000-0000-0000-0000-000000000200",
		"document_id":    input.DocumentID,
		"version_no":     1,
		"commit_message": input.CommitMessage,
		"file_name":      input.FileName,
		"current_status": "in_progress",
	}, nil
}
