package service

import (
	"context"
	"errors"
	"strings"

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

func (s AuthService) GetProfile(ctx context.Context, userID string) (*auth.UserProfile, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, ErrUnauthorized
	}
	return s.userRepo.GetUserProfile(ctx, userID)
}

func (s AuthService) UpdateProfile(ctx context.Context, userID string, input auth.ProfileUpdateInput) (*auth.UserProfile, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, ErrUnauthorized
	}

	normalized := auth.ProfileUpdateInput{
		DisplayName: strings.TrimSpace(input.DisplayName),
		Email:       strings.TrimSpace(input.Email),
		Phone:       strings.TrimSpace(input.Phone),
		Wechat:      strings.TrimSpace(input.Wechat),
	}
	if normalized.DisplayName == "" {
		return nil, ErrValidation
	}
	if len([]rune(normalized.DisplayName)) > 64 ||
		len(normalized.Email) > 128 ||
		len(normalized.Phone) > 32 ||
		len(normalized.Wechat) > 64 {
		return nil, ErrValidation
	}
	if normalized.Email != "" && !strings.Contains(normalized.Email, "@") {
		return nil, ErrValidation
	}

	return s.userRepo.UpdateUserProfile(ctx, userID, normalized)
}
