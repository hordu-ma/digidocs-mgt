package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/textsearch"
)

type DocumentRepository struct {
	db DBTX
}

func NewDocumentRepository(db DBTX) DocumentRepository {
	return DocumentRepository{db: db}
}

// documentSortSQL whitelists the list orderings; anything else falls back to
// most recently updated. Ties break on id so pagination is stable.
var documentSortSQL = map[string]string{
	"updated_desc": "d.updated_at DESC, d.id",
	"updated_asc":  "d.updated_at ASC, d.id",
	"title_asc":    "d.title ASC, d.id",
	"owner_asc":    "u.display_name ASC, d.updated_at DESC, d.id",
}

// documentListWhere builds the shared WHERE clause of the list and its counts.
// It returns the clause and its bind arguments ($1..$n). skipProject leaves the
// project filter out, for the per-project counts shown in the side rail.
func documentListWhere(filter query.DocumentListFilter, skipProject bool) (string, []any) {
	isAdmin := filter.ActorRole == "admin"
	projectID := filter.ProjectID
	if skipProject {
		projectID = ""
	}
	keyword := ""
	if k := strings.TrimSpace(filter.Keyword); k != "" {
		keyword = textsearch.LikePatterns([]string{k})[0]
	}
	args := []any{
		filter.TeamSpaceID, projectID, filter.FolderID, filter.OwnerID, filter.Status,
		keyword, filter.IncludeArchived, filter.ActorID, isAdmin, filter.UpdatedWithinDays, filter.DeletedOnly,
	}
	where := `
		WHERE ($1 = '' OR d.team_space_id::text = $1)
		  AND ($2 = '' OR d.project_id::text = $2)
		  AND ($3 = '' OR d.folder_id::text = $3)
		  AND ($4 = '' OR d.current_owner_id::text = $4)
		  AND ($5 = '' OR d.current_status::text = $5)
		  AND ($6 = '' OR d.title ILIKE $6)
		  AND ($7 = true OR $11 = true OR d.is_archived = false)
		  AND ($10::int <= 0 OR d.updated_at >= NOW() - make_interval(days => $10::int))
		  AND d.is_deleted = $11
		  AND (
			$9 = true
			OR (
				-- 回收站只对课题 owner/manager 可见；其余列表对课题成员可见。
				$11 = false AND EXISTS (
					SELECT 1 FROM projects sp
					LEFT JOIN project_members spm ON spm.project_id = sp.id AND spm.user_id::text = $8
					WHERE sp.id = d.project_id AND (sp.owner_id::text = $8 OR spm.id IS NOT NULL)
				)
			)
			OR (
				$11 = true AND EXISTS (
					SELECT 1 FROM projects sp
					LEFT JOIN project_members spm ON spm.project_id = sp.id AND spm.user_id::text = $8
					 AND spm.project_role IN ('owner', 'manager')
					WHERE sp.id = d.project_id AND (sp.owner_id::text = $8 OR spm.id IS NOT NULL)
				)
			)
		  )`
	return where, args
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
	orderBy, ok := documentSortSQL[filter.Sort]
	if !ok {
		orderBy = documentSortSQL["updated_desc"]
	}
	where, args := documentListWhere(filter, false)

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
			TO_CHAR(d.updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			d.is_archived,
			d.is_deleted,
			d.project_id::text
		FROM documents d
		LEFT JOIN projects p ON p.id = d.project_id
		LEFT JOIN folders f ON f.id = d.folder_id
		LEFT JOIN users u ON u.id = d.current_owner_id
		LEFT JOIN document_versions dv ON dv.id = d.current_version_id
		`+where+`
		ORDER BY `+orderBy+`
		LIMIT $12 OFFSET $13
		`,
		append(args, pageSize, (page-1)*pageSize)...,
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
			&item.IsArchived,
			&item.IsDeleted,
			&item.ProjectID,
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
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM documents d LEFT JOIN users u ON u.id = d.current_owner_id `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// CountDocumentsByProject returns, for the same filters minus the project,
// how many documents each visible project has — the counts of the side rail,
// which must not depend on which page of the list is loaded.
func (r DocumentRepository) CountDocumentsByProject(ctx context.Context, filter query.DocumentListFilter) ([]query.ProjectDocumentCount, error) {
	where, args := documentListWhere(filter, true)
	rows, err := r.db.QueryContext(ctx, `
		SELECT d.project_id::text, COALESCE(p.name, ''), COUNT(1)::int
		FROM documents d
		LEFT JOIN projects p ON p.id = d.project_id
		LEFT JOIN users u ON u.id = d.current_owner_id
		`+where+`
		GROUP BY d.project_id, p.name
		ORDER BY p.name
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]query.ProjectDocumentCount, 0)
	for rows.Next() {
		var c query.ProjectDocumentCount
		if err := rows.Scan(&c.ProjectID, &c.ProjectName, &c.Count); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
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
			d.is_archived,
 d.project_id::text
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
		&item.ProjectID,
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
		"id":                 input.DocumentID,
		"title":              title,
		"description":        description,
		"current_status":     status,
		"current_owner":      map[string]string{"id": owner.ID, "display_name": owner.DisplayName},
		"current_version_id": currentVersionID,
		"is_archived":        isArchived,
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
