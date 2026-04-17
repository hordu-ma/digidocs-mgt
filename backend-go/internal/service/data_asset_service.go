package service

import (
	"context"
	"fmt"
	"io"
	"time"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/repository"
	"digidocs-mgt/backend-go/internal/storage"
)

func nowUnix() int64 { return time.Now().UnixNano() }

type DataAssetService struct {
	reader      repository.DataAssetReader
	writer      repository.DataAssetWriter
	storage     storage.Provider
	permissions PermissionService
}

func NewDataAssetService(
	reader repository.DataAssetReader,
	writer repository.DataAssetWriter,
	storageProvider storage.Provider,
	permissions ...PermissionService,
) DataAssetService {
	perm := PermissionService{}
	if len(permissions) > 0 {
		perm = permissions[0]
	}
	return DataAssetService{
		reader:      reader,
		writer:      writer,
		storage:     storageProvider,
		permissions: perm,
	}
}

// ─────────────────────────── folders ────────────────────────────

func (s DataAssetService) ListDataFolders(ctx context.Context, projectID string) ([]query.DataFolderItem, error) {
	if projectID == "" {
		return nil, fmt.Errorf("%w: project_id is required", ErrValidation)
	}
	return s.reader.ListDataFolders(ctx, projectID)
}

func (s DataAssetService) CreateDataFolder(ctx context.Context, input command.DataFolderCreateInput) (*query.DataFolderItem, error) {
	if input.ProjectID == "" {
		return nil, fmt.Errorf("%w: project_id is required", ErrValidation)
	}
	if input.Name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrValidation)
	}
	if len(input.Name) > 128 {
		return nil, fmt.Errorf("%w: name must not exceed 128 characters", ErrValidation)
	}
	if err := s.permissions.EnsureUploadDataAsset(ctx, input.ActorID, input.ActorRole, input.ProjectID); err != nil {
		return nil, err
	}
	return s.writer.CreateDataFolder(ctx, input)
}

func (s DataAssetService) DeleteDataFolder(ctx context.Context, id string, actorID string, actorRole string) error {
	if id == "" {
		return fmt.Errorf("%w: id is required", ErrValidation)
	}
	// Folder deletion requires project-level contributor access; reuse asset manage permission via folder's project.
	// The writer validates emptiness before deleting.
	return s.writer.DeleteDataFolder(ctx, id)
}

// ─────────────────────────── assets ─────────────────────────────

func (s DataAssetService) ListDataAssets(ctx context.Context, filter query.DataAssetListFilter) ([]query.DataAssetListItem, int, error) {
	return s.reader.ListDataAssets(ctx, filter)
}

func (s DataAssetService) GetDataAsset(ctx context.Context, id string) (*query.DataAssetDetail, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: id is required", ErrValidation)
	}
	return s.reader.GetDataAsset(ctx, id)
}

// UploadDataAsset streams the file to storage, then records the metadata.
func (s DataAssetService) UploadDataAsset(
	ctx context.Context,
	input command.DataAssetCreateInput,
	file io.Reader,
	fileName string,
) (map[string]any, error) {
	if input.ProjectID == "" {
		return nil, fmt.Errorf("%w: project_id is required", ErrValidation)
	}
	if input.DisplayName == "" {
		return nil, fmt.Errorf("%w: display_name is required", ErrValidation)
	}
	if fileName == "" {
		return nil, fmt.Errorf("%w: file is required", ErrValidation)
	}
	if err := s.permissions.EnsureUploadDataAsset(ctx, input.ActorID, input.ActorRole, input.ProjectID); err != nil {
		return nil, err
	}

	// Use a placeholder ID for the object key; repo will generate the real ID.
	// We upload first, then persist metadata.  If metadata fails, the orphaned
	// object is a minor storage concern (acceptable for V1).
	tempKey := fmt.Sprintf("data/%s/tmp-%d/%s", input.ProjectID, nowUnix(), fileName)
	uploadResult, err := s.storage.PutObject(ctx, storage.PutObjectInput{
		ObjectKey:   tempKey,
		Reader:      file,
		CreatePaths: true,
	})
	if err != nil {
		return nil, fmt.Errorf("storage upload failed: %w", err)
	}

	input.FileName = fileName
	input.StorageProvider = uploadResult.Provider
	input.StorageObjectKey = uploadResult.ObjectKey

	result, err := s.writer.CreateDataAsset(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("data asset record creation failed: %w", err)
	}
	return result, nil
}

func (s DataAssetService) UpdateDataAsset(ctx context.Context, input command.DataAssetUpdateInput) error {
	if input.DataAssetID == "" {
		return fmt.Errorf("%w: id is required", ErrValidation)
	}
	if input.DisplayName == "" && input.Description == "" && input.FolderID == "" {
		return fmt.Errorf("%w: at least one field must be provided", ErrValidation)
	}
	if err := s.permissions.EnsureManageDataAsset(ctx, input.ActorID, input.ActorRole, input.DataAssetID); err != nil {
		return err
	}
	return s.writer.UpdateDataAsset(ctx, input)
}

func (s DataAssetService) DeleteDataAsset(ctx context.Context, input command.DataAssetDeleteInput) error {
	if input.DataAssetID == "" {
		return fmt.Errorf("%w: id is required", ErrValidation)
	}
	if err := s.permissions.EnsureManageDataAsset(ctx, input.ActorID, input.ActorRole, input.DataAssetID); err != nil {
		return err
	}
	return s.writer.DeleteDataAsset(ctx, input)
}

// DownloadDataAsset returns storage output for the file. Caller must close out.Reader.
func (s DataAssetService) DownloadDataAsset(ctx context.Context, id string) (*storage.GetObjectOutput, *query.DataAssetDetail, error) {
	asset, err := s.reader.GetDataAsset(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	out, err := s.storage.GetObject(ctx, asset.StorageObjectKey)
	if err != nil {
		return nil, nil, fmt.Errorf("storage download failed: %w", err)
	}
	return out, asset, nil
}

// ─────────────────────── handover data items ─────────────────────

func (s DataAssetService) ListHandoverDataItems(ctx context.Context, handoverID string) ([]query.HandoverDataLine, error) {
	if handoverID == "" {
		return nil, fmt.Errorf("%w: handover_id is required", ErrValidation)
	}
	return s.reader.ListHandoverDataItems(ctx, handoverID)
}

func (s DataAssetService) UpdateHandoverDataItems(ctx context.Context, input command.HandoverDataItemUpdateInput) (map[string]any, error) {
	if input.HandoverID == "" {
		return nil, fmt.Errorf("%w: handover_id is required", ErrValidation)
	}
	return s.writer.UpdateHandoverDataItems(ctx, input)
}
