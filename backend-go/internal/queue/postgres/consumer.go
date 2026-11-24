package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"

	"digidocs-mgt/backend-go/internal/domain/task"
	"digidocs-mgt/backend-go/internal/queue"
	"github.com/lib/pq"
)

// MaxTaskAttempts is how many times a task may be claimed before a further
// timeout marks it failed.
const MaxTaskAttempts = 3

type Consumer struct {
	db *sql.DB
}

func NewConsumer(db *sql.DB) queue.Consumer {
	return Consumer{db: db}
}

func (c Consumer) Poll(ctx context.Context, limit int) []task.Message {
	if limit <= 0 {
		limit = 10
	}

	// Recover tasks stuck in 'running' for more than 10 minutes (e.g. worker crash).
	// Gated on started_at (when the task actually entered running), not
	// created_at: a task that merely sat pending for a while before being
	// claimed must not be treated as stuck the instant it starts running,
	// or it gets reset to pending and picked up a second time concurrently.
	// started_at IS NULL covers rows written before this column existed.
	//
	// Dead-letter (2026-11-10): a task that has already been claimed
	// MaxTaskAttempts times and is stuck again is marked failed instead of
	// being re-queued forever. Without this, one file that reliably wedges the
	// worker would be retried every ten minutes indefinitely, each time
	// occupying a worker slot and calling the model again.
	_, recoverErr := c.db.ExecContext(
		ctx,
		`UPDATE assistant_requests
		 SET status = CASE WHEN attempts >= $1 THEN 'failed' ELSE 'pending' END,
		     error_message = CASE WHEN attempts >= $1
		       THEN '任务多次超时未完成，已停止重试（已尝试 ' || attempts || ' 次）'
		       ELSE error_message END,
		     completed_at = CASE WHEN attempts >= $1 THEN NOW() ELSE completed_at END
		 WHERE status = 'running'
		   AND (started_at IS NULL OR started_at < NOW() - INTERVAL '10 minutes')`,
		MaxTaskAttempts,
	)
	if recoverErr != nil {
		log.Printf("postgres-queue recover-running failed: %v", recoverErr)
	}

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("postgres-queue begin tx failed: %v", err)
		return nil
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(
		ctx,
		`
		SELECT
			id::text,
			request_type,
			COALESCE(related_type, ''),
			COALESCE(related_id::text, ''),
			COALESCE(payload::text, '{}'), attempts + 1
		FROM assistant_requests
		WHERE status = 'pending'
		ORDER BY created_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT $1
		`,
		limit,
	)
	if err != nil {
		log.Printf("postgres-queue query failed: %v", err)
		return nil
	}
	defer rows.Close()

	messages := make([]task.Message, 0)
	ids := make([]string, 0)
	for rows.Next() {
		var message task.Message
		var requestType string
		var payloadText string
		if err := rows.Scan(
			&message.RequestID,
			&requestType,
			&message.RelatedType,
			&message.RelatedID,
			&payloadText,
			&message.Attempt,
		); err != nil {
			log.Printf("postgres-queue scan failed: %v", err)
			return nil
		}
		message.TaskType = task.TaskType(requestType)
		message.Payload = map[string]any{}
		if err := json.Unmarshal([]byte(payloadText), &message.Payload); err != nil {
			log.Printf("postgres-queue payload decode failed request_id=%s err=%v", message.RequestID, err)
			return nil
		}
		messages = append(messages, message)
		ids = append(ids, message.RequestID)
	}
	if err := rows.Err(); err != nil {
		log.Printf("postgres-queue rows failed: %v", err)
		return nil
	}
	if len(ids) == 0 {
		return nil
	}

	if _, err := tx.ExecContext(
		ctx,
		`
		UPDATE assistant_requests
		SET status = 'running', started_at = NOW(), attempts = attempts + 1, output = NULL, error_message = NULL
		WHERE id::text = ANY($1)
		`,
		pq.Array(ids),
	); err != nil {
		log.Printf("postgres-queue update failed: %v", err)
		return nil
	}

	if err := tx.Commit(); err != nil {
		log.Printf("postgres-queue commit failed: %v", err)
		return nil
	}

	return messages
}
