package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"digidocs-mgt/backend-go/internal/domain/query"
)

type MembershipRepo interface {
	ListProjectMembers(ctx context.Context, projectID string) ([]query.ProjectMemberItem, error)
	AddProjectMember(ctx context.Context, projectID, userID, projectRole string) (*query.ProjectMemberItem, error)
	UpdateProjectMember(ctx context.Context, memberID, projectRole string) (*query.ProjectMemberItem, error)
	RemoveProjectMember(ctx context.Context, memberID string) error
	GetProjectMember(ctx context.Context, memberID string) (*query.ProjectMemberItem, error)
	FindProjectMember(ctx context.Context, projectID, userID string) (*query.ProjectMemberItem, error)
	CountOwners(ctx context.Context, projectID string) (int, error)
	CreateInvite(ctx context.Context, projectID, token, projectRole, createdBy string, expiresAt time.Time) (*query.ProjectInviteItem, error)
	ListInvites(ctx context.Context, projectID string) ([]query.ProjectInviteItem, error)
	GetInviteByID(ctx context.Context, projectID, inviteID string) (*query.ProjectInviteItem, error)
	GetInviteByToken(ctx context.Context, token string) (*query.ProjectInviteItem, error)
	RevokeInvite(ctx context.Context, projectID, inviteID, actorID string) (*query.ProjectInviteItem, error)
	GetInvitePreview(ctx context.Context, token string) (*query.ProjectInvitePreview, error)
}

type MembershipService struct {
	repo MembershipRepo
	perm PermissionService
}

func NewMembershipService(repo MembershipRepo, perm PermissionService) MembershipService {
	return MembershipService{repo: repo, perm: perm}
}

var managerAssignableRoles = map[string]bool{"contributor": true, "viewer": true, "manager": true}

func (s MembershipService) ListMembers(ctx context.Context, actorID, actorRole, projectID string) ([]query.ProjectMemberItem, error) {
	if projectID == "" {
		return nil, fmt.Errorf("%w: project_id is required", ErrValidation)
	}
	if err := s.perm.EnsureViewProjectMembers(ctx, actorID, actorRole, projectID); err != nil {
		return nil, err
	}
	return s.repo.ListProjectMembers(ctx, projectID)
}

func (s MembershipService) AddMember(ctx context.Context, actorID, actorRole, projectID, userID, projectRole string) (*query.ProjectMemberItem, error) {
	if projectID == "" || strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("%w: project_id and user_id are required", ErrValidation)
	}
	if err := s.perm.EnsureManageProjectMembers(ctx, actorID, actorRole, projectID); err != nil {
		return nil, err
	}
	role := strings.TrimSpace(projectRole)
	if role == "" {
		role = "contributor"
	}
	if actorRole != "admin" && role == "owner" {
		return nil, fmt.Errorf("%w: only admin can assign owner", ErrForbidden)
	}
	if actorRole != "admin" && !managerAssignableRoles[role] {
		return nil, fmt.Errorf("%w: project_role must be manager, contributor or viewer", ErrValidation)
	}
	if actorRole == "admin" && !allowedProjectRoles[role] {
		return nil, fmt.Errorf("%w: invalid project_role", ErrValidation)
	}
	item, err := s.repo.AddProjectMember(ctx, projectID, strings.TrimSpace(userID), role)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (s MembershipService) UpdateMember(ctx context.Context, actorID, actorRole, projectID, memberID, projectRole string) (*query.ProjectMemberItem, error) {
	if projectID == "" || memberID == "" || strings.TrimSpace(projectRole) == "" {
		return nil, fmt.Errorf("%w: project_id, member_id and project_role are required", ErrValidation)
	}
	if err := s.perm.EnsureManageProjectMembers(ctx, actorID, actorRole, projectID); err != nil {
		return nil, err
	}
	current, err := s.repo.GetProjectMember(ctx, memberID)
	if err != nil {
		return nil, err
	}
	role := strings.TrimSpace(projectRole)
	if actorRole != "admin" && (current.ProjectRole == "owner" || role == "owner") {
		return nil, fmt.Errorf("%w: only admin can change owner", ErrForbidden)
	}
	if actorRole != "admin" && !managerAssignableRoles[role] {
		return nil, fmt.Errorf("%w: invalid project_role", ErrValidation)
	}
	if current.ProjectRole == "owner" && role != "owner" {
		owners, err := s.repo.CountOwners(ctx, projectID)
		if err != nil {
			return nil, err
		}
		if owners <= 1 {
			return nil, fmt.Errorf("%w: cannot demote the last owner", ErrValidation)
		}
	}
	return s.repo.UpdateProjectMember(ctx, memberID, role)
}

func (s MembershipService) RemoveMember(ctx context.Context, actorID, actorRole, projectID, memberID string) error {
	if projectID == "" || memberID == "" {
		return fmt.Errorf("%w: project_id and member_id are required", ErrValidation)
	}
	if err := s.perm.EnsureManageProjectMembers(ctx, actorID, actorRole, projectID); err != nil {
		return err
	}
	current, err := s.repo.GetProjectMember(ctx, memberID)
	if err != nil {
		return err
	}
	if current.ProjectRole == "owner" {
		owners, err := s.repo.CountOwners(ctx, projectID)
		if err != nil {
			return err
		}
		if owners <= 1 {
			return fmt.Errorf("%w: cannot remove the last owner", ErrValidation)
		}
		if actorRole != "admin" {
			return fmt.Errorf("%w: only admin can remove an owner", ErrForbidden)
		}
	}
	return s.repo.RemoveProjectMember(ctx, memberID)
}

func (s MembershipService) CreateInvite(ctx context.Context, actorID, actorRole, projectID string) (*query.ProjectInviteItem, error) {
	if projectID == "" {
		return nil, fmt.Errorf("%w: project_id is required", ErrValidation)
	}
	if err := s.perm.EnsureManageProjectMembers(ctx, actorID, actorRole, projectID); err != nil {
		return nil, err
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, fmt.Errorf("failed to generate invite token: %w", err)
	}
	return s.repo.CreateInvite(ctx, projectID, hex.EncodeToString(raw), "contributor", actorID, time.Now().Add(7*24*time.Hour))
}

func (s MembershipService) ListInvites(ctx context.Context, actorID, actorRole, projectID string) ([]query.ProjectInviteItem, error) {
	if projectID == "" {
		return nil, fmt.Errorf("%w: project_id is required", ErrValidation)
	}
	if err := s.perm.EnsureManageProjectMembers(ctx, actorID, actorRole, projectID); err != nil {
		return nil, err
	}
	return s.repo.ListInvites(ctx, projectID)
}

func (s MembershipService) RevokeInvite(ctx context.Context, actorID, actorRole, projectID, inviteID string) (*query.ProjectInviteItem, error) {
	if projectID == "" || inviteID == "" {
		return nil, fmt.Errorf("%w: project_id and invite_id are required", ErrValidation)
	}
	if err := s.perm.EnsureManageProjectMembers(ctx, actorID, actorRole, projectID); err != nil {
		return nil, err
	}
	return s.repo.RevokeInvite(ctx, projectID, inviteID, actorID)
}

func (s MembershipService) PreviewInvite(ctx context.Context, token string) (*query.ProjectInvitePreview, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("%w: token is required", ErrValidation)
	}
	return s.repo.GetInvitePreview(ctx, token)
}

func (s MembershipService) AcceptInvite(ctx context.Context, actorID, token string) (*query.ProjectMemberItem, error) {
	token = strings.TrimSpace(token)
	if actorID == "" || token == "" {
		return nil, fmt.Errorf("%w: token is required", ErrValidation)
	}
	invite, err := s.repo.GetInviteByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if invite.Status != "active" {
		return nil, fmt.Errorf("%w: invite is not active", ErrValidation)
	}
	expires, err := time.Parse(time.RFC3339, invite.ExpiresAt)
	if err != nil {
		return nil, err
	}
	if time.Now().UTC().After(expires.UTC()) {
		return nil, fmt.Errorf("%w: invite has expired", ErrValidation)
	}
	existing, err := s.repo.FindProjectMember(ctx, invite.ProjectID, actorID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	return s.repo.AddProjectMember(ctx, invite.ProjectID, actorID, invite.ProjectRole)
}
