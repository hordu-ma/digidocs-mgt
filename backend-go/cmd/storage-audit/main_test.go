package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"digidocs-mgt/backend-go/internal/storage"
)

type fakeTree map[string][]storage.FileInfo

func (f fakeTree) ListDir(_ context.Context, dir string) ([]storage.FileInfo, error) {
	if dir == "data" {
		return nil, errors.New("permission denied")
	}
	return f[dir], nil
}

func TestWalkAndReconcile(t *testing.T) {
	tree := fakeTree{
		"documents":    {{Name: "d1", Path: "/DigiDocs/documents/d1", IsDir: true}},
		"documents/d1": {{Name: "a.docx", Path: "/DigiDocs/documents/d1/a.docx", Size: 10}, {Name: "old.docx", Path: "/DigiDocs/documents/d1/old.docx", Size: 7}},
	}
	stored, errs := walk(context.Background(), tree, "/DigiDocs")
	if len(stored) != 2 || len(errs) != 1 {
		t.Fatalf("walk: %#v errs=%v", stored, errs)
	}
	rep := reconcile(stored, map[string]bool{
		"documents/d1/a.docx":   true,
		"documents/d2/gone.pdf": true,
		"code/repo/file.go":     true, // outside audited prefixes: never "missing"
	})
	if len(rep.Orphans) != 1 || rep.Orphans[0].Key != "documents/d1/old.docx" || rep.OrphanBytes != 7 {
		t.Fatalf("orphans: %#v", rep)
	}
	if len(rep.MissingObjects) != 1 || rep.MissingObjects[0] != "documents/d2/gone.pdf" {
		t.Fatalf("missing: %#v", rep.MissingObjects)
	}
}

type fakeStore struct {
	deleted []string
	content map[string]string
}

func (f *fakeStore) ListDir(context.Context, string) ([]storage.FileInfo, error) { return nil, nil }
func (f *fakeStore) GetObject(_ context.Context, k string) (*storage.GetObjectOutput, error) {
	return &storage.GetObjectOutput{Reader: io.NopCloser(strings.NewReader(f.content[k]))}, nil
}
func (f *fakeStore) DeleteObject(_ context.Context, k string) error {
	f.deleted = append(f.deleted, k)
	return nil
}

func TestDeleteOrphansValidatesEveryKeyFirst(t *testing.T) {
	store := &fakeStore{content: map[string]string{"documents/d/old.txt": "x"}}
	rep := report{Orphans: []object{{Key: "documents/d/old.txt"}}}
	err := deleteOrphans(context.Background(), store, rep, []string{"documents/d/old.txt", "documents/d/used.txt"}, t.TempDir(), io.Discard)
	if err == nil || len(store.deleted) != 0 {
		t.Fatalf("a referenced key must abort the run before any delete: err=%v deleted=%v", err, store.deleted)
	}
	if err := deleteOrphans(context.Background(), store, report{Orphans: rep.Orphans, ListErrors: []string{"x"}}, []string{"documents/d/old.txt"}, t.TempDir(), io.Discard); err == nil {
		t.Fatal("incomplete listing must refuse deletion")
	}
	dir := t.TempDir()
	if err := deleteOrphans(context.Background(), store, rep, []string{"documents/d/old.txt"}, dir, io.Discard); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "documents", "d", "old.txt"))
	if err != nil || string(b) != "x" || len(store.deleted) != 1 {
		t.Fatalf("backup then delete: %q %v %v", b, err, store.deleted)
	}
}
