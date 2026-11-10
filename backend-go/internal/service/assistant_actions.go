package service

import (
	"context"
	"fmt"
	"strings"

	"digidocs-mgt/backend-go/internal/domain/command"
)

// suggestedFlowActions is the whitelist of flow actions an AI suggestion may
// propose. transfer needs a target the model cannot be trusted to pick, and
// unarchive / accept_transfer are decisions about people, not documents.
var suggestedFlowActions = map[string]bool{
	"mark_in_progress": true,
	"finalize":         true,
	"archive":          true,
}

// NormalizeSuggestedAction returns the action if whitelisted, else "".
func NormalizeSuggestedAction(action string) string {
	action = strings.TrimSpace(strings.ToLower(action))
	if suggestedFlowActions[action] {
		return action
	}
	return ""
}

// FlowActionExecutor is implemented by FlowService; kept as an interface so the
// assistant service does not need the whole flow service to be testable.
type FlowActionExecutor interface {
	ApplyAction(ctx context.Context, input command.FlowActionInput) (map[string]any, error)
}

// WithFlowExecutor enables "confirm and execute" on suggestions.
func (s AssistantService) WithFlowExecutor(executor FlowActionExecutor) AssistantService {
	s.flows = executor
	return s
}

// ConfirmAndExecuteSuggestion runs the suggestion's flow action *as the
// confirming user* and only then marks it confirmed. The flow service applies
// the normal permission and state-machine checks, so an AI suggestion can
// never do anything the user could not do by clicking the button themselves.
// If the action fails, the suggestion stays pending.
func (s AssistantService) ConfirmAndExecuteSuggestion(
	ctx context.Context,
	suggestionID string,
	actorID string,
	actorRole string,
	note string,
) (map[string]any, error) {
	if err := s.permissions.EnsureViewAssistantSuggestion(ctx, actorID, actorRole, suggestionID); err != nil {
		return nil, err
	}
	if s.flows == nil {
		return nil, fmt.Errorf("%w: suggestion execution is not available", ErrValidation)
	}
	suggestion, err := s.repo.GetSuggestion(ctx, suggestionID)
	if err != nil {
		return nil, err
	}
	if suggestion.Status != "pending" {
		return nil, fmt.Errorf("%w: suggestion is already %s", ErrValidation, suggestion.Status)
	}
	action := NormalizeSuggestedAction(suggestion.SuggestedAction)
	if action == "" || suggestion.RelatedType != "document" {
		return nil, fmt.Errorf("%w: suggestion has no executable action", ErrValidation)
	}
	flowNote := "确认 AI 建议：" + suggestion.Title
	if strings.TrimSpace(note) != "" {
		flowNote += "；" + strings.TrimSpace(note)
	}
	if _, err := s.flows.ApplyAction(ctx, command.FlowActionInput{
		DocumentID: suggestion.RelatedID,
		Action:     action,
		Note:       flowNote,
		ActorID:    actorID,
		ActorRole:  actorRole,
	}); err != nil {
		return nil, err
	}
	result, err := s.repo.ConfirmSuggestion(ctx, suggestionID, actorID, note)
	if err != nil {
		return nil, err
	}
	result["executed_action"] = action
	return result, nil
}
