package service

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/domain/task"
	"digidocs-mgt/backend-go/internal/queue"
	"digidocs-mgt/backend-go/internal/repository"
)

type AssistantService struct {
	publisher queue.Publisher
	repo      repository.AssistantRepository
}

func NewAssistantService(
	publisher queue.Publisher,
	repo repository.AssistantRepository,
) AssistantService {
	return AssistantService{
		publisher: publisher,
		repo:      repo,
	}
}

func (s AssistantService) QueueTask(
	ctx context.Context,
	taskType task.TaskType,
	relatedType string,
	relatedID string,
	payload map[string]any,
	actorID string,
) (task.Message, error) {
	message := task.Message{
		RequestID:   newRequestID(),
		TaskType:    taskType,
		RelatedType: relatedType,
		RelatedID:   relatedID,
		Payload:     payload,
	}
	if message.Payload == nil {
		message.Payload = map[string]any{}
	}

	if err := s.repo.CreateAssistantRequest(ctx, message, actorID); err != nil {
		return task.Message{}, err
	}
	if err := s.publisher.Publish(ctx, message); err != nil {
		return task.Message{}, err
	}

	return message, nil
}

func (s AssistantService) ReceiveResult(ctx context.Context, result task.Result) error {
	return s.repo.CompleteAssistantRequest(ctx, result)
}

func (s AssistantService) GetRequest(
	ctx context.Context,
	requestID string,
) (*query.AssistantRequestItem, error) {
	return s.repo.GetAssistantRequest(ctx, requestID)
}

func (s AssistantService) GetLatestDocumentExtractedText(
	ctx context.Context,
	documentID string,
) (string, error) {
	return s.repo.GetLatestDocumentExtractedText(ctx, documentID)
}

func (s AssistantService) ListSuggestions(
	ctx context.Context,
	filter query.AssistantSuggestionFilter,
) ([]query.AssistantSuggestionItem, error) {
	return s.repo.ListSuggestions(ctx, filter)
}

func (s AssistantService) ConfirmSuggestion(
	ctx context.Context,
	suggestionID string,
	actorID string,
	note string,
) (map[string]any, error) {
	return s.repo.ConfirmSuggestion(ctx, suggestionID, actorID, note)
}

func (s AssistantService) DismissSuggestion(
	ctx context.Context,
	suggestionID string,
	actorID string,
	reason string,
) (map[string]any, error) {
	return s.repo.DismissSuggestion(ctx, suggestionID, actorID, reason)
}
