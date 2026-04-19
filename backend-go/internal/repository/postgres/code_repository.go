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
)

type CodeRepositoryRepository struct {
	db *sql.DB
}

func NewCodeRepositoryRepository(db *sql.DB) CodeRepositoryRepository {
	return CodeRepositoryRepository{db: db}
}

func (r CodeRepositoryRepository) ListCodeRepositories(ctx context.Context, filter query.CodeRepositoryListFilter) ([]query.CodeRepositoryItem, int, error) {
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 30
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT
			cr.id::text,
			cr.team_space_id::text,
			cr.project_id::text,
			COALESCE(p.name, ''),
			cr.name,
			cr.slug,
			COALESCE(cr.description, ''),
			cr.default_branch,
			cr.target_folder_path,
			cr.repo_storage_path,
			COALESCE(cr.last_commit_sha, ''),
			COALESCE(TO_CHAR(cr.last_pushed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
			cr.status,
			COALESCE(u.display_name, ''),
			TO_CHAR(cr.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			TO_CHAR(cr.updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM code_repositories cr
		LEFT JOIN projects p ON p.id = cr.project_id
		LEFT JOIN users u ON u.id = cr.created_by
		WHERE cr.is_deleted = false
		  AND ($1 = '' OR cr.project_id::text = $1)
		  AND ($2 = '' OR cr.name ILIKE '%' || $2 || '%' OR cr.slug ILIKE '%' || $2 || '%')
		ORDER BY cr.updated_at DESC
		LIMIT $3 OFFSET $4
	`, filter.ProjectID, filter.Keyword, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]query.CodeRepositoryItem, 0)
	for rows.Next() {
		var item query.CodeRepositoryItem
		if err := rows.Scan(
			&item.ID, &item.TeamSpaceID, &item.ProjectID, &item.ProjectName,
			&item.Name, &item.Slug, &item.Description, &item.DefaultBranch,
			&item.TargetFolderPath, &item.RepoStoragePath, &item.LastCommitSHA,
			&item.LastPushedAt, &item.Status, &item.CreatedByName, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	var total int
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM code_repositories
		WHERE is_deleted = false
		  AND ($1 = '' OR project_id::text = $1)
		  AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR slug ILIKE '%' || $2 || '%')
	`, filter.ProjectID, filter.Keyword).Scan(&total); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r CodeRepositoryRepository) GetCodeRepository(ctx context.Context, id string) (*query.CodeRepositoryDetail, error) {
	return r.get(ctx, `cr.id::text = $1`, id, false)
}

func (r CodeRepositoryRepository) GetCodeRepositoryBySlug(ctx context.Context, slug string) (*query.CodeRepositoryDetail, error) {
	return r.get(ctx, `cr.slug = $1`, slug, true)
}

func (r CodeRepositoryRepository) get(ctx context.Context, where string, value string, includeToken bool) (*query.CodeRepositoryDetail, error) {
	selectToken := "''"
	if includeToken {
		selectToken = "cr.push_token"
	}
	row := r.db.QueryRowContext(ctx, `
		SELECT
			cr.id::text,
			cr.team_space_id::text,
			cr.project_id::text,
			COALESCE(p.name, ''),
			cr.name,
			cr.slug,
			COALESCE(cr.description, ''),
			cr.default_branch,
			cr.target_folder_path,
			cr.repo_storage_path,
			COALESCE(cr.last_commit_sha, ''),
			COALESCE(TO_CHAR(cr.last_pushed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), ''),
			cr.status,
			COALESCE(u.display_name, ''),
			TO_CHAR(cr.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			TO_CHAR(cr.updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			`+selectToken+`
		FROM code_repositories cr
		LEFT JOIN projects p ON p.id = cr.project_id
		LEFT JOIN users u ON u.id = cr.created_by
		WHERE cr.is_deleted = false
		  AND `+where, value)

	var item query.CodeRepositoryDetail
	if err := row.Scan(
		&item.ID, &item.TeamSpaceID, &item.ProjectID, &item.ProjectName,
		&item.Name, &item.Slug, &item.Description, &item.DefaultBranch,
		&item.TargetFolderPath, &item.RepoStoragePath, &item.LastCommitSHA,
		&item.LastPushedAt, &item.Status, &item.CreatedByName, &item.CreatedAt, &item.UpdatedAt,
		&item.PushToken,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (r CodeRepositoryRepository) CreateCodeRepository(ctx context.Context, input command.CodeRepositoryCreateInput) (*query.CodeRepositoryDetail, error) {
	id := newID()
	now := time.Now().UTC()
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO code_repositories (
			id, team_space_id, project_id, name, slug, description, default_branch,
			target_folder_path, repo_storage_path, push_token, status, created_by, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'active', $11, $12, $12)
	`, id, input.TeamSpaceID, input.ProjectID, input.Name, input.Slug, input.Description,
		input.DefaultBranch, input.TargetFolderPath, input.RepoStoragePath, input.PushToken, input.ActorID, now); err != nil {
		if strings.Contains(err.Error(), "code_repositories") && strings.Contains(err.Error(), "duplicate") {
			return nil, service.ErrConflict
		}
		return nil, err
	}
	item, err := r.GetCodeRepository(ctx, id)
	if err != nil {
		return nil, err
	}
	item.PushToken = input.PushToken
	return item, nil
}

func (r CodeRepositoryRepository) UpdateCodeRepository(ctx context.Context, input command.CodeRepositoryUpdateInput) (*query.CodeRepositoryDetail, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE code_repositories
		SET name = COALESCE(NULLIF($2, ''), name),
		    description = $3,
		    default_branch = COALESCE(NULLIF($4, ''), default_branch),
		    target_folder_path = COALESCE(NULLIF($5, ''), target_folder_path),
		    updated_at = NOW()
		WHERE id::text = $1
		  AND is_deleted = false
	`, input.RepositoryID, input.Name, input.Description, input.DefaultBranch, input.TargetFolderPath)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, service.ErrNotFound
	}
	return r.GetCodeRepository(ctx, input.RepositoryID)
}

func (r CodeRepositoryRepository) CreateCodePushEvent(ctx context.Context, input command.CodePushEventCreateInput) (*query.CodePushEventItem, error) {
	id := newID()
	now := time.Now().UTC()
	var pusher any
	if input.PusherID != "" {
		pusher = input.PusherID
	}
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO code_push_events (
			id, code_repository_id, branch, before_sha, after_sha, commit_message,
			pusher_id, sync_status, error_message, created_at, completed_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
	`, id, input.RepositoryID, input.Branch, input.BeforeSHA, input.AfterSHA,
		input.CommitMessage, pusher, input.SyncStatus, input.ErrorMessage, now); err != nil {
		return nil, err
	}
	events, err := r.ListCodePushEvents(ctx, input.RepositoryID)
	if err != nil {
		return nil, err
	}
	for _, item := range events {
		if item.ID == id {
			return &item, nil
		}
	}
	return nil, service.ErrNotFound
}

func (r CodeRepositoryRepository) UpdateCodeRepositoryAfterPush(ctx context.Context, repositoryID string, commitSHA string, status string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE code_repositories
		SET last_commit_sha = $2,
		    last_pushed_at = NOW(),
		    status = $3,
		    updated_at = NOW()
		WHERE id::text = $1
		  AND is_deleted = false
	`, repositoryID, commitSHA, status)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return service.ErrNotFound
	}
	return nil
}

func (r CodeRepositoryRepository) ListCodePushEvents(ctx context.Context, repositoryID string) ([]query.CodePushEventItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			cpe.id::text,
			cpe.code_repository_id::text,
			cpe.branch,
			COALESCE(cpe.before_sha, ''),
			COALESCE(cpe.after_sha, ''),
			COALESCE(cpe.commit_message, ''),
			COALESCE(u.display_name, ''),
			cpe.sync_status,
			COALESCE(cpe.error_message, ''),
			TO_CHAR(cpe.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			COALESCE(TO_CHAR(cpe.completed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')
		FROM code_push_events cpe
		LEFT JOIN users u ON u.id = cpe.pusher_id
		WHERE cpe.code_repository_id::text = $1
		ORDER BY cpe.created_at DESC
		LIMIT 50
	`, repositoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]query.CodePushEventItem, 0)
	for rows.Next() {
		var item query.CodePushEventItem
		if err := rows.Scan(
			&item.ID, &item.RepositoryID, &item.Branch, &item.BeforeSHA, &item.AfterSHA,
			&item.CommitMessage, &item.PusherName, &item.SyncStatus, &item.ErrorMessage,
			&item.CreatedAt, &item.CompletedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
