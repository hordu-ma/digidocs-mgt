package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"digidocs-mgt/backend-go/internal/domain/auth"
)

// --- mock: UserAuthReader ---

type mockUserAuthReader struct {
	user         *auth.UserRecord
	profile      *auth.UserProfile
	err          error
	registerErr  error
	lastLookup   string
	registered   *auth.RegisterUserInput
	registerUser *auth.UserRecord
}

func (m *mockUserAuthReader) FindUserByIdentifier(_ context.Context, identifier string) (*auth.UserRecord, error) {
	m.lastLookup = identifier
	return m.user, m.err
}

func (m *mockUserAuthReader) RegisterUser(_ context.Context, input auth.RegisterUserInput) (*auth.UserRecord, error) {
	m.registered = &input
	if m.registerErr != nil {
		return nil, m.registerErr
	}
	if m.registerUser != nil {
		return m.registerUser, nil
	}
	return &auth.UserRecord{
		ID:           "u-new",
		Username:     "",
		PasswordHash: input.PasswordHash,
		DisplayName:  input.DisplayName,
		Role:         "member",
	}, nil
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

func newAuthService(repo *mockUserAuthReader) (AuthService, TokenService) {
	tokenSvc := NewTokenService("test-secret")
	return NewAuthService(repo, tokenSvc), tokenSvc
}

// --- tests ---

func TestLogin_SuccessUsesStoredUsername(t *testing.T) {
	password := "correct-password"
	repo := &mockUserAuthReader{
		user: &auth.UserRecord{
			ID:           "u-1",
			Username:     "alice",
			PasswordHash: hashPassword(t, password),
			DisplayName:  "Alice",
			Role:         "admin",
		},
	}
	svc, tokenSvc := newAuthService(repo)

	token, claims, err := svc.Login(context.Background(), "alice@example.com", password)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if repo.lastLookup != "alice@example.com" {
		t.Errorf("lookup identifier = %q, want email", repo.lastLookup)
	}
	if claims.UserID != "u-1" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "u-1")
	}
	if claims.Username != "alice" {
		t.Errorf("Username = %q, want stored username, not login identifier", claims.Username)
	}
	if claims.DisplayName != "Alice" {
		t.Errorf("DisplayName = %q, want %q", claims.DisplayName, "Alice")
	}
	if claims.Role != "admin" {
		t.Errorf("Role = %q, want %q", claims.Role, "admin")
	}

	parsed, err := tokenSvc.Parse(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if parsed.UserID != claims.UserID || parsed.Username != "alice" {
		t.Errorf("parsed claims = %#v, want stored username", parsed)
	}
}

func TestLogin_AcceptsUsernameEmailAndPhone(t *testing.T) {
	password := "pw-123456"
	repo := &mockUserAuthReader{
		user: &auth.UserRecord{
			ID:           "u-1",
			Username:     "zhangsan",
			PasswordHash: hashPassword(t, password),
			DisplayName:  "张三",
			Role:         "member",
		},
	}
	svc, _ := newAuthService(repo)

	for _, identifier := range []string{"zhangsan", "zhangsan@example.com", "13800138000"} {
		if _, claims, err := svc.Login(context.Background(), identifier, password); err != nil {
			t.Fatalf("login %q: %v", identifier, err)
		} else if claims.Username != "zhangsan" {
			t.Fatalf("login %q username = %q, want zhangsan", identifier, claims.Username)
		}
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	repo := &mockUserAuthReader{err: ErrNotFound}
	svc, _ := newAuthService(repo)

	_, _, err := svc.Login(context.Background(), "nobody", "pw")
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("err = %v, want ErrUnauthorized", err)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := &mockUserAuthReader{
		user: &auth.UserRecord{
			ID:           "u-2",
			Username:     "bob",
			PasswordHash: hashPassword(t, "real-password"),
			DisplayName:  "Bob",
			Role:         "user",
		},
	}
	svc, _ := newAuthService(repo)

	_, _, err := svc.Login(context.Background(), "bob", "wrong-password")
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("err = %v, want ErrUnauthorized", err)
	}
}

func TestLogin_RepoError(t *testing.T) {
	repoErr := errors.New("db connection lost")
	repo := &mockUserAuthReader{err: repoErr}
	svc, _ := newAuthService(repo)

	_, _, err := svc.Login(context.Background(), "alice", "pw")
	if !errors.Is(err, repoErr) {
		t.Errorf("err = %v, want %v", err, repoErr)
	}
}

func TestRegister_SuccessIssuesToken(t *testing.T) {
	repo := &mockUserAuthReader{}
	svc, tokenSvc := newAuthService(repo)

	token, claims, err := svc.Register(context.Background(), "user@example.com", "13800138000", "password1", "新用户")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected token")
	}
	if claims.Role != "member" || claims.Username != "" || claims.DisplayName != "新用户" {
		t.Fatalf("unexpected claims: %#v", claims)
	}
	if repo.registered == nil || repo.registered.Email != "user@example.com" || repo.registered.Phone != "13800138000" {
		t.Fatalf("unexpected register payload: %#v", repo.registered)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(repo.registered.PasswordHash), []byte("password1")); err != nil {
		t.Fatalf("stored hash does not match password: %v", err)
	}
	if _, err := tokenSvc.Parse(token); err != nil {
		t.Fatalf("parse token: %v", err)
	}
}

func TestRegister_ValidationBranches(t *testing.T) {
	svc, _ := newAuthService(&mockUserAuthReader{})

	cases := []struct {
		name        string
		email       string
		phone       string
		password    string
		displayName string
	}{
		{name: "missing contact", password: "password1", displayName: "用户"},
		{name: "short password", email: "a@b.com", password: "short", displayName: "用户"},
		{name: "long password", email: "a@b.com", password: strings.Repeat("x", 73), displayName: "用户"},
		{name: "bad phone", phone: "12345", password: "password1", displayName: "用户"},
		{name: "bad email", email: "not-an-email", password: "password1", displayName: "用户"},
		{name: "missing display name", email: "a@b.com", password: "password1"},
		{name: "long display name", email: "a@b.com", password: "password1", displayName: strings.Repeat("名", 65)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := svc.Register(context.Background(), tc.email, tc.phone, tc.password, tc.displayName)
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("err = %v, want ErrValidation", err)
			}
		})
	}
}

func TestRegister_ConflictPropagates(t *testing.T) {
	repo := &mockUserAuthReader{registerErr: fmt.Errorf("%w: 该邮箱已被注册", ErrConflict)}
	svc, _ := newAuthService(repo)

	_, _, err := svc.Register(context.Background(), "user@example.com", "", "password1", "用户")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("err = %v, want ErrConflict", err)
	}
}

func TestUpdateProfile_Success(t *testing.T) {
	svc, _ := newAuthService(&mockUserAuthReader{})

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
	svc, _ := newAuthService(&mockUserAuthReader{})

	_, err := svc.UpdateProfile(context.Background(), "u-1", auth.ProfileUpdateInput{
		DisplayName: " ",
	})
	if !errors.Is(err, ErrValidation) {
		t.Errorf("err = %v, want ErrValidation", err)
	}
}
