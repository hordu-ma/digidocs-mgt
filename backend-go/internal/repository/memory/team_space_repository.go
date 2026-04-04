package memory

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/query"
)

type TeamSpaceRepository struct{}

func NewTeamSpaceRepository() TeamSpaceRepository {
	return TeamSpaceRepository{}
}

func (r TeamSpaceRepository) ListTeamSpaces(ctx context.Context) ([]query.TeamSpaceSummary, error) {
	_ = ctx

	return []query.TeamSpaceSummary{
		{
			ID:   "00000000-0000-0000-0000-000000000010",
			Name: "随机控制实验室",
			Code: "lab-rc",
		},
	}, nil
}
