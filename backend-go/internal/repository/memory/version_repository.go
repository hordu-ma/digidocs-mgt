package memory

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/query"
)

type VersionRepository struct{}

func NewVersionRepository() VersionRepository {
	return VersionRepository{}
}

func (r VersionRepository) ListVersions(ctx context.Context, documentID string) ([]query.VersionItem, error) {
	_ = ctx
	_ = documentID

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

func (r VersionRepository) GetVersion(ctx context.Context, versionID string) (*query.VersionDetail, error) {
	_ = ctx

	return &query.VersionDetail{
		ID:               versionID,
		PreviewType:      "pdf",
		WatermarkEnabled: true,
	}, nil
}
