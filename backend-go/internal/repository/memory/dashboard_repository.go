package memory

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/query"
)

type DashboardRepository struct{}

func NewDashboardRepository() DashboardRepository {
	return DashboardRepository{}
}

func (r DashboardRepository) GetOverview(ctx context.Context, projectID string, scope query.ActorScope) (query.DashboardOverview, error) {
	_ = ctx
	_ = projectID
	_ = scope

	return query.DashboardOverview{
		DocumentTotal:        0,
		StatusCounts:         map[string]int{},
		HandoverPendingCount: 0,
		RiskDocumentCount:    0,
	}, nil
}

func (r DashboardRepository) ListRecentFlows(
	ctx context.Context,
	projectID string,
	scope query.ActorScope,
) ([]query.RecentFlowItem, error) {
	_ = ctx
	_ = projectID
	_ = scope

	return []query.RecentFlowItem{}, nil
}

func (r DashboardRepository) ListRiskDocuments(
	ctx context.Context,
	projectID string,
	scope query.ActorScope,
) ([]query.RiskDocumentItem, error) {
	_ = ctx
	_ = projectID
	_ = scope

	return []query.RiskDocumentItem{}, nil
}
