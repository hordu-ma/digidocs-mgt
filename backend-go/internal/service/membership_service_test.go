package service

import (
	"context"
	"testing"
	"time"

	"digidocs-mgt/backend-go/internal/domain/query"
)

type stubMembershipRepo struct {
	members []query.ProjectMemberItem
	invites []query.ProjectInviteItem
}

func (s *stubMembershipRepo) ListProjectMembers(context.Context, string) ([]query.ProjectMemberItem, error) {
	return s.members, nil
}
func (s *stubMembershipRepo) AddProjectMember(_ context.Context, _, userID, role string) (*query.ProjectMemberItem, error) {
	item := query.ProjectMemberItem{ID: "m-new", UserID: userID, ProjectRole: role}
	s.members = append(s.members, item)
	return &item, nil
}
func (s *stubMembershipRepo) UpdateProjectMember(_ context.Context, memberID, role string) (*query.ProjectMemberItem, error) {
	return &query.ProjectMemberItem{ID: memberID, ProjectRole: role}, nil
}
func (s *stubMembershipRepo) RemoveProjectMember(context.Context, string) error { return nil }
func (s *stubMembershipRepo) GetProjectMember(_ context.Context, memberID string) (*query.ProjectMemberItem, error) {
	for _, m := range s.members {
		if m.ID == memberID {
			return &m, nil
		}
	}
	return nil, ErrNotFound
}
func (s *stubMembershipRepo) FindProjectMember(_ context.Context, _, userID string) (*query.ProjectMemberItem, error) {
	for _, m := range s.members {
		if m.UserID == userID {
			return &m, nil
		}
	}
	return nil, ErrNotFound
}
func (s *stubMembershipRepo) CountOwners(context.Context, string) (int, error) {
	n := 0
	for _, m := range s.members {
		if m.ProjectRole == "owner" {
			n++
		}
	}
	return n, nil
}
func (s *stubMembershipRepo) CreateInvite(_ context.Context, projectID, token, role, _ string, expires time.Time) (*query.ProjectInviteItem, error) {
	item := query.ProjectInviteItem{
		ID: "inv-1", ProjectID: projectID, Token: token, ProjectRole: role,
		Status: "active", ExpiresAt: expires.UTC().Format(time.RFC3339),
	}
	s.invites = append(s.invites, item)
	return &item, nil
}
func (s *stubMembershipRepo) ListInvites(context.Context, string) ([]query.ProjectInviteItem, error) {
	return s.invites, nil
}
func (s *stubMembershipRepo) GetInviteByID(context.Context, string, string) (*query.ProjectInviteItem, error) {
	return &s.invites[0], nil
}
func (s *stubMembershipRepo) GetInviteByToken(_ context.Context, token string) (*query.ProjectInviteItem, error) {
	for i := range s.invites {
		if s.invites[i].Token == token {
			return &s.invites[i], nil
		}
	}
	return nil, ErrNotFound
}
func (s *stubMembershipRepo) RevokeInvite(context.Context, string, string, string) (*query.ProjectInviteItem, error) {
	s.invites[0].Status = "revoked"
	return &s.invites[0], nil
}
func (s *stubMembershipRepo) GetInvitePreview(context.Context, string) (*query.ProjectInvitePreview, error) {
	return &query.ProjectInvitePreview{ProjectName: "课题A", Status: "active"}, nil
}

func TestMembershipServiceAddAndInvite(t *testing.T) {
	repo := &stubMembershipRepo{members: []query.ProjectMemberItem{{ID: "m-1", UserID: "owner", ProjectRole: "owner"}}}
	svc := NewMembershipService(repo, NewPermissionService(mockPermissionReader{ok: true}))

	added, err := svc.AddMember(context.Background(), "owner", "member", "p-1", "u-2", "")
	if err != nil || added.ProjectRole != "contributor" {
		t.Fatalf("AddMember got %#v err=%v", added, err)
	}
	if _, err := svc.AddMember(context.Background(), "owner", "member", "p-1", "u-3", "owner"); err == nil {
		t.Fatal("non-admin should not assign owner")
	}

	invite, err := svc.CreateInvite(context.Background(), "owner", "member", "p-1")
	if err != nil || invite.Token == "" || invite.ProjectRole != "contributor" {
		t.Fatalf("CreateInvite got %#v err=%v", invite, err)
	}

	joined, err := svc.AcceptInvite(context.Background(), "u-9", invite.Token)
	if err != nil || joined.UserID != "u-9" {
		t.Fatalf("AcceptInvite got %#v err=%v", joined, err)
	}
	again, err := svc.AcceptInvite(context.Background(), "u-9", invite.Token)
	if err != nil || again.UserID != "u-9" {
		t.Fatalf("idempotent accept got %#v err=%v", again, err)
	}

	if err := svc.RemoveMember(context.Background(), "owner", "member", "p-1", "m-1"); err == nil {
		t.Fatal("should not remove last owner")
	}
}

func TestMembershipServiceRequiresPermission(t *testing.T) {
	svc := NewMembershipService(&stubMembershipRepo{}, NewPermissionService(mockPermissionReader{ok: false}))
	if _, err := svc.AddMember(context.Background(), "u", "member", "p-1", "u-2", "contributor"); err == nil {
		t.Fatal("expected forbidden")
	}
}
