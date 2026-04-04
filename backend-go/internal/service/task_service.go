package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"digidocs-mgt/backend-go/internal/domain/task"
	"digidocs-mgt/backend-go/internal/queue"
)

type TaskService struct {
	publisher queue.Publisher
}

func NewTaskService(publisher queue.Publisher) TaskService {
	return TaskService{publisher: publisher}
}

func (s TaskService) Publish(
	ctx context.Context,
	taskType task.TaskType,
	relatedType string,
	relatedID string,
	payload map[string]any,
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

	if err := s.publisher.Publish(ctx, message); err != nil {
		return task.Message{}, err
	}

	return message, nil
}

func newRequestID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "request-id-unavailable"
	}

	return hex.EncodeToString(buf)
}
