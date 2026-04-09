package memory

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/query"
)

type UserQueryRepository struct{}

func NewUserQueryRepository() UserQueryRepository {
	return UserQueryRepository{}
}

func (r UserQueryRepository) ListUsers(ctx context.Context) ([]query.UserOption, error) {
	_ = ctx

	return []query.UserOption{
		{
			ID:          "00000000-0000-0000-0000-000000000001",
			DisplayName: "开发管理员",
			Role:        "admin",
		},
		{
			ID:          "00000000-0000-0000-0000-000000000010",
			DisplayName: "李老师",
			Role:        "project_lead",
		},
		{
			ID:          "00000000-0000-0000-0000-000000000011",
			DisplayName: "张三",
			Role:        "member",
		},
		{
			ID:          "00000000-0000-0000-0000-000000000012",
			DisplayName: "王五",
			Role:        "member",
		},
	}, nil
}
