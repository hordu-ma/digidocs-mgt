package service

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	memstorage "digidocs-mgt/backend-go/internal/storage/memory"
)

type fakeCodeRepoStore struct {
	repos  []query.CodeRepositoryDetail
	events []query.CodePushEventItem
}

func (s *fakeCodeRepoStore) ListCodeRepositories(_ context.Context, filter query.CodeRepositoryListFilter) ([]query.CodeRepositoryItem, int, error) {
	keyword := strings.ToLower(filter.Keyword)
	items := make([]query.CodeRepositoryItem, 0)
	for _, repo := range s.repos {
		if filter.ProjectID != "" && repo.ProjectID != filter.ProjectID {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(repo.Name), keyword) && !strings.Contains(strings.ToLower(repo.Slug), keyword) {
			continue
		}
		items = append(items, repo.CodeRepositoryItem)
	}
	return items, len(items), nil
}

func (s *fakeCodeRepoStore) GetCodeRepository(_ context.Context, id string) (*query.CodeRepositoryDetail, error) {
	for _, repo := range s.repos {
		if repo.ID == id {
			copy := repo
			return &copy, nil
		}
	}
	return nil, ErrNotFound
}

func (s *fakeCodeRepoStore) GetCodeRepositoryBySlug(_ context.Context, slug string) (*query.CodeRepositoryDetail, error) {
	for _, repo := range s.repos {
		if repo.Slug == slug {
			copy := repo
			return &copy, nil
		}
	}
	return nil, ErrNotFound
}

func (s *fakeCodeRepoStore) ListCodePushEvents(_ context.Context, repositoryID string) ([]query.CodePushEventItem, error) {
	items := make([]query.CodePushEventItem, 0)
	for _, event := range s.events {
		if event.RepositoryID == repositoryID {
			items = append(items, event)
		}
	}
	return items, nil
}

func (s *fakeCodeRepoStore) CreateCodeRepository(_ context.Context, input command.CodeRepositoryCreateInput) (*query.CodeRepositoryDetail, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	repo := query.CodeRepositoryDetail{
		CodeRepositoryItem: query.CodeRepositoryItem{
			ID:               "repo-" + input.Slug,
			TeamSpaceID:      input.TeamSpaceID,
			ProjectID:        input.ProjectID,
			Name:             input.Name,
			Slug:             input.Slug,
			Description:      input.Description,
			DefaultBranch:    input.DefaultBranch,
			TargetFolderPath: input.TargetFolderPath,
			RepoStoragePath:  input.RepoStoragePath,
			Status:           "active",
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		PushToken: input.PushToken,
	}
	s.repos = append(s.repos, repo)
	return &repo, nil
}

func (s *fakeCodeRepoStore) UpdateCodeRepository(_ context.Context, input command.CodeRepositoryUpdateInput) (*query.CodeRepositoryDetail, error) {
	for i := range s.repos {
		if s.repos[i].ID == input.RepositoryID {
			if input.Name != "" {
				s.repos[i].Name = input.Name
			}
			if input.TargetFolderPath != "" {
				s.repos[i].TargetFolderPath = input.TargetFolderPath
			}
			if input.DefaultBranch != "" {
				s.repos[i].DefaultBranch = input.DefaultBranch
			}
			s.repos[i].Description = input.Description
			copy := s.repos[i]
			return &copy, nil
		}
	}
	return nil, ErrNotFound
}

func (s *fakeCodeRepoStore) CreateCodePushEvent(_ context.Context, input command.CodePushEventCreateInput) (*query.CodePushEventItem, error) {
	event := query.CodePushEventItem{
		ID:            "event-1",
		RepositoryID:  input.RepositoryID,
		Branch:        input.Branch,
		BeforeSHA:     input.BeforeSHA,
		AfterSHA:      input.AfterSHA,
		CommitMessage: input.CommitMessage,
		PusherName:    input.PusherID,
		SyncStatus:    input.SyncStatus,
		ErrorMessage:  input.ErrorMessage,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		CompletedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	s.events = append([]query.CodePushEventItem{event}, s.events...)
	return &event, nil
}

func (s *fakeCodeRepoStore) UpdateCodeRepositoryAfterPush(_ context.Context, repositoryID string, commitSHA string, status string) error {
	for i := range s.repos {
		if s.repos[i].ID == repositoryID {
			s.repos[i].LastCommitSHA = commitSHA
			s.repos[i].Status = status
			return nil
		}
	}
	return ErrNotFound
}

func TestCodeRepositoryService_CreateInitializesBareRepoAndRecord(t *testing.T) {
	repo := &fakeCodeRepoStore{}
	root := t.TempDir()
	svc := NewCodeRepositoryService(repo, repo, PermissionService{}, root, nil)

	got, err := svc.Create(context.Background(), command.CodeRepositoryCreateInput{
		TeamSpaceID: "ts-1", ProjectID: "p-1", Name: "Demo Repo", TargetFolderPath: "/projects/demo", ActorID: "u-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID == "" || !strings.HasPrefix(got.Slug, "demo-repo-") || got.PushToken == "" {
		t.Fatalf("got = %#v, want generated id/slug/token", got)
	}
	if got.DefaultBranch != "main" {
		t.Fatalf("default branch = %q, want main", got.DefaultBranch)
	}
	if _, err := os.Stat(filepath.Join(root, got.Slug+".git", "HEAD")); err != nil {
		t.Fatalf("bare repo HEAD missing: %v", err)
	}
}

func TestCodeRepositoryService_CreateValidationBranches(t *testing.T) {
	repo := &fakeCodeRepoStore{}
	svc := NewCodeRepositoryService(repo, repo, PermissionService{}, t.TempDir(), nil)
	valid := command.CodeRepositoryCreateInput{TeamSpaceID: "ts-1", ProjectID: "p-1", Name: "Repo", TargetFolderPath: "/target"}
	cases := []command.CodeRepositoryCreateInput{
		{ProjectID: valid.ProjectID, Name: valid.Name, TargetFolderPath: valid.TargetFolderPath},
		{TeamSpaceID: valid.TeamSpaceID, Name: valid.Name, TargetFolderPath: valid.TargetFolderPath},
		{TeamSpaceID: valid.TeamSpaceID, ProjectID: valid.ProjectID, TargetFolderPath: valid.TargetFolderPath},
		{TeamSpaceID: valid.TeamSpaceID, ProjectID: valid.ProjectID, Name: valid.Name},
		{TeamSpaceID: valid.TeamSpaceID, ProjectID: valid.ProjectID, Name: valid.Name, TargetFolderPath: "relative"},
		{TeamSpaceID: valid.TeamSpaceID, ProjectID: valid.ProjectID, Name: valid.Name, TargetFolderPath: "/bad/../path"},
	}
	for _, tc := range cases {
		if _, err := svc.Create(context.Background(), tc); err == nil {
			t.Fatalf("Create(%#v) got nil error, want validation error", tc)
		}
	}
}

func TestCodeRepositoryService_GetUpdateListAndAuthenticate(t *testing.T) {
	repo := &fakeCodeRepoStore{}
	svc := NewCodeRepositoryService(repo, repo, PermissionService{}, t.TempDir(), nil)
	created, err := svc.Create(context.Background(), command.CodeRepositoryCreateInput{
		TeamSpaceID: "ts-1", ProjectID: "p-1", Name: "Demo Repo", Slug: "custom", DefaultBranch: "main", TargetFolderPath: "/target",
	})
	if err != nil {
		t.Fatalf("create unexpected error: %v", err)
	}
	got, err := svc.Get(context.Background(), created.ID)
	if err != nil || got.ID != created.ID {
		t.Fatalf("Get got %#v err %v, want created repo", got, err)
	}
	if _, err := svc.Get(context.Background(), ""); err == nil {
		t.Fatal("Get empty id got nil error, want validation error")
	}
	if _, err := svc.GetBySlug(context.Background(), ""); err == nil {
		t.Fatal("GetBySlug empty slug got nil error, want validation error")
	}

	updated, err := svc.Update(context.Background(), command.CodeRepositoryUpdateInput{
		RepositoryID: created.ID, Name: "Renamed", TargetFolderPath: "/new-target",
	})
	if err != nil {
		t.Fatalf("update unexpected error: %v", err)
	}
	if updated.Name != "Renamed" || updated.TargetFolderPath != "/new-target" {
		t.Fatalf("updated = %#v, want renamed target", updated)
	}
	if _, err := svc.Update(context.Background(), command.CodeRepositoryUpdateInput{}); err == nil {
		t.Fatal("Update empty id got nil error, want validation error")
	}

	items, total, err := svc.List(context.Background(), query.CodeRepositoryListFilter{Keyword: "rename"})
	if err != nil || total != 1 || len(items) != 1 {
		t.Fatalf("List got items=%#v total=%d err=%v, want one item", items, total, err)
	}
	authRepo, ok, err := svc.AuthenticatePush(context.Background(), created.Slug, created.PushToken)
	if err != nil || !ok || authRepo.ID != created.ID {
		t.Fatalf("AuthenticatePush ok=%v repo=%#v err=%v, want success", ok, authRepo, err)
	}
	if _, ok, err := svc.AuthenticatePush(context.Background(), created.Slug, "bad"); err != nil || ok {
		t.Fatalf("AuthenticatePush bad token ok=%v err=%v, want false nil", ok, err)
	}
}

func TestCodeRepositoryService_RecordPushWithoutDefaultBranchRecordsFailure(t *testing.T) {
	repo := &fakeCodeRepoStore{}
	svc := NewCodeRepositoryService(repo, repo, PermissionService{}, t.TempDir(), nil)
	created, err := svc.Create(context.Background(), command.CodeRepositoryCreateInput{
		TeamSpaceID: "ts-1", ProjectID: "p-1", Name: "Empty Repo", TargetFolderPath: "/target",
	})
	if err != nil {
		t.Fatalf("create unexpected error: %v", err)
	}
	event, err := svc.RecordPush(context.Background(), created, "u-1")
	if err != nil {
		t.Fatalf("record push unexpected error: %v", err)
	}
	if event.SyncStatus != "failed" || !strings.Contains(event.ErrorMessage, "default branch") {
		t.Fatalf("event = %#v, want failed default branch event", event)
	}
}

func TestCodeRepositoryService_RecordPushSyncsDefaultBranchToStorage(t *testing.T) {
	ctx := context.Background()
	repo := &fakeCodeRepoStore{}
	storageProvider := memstorage.NewProvider()
	svc := NewCodeRepositoryService(repo, repo, PermissionService{}, t.TempDir(), storageProvider)
	created, err := svc.Create(ctx, command.CodeRepositoryCreateInput{
		TeamSpaceID: "ts-1", ProjectID: "p-1", Name: "Sync Repo", TargetFolderPath: "/code/sync",
	})
	if err != nil {
		t.Fatalf("create unexpected error: %v", err)
	}

	worktree := t.TempDir()
	runTestGit(t, worktree, "init", "-b", created.DefaultBranch)
	if err := os.WriteFile(filepath.Join(worktree, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	runTestGit(t, worktree, "add", "README.md")
	runTestGit(t, worktree, "-c", "user.name=Test User", "-c", "user.email=test@example.com", "commit", "-m", "initial commit")
	runTestGit(t, worktree, "remote", "add", "origin", created.RepoStoragePath)
	runTestGit(t, worktree, "push", "origin", created.DefaultBranch)

	event, err := svc.RecordPush(ctx, created, "u-1")
	if err != nil {
		t.Fatalf("record push unexpected error: %v", err)
	}
	if event.SyncStatus != "synced" || event.AfterSHA == "" || event.CommitMessage != "initial commit" {
		t.Fatalf("event = %#v, want synced commit event", event)
	}
	out, err := storageProvider.GetObject(ctx, "code/sync/README.md")
	if err != nil {
		t.Fatalf("synced README not found in storage: %v", err)
	}
	defer out.Reader.Close()
}

func TestCodeRepositoryHelpers(t *testing.T) {
	if got := makeSlug("  My_Repo: 研究 Version  "); got != "my-repo-version" {
		t.Fatalf("makeSlug = %q, want my-repo-version", got)
	}
	if got := makeSlug(strings.Repeat("a", 80)); len(got) != 48 {
		t.Fatalf("long slug length = %d, want 48", len(got))
	}
	if err := validateTargetFolder("/valid/path"); err != nil {
		t.Fatalf("valid target folder err = %v", err)
	}
	if err := validateTargetFolder("relative/path"); err == nil {
		t.Fatal("relative path got nil error, want validation error")
	}
	if token := shortToken(2); len(token) != 4 {
		t.Fatalf("shortToken length = %d, want 4", len(token))
	}
}

func runTestGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, strings.TrimSpace(string(out)))
	}
}
