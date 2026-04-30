package memory

import (
	"context"
	"testing"

	"digidocs-mgt/backend-go/internal/domain/task"
)

func TestPublisherPollDrainsUpToLimit(t *testing.T) {
	ctx := context.Background()
	publisher := NewPublisher()

	messages := []task.Message{
		{RequestID: "req-1", TaskType: task.TaskTypeAssistantAsk},
		{RequestID: "req-2", TaskType: task.TaskTypeDocumentSummarize},
		{RequestID: "req-3", TaskType: task.TaskTypeHandoverSummarize},
	}
	for _, message := range messages {
		if err := publisher.Publish(ctx, message); err != nil {
			t.Fatalf("publish failed: %v", err)
		}
	}

	first := publisher.Poll(ctx, 2)
	if len(first) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(first))
	}
	if first[0].RequestID != "req-1" || first[1].RequestID != "req-2" {
		t.Fatalf("unexpected poll order: %+v", first)
	}

	second := publisher.Poll(ctx, 10)
	if len(second) != 1 || second[0].RequestID != "req-3" {
		t.Fatalf("unexpected second batch: %+v", second)
	}
	if third := publisher.Poll(ctx, 10); third != nil {
		t.Fatalf("expected empty queue to return nil, got %+v", third)
	}
}
