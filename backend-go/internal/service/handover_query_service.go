package service

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/repository"
)

type HandoverQueryService struct {
	handovers repository.HandoverReader
}

func NewHandoverQueryService(handovers repository.HandoverReader) HandoverQueryService {
	return HandoverQueryService{handovers: handovers}
}

func (s HandoverQueryService) List(ctx context.Context) ([]query.HandoverItem, error) {
	return s.handovers.ListHandovers(ctx)
}

func (s HandoverQueryService) Get(ctx context.Context, handoverID string) (*query.HandoverItem, error) {
	return s.handovers.GetHandover(ctx, handoverID)
}
