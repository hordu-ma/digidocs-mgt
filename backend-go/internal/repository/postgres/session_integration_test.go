package postgres_test

import (
	"context"
	"testing"
	"time"

	pgrepo "digidocs-mgt/backend-go/internal/repository/postgres"
	"digidocs-mgt/backend-go/internal/service"

	"golang.org/x/crypto/bcrypt"
)

func TestPostgresIntegrationPasswordChangeRevokesSessions(t *testing.T) {
	ctx := context.Background()
	database := openIntegrationDB(t)
	seedIntegrationBaseRows(t, ctx, database)
	const userID = "00000000-0000-0000-0000-000000000901"
	hash, _ := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.MinCost)
	if _, err := database.ExecContext(ctx, `INSERT INTO users(id,username,password_hash,display_name,role,status) VALUES($1,'sess','`+string(hash)+`','会话','member','active')`, userID); err != nil {
		t.Fatal(err)
	}
	tokens := service.NewTokenService("integration-secret")
	authSvc := service.NewAuthService(pgrepo.NewUserAuthRepository(database), tokens)

	oldToken, _, err := authSvc.Login(ctx, "sess", "old-password")
	if err != nil {
		t.Fatal(err)
	}
	oldClaims, _ := tokens.Parse(oldToken)
	if !authSvc.SessionValid(ctx, oldClaims) {
		t.Fatal("fresh token must be valid")
	}
	// JWT iat has one-second resolution; step past the issuing second.
	time.Sleep(1100 * time.Millisecond)
	if err := authSvc.ChangePassword(ctx, userID, "old-password", "new-password-123"); err != nil {
		t.Fatal(err)
	}
	if authSvc.SessionValid(ctx, oldClaims) {
		t.Fatal("token issued before the password change must be revoked")
	}
	newToken, err := authSvc.ReissueToken(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	newClaims, _ := tokens.Parse(newToken)
	if !authSvc.SessionValid(ctx, newClaims) {
		t.Fatal("reissued token must be valid")
	}

	if _, err := database.ExecContext(ctx, `UPDATE users SET status='disabled' WHERE id=$1`, userID); err != nil {
		t.Fatal(err)
	}
	fresh := service.NewAuthService(pgrepo.NewUserAuthRepository(database), tokens) // no cached state
	if fresh.SessionValid(ctx, newClaims) {
		t.Fatal("a disabled account's token must be rejected")
	}
}
