package memory

import (
	"context"
	"fmt"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/service"
)

type ActionRepository struct{}

func NewActionRepository() ActionRepository {
	return ActionRepository{}
}

// validFlowTransitions mirrors the postgres implementation to keep dev/test parity.
var validFlowTransitions = map[string][]string{
	"mark_in_progress": {"draft", "handed_over", "in_progress"},
	"transfer":         {"in_progress"},
	"accept_transfer":  {"pending_handover"},
	"finalize":         {"in_progress", "handed_over"},
	"archive":          {"finalized", "handed_over"},
	"unarchive":        {"archived"},
}

// validHandoverTransitions mirrors the postgres implementation.
var validHandoverTransitions = map[string][]string{
	"confirm":  {"generated"},
	"complete": {"pending_confirm"},
	"cancel":   {"generated", "pending_confirm"},
}

func (r ActionRepository) CreateFlowRecord(ctx context.Context, input command.FlowActionInput) (map[string]any, error) {
	_ = ctx

	// Memory mode has no persisted state, so skip the from-status lookup.
	// Still validate that the action itself is recognised.
	if _, ok := validFlowTransitions[input.Action]; !ok {
		return nil, fmt.Errorf("%w: action=%s", service.ErrInvalidTransition, input.Action)
	}

	toStatus := memFlowActionToStatus(input.Action)
	data := map[string]any{
		"document_id":    input.DocumentID,
		"action":         input.Action,
		"current_status": toStatus,
	}
	if input.Note != "" {
		data["note"] = input.Note
	}
	if input.ToUserID != "" {
		data["to_user_id"] = input.ToUserID
	}

	return data, nil
}

func memFlowActionToStatus(action string) string {
	switch action {
	case "archive":
		return "archived"
	case "finalize":
		return "finalized"
	case "transfer":
		return "pending_handover"
	case "accept_transfer", "mark_in_progress", "unarchive":
		return "in_progress"
	default:
		return "in_progress"
	}
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

	if _, ok := validHandoverTransitions[input.Action]; !ok {
		return nil, fmt.Errorf("%w: action=%s", service.ErrInvalidTransition, input.Action)
	}

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
