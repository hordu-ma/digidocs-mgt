package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"digidocs-mgt/backend-go/internal/domain/auth"
	"digidocs-mgt/backend-go/internal/repository"
)

var mainlandMobileRE = regexp.MustCompile(`^1[3-9]\d{9}$`)

type AuthService struct {
	userRepo repository.UserAuthReader
	tokenSvc TokenService
}

func NewAuthService(userRepo repository.UserAuthReader, tokenSvc TokenService) AuthService {
	return AuthService{userRepo: userRepo, tokenSvc: tokenSvc}
}

// Login verifies credentials against username, email or phone and returns a signed JWT.
func (s AuthService) Login(ctx context.Context, identifier, password string) (string, auth.Claims, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" || password == "" {
		return "", auth.Claims{}, ErrUnauthorized
	}

	user, err := s.userRepo.FindUserByIdentifier(ctx, identifier)
	if errors.Is(err, ErrNotFound) {
		return "", auth.Claims{}, ErrUnauthorized
	}
	if err != nil {
		return "", auth.Claims{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", auth.Claims{}, ErrUnauthorized
	}

	return s.issueToken(user)
}

// Register creates a member account and immediately issues a JWT.
func (s AuthService) Register(ctx context.Context, email, phone, password, displayName string) (string, auth.Claims, error) {
	email = strings.TrimSpace(email)
	phone = strings.TrimSpace(phone)
	displayName = strings.TrimSpace(displayName)

	if email == "" && phone == "" {
		return "", auth.Claims{}, fmt.Errorf("%w: email or phone is required", ErrValidation)
	}
	if displayName == "" {
		return "", auth.Claims{}, fmt.Errorf("%w: display_name is required", ErrValidation)
	}
	if utf8.RuneCountInString(displayName) > 64 {
		return "", auth.Claims{}, fmt.Errorf("%w: display_name must not exceed 64 characters", ErrValidation)
	}
	if len(password) < 8 {
		return "", auth.Claims{}, fmt.Errorf("%w: password must be at least 8 characters", ErrValidation)
	}
	if len(password) > 72 {
		return "", auth.Claims{}, fmt.Errorf("%w: password exceeds max length", ErrValidation)
	}
	if email != "" && !isBasicEmail(email) {
		return "", auth.Claims{}, fmt.Errorf("%w: invalid email", ErrValidation)
	}
	if phone != "" && !mainlandMobileRE.MatchString(phone) {
		return "", auth.Claims{}, fmt.Errorf("%w: invalid phone", ErrValidation)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", auth.Claims{}, fmt.Errorf("failed to hash password: %w", err)
	}

	user, err := s.userRepo.RegisterUser(ctx, auth.RegisterUserInput{
		Email:        email,
		Phone:        phone,
		PasswordHash: string(hash),
		DisplayName:  displayName,
	})
	if err != nil {
		return "", auth.Claims{}, err
	}

	return s.issueToken(user)
}

func (s AuthService) issueToken(user *auth.UserRecord) (string, auth.Claims, error) {
	claims := auth.Claims{
		UserID:      user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Role:        user.Role,
	}

	token, err := s.tokenSvc.Generate(claims)
	if err != nil {
		return "", auth.Claims{}, err
	}
	return token, claims, nil
}

func isBasicEmail(email string) bool {
	at := strings.IndexByte(email, '@')
	if at <= 0 || at != strings.LastIndexByte(email, '@') || at == len(email)-1 {
		return false
	}
	if strings.ContainsAny(email, " \t\r\n") {
		return false
	}
	return strings.Contains(email[at+1:], ".")
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
