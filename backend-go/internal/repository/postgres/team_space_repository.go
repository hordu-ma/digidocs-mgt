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

func (r TeamSpaceRepository) ListTeamSpaces(ctx context.Context, actorID, actorRole string) ([]query.TeamSpaceSummary, error) {
	var q string
	var args []any
	if actorRole == "admin" {
		q = `SELECT id::text, name, code FROM team_spaces ORDER BY name ASC`
	} else {
		q = `SELECT DISTINCT ts.id::text, ts.name, ts.code
			FROM team_spaces ts
			JOIN projects p ON p.team_space_id = ts.id
			JOIN project_members pm ON pm.project_id = p.id
			WHERE pm.user_id = $1::uuid
			ORDER BY ts.name ASC`
		args = append(args, actorID)
	}
	rows, err := r.db.QueryContext(ctx, q, args...)
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
