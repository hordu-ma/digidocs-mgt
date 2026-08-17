package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/service"
)

type MembershipRepository struct {
	db    *sql.DB
	admin AdminRepository
}

func NewMembershipRepository(db *sql.DB) MembershipRepository {
	return MembershipRepository{db: db, admin: NewAdminRepository(db)}
}

func (r MembershipRepository) ListProjectMembers(ctx context.Context, projectID string) ([]query.ProjectMemberItem, error) {
	return r.admin.ListProjectMembers(ctx, projectID)
}

func (r MembershipRepository) AddProjectMember(ctx context.Context, projectID, userID, projectRole string) (*query.ProjectMemberItem, error) {
	return r.admin.AddProjectMember(ctx, projectID, userID, projectRole)
}

func (r MembershipRepository) UpdateProjectMember(ctx context.Context, memberID, projectRole string) (*query.ProjectMemberItem, error) {
	return r.admin.UpdateProjectMember(ctx, memberID, projectRole)
}

func (r MembershipRepository) RemoveProjectMember(ctx context.Context, memberID string) error {
	return r.admin.RemoveProjectMember(ctx, memberID)
}

func (r MembershipRepository) GetProjectMember(ctx context.Context, memberID string) (*query.ProjectMemberItem, error) {
	var item query.ProjectMemberItem
	err := r.db.QueryRowContext(ctx, `
		SELECT pm.id::text, pm.user_id::text, u.display_name, COALESCE(u.username, ''), pm.project_role
		FROM project_members pm
		JOIN users u ON u.id = pm.user_id
		WHERE pm.id = $1
	`, memberID).Scan(&item.ID, &item.UserID, &item.DisplayName, &item.Username, &item.ProjectRole)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r MembershipRepository) FindProjectMember(ctx context.Context, projectID, userID string) (*query.ProjectMemberItem, error) {
	var item query.ProjectMemberItem
	err := r.db.QueryRowContext(ctx, `
		SELECT pm.id::text, pm.user_id::text, u.display_name, COALESCE(u.username, ''), pm.project_role
		FROM project_members pm
		JOIN users u ON u.id = pm.user_id
		WHERE pm.project_id::text = $1 AND pm.user_id::text = $2
	`, projectID, userID).Scan(&item.ID, &item.UserID, &item.DisplayName, &item.Username, &item.ProjectRole)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r MembershipRepository) CountOwners(ctx context.Context, projectID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM project_members
		WHERE project_id::text = $1 AND project_role = 'owner'
	`, projectID).Scan(&n)
	return n, err
}

func (r MembershipRepository) CreateInvite(ctx context.Context, projectID, token, projectRole, createdBy string, expiresAt time.Time) (*query.ProjectInviteItem, error) {
	var exists int
	if err := r.db.QueryRowContext(ctx, `SELECT 1 FROM projects WHERE id::text = $1`, projectID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrNotFound
		}
		return nil, err
	}
	var item query.ProjectInviteItem
	var expires, created time.Time
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO project_invites (project_id, token, project_role, status, expires_at, created_by)
		VALUES ($1::uuid, $2, $3, 'active', $4, $5::uuid)
		RETURNING id::text, project_id::text, token, project_role, status, expires_at, created_at
	`, projectID, token, projectRole, expiresAt.UTC(), createdBy).Scan(
		&item.ID, &item.ProjectID, &item.Token, &item.ProjectRole, &item.Status, &expires, &created,
	)
	if err != nil {
		return nil, err
	}
	item.ExpiresAt = expires.UTC().Format(time.RFC3339)
	item.CreatedAt = created.UTC().Format(time.RFC3339)
	return &item, nil
}

func (r MembershipRepository) ListInvites(ctx context.Context, projectID string) ([]query.ProjectInviteItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, project_id::text, token, project_role, status, expires_at, created_at
		FROM project_invites
		WHERE project_id::text = $1
		ORDER BY created_at DESC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]query.ProjectInviteItem, 0)
	for rows.Next() {
		var item query.ProjectInviteItem
		var expires, created time.Time
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.Token, &item.ProjectRole, &item.Status, &expires, &created); err != nil {
			return nil, err
		}
		item.ExpiresAt = expires.UTC().Format(time.RFC3339)
		item.CreatedAt = created.UTC().Format(time.RFC3339)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r MembershipRepository) GetInviteByID(ctx context.Context, projectID, inviteID string) (*query.ProjectInviteItem, error) {
	return r.scanInvite(ctx, `
		SELECT id::text, project_id::text, token, project_role, status, expires_at, created_at
		FROM project_invites
		WHERE id::text = $1 AND project_id::text = $2
	`, inviteID, projectID)
}

func (r MembershipRepository) GetInviteByToken(ctx context.Context, token string) (*query.ProjectInviteItem, error) {
	return r.scanInvite(ctx, `
		SELECT id::text, project_id::text, token, project_role, status, expires_at, created_at
		FROM project_invites
		WHERE token = $1
	`, token)
}

func (r MembershipRepository) scanInvite(ctx context.Context, q string, args ...any) (*query.ProjectInviteItem, error) {
	var item query.ProjectInviteItem
	var expires, created time.Time
	err := r.db.QueryRowContext(ctx, q, args...).Scan(
		&item.ID, &item.ProjectID, &item.Token, &item.ProjectRole, &item.Status, &expires, &created,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	item.ExpiresAt = expires.UTC().Format(time.RFC3339)
	item.CreatedAt = created.UTC().Format(time.RFC3339)
	return &item, nil
}

func (r MembershipRepository) RevokeInvite(ctx context.Context, projectID, inviteID, actorID string) (*query.ProjectInviteItem, error) {
	var item query.ProjectInviteItem
	var expires, created time.Time
	err := r.db.QueryRowContext(ctx, `
		UPDATE project_invites
		SET status = 'revoked', revoked_at = NOW(), revoked_by = $3::uuid
		WHERE id::text = $1 AND project_id::text = $2 AND status = 'active'
		RETURNING id::text, project_id::text, token, project_role, status, expires_at, created_at
	`, inviteID, projectID, actorID).Scan(
		&item.ID, &item.ProjectID, &item.Token, &item.ProjectRole, &item.Status, &expires, &created,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	item.ExpiresAt = expires.UTC().Format(time.RFC3339)
	item.CreatedAt = created.UTC().Format(time.RFC3339)
	return &item, nil
}

func (r MembershipRepository) GetInvitePreview(ctx context.Context, token string) (*query.ProjectInvitePreview, error) {
	var preview query.ProjectInvitePreview
	var expires time.Time
	err := r.db.QueryRowContext(ctx, `
		SELECT p.id::text, p.name, ts.name, i.project_role, i.status, i.expires_at
		FROM project_invites i
		JOIN projects p ON p.id = i.project_id
		JOIN team_spaces ts ON ts.id = p.team_space_id
		WHERE i.token = $1
	`, token).Scan(
		&preview.ProjectID, &preview.ProjectName, &preview.TeamSpaceName, &preview.ProjectRole, &preview.Status, &expires,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	preview.ExpiresAt = expires.UTC().Format(time.RFC3339)
	preview.Expired = time.Now().UTC().After(expires.UTC())
	return &preview, nil
}
