package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/service"
)

type dataFolderRecord struct {
	id        string
	projectID string
	parentID  string
	depth     int
	name      string
	createdAt time.Time
}

type dataAssetRecord struct {
	id               string
	teamSpaceID      string
	projectID        string
	folderID         string
	displayName      string
	fileName         string
	description      string
	mimeType         string
	fileSize         int64
	storageProvider  string
	storageObjectKey string
	createdBy        string
	createdAt        time.Time
	updatedAt        time.Time
	isDeleted        bool
}

type handoverDataItemRecord struct {
	id          string
	handoverID  string
	dataAssetID string
	selected    bool
	note        string
}

type DataAssetRepository struct {
	mu      sync.RWMutex
	folders []dataFolderRecord
	assets  []dataAssetRecord
	items   []handoverDataItemRecord
	seq     int
}

func NewDataAssetRepository() *DataAssetRepository {
	return &DataAssetRepository{}
}

func (r *DataAssetRepository) nextID() string {
	r.seq++
	return fmt.Sprintf("mem-da-%d", r.seq)
}

// ─────────────────────────── folders ────────────────────────────

func (r *DataAssetRepository) ListDataFolders(_ context.Context, projectID string) ([]query.DataFolderItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]query.DataFolderItem, 0)
	for _, f := range r.folders {
		if f.projectID == projectID {
			items = append(items, query.DataFolderItem{
				ID:        f.id,
				ProjectID: f.projectID,
				ParentID:  f.parentID,
				Depth:     f.depth,
				Name:      f.name,
				CreatedAt: f.createdAt.Format(time.RFC3339),
			})
		}
	}
	return items, nil
}

func (r *DataAssetRepository) CreateDataFolder(_ context.Context, input command.DataFolderCreateInput) (*query.DataFolderItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	depth := 0
	if input.ParentID != "" {
		found := false
		for _, f := range r.folders {
			if f.id == input.ParentID {
				depth = f.depth + 1
				found = true
				break
			}
		}
		if !found {
			return nil, service.ErrNotFound
		}
		if depth > 2 {
			return nil, fmt.Errorf("%w: max folder depth is 2", service.ErrValidation)
		}
	}

	item := dataFolderRecord{
		id:        r.nextID(),
		projectID: input.ProjectID,
		parentID:  input.ParentID,
		depth:     depth,
		name:      input.Name,
		createdAt: time.Now(),
	}
	r.folders = append(r.folders, item)

	return &query.DataFolderItem{
		ID:        item.id,
		ProjectID: item.projectID,
		ParentID:  item.parentID,
		Depth:     item.depth,
		Name:      item.name,
		CreatedAt: item.createdAt.Format(time.RFC3339),
	}, nil
}

func (r *DataAssetRepository) DeleteDataFolder(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, f := range r.folders {
		if f.id == id {
			r.folders = append(r.folders[:i], r.folders[i+1:]...)
			return nil
		}
	}
	return service.ErrNotFound
}

// ─────────────────────────── assets ─────────────────────────────

func (r *DataAssetRepository) ListDataAssets(_ context.Context, filter query.DataAssetListFilter) ([]query.DataAssetListItem, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]query.DataAssetListItem, 0)
	for _, a := range r.assets {
		if a.isDeleted {
			continue
		}
		if filter.ProjectID != "" && a.projectID != filter.ProjectID {
			continue
		}
		if filter.FolderID != "" && a.folderID != filter.FolderID {
			continue
		}
		items = append(items, query.DataAssetListItem{
			ID:          a.id,
			ProjectID:   a.projectID,
			FolderID:    a.folderID,
			DisplayName: a.displayName,
			FileName:    a.fileName,
			MimeType:    a.mimeType,
			FileSize:    a.fileSize,
			CreatedAt:   a.createdAt.Format(time.RFC3339),
		})
	}
	return items, len(items), nil
}

func (r *DataAssetRepository) GetDataAsset(_ context.Context, id string) (*query.DataAssetDetail, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, a := range r.assets {
		if a.id == id && !a.isDeleted {
			return &query.DataAssetDetail{
				ID:               a.id,
				TeamSpaceID:      a.teamSpaceID,
				ProjectID:        a.projectID,
				FolderID:         a.folderID,
				DisplayName:      a.displayName,
				FileName:         a.fileName,
				Description:      a.description,
				MimeType:         a.mimeType,
				FileSize:         a.fileSize,
				StorageProvider:  a.storageProvider,
				StorageObjectKey: a.storageObjectKey,
				CreatedAt:        a.createdAt.Format(time.RFC3339),
				UpdatedAt:        a.updatedAt.Format(time.RFC3339),
			}, nil
		}
	}
	return nil, service.ErrNotFound
}

func (r *DataAssetRepository) CreateDataAsset(_ context.Context, input command.DataAssetCreateInput) (map[string]any, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	item := dataAssetRecord{
		id:               r.nextID(),
		teamSpaceID:      input.TeamSpaceID,
		projectID:        input.ProjectID,
		folderID:         input.FolderID,
		displayName:      input.DisplayName,
		fileName:         input.FileName,
		description:      input.Description,
		mimeType:         input.MimeType,
		fileSize:         input.FileSize,
		storageProvider:  input.StorageProvider,
		storageObjectKey: input.StorageObjectKey,
		createdBy:        input.ActorID,
		createdAt:        now,
		updatedAt:        now,
	}
	r.assets = append(r.assets, item)
	return map[string]any{
		"id":           item.id,
		"project_id":   item.projectID,
		"display_name": item.displayName,
		"file_name":    item.fileName,
		"file_size":    item.fileSize,
		"created_at":   item.createdAt.Format(time.RFC3339),
	}, nil
}

func (r *DataAssetRepository) UpdateDataAsset(_ context.Context, input command.DataAssetUpdateInput) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, a := range r.assets {
		if a.id == input.DataAssetID && !a.isDeleted {
			r.assets[i].displayName = input.DisplayName
			r.assets[i].description = input.Description
			r.assets[i].folderID = input.FolderID
			r.assets[i].updatedAt = time.Now()
			return nil
		}
	}
	return service.ErrNotFound
}

func (r *DataAssetRepository) DeleteDataAsset(_ context.Context, input command.DataAssetDeleteInput) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, a := range r.assets {
		if a.id == input.DataAssetID && !a.isDeleted {
			r.assets[i].isDeleted = true
			r.assets[i].updatedAt = time.Now()
			return nil
		}
	}
	return service.ErrNotFound
}

// ─────────────────────── handover data items ─────────────────────

func (r *DataAssetRepository) ListHandoverDataItems(_ context.Context, handoverID string) ([]query.HandoverDataLine, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]query.HandoverDataLine, 0)
	for _, it := range r.items {
		if it.handoverID == handoverID {
			items = append(items, query.HandoverDataLine{
				DataAssetID: it.dataAssetID,
				Selected:    it.selected,
				Note:        it.note,
			})
		}
	}
	return items, nil
}

func (r *DataAssetRepository) UpdateHandoverDataItems(_ context.Context, input command.HandoverDataItemUpdateInput) (map[string]any, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	filtered := make([]handoverDataItemRecord, 0)
	for _, it := range r.items {
		if it.handoverID != input.HandoverID {
			filtered = append(filtered, it)
		}
	}

	for _, item := range input.Items {
		filtered = append(filtered, handoverDataItemRecord{
			id:          r.nextID(),
			handoverID:  input.HandoverID,
			dataAssetID: item.DataAssetID,
			selected:    item.Selected,
			note:        item.Note,
		})
	}
	r.items = filtered

	return map[string]any{"handover_id": input.HandoverID, "count": len(input.Items)}, nil
}
