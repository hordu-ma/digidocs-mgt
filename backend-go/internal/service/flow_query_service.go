package service

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/repository"
)

type FlowQueryService struct {
	flows repository.FlowReader
}

func NewFlowQueryService(flows repository.FlowReader) FlowQueryService {
	return FlowQueryService{flows: flows}
}

func (s FlowQueryService) List(ctx context.Context, documentID string) ([]query.FlowItem, error) {
	return s.flows.ListFlows(ctx, documentID)
}
