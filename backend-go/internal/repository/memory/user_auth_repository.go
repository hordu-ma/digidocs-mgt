package memory

import (
	"context"
	"os"

	"golang.org/x/crypto/bcrypt"

	"digidocs-mgt/backend-go/internal/domain/auth"
	"digidocs-mgt/backend-go/internal/service"
)

// UserAuthRepository provides a single dev admin account for memory-backend mode.
// Configure via env vars DEV_ADMIN_USERNAME and DEV_ADMIN_PASSWORD_HASH (bcrypt).
// Default username: admin, default password hash: bcrypt("admin123").
type UserAuthRepository struct {
	username     string
	passwordHash string
}

// defaultDevHash is bcrypt("admin123", cost=10) — for local development only.
const defaultDevHash = "$2a$10$PNs2z4ZC0G7FH5yhWK8bEOI3QFZ/BOFbxd3U9Ozo/VC9i7zwR.fpa"

func NewUserAuthRepository() UserAuthRepository {
	username := os.Getenv("DEV_ADMIN_USERNAME")
	if username == "" {
		username = "admin"
	}

	passwordHash := os.Getenv("DEV_ADMIN_PASSWORD_HASH")
	if passwordHash == "" {
		passwordHash = defaultDevHash
	}

	return UserAuthRepository{username: username, passwordHash: passwordHash}
}

func (r UserAuthRepository) FindUserByUsername(_ context.Context, username string) (*auth.UserRecord, error) {
	if username != r.username {
		return nil, service.ErrNotFound
	}

	// Sanity-check that the stored hash is actually a bcrypt hash before returning it.
	if len(r.passwordHash) < 7 || r.passwordHash[:4] != "$2a$" && r.passwordHash[:4] != "$2b$" {
		return nil, service.ErrNotFound
	}

	// Validate hash is parseable (not a timing check — the actual comparison is in AuthService).
	if err := bcrypt.CompareHashAndPassword([]byte(r.passwordHash), []byte("")); err == bcrypt.ErrMismatchedHashAndPassword {
		// Expected mismatch — hash is structurally valid.
	} else if err != nil {
		return nil, service.ErrNotFound
	}

	return &auth.UserRecord{
		ID:           "00000000-0000-0000-0000-000000000001",
		PasswordHash: r.passwordHash,
		DisplayName:  "开发管理员",
		Role:         "admin",
	}, nil
}

func (r UserAuthRepository) GetUserProfile(_ context.Context, userID string) (*auth.UserProfile, error) {
	if userID != "00000000-0000-0000-0000-000000000001" {
		return nil, service.ErrNotFound
	}

	return &auth.UserProfile{
		ID:          userID,
		Username:    r.username,
		DisplayName: "开发管理员",
		Role:        "admin",
		Email:       "admin@example.com",
		Phone:       "13800000000",
		Wechat:      "admin_wechat",
		Status:      "active",
	}, nil
}

func (r UserAuthRepository) UpdateUserProfile(_ context.Context, userID string, input auth.ProfileUpdateInput) (*auth.UserProfile, error) {
	if userID != "00000000-0000-0000-0000-000000000001" {
		return nil, service.ErrNotFound
	}

	return &auth.UserProfile{
		ID:          userID,
		Username:    r.username,
		DisplayName: input.DisplayName,
		Role:        "admin",
		Email:       input.Email,
		Phone:       input.Phone,
		Wechat:      input.Wechat,
		Status:      "active",
	}, nil
}
