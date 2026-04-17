package postgres

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/query"
)

type UserQueryRepository struct {
	db DBTX
}

func NewUserQueryRepository(db DBTX) UserQueryRepository {
	return UserQueryRepository{db: db}
}

func (r UserQueryRepository) ListUsers(ctx context.Context) ([]query.UserOption, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT
			id::text,
			username,
			display_name,
			role::text,
			COALESCE(email, ''),
			COALESCE(phone, ''),
			COALESCE(wechat, ''),
			status
		FROM users
		WHERE status = 'active'
		ORDER BY
			CASE role::text
				WHEN 'admin' THEN 1
				WHEN 'project_lead' THEN 2
				ELSE 3
			END,
			display_name ASC
		`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]query.UserOption, 0)
	for rows.Next() {
		var item query.UserOption
		if err := rows.Scan(
			&item.ID,
			&item.Username,
			&item.DisplayName,
			&item.Role,
			&item.Email,
			&item.Phone,
			&item.Wechat,
			&item.Status,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}
