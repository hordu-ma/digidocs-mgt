package service

import (
	"context"
	"fmt"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/repository"
)

var validHandoverActions = map[string]bool{
	"confirm":  true,
	"complete": true,
	"cancel":   true,
}

type HandoverService struct {
	reader repository.HandoverReader
	writer repository.ActionWriter
}

func NewHandoverService(reader repository.HandoverReader, writer repository.ActionWriter) HandoverService {
	return HandoverService{reader: reader, writer: writer}
}

func (s HandoverService) Create(ctx context.Context, input command.HandoverCreateInput) (map[string]any, error) {
	if input.TargetUserID == "" {
		return nil, fmt.Errorf("%w: target_user_id is required", ErrValidation)
	}
	if input.ReceiverUserID == "" {
		return nil, fmt.Errorf("%w: receiver_user_id is required", ErrValidation)
	}
	if input.ActorID == "" {
		return nil, fmt.Errorf("%w: actor_id is required", ErrValidation)
	}
	return s.writer.CreateHandover(ctx, input)
}

func (s HandoverService) UpdateItems(ctx context.Context, input command.HandoverItemUpdateInput) (map[string]any, error) {
	if input.HandoverID == "" {
		return nil, fmt.Errorf("%w: handover_id is required", ErrValidation)
	}
	if len(input.Items) == 0 {
		return nil, fmt.Errorf("%w: items must not be empty", ErrValidation)
	}
	return s.writer.UpdateHandoverItems(ctx, input)
}

func (s HandoverService) ApplyAction(ctx context.Context, input command.HandoverActionInput) (map[string]any, error) {
	if input.HandoverID == "" {
		return nil, fmt.Errorf("%w: handover_id is required", ErrValidation)
	}
	if !validHandoverActions[input.Action] {
		return nil, fmt.Errorf("%w: unknown action %q", ErrValidation, input.Action)
	}
	if input.ActorID == "" {
		return nil, fmt.Errorf("%w: actor_id is required", ErrValidation)
	}
	return s.writer.ApplyHandover(ctx, input)
}

func (s HandoverService) List(ctx context.Context) ([]query.HandoverItem, error) {
	return s.reader.ListHandovers(ctx)
}

func (s HandoverService) Get(ctx context.Context, handoverID string) (*query.HandoverItem, error) {
	if handoverID == "" {
		return nil, fmt.Errorf("%w: handover_id is required", ErrValidation)
	}
	return s.reader.GetHandover(ctx, handoverID)
}
