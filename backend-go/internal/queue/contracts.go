package queue

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/task"
)

type Publisher interface {
	Publish(ctx context.Context, message task.Message) error
}

// Consumer allows draining pending task messages (used by internal HTTP handler for worker polling).
type Consumer interface {
	Poll(ctx context.Context, limit int) []task.Message
}
