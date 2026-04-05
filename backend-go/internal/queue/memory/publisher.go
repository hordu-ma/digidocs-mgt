package memory

import (
	"context"
	"log"
	"sync"

	"digidocs-mgt/backend-go/internal/domain/task"
)

type Publisher struct {
	mu    sync.Mutex
	queue []task.Message
}

func NewPublisher() *Publisher {
	return &Publisher{}
}

func (p *Publisher) Publish(_ context.Context, message task.Message) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.queue = append(p.queue, message)
	log.Printf("memory-queue publish task=%s request_id=%s queue_len=%d", message.TaskType, message.RequestID, len(p.queue))
	return nil
}

// Poll atomically drains up to limit messages from the queue.
func (p *Publisher) Poll(_ context.Context, limit int) []task.Message {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.queue) == 0 {
		return nil
	}

	n := limit
	if n > len(p.queue) {
		n = len(p.queue)
	}

	batch := make([]task.Message, n)
	copy(batch, p.queue[:n])
	p.queue = p.queue[n:]
	return batch
}
