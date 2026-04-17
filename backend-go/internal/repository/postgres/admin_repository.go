package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/service"
)

type AdminRepository struct {
	db *sql.DB
}

func NewAdminRepository(db *sql.DB) AdminRepository {
	return AdminRepository{db: db}
}

// CreateTeamSpace inserts a new team space and returns its summary.
func (r AdminRepository) CreateTeamSpace(ctx context.Context, name, code, description, createdBy string) (*query.TeamSpaceSummary, error) {
	var item query.TeamSpaceSummary
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO team_spaces (id, name, code, description, created_by)
		VALUES (gen_random_uuid(), $1, $2, NULLIF($3, ''), $4::uuid)
		RETURNING id::text, name, code
	`, name, code, description, createdBy).Scan(&item.ID, &item.Name, &item.Code)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, service.ErrConflict
		}
		return nil, err
	}
	return &item, nil
}

// CreateProjectWithOwner inserts a new project and its owner membership in one transaction.
func (r AdminRepository) CreateProjectWithOwner(ctx context.Context, teamSpaceID, name, code, description, ownerID string) (*query.ProjectSummary, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var item query.ProjectSummary
	err = tx.QueryRowContext(ctx, `
		INSERT INTO projects (id, team_space_id, name, code, description, owner_id)
		VALUES (gen_random_uuid(), $1::uuid, $2, $3, NULLIF($4, ''), $5::uuid)
		RETURNING id::text, team_space_id::text, name, code
	`, teamSpaceID, name, code, description, ownerID).Scan(
		&item.ID, &item.TeamSpaceID, &item.Name, &item.Code,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, service.ErrConflict
		}
		return nil, err
	}

	// Mirror owner into project_members
	_, err = tx.ExecContext(ctx, `
		INSERT INTO project_members (project_id, user_id, project_role)
		VALUES ($1::uuid, $2::uuid, 'owner')
	`, item.ID, ownerID)
	if err != nil {
		return nil, err
	}

	// Populate owner display name
	err = tx.QueryRowContext(ctx, `
		SELECT id::text, display_name FROM users WHERE id = $1::uuid
	`, ownerID).Scan(&item.Owner.ID, &item.Owner.DisplayName)
	if err != nil {
		return nil, err
	}

	return &item, tx.Commit()
}

// CreateUser inserts a new user with a pre-hashed password.
func (r AdminRepository) CreateUser(ctx context.Context, username, passwordHash, displayName, role, email, phone string) (*query.UserOption, error) {
	var item query.UserOption
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO users (id, username, password_hash, display_name, role, email, phone)
		VALUES (gen_random_uuid(), $1, $2, $3, $4::user_role, NULLIF($5, ''), NULLIF($6, ''))
		RETURNING id::text, username, display_name, role::text,
			COALESCE(email, ''), COALESCE(phone, ''), COALESCE(wechat, ''), status
	`, username, passwordHash, displayName, role, email, phone).Scan(
		&item.ID, &item.Username, &item.DisplayName, &item.Role,
		&item.Email, &item.Phone, &item.Wechat, &item.Status,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, service.ErrConflict
		}
		return nil, err
	}
	return &item, nil
}

// UpdateUser updates mutable fields of a user.
func (r AdminRepository) UpdateUser(ctx context.Context, userID string, fields map[string]any) (*query.UserOption, error) {
	setClauses := make([]string, 0, len(fields))
	args := make([]any, 0, len(fields)+1)
	args = append(args, userID) // $1
	idx := 2
	for col, val := range fields {
		switch col {
		case "role":
			setClauses = append(setClauses, fmt.Sprintf("%s = $%d::user_role", col, idx))
		default:
			setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, idx))
		}
		args = append(args, val)
		idx++
	}
	setClauses = append(setClauses, "updated_at = NOW()")

	q := fmt.Sprintf(`
		UPDATE users SET %s
		WHERE id = $1
		RETURNING id::text, username, display_name, role::text,
			COALESCE(email, ''), COALESCE(phone, ''), COALESCE(wechat, ''), status
	`, strings.Join(setClauses, ", "))

	var item query.UserOption
	err := r.db.QueryRowContext(ctx, q, args...).Scan(
		&item.ID, &item.Username, &item.DisplayName, &item.Role,
		&item.Email, &item.Phone, &item.Wechat, &item.Status,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

// ListAllUsers returns all users including inactive ones for admin management.
func (r AdminRepository) ListAllUsers(ctx context.Context) ([]query.UserOption, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id::text, username, display_name, role::text,
			COALESCE(email, ''), COALESCE(phone, ''), COALESCE(wechat, ''), status
		FROM users
		ORDER BY
			CASE role::text WHEN 'admin' THEN 1 WHEN 'project_lead' THEN 2 ELSE 3 END,
			display_name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]query.UserOption, 0)
	for rows.Next() {
		var item query.UserOption
		if err := rows.Scan(&item.ID, &item.Username, &item.DisplayName, &item.Role,
			&item.Email, &item.Phone, &item.Wechat, &item.Status); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ListProjectMembers returns all members of a project with user info.
func (r AdminRepository) ListProjectMembers(ctx context.Context, projectID string) ([]query.ProjectMemberItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT pm.id::text, pm.user_id::text, u.display_name, u.username, pm.project_role
		FROM project_members pm
		JOIN users u ON u.id = pm.user_id
		WHERE pm.project_id::text = $1
		ORDER BY
			CASE pm.project_role WHEN 'owner' THEN 1 WHEN 'manager' THEN 2 WHEN 'contributor' THEN 3 ELSE 4 END,
			u.display_name ASC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]query.ProjectMemberItem, 0)
	for rows.Next() {
		var item query.ProjectMemberItem
		if err := rows.Scan(&item.ID, &item.UserID, &item.DisplayName, &item.Username, &item.ProjectRole); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// AddProjectMember inserts a new project member.
func (r AdminRepository) AddProjectMember(ctx context.Context, projectID, userID, projectRole string) (*query.ProjectMemberItem, error) {
	var item query.ProjectMemberItem
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO project_members (project_id, user_id, project_role)
		VALUES ($1::uuid, $2::uuid, $3)
		RETURNING id::text
	`, projectID, userID, projectRole).Scan(&item.ID)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, service.ErrConflict
		}
		return nil, err
	}
	item.UserID = userID
	item.ProjectRole = projectRole

	// Populate user display info
	_ = r.db.QueryRowContext(ctx, `SELECT display_name, username FROM users WHERE id = $1::uuid`,
		userID).Scan(&item.DisplayName, &item.Username)

	return &item, nil
}

// UpdateProjectMember changes the role of a project member.
func (r AdminRepository) UpdateProjectMember(ctx context.Context, memberID, projectRole string) (*query.ProjectMemberItem, error) {
	var item query.ProjectMemberItem
	err := r.db.QueryRowContext(ctx, `
		UPDATE project_members SET project_role = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id::text, user_id::text, project_role
	`, memberID, projectRole).Scan(&item.ID, &item.UserID, &item.ProjectRole)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, service.ErrNotFound
		}
		return nil, err
	}

	_ = r.db.QueryRowContext(ctx, `SELECT display_name, username FROM users WHERE id = $1::uuid`,
		item.UserID).Scan(&item.DisplayName, &item.Username)

	return &item, nil
}

// RemoveProjectMember deletes a project member record.
func (r AdminRepository) RemoveProjectMember(ctx context.Context, memberID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM project_members WHERE id = $1`, memberID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return service.ErrNotFound
	}
	return nil
}

// isUniqueViolation checks for PostgreSQL unique constraint violation (code 23505).
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}
