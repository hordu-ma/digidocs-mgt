package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"digidocs-mgt/backend-go/internal/domain/query"
)

type mockAdminRepo struct {
	teamSpace *query.TeamSpaceSummary
	project   *query.ProjectSummary
	user      *query.UserOption
	member    *query.ProjectMemberItem

	allUsers []query.UserOption
	members  []query.ProjectMemberItem

	createUserHash string
	updateFields   map[string]any
	err            error
	removedID      string
}

func (m *mockAdminRepo) CreateTeamSpace(_ context.Context, name, code, _ string, _ string) (*query.TeamSpaceSummary, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.teamSpace != nil {
		return m.teamSpace, nil
	}
	return &query.TeamSpaceSummary{ID: "ts-1", Name: name, Code: code}, nil
}

func (m *mockAdminRepo) CreateProjectWithOwner(_ context.Context, teamSpaceID, name, code, _ string, ownerID string) (*query.ProjectSummary, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &query.ProjectSummary{ID: "p-1", TeamSpaceID: teamSpaceID, Name: name, Code: code, Owner: query.UserSummary{ID: ownerID}}, nil
}

func (m *mockAdminRepo) CreateUser(_ context.Context, username, passwordHash, displayName, role, email, phone string) (*query.UserOption, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.createUserHash = passwordHash
	return &query.UserOption{ID: "u-1", Username: username, DisplayName: displayName, Role: role, Email: email, Phone: phone}, nil
}

func (m *mockAdminRepo) UpdateUser(_ context.Context, userID string, fields map[string]any) (*query.UserOption, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.updateFields = fields
	return &query.UserOption{ID: userID, DisplayName: valueString(fields["display_name"]), Role: valueString(fields["role"]), Status: valueString(fields["status"])}, nil
}

func (m *mockAdminRepo) ListAllUsers(_ context.Context) ([]query.UserOption, error) {
	return m.allUsers, m.err
}

func (m *mockAdminRepo) ListProjectMembers(_ context.Context, projectID string) ([]query.ProjectMemberItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.members, nil
}

func (m *mockAdminRepo) AddProjectMember(_ context.Context, projectID, userID, projectRole string) (*query.ProjectMemberItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &query.ProjectMemberItem{ID: "pm-1", UserID: userID, ProjectRole: projectRole}, nil
}

func (m *mockAdminRepo) UpdateProjectMember(_ context.Context, memberID, projectRole string) (*query.ProjectMemberItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &query.ProjectMemberItem{ID: memberID, ProjectRole: projectRole}, nil
}

func (m *mockAdminRepo) RemoveProjectMember(_ context.Context, memberID string) error {
	m.removedID = memberID
	return m.err
}

func valueString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func TestAdminService_CreateTeamSpace_TrimsAndDelegates(t *testing.T) {
	repo := &mockAdminRepo{}
	svc := NewAdminService(repo)

	got, err := svc.CreateTeamSpace(context.Background(), CreateTeamSpaceInput{Name: "  Lab  ", Code: "  lab-a  ", ActorID: "admin-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Lab" || got.Code != "lab-a" {
		t.Fatalf("got = %#v, want trimmed name/code", got)
	}
}

func TestAdminService_CreateTeamSpace_ValidatesRequiredAndLength(t *testing.T) {
	svc := NewAdminService(&mockAdminRepo{})
	if _, err := svc.CreateTeamSpace(context.Background(), CreateTeamSpaceInput{Name: "Lab"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("missing code err = %v, want ErrValidation", err)
	}
	if _, err := svc.CreateTeamSpace(context.Background(), CreateTeamSpaceInput{Name: strings.Repeat("长", 129), Code: "lab"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("long name err = %v, want ErrValidation", err)
	}
}

func TestAdminService_CreateProject_ValidatesAndDelegates(t *testing.T) {
	svc := NewAdminService(&mockAdminRepo{})
	got, err := svc.CreateProject(context.Background(), CreateProjectInput{
		TeamSpaceID: "ts-1", Name: "  Project  ", Code: "  p-a  ", OwnerID: "u-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Project" || got.Code != "p-a" || got.Owner.ID != "u-1" {
		t.Fatalf("got = %#v, want delegated project", got)
	}

	if _, err := svc.CreateProject(context.Background(), CreateProjectInput{Name: "Project", Code: "p-a", OwnerID: "u-1"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("missing team_space_id err = %v, want ErrValidation", err)
	}
}

func TestAdminService_CreateUser_HashesPasswordAndValidatesRole(t *testing.T) {
	repo := &mockAdminRepo{}
	svc := NewAdminService(repo)
	got, err := svc.CreateUser(context.Background(), CreateUserInput{
		Username: "zhangsan", Password: "secret123", DisplayName: "张三", Role: "member", Email: "z@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Username != "zhangsan" || got.Role != "member" {
		t.Fatalf("got = %#v, want created user", got)
	}
	if repo.createUserHash == "" || repo.createUserHash == "secret123" {
		t.Fatalf("password hash was not generated: %q", repo.createUserHash)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(repo.createUserHash), []byte("secret123")); err != nil {
		t.Fatalf("password hash does not match original password: %v", err)
	}

	if _, err := svc.CreateUser(context.Background(), CreateUserInput{Username: "u", Password: "p", DisplayName: "U", Role: "owner"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("invalid role err = %v, want ErrValidation", err)
	}
	if _, err := svc.CreateUser(context.Background(), CreateUserInput{Username: "u", Password: strings.Repeat("x", 73), DisplayName: "U", Role: "member"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("long password err = %v, want ErrValidation", err)
	}
}

func TestAdminService_UpdateUser_BuildsAllowedFields(t *testing.T) {
	repo := &mockAdminRepo{}
	svc := NewAdminService(repo)
	displayName := "  李四  "
	role := "project_lead"
	email := " li@example.com "
	phone := " 13800000000 "
	status := "inactive"

	got, err := svc.UpdateUser(context.Background(), UpdateUserInput{
		UserID: "u-1", DisplayName: &displayName, Role: &role, Email: &email, Phone: &phone, Status: &status,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "u-1" || repo.updateFields["display_name"] != "李四" || repo.updateFields["email"] != "li@example.com" {
		t.Fatalf("got = %#v fields = %#v, want trimmed update fields", got, repo.updateFields)
	}
}

func TestAdminService_UpdateUser_ValidationBranches(t *testing.T) {
	svc := NewAdminService(&mockAdminRepo{})
	displayName := " "
	role := "bad"
	status := "locked"

	cases := []UpdateUserInput{
		{},
		{UserID: "u-1"},
		{UserID: "u-1", DisplayName: &displayName},
		{UserID: "u-1", Role: &role},
		{UserID: "u-1", Status: &status},
	}
	for _, tc := range cases {
		if _, err := svc.UpdateUser(context.Background(), tc); !errors.Is(err, ErrValidation) {
			t.Fatalf("input %#v err = %v, want ErrValidation", tc, err)
		}
	}
}

func TestAdminService_UserAndProjectMemberDelegates(t *testing.T) {
	repo := &mockAdminRepo{
		allUsers: []query.UserOption{{ID: "u-1"}},
		members:  []query.ProjectMemberItem{{ID: "pm-1", UserID: "u-1", ProjectRole: "owner"}},
	}
	svc := NewAdminService(repo)

	users, err := svc.ListAllUsers(context.Background())
	if err != nil || len(users) != 1 {
		t.Fatalf("users = %#v err = %v, want one user", users, err)
	}
	members, err := svc.ListProjectMembers(context.Background(), "p-1")
	if err != nil || len(members) != 1 {
		t.Fatalf("members = %#v err = %v, want one member", members, err)
	}
	added, err := svc.AddProjectMember(context.Background(), AddProjectMemberInput{ProjectID: "p-1", UserID: "u-2", ProjectRole: "contributor"})
	if err != nil || added.ProjectRole != "contributor" {
		t.Fatalf("added = %#v err = %v, want contributor", added, err)
	}
	updated, err := svc.UpdateProjectMember(context.Background(), "pm-1", "viewer")
	if err != nil || updated.ProjectRole != "viewer" {
		t.Fatalf("updated = %#v err = %v, want viewer", updated, err)
	}
	if err := svc.RemoveProjectMember(context.Background(), "pm-1"); err != nil {
		t.Fatalf("remove unexpected error: %v", err)
	}
	if repo.removedID != "pm-1" {
		t.Fatalf("removedID = %q, want pm-1", repo.removedID)
	}
}

func TestAdminService_ProjectMemberValidationBranches(t *testing.T) {
	svc := NewAdminService(&mockAdminRepo{})
	if _, err := svc.ListProjectMembers(context.Background(), ""); !errors.Is(err, ErrValidation) {
		t.Fatalf("empty project err = %v, want ErrValidation", err)
	}
	if _, err := svc.AddProjectMember(context.Background(), AddProjectMemberInput{ProjectID: "p-1", UserID: "u-1", ProjectRole: "bad"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("bad role err = %v, want ErrValidation", err)
	}
	if _, err := svc.UpdateProjectMember(context.Background(), "", "viewer"); !errors.Is(err, ErrValidation) {
		t.Fatalf("empty member err = %v, want ErrValidation", err)
	}
	if _, err := svc.UpdateProjectMember(context.Background(), "pm-1", "bad"); !errors.Is(err, ErrValidation) {
		t.Fatalf("bad update role err = %v, want ErrValidation", err)
	}
	if err := svc.RemoveProjectMember(context.Background(), ""); !errors.Is(err, ErrValidation) {
		t.Fatalf("empty remove err = %v, want ErrValidation", err)
	}
}
