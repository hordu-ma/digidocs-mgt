package memory

import (
	"context"
	"log"

	"digidocs-mgt/backend-go/internal/domain/task"
)

type Publisher struct{}

func NewPublisher() Publisher {
	return Publisher{}
}

func (p Publisher) Publish(ctx context.Context, message task.Message) error {
	_ = ctx

	log.Printf("memory-queue publish task=%s request_id=%s", message.TaskType, message.RequestID)
	return nil
}
