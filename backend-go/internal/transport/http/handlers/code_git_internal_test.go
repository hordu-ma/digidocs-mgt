package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"digidocs-mgt/backend-go/internal/service"
)

// runGitHTTPBackend streams git's CGI output instead of buffering it. That
// rewrite is easy to get subtly wrong (header/body split, status parsing,
// truncation at the boundary), and a broken git remote only shows up when
// someone actually pushes — so exercise the real `git http-backend` here.

func gitAvailable(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

// newBareRepoWithCommit builds a bare repository containing one commit and
// returns its path.
func newBareRepoWithCommit(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	bare := filepath.Join(root, "sample.git")
	work := filepath.Join(root, "work")

	run := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	run(root, "init", "--bare", bare)
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatalf("mkdir work: %v", err)
	}
	run(root, "init", work)
	if err := os.WriteFile(filepath.Join(work, "README.md"), []byte("# sample\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	run(work, "add", ".")
	run(work, "commit", "-m", "initial")
	run(work, "branch", "-M", "main")
	run(work, "remote", "add", "origin", bare)
	run(work, "push", "origin", "main")

	return bare
}

func TestRunGitHTTPBackendStreamsInfoRefs(t *testing.T) {
	gitAvailable(t)
	bare := newBareRepoWithCommit(t)

	h := NewCodeRepositoryHandler(service.CodeRepositoryService{}, "/api/v1")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/git/sample.git/info/refs?service=git-upload-pack", nil)

	if err := h.runGitHTTPBackend(rec, req, bare, "/sample.git/info/refs"); err != nil {
		t.Fatalf("runGitHTTPBackend: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %q)", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/x-git-upload-pack-advertisement" {
		t.Errorf("Content-Type = %q, want the smart-HTTP advertisement type", ct)
	}

	body := rec.Body.String()
	// The smart-HTTP preamble must survive the header/body split intact.
	if !strings.HasPrefix(body, "001e# service=git-upload-pack") {
		t.Fatalf("body does not start with the git service preamble: %q", body[:min(60, len(body))])
	}
	// The ref advertisement must reach the client, i.e. the body was not cut
	// short at the header boundary.
	if !strings.Contains(body, "refs/heads/main") {
		t.Errorf("advertisement does not mention refs/heads/main: %q", body)
	}
}

func TestRunGitHTTPBackendPropagatesNotFoundStatus(t *testing.T) {
	gitAvailable(t)
	root := t.TempDir()

	h := NewCodeRepositoryHandler(service.CodeRepositoryService{}, "/api/v1")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/git/missing.git/info/refs?service=git-upload-pack", nil)

	// git http-backend answers a nonexistent repository with a CGI Status line;
	// the handler must forward that status rather than always writing 200.
	if err := h.runGitHTTPBackend(rec, req, filepath.Join(root, "missing.git"), "/missing.git/info/refs"); err != nil {
		t.Fatalf("runGitHTTPBackend: %v", err)
	}
	if rec.Code == http.StatusOK {
		t.Errorf("status = 200 for a missing repository, want the CGI error status")
	}
}
