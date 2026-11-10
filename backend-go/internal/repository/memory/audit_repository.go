package memory

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/command"
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

func (r AuditRepository) GetAuditSummary(ctx context.Context, projectID string, scope query.ActorScope) (query.AuditSummary, error) {
	_ = ctx
	_ = scope

	return query.AuditSummary{
		ProjectID:      projectID,
		DownloadCount:  0,
		UploadCount:    0,
		TransferCount:  0,
		ArchiveCount:   0,
		TopActiveUsers: []query.AuditUserMetric{},
	}, nil
}

// RecordAuditEvent is a no-op in memory mode; the in-memory audit log is
// always empty (see ListAuditEvents), matching the rest of this repository.
func (r AuditRepository) RecordAuditEvent(ctx context.Context, event command.AuditEventInput) error {
	_ = ctx
	_ = event
	return nil
}
