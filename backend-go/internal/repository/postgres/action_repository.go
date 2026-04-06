package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/shared"
)

type ActionRepository struct {
	db *sql.DB
}

func NewActionRepository(db *sql.DB) ActionRepository {
	return ActionRepository{db: db}
}

func (r ActionRepository) CreateFlowRecord(ctx context.Context, input command.FlowActionInput) (map[string]any, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var fromUserID string
	var fromStatus string
	if err := tx.QueryRowContext(
		ctx,
		`
		SELECT
			current_owner_id::text,
			current_status::text
		FROM documents
		WHERE id::text = $1
		`,
		input.DocumentID,
	).Scan(&fromUserID, &fromStatus); err != nil {
		return nil, err
	}

	if !isValidFlowTransition(fromStatus, input.Action) {
		return nil, fmt.Errorf("%w: action=%s from_status=%s", service.ErrInvalidTransition, input.Action, fromStatus)
	}

	toStatus := flowActionToStatus(input.Action)
	nextOwnerID := nextOwnerID(input, fromUserID)
	now := nowUTC()

	_, err = tx.ExecContext(
		ctx,
		`
		INSERT INTO flow_records (
			id,
			document_id,
			from_user_id,
			to_user_id,
			from_status,
			to_status,
			action,
			note,
			created_by,
			created_at
		)
		VALUES (
			$1::uuid,
			$2::uuid,
			NULLIF($3, '')::uuid,
			NULLIF($4, '')::uuid,
			NULLIF($5, '')::document_status,
			$6::document_status,
			$7,
			NULLIF($8, ''),
			$9::uuid,
			$10
		)
		`,
		newID(),
		input.DocumentID,
		fromUserID,
		nextOwnerID,
		fromStatus,
		toStatus,
		input.Action,
		input.Note,
		actorOrSystem(input.ActorID),
		now,
	)
	if err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(
		ctx,
		`
		UPDATE documents
		SET current_owner_id = COALESCE(NULLIF($2, '')::uuid, current_owner_id),
		    current_status = $3::document_status,
		    is_archived = $4,
		    updated_at = $5
		WHERE id::text = $1
		`,
		input.DocumentID,
		nullableOwnerID(input, nextOwnerID),
		toStatus,
		toStatus == "archived",
		now,
	); err != nil {
		return nil, err
	}

	if err := insertAuditEvent(
		ctx,
		tx,
		input.DocumentID,
		"",
		mappedAuditAction(input.Action),
		input.ActorID,
		map[string]any{
			"note":       input.Note,
			"to_user_id": nextOwnerID,
			"to_status":  toStatus,
		},
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	data := map[string]any{
		"document_id":    input.DocumentID,
		"action":         input.Action,
		"current_status": toStatus,
	}
	if input.Note != "" {
		data["note"] = input.Note
	}
	if nextOwnerID != "" {
		data["current_owner_id"] = nextOwnerID
	}

	return data, nil
}

func (r ActionRepository) CreateHandover(ctx context.Context, input command.HandoverCreateInput) (map[string]any, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	id := newID()
	now := nowUTC()

	_, err = tx.ExecContext(
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
		actorOrSystem(input.ActorID),
		now,
	)
	if err != nil {
		return nil, err
	}

	if err := insertAuditEvent(
		ctx,
		tx,
		"",
		"",
		"handover_generate",
		input.ActorID,
		map[string]any{
			"handover_id":      id,
			"target_user_id":   input.TargetUserID,
			"receiver_user_id": input.ReceiverUserID,
			"project_id":       input.ProjectID,
			"remark":           input.Remark,
		},
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
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

func (r ActionRepository) UpdateHandoverItems(
	ctx context.Context,
	input command.HandoverItemUpdateInput,
) (map[string]any, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM graduation_handover_items WHERE handover_id::text = $1`,
		input.HandoverID,
	); err != nil {
		return nil, err
	}

	now := nowUTC()
	for _, item := range input.Items {
		if item.DocumentID == "" {
			continue
		}
		if _, err := tx.ExecContext(
			ctx,
			`
			INSERT INTO graduation_handover_items (
				id,
				handover_id,
				document_id,
				selected,
				note,
				created_at
			)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4, NULLIF($5, ''), $6)
			`,
			newID(),
			input.HandoverID,
			item.DocumentID,
			item.Selected,
			item.Note,
			now,
		); err != nil {
			return nil, err
		}
	}

	if err := insertAuditEvent(
		ctx,
		tx,
		"",
		"",
		"admin_update",
		input.ActorID,
		map[string]any{
			"handover_id": input.HandoverID,
			"item_count":  len(input.Items),
		},
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return map[string]any{
		"id":    input.HandoverID,
		"items": input.Items,
	}, nil
}

func (r ActionRepository) ApplyHandover(ctx context.Context, input command.HandoverActionInput) (map[string]any, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var currentStatus string
	if err := tx.QueryRowContext(
		ctx,
		`
		SELECT status::text
		FROM graduation_handovers
		WHERE id::text = $1
		`,
		input.HandoverID,
	).Scan(&currentStatus); err != nil {
		return nil, err
	}

	if !isValidHandoverTransition(currentStatus, input.Action) {
		return nil, fmt.Errorf("%w: action=%s from_status=%s", service.ErrInvalidTransition, input.Action, currentStatus)
	}

	status, field := handoverActionToUpdate(input.Action)
	now := nowUTC()

	query := fmt.Sprintf(
		`
		UPDATE graduation_handovers
		SET status = $2::handover_status,
		    %s = $3
		WHERE id::text = $1
		`,
		field,
	)

	if _, err := tx.ExecContext(ctx, query, input.HandoverID, status, now); err != nil {
		return nil, err
	}

	if input.Action == "complete" {
		var receiverUserID string
		if err := tx.QueryRowContext(
			ctx,
			`
			SELECT receiver_user_id::text
			FROM graduation_handovers
			WHERE id::text = $1
			`,
			input.HandoverID,
		).Scan(&receiverUserID); err != nil {
			return nil, err
		}

		if _, err := tx.ExecContext(
			ctx,
			`
			UPDATE documents d
			SET current_owner_id = $2::uuid,
			    current_status = 'handed_over'::document_status,
			    updated_at = $3
			FROM graduation_handover_items ghi
			WHERE ghi.handover_id::text = $1
			  AND ghi.selected = TRUE
			  AND d.id = ghi.document_id
			`,
			input.HandoverID,
			receiverUserID,
			now,
		); err != nil {
			return nil, err
		}
	}

	if err := insertAuditEvent(
		ctx,
		tx,
		"",
		"",
		mappedHandoverAuditAction(input.Action),
		input.ActorID,
		map[string]any{
			"handover_id": input.HandoverID,
			"note":        input.Note,
			"reason":      input.Reason,
			"status":      status,
		},
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	data := map[string]any{
		"id":     input.HandoverID,
		"action": input.Action,
		"status": status,
	}
	if input.Note != "" {
		data["note"] = input.Note
	}
	if input.Reason != "" {
		data["reason"] = input.Reason
	}

	return data, nil
}

func insertAuditEvent(
	ctx context.Context,
	db DBTX,
	documentID string,
	versionID string,
	actionType string,
	actorID string,
	extra map[string]any,
) error {
	extraData, err := json.Marshal(extra)
	if err != nil {
		return err
	}

	userID := actorID
	if userID == "" {
		userID = systemUserID()
	}

	reqID := shared.RequestIDFromContext(ctx)

	_, err = db.ExecContext(
		ctx,
		`
		INSERT INTO audit_events (
			id,
			document_id,
			version_id,
			user_id,
			action_type,
			request_id,
			terminal_info,
			extra_data,
			created_at
		)
		VALUES (
			$1::uuid,
			NULLIF($2, '')::uuid,
			NULLIF($3, '')::uuid,
			$4::uuid,
			$5::audit_action_type,
			NULLIF($8, ''),
			'backend-go',
			$6::jsonb,
			$7
		)
		`,
		newID(),
		documentID,
		versionID,
		userID,
		actionType,
		string(extraData),
		nowUTC(),
		reqID,
	)
	return err
}

func nextOwnerID(input command.FlowActionInput, currentOwnerID string) string {
	switch input.Action {
	case "transfer":
		if input.ToUserID != "" {
			return input.ToUserID
		}
		return currentOwnerID
	default:
		return currentOwnerID
	}
}

func nullableOwnerID(input command.FlowActionInput, nextOwnerID string) string {
	switch input.Action {
	case "transfer":
		return nextOwnerID
	default:
		return ""
	}
}

func mappedAuditAction(action string) string {
	switch action {
	case "transfer":
		return "transfer"
	case "accept_transfer":
		return "receive_transfer"
	case "finalize":
		return "finalize"
	case "archive":
		return "archive"
	case "unarchive":
		return "restore"
	default:
		return "admin_update"
	}
}

func mappedHandoverAuditAction(action string) string {
	switch action {
	case "confirm":
		return "handover_confirm"
	case "complete":
		return "handover_complete"
	default:
		return "admin_update"
	}
}

func isValidFlowTransition(currentStatus string, action string) bool {
	switch action {
	case "mark_in_progress":
		return currentStatus == "draft" || currentStatus == "handed_over" || currentStatus == "in_progress"
	case "transfer":
		return currentStatus == "in_progress"
	case "accept_transfer":
		return currentStatus == "pending_handover"
	case "finalize":
		return currentStatus == "in_progress" || currentStatus == "handed_over"
	case "archive":
		return currentStatus == "finalized" || currentStatus == "handed_over"
	case "unarchive":
		return currentStatus == "archived"
	default:
		return false
	}
}

func isValidHandoverTransition(currentStatus string, action string) bool {
	switch action {
	case "confirm":
		return currentStatus == "generated"
	case "complete":
		return currentStatus == "pending_confirm"
	case "cancel":
		return currentStatus == "generated" || currentStatus == "pending_confirm"
	default:
		return false
	}
}

func newID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		// rand.Read failing is an unrecoverable OS-level error; panic to surface it.
		panic("crypto/rand unavailable: " + err.Error())
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
	case "transfer":
		// Transfer puts the document into pending_handover, waiting for accept.
		return "pending_handover"
	case "accept_transfer", "mark_in_progress":
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

func actorOrSystem(actorID string) string {
	if actorID == "" {
		return systemUserID()
	}
	return actorID
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
