package service

import (
	"context"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"digidocs-mgt/backend-go/internal/domain/auth"
)

// --- mock: UserAuthReader ---

type mockUserAuthReader struct {
	user    *auth.UserRecord
	profile *auth.UserProfile
	err     error
}

func (m *mockUserAuthReader) FindUserByUsername(_ context.Context, _ string) (*auth.UserRecord, error) {
	return m.user, m.err
}

func (m *mockUserAuthReader) GetUserProfile(_ context.Context, _ string) (*auth.UserProfile, error) {
	return m.profile, m.err
}

func (m *mockUserAuthReader) UpdateUserProfile(_ context.Context, userID string, input auth.ProfileUpdateInput) (*auth.UserProfile, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &auth.UserProfile{
		ID:          userID,
		Username:    "alice",
		DisplayName: input.DisplayName,
		Role:        "admin",
		Email:       input.Email,
		Phone:       input.Phone,
		Wechat:      input.Wechat,
		Status:      "active",
	}, nil
}

// --- helpers ---

func hashPassword(t *testing.T, plain string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt hash: %v", err)
	}
	return string(h)
}

// --- tests ---

func TestLogin_Success(t *testing.T) {
	password := "correct-password"
	repo := &mockUserAuthReader{
		user: &auth.UserRecord{
			ID:           "u-1",
			PasswordHash: hashPassword(t, password),
			DisplayName:  "Alice",
			Role:         "admin",
		},
	}
	tokenSvc := NewTokenService("test-secret")
	svc := NewAuthService(repo, tokenSvc)

	token, claims, err := svc.Login(context.Background(), "alice", password)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if claims.UserID != "u-1" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "u-1")
	}
	if claims.Username != "alice" {
		t.Errorf("Username = %q, want %q", claims.Username, "alice")
	}
	if claims.DisplayName != "Alice" {
		t.Errorf("DisplayName = %q, want %q", claims.DisplayName, "Alice")
	}
	if claims.Role != "admin" {
		t.Errorf("Role = %q, want %q", claims.Role, "admin")
	}

	// token should be parseable
	parsed, err := tokenSvc.Parse(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if parsed.UserID != claims.UserID {
		t.Errorf("parsed UserID = %q, want %q", parsed.UserID, claims.UserID)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	repo := &mockUserAuthReader{err: ErrNotFound}
	svc := NewAuthService(repo, NewTokenService("s"))

	_, _, err := svc.Login(context.Background(), "nobody", "pw")
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("err = %v, want ErrUnauthorized", err)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := &mockUserAuthReader{
		user: &auth.UserRecord{
			ID:           "u-2",
			PasswordHash: hashPassword(t, "real-password"),
			DisplayName:  "Bob",
			Role:         "user",
		},
	}
	svc := NewAuthService(repo, NewTokenService("s"))

	_, _, err := svc.Login(context.Background(), "bob", "wrong-password")
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("err = %v, want ErrUnauthorized", err)
	}
}

func TestLogin_RepoError(t *testing.T) {
	repoErr := errors.New("db connection lost")
	repo := &mockUserAuthReader{err: repoErr}
	svc := NewAuthService(repo, NewTokenService("s"))

	_, _, err := svc.Login(context.Background(), "alice", "pw")
	if !errors.Is(err, repoErr) {
		t.Errorf("err = %v, want %v", err, repoErr)
	}
}

func TestUpdateProfile_Success(t *testing.T) {
	svc := NewAuthService(&mockUserAuthReader{}, NewTokenService("s"))

	profile, err := svc.UpdateProfile(context.Background(), "u-1", auth.ProfileUpdateInput{
		DisplayName: "Alice",
		Email:       "alice@example.com",
		Phone:       "13900000000",
		Wechat:      "alice_wechat",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.DisplayName != "Alice" || profile.Email != "alice@example.com" {
		t.Fatalf("unexpected profile: %#v", profile)
	}
}

func TestUpdateProfile_RequiresDisplayName(t *testing.T) {
	svc := NewAuthService(&mockUserAuthReader{}, NewTokenService("s"))

	_, err := svc.UpdateProfile(context.Background(), "u-1", auth.ProfileUpdateInput{
		DisplayName: " ",
	})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}
