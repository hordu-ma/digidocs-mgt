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

func (s DashboardQueryService) Overview(ctx context.Context, projectID string) (query.DashboardOverview, error) {
	return s.dashboard.GetOverview(ctx, projectID)
}

func (s DashboardQueryService) RecentFlows(ctx context.Context, projectID string) ([]query.RecentFlowItem, error) {
	return s.dashboard.ListRecentFlows(ctx, projectID)
}

func (s DashboardQueryService) RiskDocuments(ctx context.Context, projectID string) ([]query.RiskDocumentItem, error) {
	return s.dashboard.ListRiskDocuments(ctx, projectID)
}
