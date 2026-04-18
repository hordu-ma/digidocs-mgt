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
	_, recoverErr := c.db.ExecContext(
		ctx,
		`UPDATE assistant_requests
		 SET status = 'pending', updated_at = NOW()
		 WHERE status = 'running'
		   AND updated_at < NOW() - INTERVAL '10 minutes'`,
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
			COALESCE(payload::text, '{}')
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
		SET status = 'running'
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
