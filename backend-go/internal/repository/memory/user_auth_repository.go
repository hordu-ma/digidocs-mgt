package memory

import (
	"context"
	"fmt"
	"os"
	"sync"

	"golang.org/x/crypto/bcrypt"

	"digidocs-mgt/backend-go/internal/domain/auth"
	"digidocs-mgt/backend-go/internal/service"
)

const defaultDevAdminID = "00000000-0000-0000-0000-000000000001"

// defaultDevHash is bcrypt("admin123", cost=10) — for local development only.
const defaultDevHash = "$2a$10$PNs2z4ZC0G7FH5yhWK8bEOI3QFZ/BOFbxd3U9Ozo/VC9i7zwR.fpa"

type memoryUser struct {
	ID           string
	Username     string
	PasswordHash string
	DisplayName  string
	Role         string
	Email        string
	Phone        string
	Wechat       string
	Status       string
}

// UserAuthRepository is a stateful in-memory auth store used by handler tests
// and local memory-backend mode. It keeps a default development admin account
// and supports dynamic self-registration.
type UserAuthRepository struct {
	mu    sync.RWMutex
	users map[string]*memoryUser
	seq   int
}

func NewUserAuthRepository() *UserAuthRepository {
	username := os.Getenv("DEV_ADMIN_USERNAME")
	if username == "" {
		username = "admin"
	}

	passwordHash := os.Getenv("DEV_ADMIN_PASSWORD_HASH")
	if passwordHash == "" {
		passwordHash = defaultDevHash
	}

	repo := &UserAuthRepository{
		users: make(map[string]*memoryUser),
	}
	repo.users[defaultDevAdminID] = &memoryUser{
		ID:           defaultDevAdminID,
		Username:     username,
		PasswordHash: passwordHash,
		DisplayName:  "开发管理员",
		Role:         "admin",
		Email:        "admin@example.com",
		Phone:        "13800000000",
		Wechat:       "admin_wechat",
		Status:       "active",
	}
	return repo
}

func (r *UserAuthRepository) FindUserByIdentifier(_ context.Context, identifier string) (*auth.UserRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.findByIdentifierLocked(identifier)
	if !ok {
		return nil, service.ErrNotFound
	}
	if !validBcryptHash(user.PasswordHash) {
		return nil, service.ErrNotFound
	}
	return toUserRecord(user), nil
}

func (r *UserAuthRepository) RegisterUser(_ context.Context, input auth.RegisterUserInput) (*auth.UserRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if input.Email != "" {
		if _, exists := r.findByFieldLocked(func(u *memoryUser) bool { return u.Email == input.Email }); exists {
			return nil, fmt.Errorf("%w: 该邮箱已被注册", service.ErrConflict)
		}
	}
	if input.Phone != "" {
		if _, exists := r.findByFieldLocked(func(u *memoryUser) bool { return u.Phone == input.Phone }); exists {
			return nil, fmt.Errorf("%w: 该手机号已被注册", service.ErrConflict)
		}
	}

	r.seq++
	user := &memoryUser{
		ID:           fmt.Sprintf("mem-user-%d", r.seq),
		PasswordHash: input.PasswordHash,
		DisplayName:  input.DisplayName,
		Role:         "member",
		Email:        input.Email,
		Phone:        input.Phone,
		Status:       "active",
	}
	r.users[user.ID] = user
	return toUserRecord(user), nil
}

func (r *UserAuthRepository) GetUserProfile(_ context.Context, userID string) (*auth.UserProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.users[userID]
	if !ok || user.Status != "active" {
		return nil, service.ErrNotFound
	}
	return toUserProfile(user), nil
}

func (r *UserAuthRepository) UpdateUserProfile(_ context.Context, userID string, input auth.ProfileUpdateInput) (*auth.UserProfile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, ok := r.users[userID]
	if !ok || user.Status != "active" {
		return nil, service.ErrNotFound
	}
	if input.Email != "" {
		if existing, exists := r.findByFieldLocked(func(u *memoryUser) bool {
			return u.ID != userID && u.Email == input.Email
		}); exists && existing != nil {
			return nil, fmt.Errorf("%w: 该邮箱已被注册", service.ErrConflict)
		}
	}
	if input.Phone != "" {
		if existing, exists := r.findByFieldLocked(func(u *memoryUser) bool {
			return u.ID != userID && u.Phone == input.Phone
		}); exists && existing != nil {
			return nil, fmt.Errorf("%w: 该手机号已被注册", service.ErrConflict)
		}
	}

	user.DisplayName = input.DisplayName
	user.Email = input.Email
	user.Phone = input.Phone
	user.Wechat = input.Wechat
	return toUserProfile(user), nil
}

func (r *UserAuthRepository) findByIdentifierLocked(identifier string) (*memoryUser, bool) {
	if identifier == "" {
		return nil, false
	}
	return r.findByFieldLocked(func(u *memoryUser) bool {
		return u.Status == "active" && (u.Username == identifier || u.Email == identifier || u.Phone == identifier)
	})
}

func (r *UserAuthRepository) findByFieldLocked(match func(*memoryUser) bool) (*memoryUser, bool) {
	for _, user := range r.users {
		if match(user) {
			return user, true
		}
	}
	return nil, false
}

func toUserRecord(user *memoryUser) *auth.UserRecord {
	return &auth.UserRecord{
		ID:           user.ID,
		Username:     user.Username,
		PasswordHash: user.PasswordHash,
		DisplayName:  user.DisplayName,
		Role:         user.Role,
	}
}

func toUserProfile(user *memoryUser) *auth.UserProfile {
	return &auth.UserProfile{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		Email:       user.Email,
		Phone:       user.Phone,
		Wechat:      user.Wechat,
		Status:      user.Status,
	}
}

func validBcryptHash(hash string) bool {
	if len(hash) < 7 {
		return false
	}
	prefix := hash[:4]
	if prefix != "$2a$" && prefix != "$2b$" {
		return false
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("")); err == bcrypt.ErrMismatchedHashAndPassword {
		return true
	} else if err != nil {
		return false
	}
	return true
}
