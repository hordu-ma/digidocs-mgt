package postgres

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/query"
)

type TeamSpaceRepository struct {
	db DBTX
}

func NewTeamSpaceRepository(db DBTX) TeamSpaceRepository {
	return TeamSpaceRepository{db: db}
}

func (r TeamSpaceRepository) ListTeamSpaces(ctx context.Context) ([]query.TeamSpaceSummary, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id::text, name, code FROM team_spaces ORDER BY name ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]query.TeamSpaceSummary, 0)
	for rows.Next() {
		var item query.TeamSpaceSummary
		if err := rows.Scan(&item.ID, &item.Name, &item.Code); err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, rows.Err()
}
