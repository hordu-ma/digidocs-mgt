package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"digidocs-mgt/backend-go/internal/domain/command"
)

type ActionRepository struct {
	db DBTX
}

func NewActionRepository(db DBTX) ActionRepository {
	return ActionRepository{db: db}
}

func (r ActionRepository) CreateFlowRecord(ctx context.Context, input command.FlowActionInput) (map[string]any, error) {
	id := newID()
	toStatus := flowActionToStatus(input.Action)

	_, err := r.db.ExecContext(
		ctx,
		`
		INSERT INTO flow_records (
			id,
			document_id,
			to_user_id,
			to_status,
			action,
			note,
			created_by,
			created_at
		)
		VALUES ($1::uuid, $2::uuid, NULLIF($3, '')::uuid, $4::document_status, $5, NULLIF($6, ''), $7::uuid, $8)
		`,
		id,
		input.DocumentID,
		input.ToUserID,
		toStatus,
		input.Action,
		input.Note,
		systemUserID(),
		time.Now().UTC(),
	)
	if err != nil {
		return nil, err
	}

	data := map[string]any{
		"id":          id,
		"document_id": input.DocumentID,
		"action":      input.Action,
	}
	if input.Note != "" {
		data["note"] = input.Note
	}
	if input.ToUserID != "" {
		data["to_user_id"] = input.ToUserID
	}

	return data, nil
}

func (r ActionRepository) CreateHandover(ctx context.Context, input command.HandoverCreateInput) (map[string]any, error) {
	id := newID()

	_, err := r.db.ExecContext(
		ctx,
		`
		INSERT INTO graduation_handovers (
			id,
			target_user_id,
			receiver_user_id,
			project_id,
			status,
			remark,
			generated_by,
			generated_at
		)
		VALUES ($1::uuid, $2::uuid, $3::uuid, NULLIF($4, '')::uuid, 'generated'::handover_status, NULLIF($5, ''), $6::uuid, $7)
		`,
		id,
		input.TargetUserID,
		input.ReceiverUserID,
		input.ProjectID,
		input.Remark,
		systemUserID(),
		time.Now().UTC(),
	)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"id":               id,
		"target_user_id":   input.TargetUserID,
		"receiver_user_id": input.ReceiverUserID,
		"project_id":       input.ProjectID,
		"remark":           input.Remark,
		"status":           "generated",
	}, nil
}

func (r ActionRepository) ApplyHandover(ctx context.Context, input command.HandoverActionInput) (map[string]any, error) {
	status, field := handoverActionToUpdate(input.Action)

	query := fmt.Sprintf(
		`
		UPDATE graduation_handovers
		SET status = $2::handover_status,
		    %s = $3
		WHERE id::text = $1
		`,
		field,
	)

	_, err := r.db.ExecContext(ctx, query, input.HandoverID, status, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	data := map[string]any{
		"id":     input.HandoverID,
		"action": input.Action,
	}
	if input.Note != "" {
		data["note"] = input.Note
	}
	if input.Reason != "" {
		data["reason"] = input.Reason
	}

	return data, nil
}

func newID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "00000000-0000-0000-0000-000000000000"
	}

	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80

	hexed := hex.EncodeToString(buf)
	return fmt.Sprintf("%s-%s-%s-%s-%s", hexed[0:8], hexed[8:12], hexed[12:16], hexed[16:20], hexed[20:32])
}

func flowActionToStatus(action string) string {
	switch action {
	case "archive":
		return "archived"
	case "finalize":
		return "finalized"
	case "transfer", "accept_transfer", "mark_in_progress":
		return "in_progress"
	case "unarchive":
		return "in_progress"
	default:
		return "in_progress"
	}
}

func handoverActionToUpdate(action string) (string, string) {
	switch action {
	case "confirm":
		return "pending_confirm", "confirmed_at"
	case "complete":
		return "completed", "completed_at"
	case "cancel":
		return "cancelled", "cancelled_at"
	default:
		return "generated", "generated_at"
	}
}

func systemUserID() string {
	return "00000000-0000-0000-0000-000000000001"
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
