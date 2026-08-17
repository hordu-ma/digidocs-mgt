package postgres

import (
	"context"

	"digidocs-mgt/backend-go/internal/domain/query"
)

type FlowRepository struct {
	db DBTX
}

func NewFlowRepository(db DBTX) FlowRepository {
	return FlowRepository{db: db}
}

// maxDocumentFlows caps the flow history returned for one document. Same reason
// as maxDocumentVersions: the list grows without bound over a document's life
// and the detail page fetched all of it on every load.
const maxDocumentFlows = 200

func (r FlowRepository) ListFlows(ctx context.Context, documentID string) ([]query.FlowItem, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT
			id::text,
			action,
			COALESCE(from_status::text, ''),
			to_status::text,
			TO_CHAR(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM flow_records
		WHERE document_id::text = $1
		ORDER BY created_at DESC
		LIMIT $2
		`,
		documentID,
		maxDocumentFlows,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]query.FlowItem, 0)
	for rows.Next() {
		var item query.FlowItem
		if err := rows.Scan(
			&item.ID,
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
