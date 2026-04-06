package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/service"
)

type DocumentRepository struct {
	db DBTX
}

func NewDocumentRepository(db DBTX) DocumentRepository {
	return DocumentRepository{db: db}
}

func (r DocumentRepository) ListDocuments(
	ctx context.Context,
	filter query.DocumentListFilter,
) ([]query.DocumentListItem, int, error) {
	page := filter.Page
	if page <= 0 {
		page = 1
	}

	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT
			d.id::text,
			d.title,
			COALESCE(p.name, '') AS project_name,
			COALESCE(f.path, '') AS folder_path,
			d.current_status::text,
			u.id::text,
			u.display_name,
			dv.version_no,
			TO_CHAR(d.updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM documents d
		LEFT JOIN projects p ON p.id = d.project_id
		LEFT JOIN folders f ON f.id = d.folder_id
		LEFT JOIN users u ON u.id = d.current_owner_id
		LEFT JOIN document_versions dv ON dv.id = d.current_version_id
		WHERE ($1 = '' OR d.team_space_id::text = $1)
		  AND ($2 = '' OR d.project_id::text = $2)
		  AND ($3 = '' OR d.folder_id::text = $3)
		  AND ($4 = '' OR d.current_owner_id::text = $4)
		  AND ($5 = '' OR d.current_status::text = $5)
		  AND ($6 = '' OR d.title ILIKE '%' || $6 || '%')
		  AND ($7 = true OR d.is_archived = false)
		  AND d.is_deleted = false
		ORDER BY d.updated_at DESC
		LIMIT $8 OFFSET $9
		`,
		filter.TeamSpaceID,
		filter.ProjectID,
		filter.FolderID,
		filter.OwnerID,
		filter.Status,
		filter.Keyword,
		filter.IncludeArchived,
		pageSize,
		(page-1)*pageSize,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]query.DocumentListItem, 0)
	for rows.Next() {
		var item query.DocumentListItem
		var owner query.UserSummary
		var versionNo *int
		var updatedAt *string

		if err := rows.Scan(
			&item.ID,
			&item.Title,
			&item.ProjectName,
			&item.FolderPath,
			&item.CurrentStatus,
			&owner.ID,
			&owner.DisplayName,
			&versionNo,
			&updatedAt,
		); err != nil {
			return nil, 0, err
		}

		item.CurrentOwner = &owner
		item.CurrentVersionNo = versionNo
		item.UpdatedAt = updatedAt
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var total int
	if err := r.db.QueryRowContext(
		ctx,
		`
		SELECT COUNT(1)
		FROM documents d
		WHERE ($1 = '' OR d.team_space_id::text = $1)
		  AND ($2 = '' OR d.project_id::text = $2)
		  AND ($3 = '' OR d.folder_id::text = $3)
		  AND ($4 = '' OR d.current_owner_id::text = $4)
		  AND ($5 = '' OR d.current_status::text = $5)
		  AND ($6 = '' OR d.title ILIKE '%' || $6 || '%')
		  AND ($7 = true OR d.is_archived = false)
		  AND d.is_deleted = false
		`,
		filter.TeamSpaceID,
		filter.ProjectID,
		filter.FolderID,
		filter.OwnerID,
		filter.Status,
		filter.Keyword,
		filter.IncludeArchived,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r DocumentRepository) GetDocument(ctx context.Context, documentID string) (*query.DocumentDetail, error) {
	var item query.DocumentDetail
	var owner query.UserSummary

	if err := r.db.QueryRowContext(
		ctx,
		`
		SELECT
			d.id::text,
			d.title,
			COALESCE(d.description, ''),
			d.current_status::text,
			u.id::text,
			u.display_name,
			COALESCE(d.current_version_id::text, ''),
			d.is_archived
		FROM documents d
		LEFT JOIN users u ON u.id = d.current_owner_id
		WHERE d.id::text = $1
		  AND d.is_deleted = false
		`,
		documentID,
	).Scan(
		&item.ID,
		&item.Title,
		&item.Description,
		&item.CurrentStatus,
		&owner.ID,
		&owner.DisplayName,
		&item.CurrentVersionID,
		&item.IsArchived,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrNotFound
		}
		return nil, err
	}

	item.CurrentOwner = &owner
	return &item, nil
}

func (r DocumentRepository) CreateDocument(ctx context.Context, input command.DocumentCreateInput) (map[string]any, error) {
	id := newID()
	now := time.Now().UTC()

	_, err := r.db.ExecContext(
		ctx,
		`
		INSERT INTO documents (
			id,
			team_space_id,
			project_id,
			folder_id,
			title,
			description,
			current_owner_id,
			current_status,
			is_archived,
			is_deleted,
			created_by,
			created_at,
			updated_at
		)
		VALUES (
			$1::uuid,
			$2::uuid,
			$3::uuid,
			NULLIF($4, '')::uuid,
			$5,
			NULLIF($6, ''),
			$7::uuid,
			'draft'::document_status,
			false,
			false,
			$8::uuid,
			$9,
			$9
		)
		`,
		id,
		input.TeamSpaceID,
		input.ProjectID,
		input.FolderID,
		input.Title,
		input.Description,
		input.CurrentOwnerID,
		actorOrSystem(input.ActorID),
		now,
	)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"id":             id,
		"title":          input.Title,
		"current_status": "draft",
	}, nil
}

func (r DocumentRepository) UpdateDocument(ctx context.Context, input command.DocumentUpdateInput) (map[string]any, error) {
	var title, description, status string
	var owner query.UserSummary
	var currentVersionID *string
	var isArchived bool

	err := r.db.QueryRowContext(
		ctx,
		`
		UPDATE documents
		SET title       = COALESCE(NULLIF($2, ''), title),
		    description = CASE WHEN $3 = '' THEN description ELSE $3 END,
		    folder_id   = COALESCE(NULLIF($4, '')::uuid, folder_id),
		    updated_at  = NOW()
		WHERE id::text = $1 AND is_deleted = false
		RETURNING
			id::text,
			title,
			COALESCE(description, ''),
			current_status::text,
			current_owner_id::text,
			(SELECT display_name FROM users WHERE id = documents.current_owner_id),
			current_version_id::text,
			is_archived
		`,
		input.DocumentID,
		input.Title,
		input.Description,
		input.FolderID,
	).Scan(&input.DocumentID, &title, &description, &status, &owner.ID, &owner.DisplayName, &currentVersionID, &isArchived)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrNotFound
		}
		return nil, err
	}

	result := map[string]any{
		"id":               input.DocumentID,
		"title":            title,
		"description":      description,
		"current_status":   status,
		"current_owner":    map[string]string{"id": owner.ID, "display_name": owner.DisplayName},
		"current_version_id": currentVersionID,
		"is_archived":      isArchived,
	}
	return result, nil
}

func (r DocumentRepository) DeleteDocument(ctx context.Context, input command.DocumentDeleteInput) error {
	res, err := r.db.ExecContext(
		ctx,
		`UPDATE documents SET is_deleted = true, updated_at = NOW() WHERE id::text = $1 AND is_deleted = false`,
		input.DocumentID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return service.ErrNotFound
	}
	return nil
}

func (r DocumentRepository) RestoreDocument(ctx context.Context, documentID string, actorID string) error {
	res, err := r.db.ExecContext(
		ctx,
		`UPDATE documents SET is_deleted = false, updated_at = NOW() WHERE id::text = $1 AND is_deleted = true`,
		documentID,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return service.ErrNotFound
	}
	return nil
}
