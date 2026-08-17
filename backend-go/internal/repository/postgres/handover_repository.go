package postgres

import (
	"context"
	"database/sql"
	"errors"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/service"
)

type HandoverRepository struct {
	db DBTX
}

func NewHandoverRepository(db DBTX) HandoverRepository {
	return HandoverRepository{db: db}
}

func (r HandoverRepository) ListHandovers(ctx context.Context, scope query.ActorScope) ([]query.HandoverItem, error) {
	isAdmin, actorID := scopeArgs(scope)
	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT
			h.id::text,
			h.target_user_id::text,
			h.receiver_user_id::text,
			COALESCE(h.project_id::text, ''),
			h.status::text,
			COALESCE(h.remark, '')
		FROM graduation_handovers h
		WHERE (
			h.target_user_id::text = $2
			OR h.receiver_user_id::text = $2
			OR `+projectScopeSQL("h.project_id", "$1", "$2")+`
		)
		ORDER BY h.generated_at DESC
		`,
		isAdmin,
		actorID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]query.HandoverItem, 0)
	for rows.Next() {
		var item query.HandoverItem
		if err := rows.Scan(
			&item.ID,
			&item.TargetUserID,
			&item.ReceiverUserID,
			&item.ProjectID,
			&item.Status,
			&item.Remark,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func (r HandoverRepository) GetHandover(ctx context.Context, handoverID string) (*query.HandoverItem, error) {
	var item query.HandoverItem

	if err := r.db.QueryRowContext(
		ctx,
		`
		SELECT
			id::text,
			target_user_id::text,
			receiver_user_id::text,
			COALESCE(project_id::text, ''),
			status::text,
			COALESCE(remark, '')
		FROM graduation_handovers
		WHERE id::text = $1
		`,
		handoverID,
	).Scan(
		&item.ID,
		&item.TargetUserID,
		&item.ReceiverUserID,
		&item.ProjectID,
		&item.Status,
		&item.Remark,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrNotFound
		}
		return nil, err
	}

	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT
			ghi.document_id::text,
			COALESCE(d.title, ''),
			COALESCE(d.current_status::text, ''),
			ghi.selected,
			COALESCE(ghi.note, '')
		FROM graduation_handover_items ghi
		LEFT JOIN documents d ON d.id = ghi.document_id
		WHERE ghi.handover_id::text = $1
		ORDER BY ghi.created_at ASC
		`,
		handoverID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	item.Items = make([]query.HandoverLine, 0)
	for rows.Next() {
		var line query.HandoverLine
		if err := rows.Scan(&line.DocumentID, &line.Title, &line.CurrentStatus, &line.Selected, &line.Note); err != nil {
			return nil, err
		}
		item.Items = append(item.Items, line)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &item, nil
}
