package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
)

type VersionRepository struct {
	mu         sync.RWMutex
	seq        int
	byID       map[string]query.VersionDetail
	byDocument map[string][]query.VersionItem
}

func NewVersionRepository() *VersionRepository {
	repo := &VersionRepository{
		seq:        200,
		byID:       make(map[string]query.VersionDetail),
		byDocument: make(map[string][]query.VersionItem),
	}
	repo.seedDefault()
	return repo
}

func (r *VersionRepository) ListVersions(ctx context.Context, documentID string) ([]query.VersionItem, error) {
	_ = ctx

	r.mu.RLock()
	items, ok := r.byDocument[documentID]
	r.mu.RUnlock()
	if ok && len(items) > 0 {
		cloned := make([]query.VersionItem, len(items))
		copy(cloned, items)
		return cloned, nil
	}

	return []query.VersionItem{
		{
			ID:            "00000000-0000-0000-0000-000000000200",
			VersionNo:     1,
			FileName:      "课题申报书.docx",
			SummaryStatus: "pending",
			CreatedAt:     "2026-04-04T00:00:00Z",
		},
	}, nil
}

func (r *VersionRepository) GetVersion(ctx context.Context, versionID string) (*query.VersionDetail, error) {
	_ = ctx

	r.mu.RLock()
	item, ok := r.byID[versionID]
	r.mu.RUnlock()
	if ok {
		cloned := item
		return &cloned, nil
	}

	return &query.VersionDetail{
		ID:               versionID,
		FileName:         "sample.docx",
		StorageProvider:  "memory",
		StorageObjectKey: "documents/sample/" + versionID,
		PreviewType:      "pdf",
		WatermarkEnabled: true,
	}, nil
}

func (r *VersionRepository) createUploadedVersion(_ context.Context, input command.VersionCreateInput) (map[string]any, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.seq++
	versionID := fmt.Sprintf("mem-version-%03d", r.seq)
	versionNo := len(r.byDocument[input.DocumentID]) + 1
	createdAt := time.Now().UTC().Format(time.RFC3339)

	detail := query.VersionDetail{
		ID:               versionID,
		DocumentID:       input.DocumentID,
		VersionNo:        versionNo,
		CommitMessage:    input.CommitMessage,
		FileName:         input.FileName,
		FileSize:         input.FileSize,
		StorageProvider:  input.StorageProvider,
		StorageObjectKey: input.StorageObjectKey,
		PreviewType:      "pdf",
		WatermarkEnabled: true,
	}
	r.byID[versionID] = detail
	r.byDocument[input.DocumentID] = append(r.byDocument[input.DocumentID], query.VersionItem{
		ID:            versionID,
		VersionNo:     versionNo,
		FileName:      input.FileName,
		SummaryStatus: "pending",
		CreatedAt:     createdAt,
	})

	return map[string]any{
		"id":             versionID,
		"document_id":    input.DocumentID,
		"version_no":     versionNo,
		"commit_message": input.CommitMessage,
		"file_name":      input.FileName,
		"current_status": "in_progress",
	}, nil
}

func (r *VersionRepository) seedDefault() {
	r.byID["00000000-0000-0000-0000-000000000200"] = query.VersionDetail{
		ID:               "00000000-0000-0000-0000-000000000200",
		DocumentID:       "00000000-0000-0000-0000-000000000100",
		VersionNo:        1,
		CommitMessage:    "initial",
		FileName:         "课题申报书.docx",
		StorageProvider:  "memory",
		StorageObjectKey: "documents/sample/00000000-0000-0000-0000-000000000200",
		PreviewType:      "pdf",
		WatermarkEnabled: true,
	}
	r.byDocument["00000000-0000-0000-0000-000000000100"] = []query.VersionItem{
		{
			ID:            "00000000-0000-0000-0000-000000000200",
			VersionNo:     1,
			FileName:      "课题申报书.docx",
			SummaryStatus: "pending",
			CreatedAt:     "2026-04-04T00:00:00Z",
		},
	}
}
