package postgres

import (
	"context"
	"strconv"

	"digidocs-mgt/backend-go/internal/domain/query"
)

type DashboardRepository struct {
	db DBTX
}

func NewDashboardRepository(db DBTX) DashboardRepository {
	return DashboardRepository{db: db}
}

func (r DashboardRepository) GetOverview(ctx context.Context, projectID string, scope query.ActorScope) (query.DashboardOverview, error) {
	isAdmin, actorID := scopeArgs(scope)
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
		  AND `+projectScopeSQL("d.project_id", "$2", "$3")+`
		  AND d.is_deleted = false
		`,
		projectID, isAdmin, actorID,
	).Scan(&overview.DocumentTotal); err != nil {
		return query.DashboardOverview{}, err
	}

	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT d.current_status::text, COUNT(*)::int
		FROM documents d
		WHERE ($1 = '' OR d.project_id::text = $1)
		  AND `+projectScopeSQL("d.project_id", "$2", "$3")+`
		  AND d.is_deleted = false
		GROUP BY d.current_status
		`,
		projectID, isAdmin, actorID,
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
		  AND `+projectScopeSQL("gh.project_id", "$2", "$3")+`
		  AND gh.status IN ('generated'::handover_status, 'pending_confirm'::handover_status)
		`,
		projectID, isAdmin, actorID,
	).Scan(&overview.HandoverPendingCount); err != nil {
		return query.DashboardOverview{}, err
	}

	if err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM (`+riskItemsSQL+`) risks`,
		projectID, isAdmin, actorID,
	).Scan(&overview.RiskDocumentCount); err != nil {
		return query.DashboardOverview{}, err
	}

	return overview, nil
}

func (r DashboardRepository) ListRecentFlows(
	ctx context.Context,
	projectID string,
	scope query.ActorScope,
) ([]query.RecentFlowItem, error) {
	isAdmin, actorID := scopeArgs(scope)
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
		  AND `+projectScopeSQL("d.project_id", "$2", "$3")+`
		  AND d.is_deleted = false
		ORDER BY fr.created_at DESC
		LIMIT 20
		`,
		projectID, isAdmin, actorID,
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
	scope query.ActorScope,
) ([]query.RiskDocumentItem, error) {
	isAdmin, actorID := scopeArgs(scope)
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT document_id, title, risk_type, risk_message, related_type, related_id, link
		FROM (`+riskItemsSQL+`) risks
		ORDER BY severity, since ASC
		LIMIT 50`,
		projectID, isAdmin, actorID,
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
			&item.RelatedType,
			&item.RelatedID,
			&item.Link,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

// Risk thresholds (days). Kept as constants rather than configuration: they
// are part of the product definition in docs/项目定义与技术架构.md §8.
const (
	riskStaleDays              = 30
	riskTransferUnacceptedDays = 7
	riskHandoverOverdueDays    = 14
)

// riskItemsSQL is the single definition of "risk" shared by the overview count
// and the risk list, so the KPI number always equals the list length (before
// the list's LIMIT). Each document appears at most once, under its most
// severe rule. Bind params: $1 project id (” = all), $2 is admin, $3 actor.
var riskItemsSQL = `
	SELECT
		d.id::text AS document_id,
		d.title,
		CASE
			WHEN u.status <> 'active' THEN 'owner_inactive'
			WHEN d.current_status = 'pending_handover' THEN 'transfer_unaccepted'
			ELSE 'stale'
		END AS risk_type,
		CASE
			WHEN u.status <> 'active' THEN '责任人「' || u.display_name || '」已停用，需要重新指定'
			WHEN d.current_status = 'pending_handover' THEN '转交后 ' || EXTRACT(DAY FROM NOW() - d.updated_at)::int || ' 天未被接收'
			ELSE EXTRACT(DAY FROM NOW() - d.updated_at)::int || ' 天未更新'
		END AS risk_message,
		'document' AS related_type,
		d.id::text AS related_id,
		'/documents/' || d.id::text AS link,
		CASE WHEN u.status <> 'active' THEN 1 WHEN d.current_status = 'pending_handover' THEN 2 ELSE 4 END AS severity,
		d.updated_at AS since
	FROM documents d
	JOIN users u ON u.id = d.current_owner_id
	WHERE ($1 = '' OR d.project_id::text = $1)
	  AND ` + projectScopeSQL("d.project_id", "$2", "$3") + `
	  AND d.is_deleted = false
	  AND d.current_status NOT IN ('archived'::document_status, 'finalized'::document_status)
	  AND (
		u.status <> 'active'
		OR (d.current_status = 'pending_handover' AND d.updated_at < NOW() - make_interval(days => ` + itoa(riskTransferUnacceptedDays) + `))
		OR d.updated_at < NOW() - make_interval(days => ` + itoa(riskStaleDays) + `)
	  )

	UNION ALL

	SELECT
		'' AS document_id,
		tu.display_name || ' → ' || ru.display_name || ' 的交接',
		'handover_overdue',
		'交接单创建 ' || EXTRACT(DAY FROM NOW() - gh.generated_at)::int || ' 天仍未完成',
		'handover',
		gh.id::text,
		'/handovers?id=' || gh.id::text,
		3,
		gh.generated_at
	FROM graduation_handovers gh
	JOIN users tu ON tu.id = gh.target_user_id
	JOIN users ru ON ru.id = gh.receiver_user_id
	WHERE ($1 = '' OR gh.project_id::text = $1)
	  AND ` + projectScopeSQL("gh.project_id", "$2", "$3") + `
	  AND gh.status IN ('generated'::handover_status, 'pending_confirm'::handover_status)
	  AND gh.generated_at < NOW() - make_interval(days => ` + itoa(riskHandoverOverdueDays) + `)
`

func itoa(n int) string { return strconv.Itoa(n) }
