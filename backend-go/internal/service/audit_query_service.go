package service

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/repository"
)

type AuditQueryService struct {
	audits      repository.AuditReader
	permissions PermissionService
}

// WithPermissions enables the per-document widening in List (see there).
func (s AuditQueryService) WithPermissions(p PermissionService) AuditQueryService {
	s.permissions = p
	return s
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

	// A document's access history is part of that document: anyone allowed to
	// open it (including a current owner who is not a member of its project,
	// e.g. after a cross-project transfer) may see it. The query stays pinned
	// to that one document id, so widening the project scope cannot leak rows
	// from anywhere else.
	if filter.DocumentID != "" && !filter.Scope.IsAdmin && s.permissions.reader != nil {
		role := ""
		if err := s.permissions.EnsureViewDocument(ctx, filter.Scope.ActorID, role, filter.DocumentID); err == nil {
			filter.Scope = query.ActorScope{ActorID: filter.Scope.ActorID, IsAdmin: true}
		}
	}

	return s.audits.ListAuditEvents(ctx, filter)
}

func (s AuditQueryService) Summary(
	ctx context.Context,
	projectID string,
	scope query.ActorScope,
) (query.AuditSummary, error) {
	return s.audits.GetAuditSummary(ctx, projectID, scope)
}
