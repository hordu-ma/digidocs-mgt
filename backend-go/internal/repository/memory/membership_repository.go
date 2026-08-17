package memory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/service"
)

type memoryMember struct {
	query.ProjectMemberItem
	ProjectID string
}

type memoryInvite struct {
	query.ProjectInviteItem
}

type MembershipRepository struct {
	mu      sync.RWMutex
	members []memoryMember
	invites []memoryInvite
	seq     int
}

func NewMembershipRepository() *MembershipRepository {
	return &MembershipRepository{
		members: []memoryMember{
			{
				ProjectID: "00000000-0000-0000-0000-000000000020",
				ProjectMemberItem: query.ProjectMemberItem{
					ID:          "mem-owner-1",
					UserID:      "00000000-0000-0000-0000-000000000001",
					DisplayName: "开发管理员",
					Username:    "admin",
					ProjectRole: "owner",
				},
			},
		},
	}
}

func (r *MembershipRepository) nextID(prefix string) string {
	r.seq++
	return fmt.Sprintf("%s-%d", prefix, r.seq)
}

func (r *MembershipRepository) ListProjectMembers(_ context.Context, projectID string) ([]query.ProjectMemberItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]query.ProjectMemberItem, 0)
	for _, m := range r.members {
		if m.ProjectID == projectID {
			items = append(items, m.ProjectMemberItem)
		}
	}
	return items, nil
}

func (r *MembershipRepository) AddProjectMember(_ context.Context, projectID, userID, projectRole string) (*query.ProjectMemberItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, m := range r.members {
		if m.ProjectID == projectID && m.UserID == userID {
			return nil, service.ErrConflict
		}
	}
	item := query.ProjectMemberItem{
		ID:          r.nextID("mem"),
		UserID:      userID,
		DisplayName: userID,
		Username:    "",
		ProjectRole: projectRole,
	}
	r.members = append(r.members, memoryMember{ProjectID: projectID, ProjectMemberItem: item})
	return &item, nil
}

func (r *MembershipRepository) UpdateProjectMember(_ context.Context, memberID, projectRole string) (*query.ProjectMemberItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, m := range r.members {
		if m.ID == memberID {
			r.members[i].ProjectRole = projectRole
			item := r.members[i].ProjectMemberItem
			return &item, nil
		}
	}
	return nil, service.ErrNotFound
}

func (r *MembershipRepository) RemoveProjectMember(_ context.Context, memberID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, m := range r.members {
		if m.ID == memberID {
			r.members = append(r.members[:i], r.members[i+1:]...)
			return nil
		}
	}
	return service.ErrNotFound
}

func (r *MembershipRepository) GetProjectMember(_ context.Context, memberID string) (*query.ProjectMemberItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, m := range r.members {
		if m.ID == memberID {
			item := m.ProjectMemberItem
			return &item, nil
		}
	}
	return nil, service.ErrNotFound
}

func (r *MembershipRepository) FindProjectMember(_ context.Context, projectID, userID string) (*query.ProjectMemberItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, m := range r.members {
		if m.ProjectID == projectID && m.UserID == userID {
			item := m.ProjectMemberItem
			return &item, nil
		}
	}
	return nil, service.ErrNotFound
}

func (r *MembershipRepository) CountOwners(_ context.Context, projectID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n := 0
	for _, m := range r.members {
		if m.ProjectID == projectID && m.ProjectRole == "owner" {
			n++
		}
	}
	return n, nil
}

func (r *MembershipRepository) CreateInvite(_ context.Context, projectID, token, projectRole, createdBy string, expiresAt time.Time) (*query.ProjectInviteItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item := query.ProjectInviteItem{
		ID:          r.nextID("inv"),
		ProjectID:   projectID,
		Token:       token,
		ProjectRole: projectRole,
		Status:      "active",
		ExpiresAt:   expiresAt.UTC().Format(time.RFC3339),
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	r.invites = append(r.invites, memoryInvite{ProjectInviteItem: item})
	_ = createdBy
	return &item, nil
}

func (r *MembershipRepository) ListInvites(_ context.Context, projectID string) ([]query.ProjectInviteItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]query.ProjectInviteItem, 0)
	for _, inv := range r.invites {
		if inv.ProjectID == projectID {
			items = append(items, inv.ProjectInviteItem)
		}
	}
	return items, nil
}

func (r *MembershipRepository) GetInviteByID(_ context.Context, projectID, inviteID string) (*query.ProjectInviteItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, inv := range r.invites {
		if inv.ID == inviteID && inv.ProjectID == projectID {
			item := inv.ProjectInviteItem
			return &item, nil
		}
	}
	return nil, service.ErrNotFound
}

func (r *MembershipRepository) GetInviteByToken(_ context.Context, token string) (*query.ProjectInviteItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, inv := range r.invites {
		if inv.Token == token {
			item := inv.ProjectInviteItem
			return &item, nil
		}
	}
	return nil, service.ErrNotFound
}

func (r *MembershipRepository) RevokeInvite(_ context.Context, projectID, inviteID, actorID string) (*query.ProjectInviteItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, inv := range r.invites {
		if inv.ID == inviteID && inv.ProjectID == projectID && inv.Status == "active" {
			r.invites[i].Status = "revoked"
			item := r.invites[i].ProjectInviteItem
			_ = actorID
			return &item, nil
		}
	}
	return nil, service.ErrNotFound
}

func (r *MembershipRepository) GetInvitePreview(_ context.Context, token string) (*query.ProjectInvitePreview, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, inv := range r.invites {
		if inv.Token == token {
			expires, _ := time.Parse(time.RFC3339, inv.ExpiresAt)
			return &query.ProjectInvitePreview{
				ProjectID:     inv.ProjectID,
				ProjectName:   "课题A",
				TeamSpaceName: "随机控制实验室",
				ProjectRole:   inv.ProjectRole,
				Status:        inv.Status,
				ExpiresAt:     inv.ExpiresAt,
				Expired:       time.Now().UTC().After(expires),
			}, nil
		}
	}
	return nil, service.ErrNotFound
}
