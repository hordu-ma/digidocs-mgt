package memory

import (
	"context"
	"io"
	"strings"
	"testing"

	"digidocs-mgt/backend-go/internal/storage"
)

func TestProviderObjectLifecycle(t *testing.T) {
	ctx := context.Background()
	provider := NewProvider()

	result, err := provider.PutObject(ctx, storage.PutObjectInput{
		ObjectKey: "project/docs/report.txt",
		Reader:    strings.NewReader("hello"),
	})
	if err != nil {
		t.Fatalf("put object failed: %v", err)
	}
	if result.Provider != "memory" || result.ObjectKey != "project/docs/report.txt" {
		t.Fatalf("unexpected put result: %+v", result)
	}

	object, err := provider.GetObject(ctx, "project/docs/report.txt")
	if err != nil {
		t.Fatalf("get object failed: %v", err)
	}
	defer object.Reader.Close()
	body, err := io.ReadAll(object.Reader)
	if err != nil {
		t.Fatalf("read object failed: %v", err)
	}
	if string(body) != "hello" || object.Size != 5 {
		t.Fatalf("unexpected object: body=%q size=%d", string(body), object.Size)
	}

	info, err := provider.Stat(ctx, "project/docs/report.txt")
	if err != nil {
		t.Fatalf("stat file failed: %v", err)
	}
	if info.Name != "report.txt" || info.IsDir || info.Size != 5 {
		t.Fatalf("unexpected file info: %+v", info)
	}

	dirInfo, err := provider.Stat(ctx, "project/docs")
	if err != nil {
		t.Fatalf("stat dir failed: %v", err)
	}
	if !dirInfo.IsDir || dirInfo.Name != "docs" {
		t.Fatalf("unexpected dir info: %+v", dirInfo)
	}

	items, err := provider.ListDir(ctx, "project")
	if err != nil {
		t.Fatalf("list dir failed: %v", err)
	}
	if len(items) != 1 || !items[0].IsDir || items[0].Name != "docs" {
		t.Fatalf("unexpected list result: %+v", items)
	}

	link, err := provider.CreateShareLink(ctx, "project/docs/report.txt", 1)
	if err != nil {
		t.Fatalf("share link failed: %v", err)
	}
	if !strings.HasPrefix(link.URL, "memory://share/") || link.ExpiresAt == nil {
		t.Fatalf("unexpected share link: %+v", link)
	}

	if err := provider.DeleteObject(ctx, "project/docs"); err != nil {
		t.Fatalf("delete dir failed: %v", err)
	}
	if _, err := provider.GetObject(ctx, "project/docs/report.txt"); err == nil {
		t.Fatal("expected deleted child object to be missing")
	}
}

func TestProviderErrorsAndFolderCreation(t *testing.T) {
	ctx := context.Background()
	provider := NewProvider()

	if _, err := provider.GetObject(ctx, "missing"); err == nil {
		t.Fatal("expected get missing object to fail")
	}
	if _, err := provider.Stat(ctx, "missing"); err == nil {
		t.Fatal("expected stat missing object to fail")
	}
	if err := provider.DeleteObject(ctx, "missing"); err == nil {
		t.Fatal("expected delete missing object to fail")
	}
	if _, err := provider.CreateShareLink(ctx, "missing", 0); err == nil {
		t.Fatal("expected share missing object to fail")
	}

	if err := provider.CreateFolder(ctx, "a/b/c"); err != nil {
		t.Fatalf("create folder failed: %v", err)
	}
	for _, folder := range []string{"a", "a/b", "a/b/c"} {
		info, err := provider.Stat(ctx, folder)
		if err != nil {
			t.Fatalf("stat created folder %q failed: %v", folder, err)
		}
		if !info.IsDir {
			t.Fatalf("expected %q to be a dir: %+v", folder, info)
		}
	}

	if _, err := provider.PutObject(ctx, storage.PutObjectInput{
		ObjectKey: "bad",
		Reader:    errorReader{},
	}); err == nil {
		t.Fatal("expected read error to be returned")
	}
}

type errorReader struct{}

func (errorReader) Read(_ []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}
