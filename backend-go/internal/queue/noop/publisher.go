package noop

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/task"
)

type Publisher struct{}

func NewPublisher() Publisher {
	return Publisher{}
}

func (Publisher) Publish(_ context.Context, _ task.Message) error {
	return nil
}
