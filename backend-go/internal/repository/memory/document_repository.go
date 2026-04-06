package memory

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
)

type DocumentRepository struct{}

func NewDocumentRepository() DocumentRepository {
	return DocumentRepository{}
}

func (r DocumentRepository) ListDocuments(
	ctx context.Context,
	filter query.DocumentListFilter,
) ([]query.DocumentListItem, int, error) {
	_ = ctx

	page := filter.Page
	if page <= 0 {
		page = 1
	}

	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	versionNo := 1
	updatedAt := "2026-04-04T00:00:00Z"

	return []query.DocumentListItem{
		{
			ID:            "00000000-0000-0000-0000-000000000100",
			Title:         "课题申报书",
			ProjectName:   "课题A",
			FolderPath:    "/申报材料",
			CurrentStatus: "draft",
			CurrentOwner: &query.UserSummary{
				ID:          "00000000-0000-0000-0000-000000000001",
				DisplayName: "张三",
			},
			CurrentVersionNo: &versionNo,
			UpdatedAt:        &updatedAt,
		},
	}, 1, nil
}

func (r DocumentRepository) GetDocument(ctx context.Context, documentID string) (*query.DocumentDetail, error) {
	_ = ctx

	return &query.DocumentDetail{
		ID:          documentID,
		Title:       "课题申报书",
		Description: "Go 迁移阶段的内存仓储占位数据",
		CurrentStatus: "draft",
		CurrentOwner: &query.UserSummary{
			ID:          "00000000-0000-0000-0000-000000000001",
			DisplayName: "系统管理员",
		},
		IsArchived: false,
	}, nil
}

func (r DocumentRepository) CreateDocument(_ context.Context, input command.DocumentCreateInput) (map[string]any, error) {
	return map[string]any{
		"id":             "00000000-0000-0000-0000-000000000200",
		"title":          input.Title,
		"current_status": "draft",
	}, nil
}
