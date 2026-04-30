package noop

import (
	"context"
	"testing"

	"digidocs-mgt/backend-go/internal/domain/task"
)

func TestPublisherPublishIsNoop(t *testing.T) {
	publisher := NewPublisher()

	if err := publisher.Publish(context.Background(), task.Message{RequestID: "req-1"}); err != nil {
		t.Fatalf("unexpected publish error: %v", err)
	}
}
