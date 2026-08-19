package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"digidocs-mgt/backend-go/internal/config"
	"digidocs-mgt/backend-go/internal/storage"
)

type fakeStore struct {
	putErr    error
	putCalls  []storage.PutObjectInput
	listDir   map[string][]storage.FileInfo
	listErr   error
	deleteErr error
	deleted   []string
}

func (f *fakeStore) PutObject(_ context.Context, input storage.PutObjectInput) (storage.PutObjectResult, error) {
	f.putCalls = append(f.putCalls, input)
	if f.putErr != nil {
		return storage.PutObjectResult{}, f.putErr
	}
	return storage.PutObjectResult{ObjectKey: input.ObjectKey, Provider: "fake"}, nil
}

func (f *fakeStore) ListDir(_ context.Context, folderPath string) ([]storage.FileInfo, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.listDir[folderPath], nil
}

func (f *fakeStore) DeleteObject(_ context.Context, objectKey string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleted = append(f.deleted, objectKey)
	return nil
}

func withStore(store *fakeStore) func(config.Config) objectStore {
	return func(config.Config) objectStore { return store }
}

func TestRunRequiresSubcommand(t *testing.T) {
	err := run(nil, func() config.Config { return config.Config{SynologyHost: "nas.local"} }, withStore(&fakeStore{}))
	if err == nil || !strings.Contains(err.Error(), "usage:") {
		t.Fatalf("expected usage error, got %v", err)
	}
}

func TestRunRequiresSynologyHost(t *testing.T) {
	err := run([]string{"upload", "local-file", "remote-key"}, func() config.Config { return config.Config{} }, withStore(&fakeStore{}))
	if err == nil || !strings.Contains(err.Error(), "SYNOLOGY_HOST") {
		t.Fatalf("expected missing-host error, got %v", err)
	}
}

func TestRunRejectsUnknownSubcommand(t *testing.T) {
	err := run([]string{"delete-everything"}, func() config.Config { return config.Config{SynologyHost: "nas.local"} }, withStore(&fakeStore{}))
	if err == nil || !strings.Contains(err.Error(), "unknown subcommand") {
		t.Fatalf("expected unknown subcommand error, got %v", err)
	}
}

func TestUploadRejectsMissingLocalFile(t *testing.T) {
	err := run([]string{"upload", "/no/such/file", "remote-key"},
		func() config.Config { return config.Config{SynologyHost: "nas.local"} }, withStore(&fakeStore{}))
	if err == nil || !strings.Contains(err.Error(), "open") {
		t.Fatalf("expected open error, got %v", err)
	}
}

func TestUploadSendsFileToObjectKey(t *testing.T) {
	dir := t.TempDir()
	localFile := filepath.Join(dir, "db.sql.gz")
	if err := os.WriteFile(localFile, []byte("payload"), 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	store := &fakeStore{}
	err := run([]string{"upload", localFile, "ops-backups/db.sql.gz"},
		func() config.Config { return config.Config{SynologyHost: "nas.local"} }, withStore(store))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if len(store.putCalls) != 1 {
		t.Fatalf("expected 1 PutObject call, got %d", len(store.putCalls))
	}
	call := store.putCalls[0]
	if call.ObjectKey != "ops-backups/db.sql.gz" || !call.Overwrite || !call.CreatePaths {
		t.Fatalf("unexpected PutObject input: %+v", call)
	}
}

func TestPruneDeletesOnlyStaleFiles(t *testing.T) {
	now := time.Now()
	store := &fakeStore{
		listDir: map[string][]storage.FileInfo{
			"ops-backups": {
				{Name: "old.sql.gz", IsDir: false, ModifiedAt: now.Add(-20 * 24 * time.Hour)},
				{Name: "fresh.sql.gz", IsDir: false, ModifiedAt: now.Add(-1 * time.Hour)},
				{Name: "subdir", IsDir: true, ModifiedAt: now.Add(-30 * 24 * time.Hour)},
			},
		},
	}

	err := run([]string{"prune", "ops-backups", "14"},
		func() config.Config { return config.Config{SynologyHost: "nas.local"} }, withStore(store))
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if len(store.deleted) != 1 || store.deleted[0] != "ops-backups/old.sql.gz" {
		t.Fatalf("expected only old.sql.gz to be pruned, got %+v", store.deleted)
	}
}

func TestPruneRejectsInvalidKeepDays(t *testing.T) {
	err := run([]string{"prune", "ops-backups", "not-a-number"},
		func() config.Config { return config.Config{SynologyHost: "nas.local"} }, withStore(&fakeStore{}))
	if err == nil || !strings.Contains(err.Error(), "keep-days") {
		t.Fatalf("expected keep-days error, got %v", err)
	}
}
