package queue

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/task"
)

type Publisher interface {
	Publish(ctx context.Context, message task.Message) error
}
