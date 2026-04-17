package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/service"
)

type DataAssetRepository struct {
	db *sql.DB
}

func NewDataAssetRepository(db *sql.DB) DataAssetRepository {
	return DataAssetRepository{db: db}
}

// ─────────────────────────── folders ────────────────────────────

func (r DataAssetRepository) ListDataFolders(ctx context.Context, projectID string) ([]query.DataFolderItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			id::text,
			project_id::text,
			COALESCE(parent_id::text, ''),
			depth,
			name,
			TO_CHAR(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM data_folders
		WHERE project_id::text = $1
		ORDER BY depth ASC, name ASC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]query.DataFolderItem, 0)
	for rows.Next() {
		var item query.DataFolderItem
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.ParentID, &item.Depth, &item.Name, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r DataAssetRepository) CreateDataFolder(ctx context.Context, input command.DataFolderCreateInput) (*query.DataFolderItem, error) {
	depth := 0
	if input.ParentID != "" {
		var parentDepth int
		if err := r.db.QueryRowContext(ctx,
			`SELECT depth FROM data_folders WHERE id::text = $1 AND project_id::text = $2`,
			input.ParentID, input.ProjectID,
		).Scan(&parentDepth); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("%w: parent folder not found", service.ErrNotFound)
			}
			return nil, err
		}
		depth = parentDepth + 1
		if depth > 2 {
			return nil, fmt.Errorf("%w: max folder depth is 2", service.ErrValidation)
		}
	}

	id := newID()
	now := time.Now().UTC()

	var parentArg any
	if input.ParentID != "" {
		parentArg = input.ParentID
	}

	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO data_folders (id, project_id, parent_id, depth, name, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
	`, id, input.ProjectID, parentArg, depth, input.Name, input.ActorID, now); err != nil {
		if strings.Contains(err.Error(), "uq_data_folder_parent_name") {
			return nil, fmt.Errorf("%w: folder name already exists", service.ErrConflict)
		}
		return nil, err
	}

	return &query.DataFolderItem{
		ID:        id,
		ProjectID: input.ProjectID,
		ParentID:  input.ParentID,
		Depth:     depth,
		Name:      input.Name,
		CreatedAt: now.Format(time.RFC3339),
	}, nil
}

func (r DataAssetRepository) DeleteDataFolder(ctx context.Context, id string) error {
	var count int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM data_assets WHERE folder_id::text = $1 AND is_deleted = false`,
		id,
	).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("%w: folder is not empty", service.ErrValidation)
	}

	var subCount int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM data_folders WHERE parent_id::text = $1`,
		id,
	).Scan(&subCount); err != nil {
		return err
	}
	if subCount > 0 {
		return fmt.Errorf("%w: folder has sub-folders", service.ErrValidation)
	}

	res, err := r.db.ExecContext(ctx, `DELETE FROM data_folders WHERE id::text = $1`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return service.ErrNotFound
	}
	return nil
}

// ─────────────────────────── assets ─────────────────────────────

func (r DataAssetRepository) ListDataAssets(ctx context.Context, filter query.DataAssetListFilter) ([]query.DataAssetListItem, int, error) {
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
			da.id::text,
			da.project_id::text,
			COALESCE(p.name, ''),
			COALESCE(da.folder_id::text, ''),
			COALESCE(df.name, ''),
			da.display_name,
			da.file_name,
			COALESCE(da.mime_type, ''),
			da.file_size,
			COALESCE(u.display_name, ''),
			TO_CHAR(da.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM data_assets da
		LEFT JOIN projects p ON p.id = da.project_id
		LEFT JOIN data_folders df ON df.id = da.folder_id
		LEFT JOIN users u ON u.id = da.created_by
		WHERE da.is_deleted = false
		  AND ($1 = '' OR da.project_id::text = $1)
		  AND ($2 = '' OR da.folder_id::text = $2)
		  AND ($3 = '' OR da.display_name ILIKE '%' || $3 || '%' OR da.file_name ILIKE '%' || $3 || '%')
		ORDER BY da.created_at DESC
		LIMIT $4 OFFSET $5
	`, filter.ProjectID, filter.FolderID, filter.Keyword, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]query.DataAssetListItem, 0)
	for rows.Next() {
		var item query.DataAssetListItem
		if err := rows.Scan(
			&item.ID, &item.ProjectID, &item.ProjectName,
			&item.FolderID, &item.FolderName,
			&item.DisplayName, &item.FileName,
			&item.MimeType, &item.FileSize,
			&item.CreatedByName, &item.CreatedAt,
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
		FROM data_assets da
		WHERE da.is_deleted = false
		  AND ($1 = '' OR da.project_id::text = $1)
		  AND ($2 = '' OR da.folder_id::text = $2)
		  AND ($3 = '' OR da.display_name ILIKE '%' || $3 || '%' OR da.file_name ILIKE '%' || $3 || '%')
	`, filter.ProjectID, filter.FolderID, filter.Keyword).Scan(&total); err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r DataAssetRepository) GetDataAsset(ctx context.Context, id string) (*query.DataAssetDetail, error) {
	var item query.DataAssetDetail
	if err := r.db.QueryRowContext(ctx, `
		SELECT
			da.id::text,
			da.team_space_id::text,
			da.project_id::text,
			COALESCE(p.name, ''),
			COALESCE(da.folder_id::text, ''),
			COALESCE(df.name, ''),
			da.display_name,
			da.file_name,
			COALESCE(da.description, ''),
			COALESCE(da.mime_type, ''),
			da.file_size,
			da.storage_provider,
			da.storage_object_key,
			COALESCE(u.display_name, ''),
			TO_CHAR(da.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			TO_CHAR(da.updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM data_assets da
		LEFT JOIN projects p ON p.id = da.project_id
		LEFT JOIN data_folders df ON df.id = da.folder_id
		LEFT JOIN users u ON u.id = da.created_by
		WHERE da.id::text = $1
		  AND da.is_deleted = false
	`, id).Scan(
		&item.ID, &item.TeamSpaceID, &item.ProjectID, &item.ProjectName,
		&item.FolderID, &item.FolderName,
		&item.DisplayName, &item.FileName,
		&item.Description, &item.MimeType, &item.FileSize,
		&item.StorageProvider, &item.StorageObjectKey,
		&item.CreatedByName, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (r DataAssetRepository) CreateDataAsset(ctx context.Context, input command.DataAssetCreateInput) (map[string]any, error) {
	id := newID()
	now := time.Now().UTC()

	mimeType := input.MimeType
	if mimeType == "" {
		ext := strings.ToLower(filepath.Ext(input.FileName))
		if mt := mime.TypeByExtension(ext); mt != "" {
			mimeType = mt
		}
	}

	var folderArg any
	if input.FolderID != "" {
		folderArg = input.FolderID
	}

	var bucketArg any
	if input.StorageBucketOrShare != "" {
		bucketArg = input.StorageBucketOrShare
	}

	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO data_assets (
			id, team_space_id, project_id, folder_id,
			display_name, file_name, description, mime_type,
			file_size, storage_provider, storage_bucket_or_share, storage_object_key,
			created_by, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$14)
	`,
		id, input.TeamSpaceID, input.ProjectID, folderArg,
		input.DisplayName, input.FileName, input.Description, mimeType,
		input.FileSize, input.StorageProvider, bucketArg, input.StorageObjectKey,
		input.ActorID, now,
	); err != nil {
		return nil, err
	}

	return map[string]any{
		"id":           id,
		"project_id":   input.ProjectID,
		"display_name": input.DisplayName,
		"file_name":    input.FileName,
		"file_size":    input.FileSize,
		"mime_type":    mimeType,
		"created_at":   now.Format(time.RFC3339),
	}, nil
}

func (r DataAssetRepository) UpdateDataAsset(ctx context.Context, input command.DataAssetUpdateInput) error {
	var folderArg any
	if input.FolderID != "" {
		folderArg = input.FolderID
	}

	res, err := r.db.ExecContext(ctx, `
		UPDATE data_assets
		SET display_name = $2, description = $3, folder_id = $4, updated_at = NOW()
		WHERE id::text = $1 AND is_deleted = false
	`, input.DataAssetID, input.DisplayName, input.Description, folderArg)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return service.ErrNotFound
	}
	return nil
}

func (r DataAssetRepository) DeleteDataAsset(ctx context.Context, input command.DataAssetDeleteInput) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE data_assets
		SET is_deleted = true, deleted_at = NOW(), deleted_by = $2, updated_at = NOW()
		WHERE id::text = $1 AND is_deleted = false
	`, input.DataAssetID, input.ActorID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return service.ErrNotFound
	}
	return nil
}

// ─────────────────────── handover data items ─────────────────────

func (r DataAssetRepository) ListHandoverDataItems(ctx context.Context, handoverID string) ([]query.HandoverDataLine, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			hi.data_asset_id::text,
			COALESCE(da.display_name, ''),
			COALESCE(da.file_name, ''),
			hi.selected,
			COALESCE(hi.note, '')
		FROM graduation_handover_data_items hi
		LEFT JOIN data_assets da ON da.id = hi.data_asset_id
		WHERE hi.handover_id::text = $1
		ORDER BY hi.created_at ASC
	`, handoverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]query.HandoverDataLine, 0)
	for rows.Next() {
		var line query.HandoverDataLine
		if err := rows.Scan(&line.DataAssetID, &line.DisplayName, &line.FileName, &line.Selected, &line.Note); err != nil {
			return nil, err
		}
		items = append(items, line)
	}
	return items, rows.Err()
}

func (r DataAssetRepository) UpdateHandoverDataItems(ctx context.Context, input command.HandoverDataItemUpdateInput) (map[string]any, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM graduation_handover_data_items WHERE handover_id::text = $1`,
		input.HandoverID,
	); err != nil {
		return nil, err
	}

	for _, item := range input.Items {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO graduation_handover_data_items (id, handover_id, data_asset_id, selected, note)
			VALUES ($1, $2, $3, $4, $5)
		`, newID(), input.HandoverID, item.DataAssetID, item.Selected, nullString(item.Note)); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return map[string]any{"handover_id": input.HandoverID, "count": len(input.Items)}, nil
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
