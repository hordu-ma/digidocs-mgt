package postgres

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/service"
)

func TestDashboardRepositoryMethods(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewDashboardRepository(db)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\)\\s+FROM documents").WithArgs("p-1").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))
	mock.ExpectQuery("SELECT d.current_status::text").WithArgs("p-1").
		WillReturnRows(sqlmock.NewRows([]string{"status", "count"}).AddRow("draft", 2).AddRow("in_progress", 3))
	mock.ExpectQuery("FROM graduation_handovers gh").WithArgs("p-1").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("AND d.current_status NOT IN").WithArgs("p-1").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(4))
	overview, err := repo.GetOverview(ctx, "p-1")
	if err != nil || overview.DocumentTotal != 10 || overview.StatusCounts["in_progress"] != 3 || overview.RiskDocumentCount != 4 {
		t.Fatalf("GetOverview got %#v err=%v", overview, err)
	}

	mock.ExpectQuery("FROM flow_records fr").WithArgs("p-1").
		WillReturnRows(sqlmock.NewRows([]string{"document_id", "title", "action", "from_status", "to_status", "created_at"}).
			AddRow("doc-1", "Report", "transfer", "draft", "pending_handover", "2026-05-01T00:00:00Z"))
	flows, err := repo.ListRecentFlows(ctx, "p-1")
	if err != nil || len(flows) != 1 || flows[0].Action != "transfer" {
		t.Fatalf("ListRecentFlows got %#v err=%v", flows, err)
	}

	mock.ExpectQuery("'超过30天未更新'").WithArgs("p-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "risk_type", "risk_message"}).
			AddRow("doc-1", "Report", "stale", "超过30天未更新"))
	risks, err := repo.ListRiskDocuments(ctx, "p-1")
	if err != nil || len(risks) != 1 || risks[0].RiskType != "stale" {
		t.Fatalf("ListRiskDocuments got %#v err=%v", risks, err)
	}
	assertExpectations(t, mock)
}

func TestAuditRepositoryMethods(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewAuditRepository(db)
	filter := query.AuditEventFilter{ProjectID: "p-1", DocumentID: "doc-1", ActionType: "download", UserID: "u-1", DateFrom: "2026-05-01", DateTo: "2026-05-01", Page: 2, PageSize: 5}
	mock.ExpectQuery("SELECT COUNT\\(\\*\\)\\s+FROM audit_events").
		WithArgs("p-1", "doc-1", "download", "u-1", "2026-05-01", "2026-05-01").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(7))
	mock.ExpectQuery("SELECT\\s+ae.id::text").
		WithArgs("p-1", "doc-1", "download", "u-1", "2026-05-01", "2026-05-01", 5, 5).
		WillReturnRows(sqlmock.NewRows([]string{"id", "document_id", "version_id", "user_id", "action_type", "request_id", "ip", "terminal", "created_at"}).
			AddRow("ae-1", "doc-1", "v-1", "u-1", "download", "req-1", "127.0.0.1", "web", "2026-05-01T00:00:00Z"))
	events, total, err := repo.ListAuditEvents(ctx, filter)
	if err != nil || total != 7 || len(events) != 1 || events[0].IPAddress != "127.0.0.1" {
		t.Fatalf("ListAuditEvents got %#v total=%d err=%v", events, total, err)
	}

	mock.ExpectQuery("COUNT\\(\\*\\) FILTER").
		WithArgs("p-1").
		WillReturnRows(sqlmock.NewRows([]string{"upload", "download", "transfer", "archive"}).AddRow(1, 2, 3, 4))
	mock.ExpectQuery("GROUP BY ae.user_id").
		WithArgs("p-1").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "display_name", "count"}).AddRow("u-1", "张三", 9))
	summary, err := repo.GetAuditSummary(ctx, "p-1")
	if err != nil || summary.DownloadCount != 2 || len(summary.TopActiveUsers) != 1 {
		t.Fatalf("GetAuditSummary got %#v err=%v", summary, err)
	}
	assertExpectations(t, mock)
}

func TestCodeRepositoryRepositoryMethods(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewCodeRepositoryRepository(db)
	codeRows := func() *sqlmock.Rows {
		return sqlmock.NewRows([]string{"id", "team_space_id", "project_id", "project_name", "name", "slug", "description", "default_branch", "target_folder_path", "repo_storage_path", "last_commit_sha", "last_pushed_at", "status", "created_by_name", "created_at", "updated_at"}).
			AddRow("r-1", "ts-1", "p-1", "Project", "Repo", "repo", "Desc", "main", "/target", "/tmp/repo.git", "abc", "2026-05-01T00:00:00Z", "active", "张三", "2026-05-01T00:00:00Z", "2026-05-01T00:00:00Z")
	}
	mock.ExpectQuery("FROM code_repositories cr").WithArgs("p-1", "repo", 10, 0).WillReturnRows(codeRows())
	mock.ExpectQuery("SELECT COUNT\\(\\*\\)\\s+FROM code_repositories").WithArgs("p-1", "repo").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	items, total, err := repo.ListCodeRepositories(ctx, query.CodeRepositoryListFilter{ProjectID: "p-1", Keyword: "repo", Page: 1, PageSize: 10})
	if err != nil || total != 1 || len(items) != 1 || items[0].Slug != "repo" {
		t.Fatalf("ListCodeRepositories got %#v total=%d err=%v", items, total, err)
	}

	mock.ExpectQuery("FROM code_repositories cr").WithArgs("r-1").WillReturnRows(codeRowsWithToken(""))
	got, err := repo.GetCodeRepository(ctx, "r-1")
	if err != nil || got.ID != "r-1" || got.PushToken != "" {
		t.Fatalf("GetCodeRepository got %#v err=%v", got, err)
	}
	mock.ExpectQuery("FROM code_repositories cr").WithArgs("repo").WillReturnRows(codeRowsWithToken("secret"))
	got, err = repo.GetCodeRepositoryBySlug(ctx, "repo")
	if err != nil || got.PushToken != "secret" {
		t.Fatalf("GetCodeRepositoryBySlug got %#v err=%v", got, err)
	}

	mock.ExpectExec("INSERT INTO code_repositories").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("FROM code_repositories cr").WithArgs(sqlmock.AnyArg()).WillReturnRows(codeRowsWithToken(""))
	created, err := repo.CreateCodeRepository(ctx, command.CodeRepositoryCreateInput{
		TeamSpaceID: "ts-1", ProjectID: "p-1", Name: "Repo", Slug: "repo", DefaultBranch: "main", TargetFolderPath: "/target", RepoStoragePath: "/tmp/repo.git", PushToken: "secret", ActorID: "u-1",
	})
	if err != nil || created.PushToken != "secret" {
		t.Fatalf("CreateCodeRepository got %#v err=%v", created, err)
	}

	mock.ExpectExec("UPDATE code_repositories").WithArgs("r-1", "Repo 2", "Desc", "main", "/target").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("FROM code_repositories cr").WithArgs("r-1").WillReturnRows(codeRowsWithToken(""))
	if _, err := repo.UpdateCodeRepository(ctx, command.CodeRepositoryUpdateInput{RepositoryID: "r-1", Name: "Repo 2", Description: "Desc", DefaultBranch: "main", TargetFolderPath: "/target"}); err != nil {
		t.Fatalf("UpdateCodeRepository unexpected err=%v", err)
	}

	mock.ExpectExec("INSERT INTO code_push_events").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("FROM code_push_events cpe").WithArgs("r-1").WillReturnRows(sqlmock.NewRows([]string{"id", "repository_id", "branch", "before_sha", "after_sha", "commit_message", "pusher", "status", "error", "created_at", "completed_at"}).
		AddRow("event-1", "r-1", "main", "", "abc", "msg", "张三", "synced", "", "2026-05-01T00:00:00Z", "2026-05-01T00:00:00Z"))
	if _, err := repo.CreateCodePushEvent(ctx, command.CodePushEventCreateInput{RepositoryID: "r-1", Branch: "main", AfterSHA: "abc", CommitMessage: "msg", PusherID: "u-1", SyncStatus: "synced"}); err == nil {
		t.Fatal("CreateCodePushEvent got nil error, want ErrNotFound when inserted random id is absent from mocked rows")
	}

	mock.ExpectExec("UPDATE code_repositories").WithArgs("r-1", "abc", "active").WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repo.UpdateCodeRepositoryAfterPush(ctx, "r-1", "abc", "active"); err != nil {
		t.Fatalf("UpdateCodeRepositoryAfterPush unexpected err=%v", err)
	}
	assertExpectations(t, mock)
}

func codeRowsWithToken(token string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "team_space_id", "project_id", "project_name", "name", "slug", "description", "default_branch", "target_folder_path", "repo_storage_path", "last_commit_sha", "last_pushed_at", "status", "created_by_name", "created_at", "updated_at", "push_token"}).
		AddRow("r-1", "ts-1", "p-1", "Project", "Repo", "repo", "Desc", "main", "/target", "/tmp/repo.git", "abc", "2026-05-01T00:00:00Z", "active", "张三", "2026-05-01T00:00:00Z", "2026-05-01T00:00:00Z", token)
}

func TestCodeRepositoryRepositoryNotFoundBranches(t *testing.T) {
	ctx := context.Background()
	db, mock := newMockDB(t)
	repo := NewCodeRepositoryRepository(db)
	mock.ExpectQuery("FROM code_repositories cr").WillReturnError(service.ErrNotFound)
	if _, err := repo.GetCodeRepository(ctx, "missing"); err == nil {
		t.Fatal("GetCodeRepository expected error")
	}
	mock.ExpectExec("UPDATE code_repositories").WillReturnResult(sqlmock.NewResult(0, 0))
	if _, err := repo.UpdateCodeRepository(ctx, command.CodeRepositoryUpdateInput{RepositoryID: "missing"}); err == nil {
		t.Fatal("UpdateCodeRepository expected error")
	}
	mock.ExpectExec("UPDATE code_repositories").WillReturnResult(sqlmock.NewResult(0, 0))
	if err := repo.UpdateCodeRepositoryAfterPush(ctx, "missing", "sha", "failed"); err == nil {
		t.Fatal("UpdateCodeRepositoryAfterPush expected error")
	}
	assertExpectations(t, mock)
}
