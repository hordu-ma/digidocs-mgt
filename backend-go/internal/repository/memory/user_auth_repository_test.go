package memory

import (
	"context"
	"errors"
	"testing"

	"digidocs-mgt/backend-go/internal/domain/auth"
	"digidocs-mgt/backend-go/internal/service"
)

func TestUserAuthRepositoryRegisterAndFind(t *testing.T) {
	repo := NewUserAuthRepository()
	ctx := context.Background()

	admin, err := repo.FindUserByIdentifier(ctx, "admin")
	if err != nil || admin.Username != "admin" {
		t.Fatalf("default admin lookup got %#v err=%v", admin, err)
	}
	if _, err := repo.FindUserByIdentifier(ctx, "admin@example.com"); err != nil {
		t.Fatalf("default admin email lookup: %v", err)
	}

	created, err := repo.RegisterUser(ctx, auth.RegisterUserInput{
		Email:        "member@example.com",
		Phone:        "13900139000",
		PasswordHash: defaultDevHash,
		DisplayName:  "成员",
	})
	if err != nil {
		t.Fatalf("RegisterUser: %v", err)
	}
	if created.Username != "" || created.Role != "member" {
		t.Fatalf("created = %#v", created)
	}

	if _, err := repo.RegisterUser(ctx, auth.RegisterUserInput{
		Email:        "member@example.com",
		PasswordHash: defaultDevHash,
		DisplayName:  "重复",
	}); !errors.Is(err, service.ErrConflict) {
		t.Fatalf("duplicate email err=%v, want ErrConflict", err)
	}

	found, err := repo.FindUserByIdentifier(ctx, "13900139000")
	if err != nil || found.ID != created.ID {
		t.Fatalf("phone lookup got %#v err=%v", found, err)
	}

	profile, err := repo.GetUserProfile(ctx, created.ID)
	if err != nil || profile.Email != "member@example.com" || profile.Username != "" {
		t.Fatalf("GetUserProfile got %#v err=%v", profile, err)
	}
}
