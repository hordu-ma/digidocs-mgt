package postgres

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/query"
)

type DashboardRepository struct {
	db DBTX
}

func NewDashboardRepository(db DBTX) DashboardRepository {
	return DashboardRepository{db: db}
}

func (r DashboardRepository) GetOverview(ctx context.Context, projectID string) (query.DashboardOverview, error) {
	overview := query.DashboardOverview{
		StatusCounts: map[string]int{
			"draft":            0,
			"in_progress":      0,
			"pending_handover": 0,
			"handed_over":      0,
			"finalized":        0,
			"archived":         0,
		},
	}

	if err := r.db.QueryRowContext(
		ctx,
		`
		SELECT COUNT(*)
		FROM documents d
		WHERE ($1 = '' OR d.project_id::text = $1)
		  AND d.is_deleted = false
		`,
		projectID,
	).Scan(&overview.DocumentTotal); err != nil {
		return query.DashboardOverview{}, err
	}

	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT d.current_status::text, COUNT(*)::int
		FROM documents d
		WHERE ($1 = '' OR d.project_id::text = $1)
		  AND d.is_deleted = false
		GROUP BY d.current_status
		`,
		projectID,
	)
	if err != nil {
		return query.DashboardOverview{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return query.DashboardOverview{}, err
		}
		overview.StatusCounts[status] = count
	}
	if err := rows.Err(); err != nil {
		return query.DashboardOverview{}, err
	}

	if err := r.db.QueryRowContext(
		ctx,
		`
		SELECT COUNT(*)
		FROM graduation_handovers gh
		WHERE ($1 = '' OR gh.project_id::text = $1)
		  AND gh.status = 'pending_confirm'::handover_status
		`,
		projectID,
	).Scan(&overview.HandoverPendingCount); err != nil {
		return query.DashboardOverview{}, err
	}

	if err := r.db.QueryRowContext(
		ctx,
		`
		SELECT COUNT(*)
		FROM documents d
		WHERE ($1 = '' OR d.project_id::text = $1)
		  AND d.is_deleted = false
		  AND d.current_status NOT IN ('archived'::document_status, 'finalized'::document_status)
		  AND d.updated_at < NOW() - INTERVAL '30 days'
		`,
		projectID,
	).Scan(&overview.RiskDocumentCount); err != nil {
		return query.DashboardOverview{}, err
	}

	return overview, nil
}

func (r DashboardRepository) ListRecentFlows(
	ctx context.Context,
	projectID string,
) ([]query.RecentFlowItem, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT
			fr.document_id::text,
			d.title,
			fr.action,
			COALESCE(fr.from_status::text, ''),
			fr.to_status::text,
			TO_CHAR(fr.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM flow_records fr
		JOIN documents d ON d.id = fr.document_id
		WHERE ($1 = '' OR d.project_id::text = $1)
		  AND d.is_deleted = false
		ORDER BY fr.created_at DESC
		LIMIT 20
		`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]query.RecentFlowItem, 0)
	for rows.Next() {
		var item query.RecentFlowItem
		if err := rows.Scan(
			&item.DocumentID,
			&item.Title,
			&item.Action,
			&item.FromStatus,
			&item.ToStatus,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r DashboardRepository) ListRiskDocuments(
	ctx context.Context,
	projectID string,
) ([]query.RiskDocumentItem, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT
			d.id::text,
			d.title,
			'stale',
			'超过30天未更新'
		FROM documents d
		WHERE ($1 = '' OR d.project_id::text = $1)
		  AND d.is_deleted = false
		  AND d.current_status NOT IN ('archived'::document_status, 'finalized'::document_status)
		  AND d.updated_at < NOW() - INTERVAL '30 days'
		ORDER BY d.updated_at ASC
		LIMIT 50
		`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]query.RiskDocumentItem, 0)
	for rows.Next() {
		var item query.RiskDocumentItem
		if err := rows.Scan(
			&item.DocumentID,
			&item.Title,
			&item.RiskType,
			&item.RiskMessage,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}
