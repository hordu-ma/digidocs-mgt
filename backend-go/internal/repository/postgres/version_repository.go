package postgres

import (
	"context"
	"database/sql"
	"errors"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/service"
)

type VersionRepository struct {
	db DBTX
}

func NewVersionRepository(db DBTX) VersionRepository {
	return VersionRepository{db: db}
}

func (r VersionRepository) ListVersions(ctx context.Context, documentID string) ([]query.VersionItem, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT
			id::text,
			version_no,
			file_name,
			summary_status,
			TO_CHAR(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM document_versions
		WHERE document_id::text = $1
		ORDER BY version_no DESC
		`,
		documentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]query.VersionItem, 0)
	for rows.Next() {
		var item query.VersionItem
		if err := rows.Scan(
			&item.ID,
			&item.VersionNo,
			&item.FileName,
			&item.SummaryStatus,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r VersionRepository) GetVersion(ctx context.Context, versionID string) (*query.VersionDetail, error) {
	var item query.VersionDetail

	if err := r.db.QueryRowContext(
		ctx,
		`
		SELECT
			id::text,
			document_id::text,
			version_no,
			COALESCE(commit_message, ''),
			file_name
		FROM document_versions
		WHERE id::text = $1
		`,
		versionID,
	).Scan(
		&item.ID,
		&item.DocumentID,
		&item.VersionNo,
		&item.CommitMessage,
		&item.FileName,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrNotFound
		}
		return nil, err
	}

	return &item, nil
}

func (r VersionRepository) CreateVersion(ctx context.Context, input command.VersionCreateInput) (map[string]any, error) {
	id := newID()

	var versionNo int
	if err := r.db.QueryRowContext(
		ctx,
		`
		SELECT COALESCE(MAX(version_no), 0) + 1
		FROM document_versions
		WHERE document_id::text = $1
		`,
		input.DocumentID,
	).Scan(&versionNo); err != nil {
		return nil, err
	}

	_, err := r.db.ExecContext(
		ctx,
		`
		INSERT INTO document_versions (
			id,
			document_id,
			version_no,
			file_name,
			file_size,
			storage_provider,
			storage_object_key,
			commit_message,
			extracted_text_status,
			summary_status,
			created_by,
			created_at
		)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, NULLIF($8, ''), 'pending', 'pending', $9::uuid, $10)
		`,
		id,
		input.DocumentID,
		versionNo,
		input.FileName,
		input.FileSize,
		input.StorageProvider,
		input.StorageObjectKey,
		input.CommitMessage,
		systemUserID(),
		nowUTC(),
	)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"id":             id,
		"document_id":    input.DocumentID,
		"version_no":     versionNo,
		"commit_message": input.CommitMessage,
		"file_name":      input.FileName,
	}, nil
}
