package service

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/repository"
)

type ActionService struct {
	writer repository.ActionWriter
}

func NewActionService(writer repository.ActionWriter) ActionService {
	return ActionService{writer: writer}
}

func (s ActionService) ApplyFlow(ctx context.Context, input command.FlowActionInput) (map[string]any, error) {
	return s.writer.CreateFlowRecord(ctx, input)
}

func (s ActionService) CreateHandover(ctx context.Context, input command.HandoverCreateInput) (map[string]any, error) {
	return s.writer.CreateHandover(ctx, input)
}

func (s ActionService) UpdateHandoverItems(
	ctx context.Context,
	input command.HandoverItemUpdateInput,
) (map[string]any, error) {
	return s.writer.UpdateHandoverItems(ctx, input)
}

func (s ActionService) ApplyHandover(ctx context.Context, input command.HandoverActionInput) (map[string]any, error) {
	return s.writer.ApplyHandover(ctx, input)
}
