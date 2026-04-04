package memory

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/query"
)

type AuditRepository struct{}

func NewAuditRepository() AuditRepository {
	return AuditRepository{}
}

func (r AuditRepository) ListAuditEvents(
	ctx context.Context,
	filter query.AuditEventFilter,
) ([]query.AuditEventItem, int, error) {
	_ = ctx
	_ = filter

	return []query.AuditEventItem{}, 0, nil
}

func (r AuditRepository) GetAuditSummary(ctx context.Context, projectID string) (query.AuditSummary, error) {
	_ = ctx

	return query.AuditSummary{
		ProjectID:      projectID,
		DownloadCount:  0,
		UploadCount:    0,
		TransferCount:  0,
		ArchiveCount:   0,
		TopActiveUsers: []query.AuditUserMetric{},
	}, nil
}
