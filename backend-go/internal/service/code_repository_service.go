package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/repository"
)

type CodeRepositoryService struct {
	reader      repository.CodeRepositoryReader
	writer      repository.CodeRepositoryWriter
	permissions PermissionService
	repoRoot    string
}

func NewCodeRepositoryService(
	reader repository.CodeRepositoryReader,
	writer repository.CodeRepositoryWriter,
	permissions PermissionService,
	repoRoot string,
) CodeRepositoryService {
	return CodeRepositoryService{
		reader:      reader,
		writer:      writer,
		permissions: permissions,
		repoRoot:    repoRoot,
	}
}

func (s CodeRepositoryService) List(ctx context.Context, filter query.CodeRepositoryListFilter) ([]query.CodeRepositoryItem, int, error) {
	return s.reader.ListCodeRepositories(ctx, filter)
}

func (s CodeRepositoryService) Get(ctx context.Context, id string) (*query.CodeRepositoryDetail, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: id is required", ErrValidation)
	}
	return s.reader.GetCodeRepository(ctx, id)
}

func (s CodeRepositoryService) GetBySlug(ctx context.Context, slug string) (*query.CodeRepositoryDetail, error) {
	if slug == "" {
		return nil, fmt.Errorf("%w: slug is required", ErrValidation)
	}
	return s.reader.GetCodeRepositoryBySlug(ctx, slug)
}

func (s CodeRepositoryService) Create(ctx context.Context, input command.CodeRepositoryCreateInput) (*query.CodeRepositoryDetail, error) {
	if input.TeamSpaceID == "" {
		return nil, fmt.Errorf("%w: team_space_id is required", ErrValidation)
	}
	if input.ProjectID == "" {
		return nil, fmt.Errorf("%w: project_id is required", ErrValidation)
	}
	if strings.TrimSpace(input.Name) == "" {
		return nil, fmt.Errorf("%w: name is required", ErrValidation)
	}
	if strings.TrimSpace(input.TargetFolderPath) == "" {
		return nil, fmt.Errorf("%w: target_folder_path is required", ErrValidation)
	}
	if err := validateTargetFolder(input.TargetFolderPath); err != nil {
		return nil, err
	}
	if input.DefaultBranch == "" {
		input.DefaultBranch = "main"
	}
	if input.Slug == "" {
		input.Slug = makeSlug(input.Name)
	}
	if input.Slug == "" {
		input.Slug = "repo"
	}
	input.Slug = input.Slug + "-" + shortToken(4)
	input.PushToken = shortToken(24)
	input.RepoStoragePath = filepath.Join(s.repoRoot, input.Slug+".git")

	if err := s.permissions.EnsureCreateCodeRepository(ctx, input.ActorID, input.ActorRole, input.ProjectID); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(s.repoRoot, 0o755); err != nil {
		return nil, err
	}
	if err := runGit(ctx, "", "init", "--bare", input.RepoStoragePath); err != nil {
		return nil, fmt.Errorf("git init failed: %w", err)
	}
	if err := runGit(ctx, input.RepoStoragePath, "symbolic-ref", "HEAD", "refs/heads/"+input.DefaultBranch); err != nil {
		return nil, fmt.Errorf("git default branch failed: %w", err)
	}
	return s.writer.CreateCodeRepository(ctx, input)
}

func (s CodeRepositoryService) Update(ctx context.Context, input command.CodeRepositoryUpdateInput) (*query.CodeRepositoryDetail, error) {
	if input.RepositoryID == "" {
		return nil, fmt.Errorf("%w: id is required", ErrValidation)
	}
	if input.TargetFolderPath != "" {
		if err := validateTargetFolder(input.TargetFolderPath); err != nil {
			return nil, err
		}
	}
	if err := s.permissions.EnsureManageCodeRepository(ctx, input.ActorID, input.ActorRole, input.RepositoryID); err != nil {
		return nil, err
	}
	return s.writer.UpdateCodeRepository(ctx, input)
}

func (s CodeRepositoryService) ListPushEvents(ctx context.Context, repositoryID string) ([]query.CodePushEventItem, error) {
	if repositoryID == "" {
		return nil, fmt.Errorf("%w: repository_id is required", ErrValidation)
	}
	return s.reader.ListCodePushEvents(ctx, repositoryID)
}

func (s CodeRepositoryService) AuthenticatePush(ctx context.Context, slug string, token string) (*query.CodeRepositoryDetail, bool, error) {
	repo, err := s.reader.GetCodeRepositoryBySlug(ctx, slug)
	if err != nil {
		return nil, false, err
	}
	return repo, token != "" && token == repo.PushToken, nil
}

func (s CodeRepositoryService) RecordPush(ctx context.Context, repo *query.CodeRepositoryDetail, pusherID string) (*query.CodePushEventItem, error) {
	branch := repo.DefaultBranch
	afterSHA := gitOutput(ctx, repo.RepoStoragePath, "rev-parse", "--verify", "refs/heads/"+branch)
	commitMessage := ""
	if afterSHA != "" {
		commitMessage = gitOutput(ctx, repo.RepoStoragePath, "log", "-1", "--pretty=%s", afterSHA)
	}
	status := "synced"
	errMsg := ""
	if afterSHA == "" {
		status = "failed"
		errMsg = "default branch was not pushed"
	}
	event, err := s.writer.CreateCodePushEvent(ctx, command.CodePushEventCreateInput{
		RepositoryID:  repo.ID,
		Branch:        branch,
		AfterSHA:      afterSHA,
		CommitMessage: commitMessage,
		PusherID:      pusherID,
		SyncStatus:    status,
		ErrorMessage:  errMsg,
	})
	if err != nil {
		return nil, err
	}
	if afterSHA != "" {
		if err := s.writer.UpdateCodeRepositoryAfterPush(ctx, repo.ID, afterSHA, "active"); err != nil {
			return nil, err
		}
	}
	return event, nil
}

func runGit(ctx context.Context, gitDir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	if gitDir != "" {
		cmd.Env = append(os.Environ(), "GIT_DIR="+gitDir)
	} else {
		cmd.Env = os.Environ()
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", args, strings.TrimSpace(string(out)))
	}
	return nil
}

func gitOutput(ctx context.Context, gitDir string, args ...string) string {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(os.Environ(), "GIT_DIR="+gitDir)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

var slugRe = regexp.MustCompile(`[^a-z0-9-]+`)

func makeSlug(value string) string {
	slug := strings.ToLower(strings.TrimSpace(value))
	slug = strings.ReplaceAll(slug, "_", "-")
	slug = slugRe.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 48 {
		slug = slug[:48]
	}
	return strings.Trim(slug, "-")
}

func validateTargetFolder(path string) error {
	if strings.Contains(path, "..") || strings.Contains(path, "\\") {
		return fmt.Errorf("%w: target folder path is invalid", ErrValidation)
	}
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("%w: target folder path must start with /", ErrValidation)
	}
	return nil
}

func shortToken(bytesLen int) string {
	b := make([]byte, bytesLen)
	if _, err := rand.Read(b); err != nil {
		return "token"
	}
	return hex.EncodeToString(b)
}
