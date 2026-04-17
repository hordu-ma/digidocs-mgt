package service

import (
	"context"
	"fmt"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/repository"
)

var validFlowActions = map[string]bool{
	"mark_in_progress": true,
	"transfer":         true,
	"accept_transfer":  true,
	"finalize":         true,
	"archive":          true,
	"unarchive":        true,
}

type FlowService struct {
	reader      repository.FlowReader
	writer      repository.ActionWriter
	permissions PermissionService
}

func NewFlowService(reader repository.FlowReader, writer repository.ActionWriter, permissions ...PermissionService) FlowService {
	permissionService := PermissionService{}
	if len(permissions) > 0 {
		permissionService = permissions[0]
	}
	return FlowService{reader: reader, writer: writer, permissions: permissionService}
}

func (s FlowService) ApplyAction(ctx context.Context, input command.FlowActionInput) (map[string]any, error) {
	if input.DocumentID == "" {
		return nil, fmt.Errorf("%w: document_id is required", ErrValidation)
	}
	if !validFlowActions[input.Action] {
		return nil, fmt.Errorf("%w: unknown action %q", ErrValidation, input.Action)
	}
	if input.Action == "transfer" && input.ToUserID == "" {
		return nil, fmt.Errorf("%w: to_user_id is required for transfer", ErrValidation)
	}
	if input.Action == "transfer" && input.ToUserID == input.ActorID {
		return nil, fmt.Errorf("%w: cannot transfer to yourself", ErrValidation)
	}
	if input.ActorID == "" {
		return nil, fmt.Errorf("%w: actor_id is required", ErrValidation)
	}
	if err := s.permissions.EnsureFlowDocument(ctx, input.ActorID, input.ActorRole, input.DocumentID, input.Action); err != nil {
		return nil, err
	}
	return s.writer.CreateFlowRecord(ctx, input)
}

func (s FlowService) ListFlows(ctx context.Context, documentID string) ([]query.FlowItem, error) {
	return s.reader.ListFlows(ctx, documentID)
}
