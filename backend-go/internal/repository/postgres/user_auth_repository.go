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
