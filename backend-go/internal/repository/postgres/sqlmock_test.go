package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"digidocs-mgt/backend-go/internal/domain/auth"
	"digidocs-mgt/backend-go/internal/service"
)

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db, mock
}

func assertExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestAdminRepository_CreateTeamSpaceAndConflict(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewAdminRepository(db)
	mock.ExpectQuery("INSERT INTO team_spaces").
		WithArgs("Lab", "lab", "", "00000000-0000-0000-0000-000000000001").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code"}).AddRow("ts-1", "Lab", "lab"))
	got, err := repo.CreateTeamSpace(ctx, "Lab", "lab", "", "00000000-0000-0000-0000-000000000001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "ts-1" || got.Name != "Lab" {
		t.Fatalf("got = %#v, want team space", got)
	}
	assertExpectations(t, mock)

	db, mock = newMockDB(t)
	repo = NewAdminRepository(db)
	mock.ExpectQuery("INSERT INTO team_spaces").
		WillReturnError(errors.New("pq: duplicate key value violates unique constraint (SQLSTATE 23505)"))
	if _, err := repo.CreateTeamSpace(ctx, "Lab", "lab", "", "u-1"); !errors.Is(err, service.ErrConflict) {
		t.Fatalf("err = %v, want ErrConflict", err)
	}
	assertExpectations(t, mock)
}

func TestAdminRepository_CreateProjectWithOwner(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewAdminRepository(db)
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO projects").
		WithArgs("ts-1", "Project", "proj", "", "u-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "team_space_id", "name", "code"}).AddRow("p-1", "ts-1", "Project", "proj"))
	mock.ExpectExec("INSERT INTO project_members").
		WithArgs("p-1", "u-1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT id::text, display_name FROM users").
		WithArgs("u-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "display_name"}).AddRow("u-1", "Owner"))
	mock.ExpectCommit()

	got, err := repo.CreateProjectWithOwner(ctx, "ts-1", "Project", "proj", "", "u-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "p-1" || got.Owner.DisplayName != "Owner" {
		t.Fatalf("got = %#v, want project with owner", got)
	}
	assertExpectations(t, mock)
}

func TestAdminRepository_UserAndMemberMethods(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewAdminRepository(db)

	mock.ExpectQuery("INSERT INTO users").
		WithArgs("zhangsan", "hash", "张三", "member", "z@example.com", "138").
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "display_name", "role", "email", "phone", "wechat", "status"}).
			AddRow("u-1", "zhangsan", "张三", "member", "z@example.com", "138", "wx", "active"))
	user, err := repo.CreateUser(ctx, "zhangsan", "hash", "张三", "member", "z@example.com", "138")
	if err != nil || user.ID != "u-1" || user.Wechat != "wx" {
		t.Fatalf("CreateUser got %#v err=%v", user, err)
	}

	mock.ExpectQuery("UPDATE users SET").
		WithArgs("u-1", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "display_name", "role", "email", "phone", "wechat", "status"}).
			AddRow("u-1", "lisi", "李四", "member", "", "", "", "inactive"))
	updated, err := repo.UpdateUser(ctx, "u-1", map[string]any{"display_name": "李四", "status": "inactive"})
	if err != nil || updated.DisplayName != "李四" || updated.Status != "inactive" {
		t.Fatalf("UpdateUser got %#v err=%v", updated, err)
	}

	mock.ExpectQuery("FROM users\\s+ORDER BY").
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "display_name", "role", "email", "phone", "wechat", "status"}).
			AddRow("u-1", "zhangsan", "张三", "member", "z@example.com", "", "", "active"))
	users, err := repo.ListAllUsers(ctx)
	if err != nil || len(users) != 1 || users[0].Username != "zhangsan" {
		t.Fatalf("ListAllUsers got %#v err=%v", users, err)
	}

	mock.ExpectQuery("FROM project_members pm").
		WithArgs("p-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "display_name", "username", "project_role"}).
			AddRow("pm-1", "u-1", "张三", "zhangsan", "owner"))
	members, err := repo.ListProjectMembers(ctx, "p-1")
	if err != nil || len(members) != 1 || members[0].ProjectRole != "owner" {
		t.Fatalf("ListProjectMembers got %#v err=%v", members, err)
	}

	mock.ExpectQuery("INSERT INTO project_members").
		WithArgs("p-1", "u-2", "contributor").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("pm-2"))
	mock.ExpectQuery("SELECT display_name, username FROM users").
		WithArgs("u-2").
		WillReturnRows(sqlmock.NewRows([]string{"display_name", "username"}).AddRow("李四", "lisi"))
	added, err := repo.AddProjectMember(ctx, "p-1", "u-2", "contributor")
	if err != nil || added.ID != "pm-2" || added.Username != "lisi" {
		t.Fatalf("AddProjectMember got %#v err=%v", added, err)
	}

	mock.ExpectQuery("UPDATE project_members SET").
		WithArgs("pm-2", "viewer").
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "project_role"}).AddRow("pm-2", "u-2", "viewer"))
	mock.ExpectQuery("SELECT display_name, username FROM users").
		WithArgs("u-2").
		WillReturnRows(sqlmock.NewRows([]string{"display_name", "username"}).AddRow("李四", "lisi"))
	member, err := repo.UpdateProjectMember(ctx, "pm-2", "viewer")
	if err != nil || member.ProjectRole != "viewer" {
		t.Fatalf("UpdateProjectMember got %#v err=%v", member, err)
	}

	mock.ExpectExec("DELETE FROM project_members").WithArgs("pm-2").WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.RemoveProjectMember(ctx, "pm-2"); err != nil {
		t.Fatalf("RemoveProjectMember unexpected err=%v", err)
	}
	assertExpectations(t, mock)
}

func TestAdminRepository_NotFoundBranches(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewAdminRepository(db)
	mock.ExpectQuery("UPDATE users SET").WillReturnError(sql.ErrNoRows)
	if _, err := repo.UpdateUser(ctx, "missing", map[string]any{"display_name": "Missing"}); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("UpdateUser err=%v, want ErrNotFound", err)
	}
	mock.ExpectQuery("UPDATE project_members SET").WillReturnError(sql.ErrNoRows)
	if _, err := repo.UpdateProjectMember(ctx, "missing", "viewer"); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("UpdateProjectMember err=%v, want ErrNotFound", err)
	}
	mock.ExpectExec("DELETE FROM project_members").WithArgs("missing").WillReturnResult(sqlmock.NewResult(0, 0))
	if err := repo.RemoveProjectMember(ctx, "missing"); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("RemoveProjectMember err=%v, want ErrNotFound", err)
	}
	assertExpectations(t, mock)
}

func TestUserAuthRepositoryMethods(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewUserAuthRepository(db)
	mock.ExpectQuery("FROM users\\s+WHERE username").
		WithArgs("zhangsan").
		WillReturnRows(sqlmock.NewRows([]string{"id", "password_hash", "display_name", "role"}).AddRow("u-1", "hash", "张三", "member"))
	record, err := repo.FindUserByUsername(ctx, "zhangsan")
	if err != nil || record.ID != "u-1" || record.PasswordHash != "hash" {
		t.Fatalf("FindUserByUsername got %#v err=%v", record, err)
	}

	loginAt := time.Date(2026, 5, 1, 1, 2, 3, 0, time.UTC)
	mock.ExpectQuery("SELECT\\s+id::text,\\s+username,\\s+display_name").
		WithArgs("u-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "display_name", "role", "email", "phone", "wechat", "status", "last_login_at"}).
			AddRow("u-1", "zhangsan", "张三", "member", "z@example.com", "", "wx", "active", loginAt))
	profile, err := repo.GetUserProfile(ctx, "u-1")
	if err != nil || profile.LastLoginAt == nil || *profile.LastLoginAt != "2026-05-01T01:02:03Z" {
		t.Fatalf("GetUserProfile got %#v err=%v", profile, err)
	}

	mock.ExpectQuery("UPDATE users").
		WithArgs("u-1", "张三", "z@example.com", "138", "wx").
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "display_name", "role", "email", "phone", "wechat", "status", "last_login_at"}).
			AddRow("u-1", "zhangsan", "张三", "member", "z@example.com", "138", "wx", "active", nil))
	updated, err := repo.UpdateUserProfile(ctx, "u-1", auth.ProfileUpdateInput{DisplayName: "张三", Email: "z@example.com", Phone: "138", Wechat: "wx"})
	if err != nil || updated.Phone != "138" || updated.LastLoginAt != nil {
		t.Fatalf("UpdateUserProfile got %#v err=%v", updated, err)
	}
	assertExpectations(t, mock)
}

func TestUserAuthRepositoryNotFound(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewUserAuthRepository(db)
	mock.ExpectQuery("FROM users\\s+WHERE username").WillReturnError(sql.ErrNoRows)
	if _, err := repo.FindUserByUsername(ctx, "missing"); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("FindUserByUsername err=%v, want ErrNotFound", err)
	}
	mock.ExpectQuery("SELECT\\s+id::text,\\s+username").WillReturnError(sql.ErrNoRows)
	if _, err := repo.GetUserProfile(ctx, "missing"); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("GetUserProfile err=%v, want ErrNotFound", err)
	}
	mock.ExpectQuery("UPDATE users").WillReturnError(sql.ErrNoRows)
	if _, err := repo.UpdateUserProfile(ctx, "missing", auth.ProfileUpdateInput{}); !errors.Is(err, service.ErrNotFound) {
		t.Fatalf("UpdateUserProfile err=%v, want ErrNotFound", err)
	}
	assertExpectations(t, mock)
}

func TestSimpleQueryRepositories(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	teamSpaces := NewTeamSpaceRepository(db)
	users := NewUserQueryRepository(db)

	mock.ExpectQuery("SELECT id::text, name, code FROM team_spaces").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code"}).AddRow("ts-1", "Lab", "lab"))
	ts, err := teamSpaces.ListTeamSpaces(ctx, "admin", "admin")
	if err != nil || len(ts) != 1 || ts[0].Code != "lab" {
		t.Fatalf("admin ListTeamSpaces got %#v err=%v", ts, err)
	}

	mock.ExpectQuery("FROM team_spaces ts").
		WithArgs("u-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "code"}).AddRow("ts-2", "Member Lab", "member-lab"))
	ts, err = teamSpaces.ListTeamSpaces(ctx, "u-1", "member")
	if err != nil || len(ts) != 1 || ts[0].ID != "ts-2" {
		t.Fatalf("member ListTeamSpaces got %#v err=%v", ts, err)
	}

	mock.ExpectQuery("FROM users\\s+WHERE status = 'active'").
		WillReturnRows(sqlmock.NewRows([]string{"id", "username", "display_name", "role", "email", "phone", "wechat", "status"}).
			AddRow("u-1", "zhangsan", "张三", "member", "z@example.com", "", "", "active"))
	gotUsers, err := users.ListUsers(ctx)
	if err != nil || len(gotUsers) != 1 || gotUsers[0].Username != "zhangsan" {
		t.Fatalf("ListUsers got %#v err=%v", gotUsers, err)
	}
	assertExpectations(t, mock)
}
