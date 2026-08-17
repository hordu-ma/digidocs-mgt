package service

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/repository"
)

type DashboardQueryService struct {
	dashboard repository.DashboardReader
}

func NewDashboardQueryService(dashboard repository.DashboardReader) DashboardQueryService {
	return DashboardQueryService{dashboard: dashboard}
}

// The dashboard aggregates across every document the actor can reach. Without a
// scope it reported platform-wide totals to every signed-in user, which was both
// a data leak and simply the wrong number for a member of a single project.

func (s DashboardQueryService) Overview(
	ctx context.Context,
	projectID string,
	scope query.ActorScope,
) (query.DashboardOverview, error) {
	return s.dashboard.GetOverview(ctx, projectID, scope)
}

func (s DashboardQueryService) RecentFlows(
	ctx context.Context,
	projectID string,
	scope query.ActorScope,
) ([]query.RecentFlowItem, error) {
	return s.dashboard.ListRecentFlows(ctx, projectID, scope)
}

func (s DashboardQueryService) RiskDocuments(
	ctx context.Context,
	projectID string,
	scope query.ActorScope,
) ([]query.RiskDocumentItem, error) {
	return s.dashboard.ListRiskDocuments(ctx, projectID, scope)
}
