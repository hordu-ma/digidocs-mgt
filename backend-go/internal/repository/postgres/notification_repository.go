package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/service"

	"github.com/lib/pq"
)

type NotificationRepository struct {
	db DBTX
}

func NewNotificationRepository(db DBTX) NotificationRepository {
	return NotificationRepository{db: db}
}

func (r NotificationRepository) CreateNotifications(ctx context.Context, input command.NotificationInput) error {
	if len(input.Recipients) == 0 {
		return nil
	}
	_, err := r.db.ExecContext(
		ctx,
		`
		INSERT INTO notifications (
			user_id, event_type, title, body, link, related_type, related_id, actor_id, created_at
		)
		SELECT
			u.id, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''),
			NULLIF($7, '')::uuid, NULLIF($8, '')::uuid, $9
		FROM users u
		WHERE u.id::text = ANY($1::text[])
		  AND u.status = 'active'
		`,
		pq.Array(input.Recipients),
		input.EventType,
		input.Title,
		input.Body,
		input.Link,
		input.RelatedType,
		input.RelatedID,
		input.ActorID,
		nowUTC(),
	)
	return err
}

func (r NotificationRepository) ListNotifications(ctx context.Context, userID string, unreadOnly bool, page int, pageSize int) ([]query.NotificationItem, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	var total int
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM notifications
		WHERE user_id::text = $1 AND (NOT $2 OR read_at IS NULL)
	`, userID, unreadOnly).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			n.id::text, n.event_type, n.title, COALESCE(n.body, ''), COALESCE(n.link, ''),
			COALESCE(n.related_type, ''), COALESCE(n.related_id::text, ''),
			COALESCE(a.display_name, ''), n.read_at IS NOT NULL,
			TO_CHAR(n.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM notifications n
		LEFT JOIN users a ON a.id = n.actor_id
		WHERE n.user_id::text = $1 AND (NOT $2 OR n.read_at IS NULL)
		ORDER BY n.created_at DESC
		OFFSET $3 LIMIT $4
	`, userID, unreadOnly, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]query.NotificationItem, 0)
	for rows.Next() {
		var item query.NotificationItem
		if err := rows.Scan(&item.ID, &item.EventType, &item.Title, &item.Body, &item.Link,
			&item.RelatedType, &item.RelatedID, &item.ActorName, &item.Read, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r NotificationRepository) CountUnread(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id::text = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

// MarkRead is scoped by user so one user cannot touch another's notification;
// a foreign or unknown id is reported as not found.
func (r NotificationRepository) MarkRead(ctx context.Context, userID string, notificationID string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE notifications SET read_at = COALESCE(read_at, $3)
		WHERE id::text = $1 AND user_id::text = $2
	`, notificationID, userID, nowUTC())
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return service.ErrNotFound
	}
	return nil
}

func (r NotificationRepository) MarkAllRead(ctx context.Context, userID string) (int, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE notifications SET read_at = $2 WHERE user_id::text = $1 AND read_at IS NULL
	`, userID, nowUTC())
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// managerOfProjectSQL is true when $1 (user id) owns or manages project col,
// or when $2 (is admin) is set.
func managerOfProjectSQL(projectColumn string) string {
	return `($2 OR EXISTS (
		SELECT 1 FROM projects mp
		LEFT JOIN project_members mpm
		  ON mpm.project_id = mp.id AND mpm.user_id::text = $1
		 AND mpm.project_role IN ('owner', 'manager')
		WHERE mp.id = ` + projectColumn + `
		  AND (mp.owner_id::text = $1 OR mpm.id IS NOT NULL)
	))`
}

// ListInbox returns the caller's open tasks, newest-relevant first per kind.
// Each branch mirrors an action the caller is actually allowed to perform, so
// the inbox never lists something whose button would then return 403.
func (r NotificationRepository) ListInbox(ctx context.Context, userID string, isAdmin bool) ([]query.InboxItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT kind, title, detail, link, related_type, related_id, project_name, since FROM (
			-- 1. 转交给我、等我接收的文档（转交后责任人即为接收人）
			SELECT 'accept_transfer' AS kind, d.title,
				COALESCE((
					SELECT fu.display_name || ' 转交给你'
					FROM flow_records fr JOIN users fu ON fu.id = fr.from_user_id
					WHERE fr.document_id = d.id AND fr.action = 'transfer'
					ORDER BY fr.created_at DESC LIMIT 1
				), '等待你接收') AS detail,
				'/documents/' || d.id::text AS link, 'document' AS related_type, d.id::text AS related_id,
				COALESCE(p.name, '') AS project_name, d.updated_at AS since, 1 AS ord
			FROM documents d LEFT JOIN projects p ON p.id = d.project_id
			WHERE d.current_owner_id::text = $1 AND d.current_status = 'pending_handover' AND d.is_deleted = false

			UNION ALL
			-- 2. 我是接收人、等我确认的交接单
			SELECT 'confirm_handover', '确认接收 ' || tu.display_name || ' 的交接',
				'清单含 ' || (
					(SELECT COUNT(*) FROM graduation_handover_items i WHERE i.handover_id = gh.id AND i.selected) +
					(SELECT COUNT(*) FROM graduation_handover_data_items di WHERE di.handover_id = gh.id AND di.selected)
				)::text || ' 项',
				'/handovers?id=' || gh.id::text, 'handover', gh.id::text,
				COALESCE(p.name, ''), gh.generated_at, 2
			FROM graduation_handovers gh
			JOIN users tu ON tu.id = gh.target_user_id
			LEFT JOIN projects p ON p.id = gh.project_id
			WHERE gh.receiver_user_id::text = $1 AND gh.status = 'generated'

			UNION ALL
			-- 3. 我管理的课题里，接收人已确认、等负责人完成的交接单
			SELECT 'complete_handover', '完成 ' || tu.display_name || ' → ' || ru.display_name || ' 的交接',
				'接收人已确认，等待负责人完成', '/handovers?id=' || gh.id::text, 'handover', gh.id::text,
				COALESCE(p.name, ''), COALESCE(gh.confirmed_at, gh.generated_at), 3
			FROM graduation_handovers gh
			JOIN users tu ON tu.id = gh.target_user_id
			JOIN users ru ON ru.id = gh.receiver_user_id
			LEFT JOIN projects p ON p.id = gh.project_id
			WHERE gh.status = 'pending_confirm' AND `+managerOfProjectSQL("gh.project_id")+`

			UNION ALL
			-- 4. 我管理的课题里，清单还是空的交接单
			SELECT 'fill_handover_items', '补全 ' || tu.display_name || ' 的交接清单',
				'交接单尚未勾选任何文档或数据', '/handovers?id=' || gh.id::text, 'handover', gh.id::text,
				COALESCE(p.name, ''), gh.generated_at, 4
			FROM graduation_handovers gh
			JOIN users tu ON tu.id = gh.target_user_id
			LEFT JOIN projects p ON p.id = gh.project_id
			WHERE gh.status = 'generated' AND `+managerOfProjectSQL("gh.project_id")+`
			  AND NOT EXISTS (SELECT 1 FROM graduation_handover_items i WHERE i.handover_id = gh.id AND i.selected)
			  AND NOT EXISTS (SELECT 1 FROM graduation_handover_data_items di WHERE di.handover_id = gh.id AND di.selected)

			UNION ALL
			-- 5. 我负责的文档上待确认的 AI 建议
			SELECT 'review_suggestion', d.title, COUNT(s.id)::text || ' 条 AI 建议待确认',
				'/documents/' || d.id::text, 'document', d.id::text,
				COALESCE(p.name, ''), MAX(s.generated_at), 5
			FROM assistant_suggestions s
			JOIN documents d ON d.id = s.related_id AND s.related_type = 'document'
			LEFT JOIN projects p ON p.id = d.project_id
			WHERE s.status = 'pending' AND d.current_owner_id::text = $1 AND d.is_deleted = false
			GROUP BY d.id, d.title, p.name

			UNION ALL
			-- 6. 我负责、超过 30 天未更新的在办文档
			SELECT 'stale_document', d.title, '超过 30 天未更新',
				'/documents/' || d.id::text, 'document', d.id::text,
				COALESCE(p.name, ''), d.updated_at, 6
			FROM documents d LEFT JOIN projects p ON p.id = d.project_id
			WHERE d.current_owner_id::text = $1 AND d.is_deleted = false
			  AND d.current_status IN ('draft', 'in_progress', 'handed_over')
			  AND d.updated_at < NOW() - INTERVAL '30 days'
		) t
		ORDER BY ord, since ASC
		LIMIT 100
	`, userID, isAdmin)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]query.InboxItem, 0)
	for rows.Next() {
		var item query.InboxItem
		var since sql.NullTime
		if err := rows.Scan(&item.Kind, &item.Title, &item.Detail, &item.Link, &item.RelatedType, &item.RelatedID, &item.ProjectName, &since); err != nil {
			return nil, err
		}
		if since.Valid {
			item.Since = since.Time.UTC().Format("2006-01-02T15:04:05Z")
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r NotificationRepository) GetUserContacts(ctx context.Context, userIDs []string) ([]query.UserContact, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, display_name, COALESCE(email, '')
		FROM users WHERE id::text = ANY($1::text[]) AND status = 'active'
	`, pq.Array(userIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]query.UserContact, 0, len(userIDs))
	for rows.Next() {
		var c query.UserContact
		if err := rows.Scan(&c.ID, &c.DisplayName, &c.Email); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r NotificationRepository) GetDocumentBrief(ctx context.Context, documentID string) (*query.DocumentBrief, error) {
	var b query.DocumentBrief
	err := r.db.QueryRowContext(ctx, `
		SELECT d.id::text, d.title, d.project_id::text, COALESCE(p.name, ''), COALESCE(d.current_owner_id::text, '')
		FROM documents d LEFT JOIN projects p ON p.id = d.project_id
		WHERE d.id::text = $1
	`, documentID).Scan(&b.ID, &b.Title, &b.ProjectID, &b.ProjectName, &b.OwnerID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrNotFound
	}
	return &b, err
}

func (r NotificationRepository) GetHandoverBrief(ctx context.Context, handoverID string) (*query.HandoverBrief, error) {
	var b query.HandoverBrief
	err := r.db.QueryRowContext(ctx, `
		SELECT gh.id::text, COALESCE(gh.project_id::text, ''), COALESCE(p.name, ''),
			gh.target_user_id::text, tu.display_name,
			gh.receiver_user_id::text, ru.display_name,
			gh.generated_by::text
		FROM graduation_handovers gh
		JOIN users tu ON tu.id = gh.target_user_id
		JOIN users ru ON ru.id = gh.receiver_user_id
		LEFT JOIN projects p ON p.id = gh.project_id
		WHERE gh.id::text = $1
	`, handoverID).Scan(&b.ID, &b.ProjectID, &b.ProjectName, &b.TargetUserID, &b.TargetName,
		&b.ReceiverUserID, &b.ReceiverName, &b.GeneratedBy)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if b.ProjectID != "" {
		managers, _, err := r.GetProjectManagers(ctx, b.ProjectID)
		if err != nil {
			return nil, err
		}
		b.ManagerIDs = managers
	}
	return &b, nil
}

func (r NotificationRepository) GetLastTransferSender(ctx context.Context, documentID string) (string, error) {
	var id sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT from_user_id::text FROM flow_records
		WHERE document_id::text = $1 AND action = 'transfer'
		ORDER BY created_at DESC LIMIT 1
	`, documentID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return id.String, err
}

func (r NotificationRepository) GetAssistantRequestBrief(ctx context.Context, requestID string) (*query.AssistantRequestBrief, error) {
	var b query.AssistantRequestBrief
	err := r.db.QueryRowContext(ctx, `
		SELECT id::text, request_type, COALESCE(related_type, ''), COALESCE(related_id::text, ''),
			COALESCE(created_by::text, ''), status
		FROM assistant_requests WHERE id::text = $1
	`, requestID).Scan(&b.ID, &b.RequestType, &b.RelatedType, &b.RelatedID, &b.CreatedBy, &b.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrNotFound
	}
	return &b, err
}

// GetProjectManagers returns owner + manager user ids and the project name.
func (r NotificationRepository) GetProjectManagers(ctx context.Context, projectID string) ([]string, string, error) {
	var name string
	if err := r.db.QueryRowContext(ctx, `SELECT name FROM projects WHERE id::text = $1`, projectID).Scan(&name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", nil
		}
		return nil, "", err
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT owner_id::text FROM projects WHERE id::text = $1 AND owner_id IS NOT NULL
		UNION
		SELECT user_id::text FROM project_members
		WHERE project_id::text = $1 AND project_role IN ('owner', 'manager')
	`, projectID)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, "", err
		}
		ids = append(ids, strings.TrimSpace(id))
	}
	return ids, name, rows.Err()
}
