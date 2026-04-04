package service

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"digidocs-mgt/backend-go/internal/domain/auth"
	"digidocs-mgt/backend-go/internal/repository"
)

type AuthService struct {
	userRepo repository.UserAuthReader
	tokenSvc TokenService
}

func NewAuthService(userRepo repository.UserAuthReader, tokenSvc TokenService) AuthService {
	return AuthService{userRepo: userRepo, tokenSvc: tokenSvc}
}

// Login verifies credentials and returns a signed JWT plus the resolved claims.
func (s AuthService) Login(ctx context.Context, username, password string) (string, auth.Claims, error) {
	user, err := s.userRepo.FindUserByUsername(ctx, username)
	if errors.Is(err, ErrNotFound) {
		return "", auth.Claims{}, ErrUnauthorized
	}
	if err != nil {
		return "", auth.Claims{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", auth.Claims{}, ErrUnauthorized
	}

	claims := auth.Claims{
		UserID:      user.ID,
		Username:    username,
		DisplayName: user.DisplayName,
		Role:        user.Role,
	}

	token, err := s.tokenSvc.Generate(claims)
	if err != nil {
		return "", auth.Claims{}, err
	}

	return token, claims, nil
}
