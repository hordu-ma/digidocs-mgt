package memory

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/command"
)

type ActionRepository struct{}

func NewActionRepository() ActionRepository {
	return ActionRepository{}
}

func (r ActionRepository) CreateFlowRecord(ctx context.Context, input command.FlowActionInput) (map[string]any, error) {
	_ = ctx

	data := map[string]any{
		"document_id": input.DocumentID,
		"action":      input.Action,
	}
	if input.Note != "" {
		data["note"] = input.Note
	}
	if input.ToUserID != "" {
		data["to_user_id"] = input.ToUserID
	}

	return data, nil
}

func (r ActionRepository) CreateHandover(ctx context.Context, input command.HandoverCreateInput) (map[string]any, error) {
	_ = ctx

	return map[string]any{
		"id":               "00000000-0000-0000-0000-000000000300",
		"target_user_id":   input.TargetUserID,
		"receiver_user_id": input.ReceiverUserID,
		"project_id":       input.ProjectID,
		"remark":           input.Remark,
		"status":           "generated",
	}, nil
}

func (r ActionRepository) UpdateHandoverItems(
	ctx context.Context,
	input command.HandoverItemUpdateInput,
) (map[string]any, error) {
	_ = ctx

	return map[string]any{
		"id":    input.HandoverID,
		"items": input.Items,
	}, nil
}

func (r ActionRepository) ApplyHandover(ctx context.Context, input command.HandoverActionInput) (map[string]any, error) {
	_ = ctx

	data := map[string]any{
		"id":     input.HandoverID,
		"action": input.Action,
	}
	if input.Note != "" {
		data["note"] = input.Note
	}
	if input.Reason != "" {
		data["reason"] = input.Reason
	}

	return data, nil
}
