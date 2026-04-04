package memory

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/query"
)

type FlowRepository struct{}

func NewFlowRepository() FlowRepository {
	return FlowRepository{}
}

func (r FlowRepository) ListFlows(ctx context.Context, documentID string) ([]query.FlowItem, error) {
	_ = ctx
	_ = documentID

	return []query.FlowItem{
		{
			ID:        "00000000-0000-0000-0000-000000000500",
			Action:    "transfer",
			ToStatus:  "in_progress",
			CreatedAt: "2026-04-04T00:00:00Z",
		},
	}, nil
}
