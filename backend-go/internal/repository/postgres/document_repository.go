package postgres

import (
	"context"
	"database/sql"
	"errors"

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
