package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"

	"digidocs-mgt/backend-go/internal/domain/auth"
	"digidocs-mgt/backend-go/internal/service"
)

const (
	usersEmailUniqueIndex = "idx_users_email_unique"
	usersPhoneUniqueIndex = "idx_users_phone_unique"
)

type UserAuthRepository struct {
	db *sql.DB
}

func NewUserAuthRepository(db *sql.DB) UserAuthRepository {
	return UserAuthRepository{db: db}
}

func (r UserAuthRepository) FindUserByIdentifier(ctx context.Context, identifier string) (*auth.UserRecord, error) {
	var record auth.UserRecord
	if err := r.db.QueryRowContext(
		ctx,
		`
		SELECT id::text, COALESCE(username, ''), password_hash, display_name, role::text
		FROM users
		WHERE (username = $1 OR email = $1 OR phone = $1)
		  AND status = 'active'
		`,
		identifier,
	).Scan(&record.ID, &record.Username, &record.PasswordHash, &record.DisplayName, &record.Role); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrNotFound
		}
		return nil, err
	}

	return &record, nil
}

func (r UserAuthRepository) RegisterUser(ctx context.Context, input auth.RegisterUserInput) (*auth.UserRecord, error) {
	var record auth.UserRecord
	err := r.db.QueryRowContext(
		ctx,
		`
		INSERT INTO users (id, username, password_hash, display_name, role, email, phone, status)
		VALUES (gen_random_uuid(), NULL, $1, $2, 'member'::user_role, NULLIF($3, ''), NULLIF($4, ''), 'active')
		RETURNING id::text, COALESCE(username, ''), password_hash, display_name, role::text
		`,
		input.PasswordHash,
		input.DisplayName,
		input.Email,
		input.Phone,
	).Scan(&record.ID, &record.Username, &record.PasswordHash, &record.DisplayName, &record.Role)
	if err != nil {
		if conflict := uniqueIndexConflict(err); conflict != nil {
			return nil, conflict
		}
		return nil, err
	}
	return &record, nil
}

func uniqueIndexConflict(err error) error {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) || pqErr.Code != "23505" {
		return nil
	}
	switch pqErr.Constraint {
	case usersEmailUniqueIndex:
		return fmt.Errorf("%w: 该邮箱已被注册", service.ErrConflict)
	case usersPhoneUniqueIndex:
		return fmt.Errorf("%w: 该手机号已被注册", service.ErrConflict)
	default:
		return service.ErrConflict
	}
}

func (r UserAuthRepository) GetUserProfile(ctx context.Context, userID string) (*auth.UserProfile, error) {
	var profile auth.UserProfile
	var lastLogin sql.NullTime
	if err := r.db.QueryRowContext(
		ctx,
		`
		SELECT
			id::text,
			COALESCE(username, ''),
			display_name,
			role::text,
			COALESCE(email, ''),
			COALESCE(phone, ''),
			COALESCE(wechat, ''),
			status,
			last_login_at
		FROM users
		WHERE id = $1
		  AND status = 'active'
		`,
		userID,
	).Scan(
		&profile.ID,
		&profile.Username,
		&profile.DisplayName,
		&profile.Role,
		&profile.Email,
		&profile.Phone,
		&profile.Wechat,
		&profile.Status,
		&lastLogin,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrNotFound
		}
		return nil, err
	}
	if lastLogin.Valid {
		value := lastLogin.Time.UTC().Format("2006-01-02T15:04:05Z")
		profile.LastLoginAt = &value
	}

	return &profile, nil
}

func (r UserAuthRepository) UpdateUserProfile(ctx context.Context, userID string, input auth.ProfileUpdateInput) (*auth.UserProfile, error) {
	var profile auth.UserProfile
	var lastLogin sql.NullTime
	if err := r.db.QueryRowContext(
		ctx,
		`
		UPDATE users
		SET
			display_name = $2,
			email = NULLIF($3, ''),
			phone = NULLIF($4, ''),
			wechat = NULLIF($5, ''),
			updated_at = now()
		WHERE id = $1
		  AND status = 'active'
		RETURNING
			id::text,
			COALESCE(username, ''),
			display_name,
			role::text,
			COALESCE(email, ''),
			COALESCE(phone, ''),
			COALESCE(wechat, ''),
			status,
			last_login_at
		`,
		userID,
		input.DisplayName,
		input.Email,
		input.Phone,
		input.Wechat,
	).Scan(
		&profile.ID,
		&profile.Username,
		&profile.DisplayName,
		&profile.Role,
		&profile.Email,
		&profile.Phone,
		&profile.Wechat,
		&profile.Status,
		&lastLogin,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrNotFound
		}
		if conflict := uniqueIndexConflict(err); conflict != nil {
			return nil, conflict
		}
		return nil, err
	}
	if lastLogin.Valid {
		value := lastLogin.Time.UTC().Format("2006-01-02T15:04:05Z")
		profile.LastLoginAt = &value
	}

	return &profile, nil
}

// GetUserCredentials loads the stored password hash so the service layer can
// verify the caller's current password before changing it.
func (r UserAuthRepository) GetUserCredentials(ctx context.Context, userID string) (*auth.UserRecord, error) {
	var record auth.UserRecord
	if err := r.db.QueryRowContext(
		ctx,
		`
		SELECT id::text, COALESCE(username, ''), password_hash, display_name, role::text
		FROM users
		WHERE id::text = $1
		  AND status = 'active'
		`,
		userID,
	).Scan(&record.ID, &record.Username, &record.PasswordHash, &record.DisplayName, &record.Role); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrNotFound
		}
		return nil, err
	}
	return &record, nil
}

func (r UserAuthRepository) UpdateUserPassword(ctx context.Context, userID string, passwordHash string) error {
	result, err := r.db.ExecContext(
		ctx,
		`
		UPDATE users
		SET password_hash = $2, updated_at = now(), password_changed_at = now()
		WHERE id::text = $1
		  AND status = 'active'
		`,
		userID,
		passwordHash,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrNotFound
	}
	return nil
}

func (r UserAuthRepository) GetSessionState(ctx context.Context, userID string) (string, *time.Time, error) {
	var status string
	var changedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `SELECT status, password_changed_at FROM users WHERE id::text = $1`, userID).Scan(&status, &changedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, service.ErrNotFound
	}
	if err != nil {
		return "", nil, err
	}
	if changedAt.Valid {
		t := changedAt.Time
		return status, &t, nil
	}
	return status, nil, nil
}
