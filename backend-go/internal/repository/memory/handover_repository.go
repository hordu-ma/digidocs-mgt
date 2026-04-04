package memory

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/query"
)

type HandoverRepository struct{}

func NewHandoverRepository() HandoverRepository {
	return HandoverRepository{}
}

func (r HandoverRepository) ListHandovers(ctx context.Context) ([]query.HandoverItem, error) {
	_ = ctx

	return []query.HandoverItem{
		{
			ID:             "00000000-0000-0000-0000-000000000300",
			TargetUserID:   "00000000-0000-0000-0000-000000000001",
			ReceiverUserID: "00000000-0000-0000-0000-000000000002",
			Status:         "generated",
			Remark:         "示例交接单",
		},
	}, nil
}

func (r HandoverRepository) GetHandover(ctx context.Context, handoverID string) (*query.HandoverItem, error) {
	_ = ctx

	return &query.HandoverItem{
		ID:             handoverID,
		TargetUserID:   "00000000-0000-0000-0000-000000000001",
		ReceiverUserID: "00000000-0000-0000-0000-000000000002",
		Status:         "generated",
		Items: []query.HandoverLine{
			{
				DocumentID: "00000000-0000-0000-0000-000000000100",
				Selected:   true,
			},
		},
	}, nil
}
