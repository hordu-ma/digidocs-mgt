package main

import (
	"context"
	"errors"
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
