package service

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/repository"
)

type VersionWorkflowService struct {
	workflow repository.VersionWorkflow
}

func NewVersionWorkflowService(workflow repository.VersionWorkflow) VersionWorkflowService {
	return VersionWorkflowService{workflow: workflow}
}

func (s VersionWorkflowService) CreateUploadedVersion(
	ctx context.Context,
	input command.VersionCreateInput,
) (map[string]any, error) {
	return s.workflow.CreateUploadedVersion(ctx, input)
}
