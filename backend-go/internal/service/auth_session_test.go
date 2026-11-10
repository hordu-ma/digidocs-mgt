package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"digidocs-mgt/backend-go/internal/domain/auth"
)

type sessionRepo struct {
	mockUserAuthReader
	status    string
	changedAt *time.Time
	err       error
	calls     int
}

func (r *sessionRepo) GetSessionState(context.Context, string) (string, *time.Time, error) {
	r.calls++
	return r.status, r.changedAt, r.err
}

func TestSessionValidRules(t *testing.T) {
	changed := time.Unix(1_800_000_000, 0)
	cases := []struct {
		name string
		repo *sessionRepo
		iat  int64
		want bool
	}{
		{"active, never changed", &sessionRepo{status: "active"}, 1, true},
		{"token before password change", &sessionRepo{status: "active", changedAt: &changed}, changed.Unix() - 1, false},
		{"token at/after password change", &sessionRepo{status: "active", changedAt: &changed}, changed.Unix(), true},
		{"disabled account", &sessionRepo{status: "disabled"}, 1, false},
		{"deleted user", &sessionRepo{err: ErrNotFound}, 1, false},
		{"lookup error fails open", &sessionRepo{err: errors.New("db down")}, 1, true},
	}
	for _, tc := range cases {
		svc := NewAuthService(tc.repo, NewTokenService("s"))
		if got := svc.SessionValid(context.Background(), auth.Claims{UserID: "u", IssuedAt: tc.iat}); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestSessionValidCachesLookups(t *testing.T) {
	repo := &sessionRepo{status: "active"}
	svc := NewAuthService(repo, NewTokenService("s"))
	for i := 0; i < 5; i++ {
		svc.SessionValid(context.Background(), auth.Claims{UserID: "u", IssuedAt: 1})
	}
	if repo.calls != 1 {
		t.Fatalf("expected one lookup within TTL, got %d", repo.calls)
	}
}

func TestTokenWithoutExpIsRejected(t *testing.T) {
	ts := NewTokenService("secret")
	token, err := ts.Generate(auth.Claims{UserID: "u"})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ts.Parse(token)
	if err != nil || claims.IssuedAt == 0 {
		t.Fatalf("issued token must parse with iat: %#v %v", claims, err)
	}
	// header.payload-without-exp, correctly signed with the same secret.
	unsigned := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidSJ9"
	if _, err := ts.Parse(unsigned + "." + ts.sign(unsigned)); err == nil {
		t.Fatal("a token without exp must be rejected")
	}
}
