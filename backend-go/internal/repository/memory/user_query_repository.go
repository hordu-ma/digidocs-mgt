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
			Username:    "admin",
			DisplayName: "开发管理员",
			Role:        "admin",
			Status:      "active",
		},
		{
			ID:          "00000000-0000-0000-0000-000000000010",
			Username:    "maliguo",
			DisplayName: "马立国",
			Role:        "project_lead",
			Status:      "active",
		},
		{
			ID:          "00000000-0000-0000-0000-000000000011",
			Username:    "qiaoanqiang",
			DisplayName: "乔安强",
			Role:        "member",
			Status:      "active",
		},
		{
			ID:          "00000000-0000-0000-0000-000000000012",
			Username:    "wangzhao",
			DisplayName: "王钊",
			Role:        "member",
			Status:      "active",
		},
		{
			ID:          "00000000-0000-0000-0000-000000000013",
			Username:    "liuzongyou",
			DisplayName: "刘宗优",
			Role:        "member",
			Status:      "active",
		},
	}, nil
}
