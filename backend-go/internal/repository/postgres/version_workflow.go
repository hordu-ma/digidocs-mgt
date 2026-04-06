package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/shared"
)

type VersionWorkflow struct {
	db *sql.DB
}

func NewVersionWorkflow(db *sql.DB) VersionWorkflow {
	return VersionWorkflow{db: db}
}

func (w VersionWorkflow) CreateUploadedVersion(ctx context.Context, input command.VersionCreateInput) (map[string]any, error) {
	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	versionRepo := NewVersionRepository(tx)
	versionData, err := versionRepo.CreateVersion(ctx, input)
	if err != nil {
		return nil, err
	}

	versionID, _ := versionData["id"].(string)
	versionNo, _ := versionData["version_no"].(int)

	if _, err := tx.ExecContext(
		ctx,
		`
		UPDATE documents
		SET current_version_id = $2::uuid,
		    current_status = 'in_progress'::document_status,
		    updated_at = $3
		WHERE id::text = $1
		`,
		input.DocumentID,
		versionID,
		time.Now().UTC(),
	); err != nil {
		return nil, err
	}

	extraData, err := json.Marshal(map[string]any{
		"file_name":  input.FileName,
		"object_key": input.StorageObjectKey,
		"provider":   input.StorageProvider,
	})
	if err != nil {
		return nil, err
	}

	reqID := shared.RequestIDFromContext(ctx)

	if _, err := tx.ExecContext(
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
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, 'replace_version'::audit_action_type, NULLIF($7, ''), 'backend-go', $5::jsonb, $6)
		`,
		newID(),
		input.DocumentID,
		versionID,
		actorOrSystem(input.ActorID),
		string(extraData),
		time.Now().UTC(),
		reqID,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return map[string]any{
		"id":             versionID,
		"document_id":    input.DocumentID,
		"version_no":     versionNo,
		"commit_message": input.CommitMessage,
		"file_name":      input.FileName,
		"current_status": "in_progress",
	}, nil
}
