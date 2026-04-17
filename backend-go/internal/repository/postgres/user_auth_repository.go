package postgres

import (
	"context"
	"database/sql"
	"errors"

	"digidocs-mgt/backend-go/internal/domain/auth"
	"digidocs-mgt/backend-go/internal/service"
)

type UserAuthRepository struct {
	db *sql.DB
}

func NewUserAuthRepository(db *sql.DB) UserAuthRepository {
	return UserAuthRepository{db: db}
}

func (r UserAuthRepository) FindUserByUsername(ctx context.Context, username string) (*auth.UserRecord, error) {
	var record auth.UserRecord
	if err := r.db.QueryRowContext(
		ctx,
		`
		SELECT id::text, password_hash, display_name, role::text
		FROM users
		WHERE username = $1
		  AND status = 'active'
		`,
		username,
	).Scan(&record.ID, &record.PasswordHash, &record.DisplayName, &record.Role); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrNotFound
		}
		return nil, err
	}

	return &record, nil
}

func (r UserAuthRepository) GetUserProfile(ctx context.Context, userID string) (*auth.UserProfile, error) {
	var profile auth.UserProfile
	var lastLogin sql.NullTime
	if err := r.db.QueryRowContext(
		ctx,
		`
		SELECT
			id::text,
			username,
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
			username,
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
		return nil, err
	}
	if lastLogin.Valid {
		value := lastLogin.Time.UTC().Format("2006-01-02T15:04:05Z")
		profile.LastLoginAt = &value
	}

	return &profile, nil
}
