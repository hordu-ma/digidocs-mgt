package service

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/repository"
)

type AuditQueryService struct {
	audits repository.AuditReader
}

func NewAuditQueryService(audits repository.AuditReader) AuditQueryService {
	return AuditQueryService{audits: audits}
}

func (s AuditQueryService) List(
	ctx context.Context,
	filter query.AuditEventFilter,
) ([]query.AuditEventItem, int, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	return s.audits.ListAuditEvents(ctx, filter)
}

func (s AuditQueryService) Summary(ctx context.Context, projectID string) (query.AuditSummary, error) {
	return s.audits.GetAuditSummary(ctx, projectID)
}
