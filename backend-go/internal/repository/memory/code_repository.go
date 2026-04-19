package memory

import (
	"context"
	"strings"
	"sync"
	"time"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/service"
)

type codeRepoRecord struct {
	item      query.CodeRepositoryDetail
	pushToken string
}

type codePushRecord struct {
	item query.CodePushEventItem
}

type CodeRepositoryRepository struct {
	mu     sync.RWMutex
	repos  []codeRepoRecord
	events []codePushRecord
}

func NewCodeRepositoryRepository() *CodeRepositoryRepository {
	return &CodeRepositoryRepository{}
}

func (r *CodeRepositoryRepository) ListCodeRepositories(ctx context.Context, filter query.CodeRepositoryListFilter) ([]query.CodeRepositoryItem, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	keyword := strings.ToLower(filter.Keyword)
	items := make([]query.CodeRepositoryItem, 0)
	for _, rec := range r.repos {
		if filter.ProjectID != "" && rec.item.ProjectID != filter.ProjectID {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(rec.item.Name), keyword) && !strings.Contains(strings.ToLower(rec.item.Slug), keyword) {
			continue
		}
		items = append(items, rec.item.CodeRepositoryItem)
	}
	return items, len(items), nil
}

func (r *CodeRepositoryRepository) GetCodeRepository(ctx context.Context, id string) (*query.CodeRepositoryDetail, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, rec := range r.repos {
		if rec.item.ID == id {
			item := rec.item
			return &item, nil
		}
	}
	return nil, service.ErrNotFound
}

func (r *CodeRepositoryRepository) GetCodeRepositoryBySlug(ctx context.Context, slug string) (*query.CodeRepositoryDetail, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, rec := range r.repos {
		if rec.item.Slug == slug {
			item := rec.item
			item.PushToken = rec.pushToken
			return &item, nil
		}
	}
	return nil, service.ErrNotFound
}

func (r *CodeRepositoryRepository) CreateCodeRepository(ctx context.Context, input command.CodeRepositoryCreateInput) (*query.CodeRepositoryDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, rec := range r.repos {
		if rec.item.Slug == input.Slug {
			return nil, service.ErrConflict
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	item := query.CodeRepositoryDetail{
		CodeRepositoryItem: query.CodeRepositoryItem{
			ID:               newID(),
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
	r.repos = append(r.repos, codeRepoRecord{item: item, pushToken: input.PushToken})
	return &item, nil
}

func (r *CodeRepositoryRepository) UpdateCodeRepository(ctx context.Context, input command.CodeRepositoryUpdateInput) (*query.CodeRepositoryDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.repos {
		if r.repos[i].item.ID == input.RepositoryID {
			if input.Name != "" {
				r.repos[i].item.Name = input.Name
			}
			r.repos[i].item.Description = input.Description
			if input.DefaultBranch != "" {
				r.repos[i].item.DefaultBranch = input.DefaultBranch
			}
			if input.TargetFolderPath != "" {
				r.repos[i].item.TargetFolderPath = input.TargetFolderPath
			}
			r.repos[i].item.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			item := r.repos[i].item
			return &item, nil
		}
	}
	return nil, service.ErrNotFound
}

func (r *CodeRepositoryRepository) CreateCodePushEvent(ctx context.Context, input command.CodePushEventCreateInput) (*query.CodePushEventItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC().Format(time.RFC3339)
	item := query.CodePushEventItem{
		ID:            newID(),
		RepositoryID:  input.RepositoryID,
		Branch:        input.Branch,
		BeforeSHA:     input.BeforeSHA,
		AfterSHA:      input.AfterSHA,
		CommitMessage: input.CommitMessage,
		SyncStatus:    input.SyncStatus,
		ErrorMessage:  input.ErrorMessage,
		CreatedAt:     now,
		CompletedAt:   now,
	}
	r.events = append([]codePushRecord{{item: item}}, r.events...)
	return &item, nil
}

func (r *CodeRepositoryRepository) UpdateCodeRepositoryAfterPush(ctx context.Context, repositoryID string, commitSHA string, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.repos {
		if r.repos[i].item.ID == repositoryID {
			r.repos[i].item.LastCommitSHA = commitSHA
			r.repos[i].item.LastPushedAt = time.Now().UTC().Format(time.RFC3339)
			r.repos[i].item.Status = status
			r.repos[i].item.UpdatedAt = r.repos[i].item.LastPushedAt
			return nil
		}
	}
	return service.ErrNotFound
}

func (r *CodeRepositoryRepository) ListCodePushEvents(ctx context.Context, repositoryID string) ([]query.CodePushEventItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]query.CodePushEventItem, 0)
	for _, rec := range r.events {
		if rec.item.RepositoryID == repositoryID {
			items = append(items, rec.item)
		}
	}
	return items, nil
}
