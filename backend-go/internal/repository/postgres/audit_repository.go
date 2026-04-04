package postgres

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/query"
)

type AuditRepository struct {
	db DBTX
}

func NewAuditRepository(db DBTX) AuditRepository {
	return AuditRepository{db: db}
}

func (r AuditRepository) ListAuditEvents(
	ctx context.Context,
	filter query.AuditEventFilter,
) ([]query.AuditEventItem, int, error) {
	offset := (filter.Page - 1) * filter.PageSize

	var total int
	if err := r.db.QueryRowContext(
		ctx,
		`
		SELECT COUNT(*)
		FROM audit_events ae
		LEFT JOIN documents d ON d.id = ae.document_id
		WHERE ($1 = '' OR d.project_id::text = $1)
		  AND ($2 = '' OR ae.document_id::text = $2)
		  AND ($3 = '' OR ae.action_type::text = $3)
		`,
		filter.ProjectID,
		filter.DocumentID,
		filter.ActionType,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT
			ae.id::text,
			COALESCE(ae.document_id::text, ''),
			COALESCE(ae.version_id::text, ''),
			COALESCE(ae.user_id::text, ''),
			ae.action_type::text,
			TO_CHAR(ae.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM audit_events ae
		LEFT JOIN documents d ON d.id = ae.document_id
		WHERE ($1 = '' OR d.project_id::text = $1)
		  AND ($2 = '' OR ae.document_id::text = $2)
		  AND ($3 = '' OR ae.action_type::text = $3)
		ORDER BY ae.created_at DESC
		OFFSET $4 LIMIT $5
		`,
		filter.ProjectID,
		filter.DocumentID,
		filter.ActionType,
		offset,
		filter.PageSize,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]query.AuditEventItem, 0)
	for rows.Next() {
		var item query.AuditEventItem
		if err := rows.Scan(
			&item.ID,
			&item.DocumentID,
			&item.VersionID,
			&item.UserID,
			&item.ActionType,
			&item.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}

	return items, total, rows.Err()
}

func (r AuditRepository) GetAuditSummary(ctx context.Context, projectID string) (query.AuditSummary, error) {
	summary := query.AuditSummary{
		ProjectID:      projectID,
		TopActiveUsers: make([]query.AuditUserMetric, 0),
	}

	if err := r.db.QueryRowContext(
		ctx,
		`
		SELECT
			COUNT(*) FILTER (WHERE ae.action_type IN ('upload', 'replace_version')),
			COUNT(*) FILTER (WHERE ae.action_type = 'download'),
			COUNT(*) FILTER (WHERE ae.action_type IN ('transfer', 'receive_transfer')),
			COUNT(*) FILTER (WHERE ae.action_type = 'archive')
		FROM audit_events ae
		LEFT JOIN documents d ON d.id = ae.document_id
		WHERE ($1 = '' OR d.project_id::text = $1)
		`,
		projectID,
	).Scan(
		&summary.UploadCount,
		&summary.DownloadCount,
		&summary.TransferCount,
		&summary.ArchiveCount,
	); err != nil {
		return query.AuditSummary{}, err
	}

	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT
			ae.user_id::text,
			COALESCE(u.display_name, ''),
			COUNT(*)::int
		FROM audit_events ae
		LEFT JOIN documents d ON d.id = ae.document_id
		LEFT JOIN users u ON u.id = ae.user_id
		WHERE ae.user_id IS NOT NULL
		  AND ($1 = '' OR d.project_id::text = $1)
		GROUP BY ae.user_id, u.display_name
		ORDER BY COUNT(*) DESC, COALESCE(u.display_name, '') ASC
		LIMIT 5
		`,
		projectID,
	)
	if err != nil {
		return query.AuditSummary{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var item query.AuditUserMetric
		if err := rows.Scan(&item.UserID, &item.DisplayName, &item.ActionCount); err != nil {
			return query.AuditSummary{}, err
		}
		summary.TopActiveUsers = append(summary.TopActiveUsers, item)
	}

	return summary, rows.Err()
}
