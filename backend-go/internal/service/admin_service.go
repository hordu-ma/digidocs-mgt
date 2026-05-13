package service

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"digidocs-mgt/backend-go/internal/domain/query"
)

// AdminRepo defines the persistence operations the admin service needs.
type AdminRepo interface {
	CreateTeamSpace(ctx context.Context, name, code, description, createdBy string) (*query.TeamSpaceSummary, error)
	CreateProjectWithOwner(ctx context.Context, teamSpaceID, name, code, description, ownerID string) (*query.ProjectSummary, error)
	CreateUser(ctx context.Context, username, passwordHash, displayName, role, email, phone string) (*query.UserOption, error)
	UpdateUser(ctx context.Context, userID string, fields map[string]any) (*query.UserOption, error)
	ListAllUsers(ctx context.Context) ([]query.UserOption, error)
	ListProjectMembers(ctx context.Context, projectID string) ([]query.ProjectMemberItem, error)
	AddProjectMember(ctx context.Context, projectID, userID, projectRole string) (*query.ProjectMemberItem, error)
	UpdateProjectMember(ctx context.Context, memberID, projectRole string) (*query.ProjectMemberItem, error)
	RemoveProjectMember(ctx context.Context, memberID string) error
}

type AdminService struct {
	repo AdminRepo
}

func NewAdminService(repo AdminRepo) AdminService {
	return AdminService{repo: repo}
}

// --- Team Space ---

type CreateTeamSpaceInput struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	ActorID     string `json:"-"`
}

func (s AdminService) CreateTeamSpace(ctx context.Context, input CreateTeamSpaceInput) (*query.TeamSpaceSummary, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Code = strings.TrimSpace(input.Code)
	if input.Name == "" || input.Code == "" {
		return nil, fmt.Errorf("%w: name and code are required", ErrValidation)
	}
	if utf8.RuneCountInString(input.Name) > 128 || len(input.Code) > 64 {
		return nil, fmt.Errorf("%w: name or code too long", ErrValidation)
	}
	return s.repo.CreateTeamSpace(ctx, input.Name, input.Code, input.Description, input.ActorID)
}

// --- Project ---

type CreateProjectInput struct {
	TeamSpaceID string `json:"team_space_id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	OwnerID     string `json:"owner_id"`
	Description string `json:"description"`
}

func (s AdminService) CreateProject(ctx context.Context, input CreateProjectInput) (*query.ProjectSummary, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Code = strings.TrimSpace(input.Code)
	if input.TeamSpaceID == "" || input.Name == "" || input.Code == "" || input.OwnerID == "" {
		return nil, fmt.Errorf("%w: team_space_id, name, code and owner_id are required", ErrValidation)
	}
	return s.repo.CreateProjectWithOwner(ctx, input.TeamSpaceID, input.Name, input.Code, input.Description, input.OwnerID)
}

// --- User ---

type CreateUserInput struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
}

var allowedRoles = map[string]bool{"admin": true, "project_lead": true, "member": true}

func (s AdminService) CreateUser(ctx context.Context, input CreateUserInput) (*query.UserOption, error) {
	input.Username = strings.TrimSpace(input.Username)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Role = strings.TrimSpace(input.Role)
	if input.Username == "" || input.Password == "" || input.DisplayName == "" {
		return nil, fmt.Errorf("%w: username, password and display_name are required", ErrValidation)
	}
	if !allowedRoles[input.Role] {
		return nil, fmt.Errorf("%w: role must be admin, project_lead or member", ErrValidation)
	}
	if len(input.Password) > 72 {
		return nil, fmt.Errorf("%w: password exceeds max length", ErrValidation)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	return s.repo.CreateUser(ctx, input.Username, string(hash), input.DisplayName, input.Role, input.Email, input.Phone)
}

type UpdateUserInput struct {
	UserID      string  `json:"-"`
	DisplayName *string `json:"display_name,omitempty"`
	Role        *string `json:"role,omitempty"`
	Email       *string `json:"email,omitempty"`
	Phone       *string `json:"phone,omitempty"`
	Status      *string `json:"status,omitempty"`
	Password    *string `json:"password,omitempty"`
}

func (s AdminService) UpdateUser(ctx context.Context, input UpdateUserInput) (*query.UserOption, error) {
	if input.UserID == "" {
		return nil, fmt.Errorf("%w: user_id is required", ErrValidation)
	}
	fields := make(map[string]any)
	if input.DisplayName != nil {
		v := strings.TrimSpace(*input.DisplayName)
		if v == "" {
			return nil, fmt.Errorf("%w: display_name cannot be empty", ErrValidation)
		}
		fields["display_name"] = v
	}
	if input.Role != nil {
		if !allowedRoles[*input.Role] {
			return nil, fmt.Errorf("%w: invalid role", ErrValidation)
		}
		fields["role"] = *input.Role
	}
	if input.Email != nil {
		fields["email"] = strings.TrimSpace(*input.Email)
	}
	if input.Phone != nil {
		fields["phone"] = strings.TrimSpace(*input.Phone)
	}
	if input.Status != nil {
		v := *input.Status
		if v != "active" && v != "inactive" {
			return nil, fmt.Errorf("%w: status must be active or inactive", ErrValidation)
		}
		fields["status"] = v
	}
	if input.Password != nil {
		v := *input.Password
		if v == "" {
			return nil, fmt.Errorf("%w: password cannot be empty", ErrValidation)
		}
		if len(v) > 72 {
			return nil, fmt.Errorf("%w: password exceeds max length", ErrValidation)
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(v), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		fields["password_hash"] = string(hash)
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("%w: no fields to update", ErrValidation)
	}
	return s.repo.UpdateUser(ctx, input.UserID, fields)
}

func (s AdminService) ListAllUsers(ctx context.Context) ([]query.UserOption, error) {
	return s.repo.ListAllUsers(ctx)
}

// --- Project Members ---

var allowedProjectRoles = map[string]bool{"owner": true, "manager": true, "contributor": true, "viewer": true}

type AddProjectMemberInput struct {
	ProjectID   string `json:"project_id"`
	UserID      string `json:"user_id"`
	ProjectRole string `json:"project_role"`
}

func (s AdminService) ListProjectMembers(ctx context.Context, projectID string) ([]query.ProjectMemberItem, error) {
	if projectID == "" {
		return nil, fmt.Errorf("%w: project_id is required", ErrValidation)
	}
	return s.repo.ListProjectMembers(ctx, projectID)
}

func (s AdminService) AddProjectMember(ctx context.Context, input AddProjectMemberInput) (*query.ProjectMemberItem, error) {
	if input.ProjectID == "" || input.UserID == "" || input.ProjectRole == "" {
		return nil, fmt.Errorf("%w: project_id, user_id and project_role are required", ErrValidation)
	}
	if !allowedProjectRoles[input.ProjectRole] {
		return nil, fmt.Errorf("%w: project_role must be owner, manager, contributor or viewer", ErrValidation)
	}
	return s.repo.AddProjectMember(ctx, input.ProjectID, input.UserID, input.ProjectRole)
}

func (s AdminService) UpdateProjectMember(ctx context.Context, memberID, projectRole string) (*query.ProjectMemberItem, error) {
	if memberID == "" || projectRole == "" {
		return nil, fmt.Errorf("%w: member_id and project_role are required", ErrValidation)
	}
	if !allowedProjectRoles[projectRole] {
		return nil, fmt.Errorf("%w: invalid project_role", ErrValidation)
	}
	return s.repo.UpdateProjectMember(ctx, memberID, projectRole)
}

func (s AdminService) RemoveProjectMember(ctx context.Context, memberID string) error {
	if memberID == "" {
		return fmt.Errorf("%w: member_id is required", ErrValidation)
	}
	return s.repo.RemoveProjectMember(ctx, memberID)
}
