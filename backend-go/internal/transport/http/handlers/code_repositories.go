package handlers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/service"
	"digidocs-mgt/backend-go/internal/transport/http/middleware"
	"digidocs-mgt/backend-go/internal/transport/http/response"
)

type CodeRepositoryHandler struct {
	service service.CodeRepositoryService
	prefix  string
}

func NewCodeRepositoryHandler(svc service.CodeRepositoryService, apiPrefix string) CodeRepositoryHandler {
	return CodeRepositoryHandler{service: svc, prefix: apiPrefix}
}

func (h CodeRepositoryHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	items, total, err := h.service.List(r.Context(), query.CodeRepositoryListFilter{
		ProjectID: r.URL.Query().Get("project_id"),
		Keyword:   r.URL.Query().Get("keyword"),
		Page:      page,
		PageSize:  pageSize,
		ActorID:   middleware.UserIDFromContext(r.Context()),
		ActorRole: middleware.UserRoleFromContext(r.Context()),
	})
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to list code repositories")
		return
	}
	response.WriteData(w, http.StatusOK, map[string]any{"items": items, "total": total, "page": page})
}

func (h CodeRepositoryHandler) Get(w http.ResponseWriter, r *http.Request) {
	actorID := middleware.UserIDFromContext(r.Context())
	actorRole := middleware.UserRoleFromContext(r.Context())
	item, err := h.service.Get(r.Context(), r.PathValue("id"), actorID, actorRole)
	if err != nil {
		writeServiceError(w, err, "code repository not found", "failed to get code repository")
		return
	}
	item.RemoteURL = h.remoteURL(r, item.Slug)
	response.WriteData(w, http.StatusOK, item)
}

func (h CodeRepositoryHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TeamSpaceID      string `json:"team_space_id"`
		ProjectID        string `json:"project_id"`
		Name             string `json:"name"`
		Description      string `json:"description"`
		DefaultBranch    string `json:"default_branch"`
		TargetFolderPath string `json:"target_folder_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid JSON")
		return
	}
	item, err := h.service.Create(r.Context(), command.CodeRepositoryCreateInput{
		TeamSpaceID:      body.TeamSpaceID,
		ProjectID:        body.ProjectID,
		Name:             body.Name,
		Description:      body.Description,
		DefaultBranch:    body.DefaultBranch,
		TargetFolderPath: body.TargetFolderPath,
		ActorID:          middleware.UserIDFromContext(r.Context()),
		ActorRole:        middleware.UserRoleFromContext(r.Context()),
	})
	if err != nil {
		writeServiceError(w, err, "code repository not found", "failed to create code repository")
		return
	}
	item.RemoteURL = h.remoteURL(r, item.Slug)
	response.WriteData(w, http.StatusCreated, item)
}

func (h CodeRepositoryHandler) Update(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name             string `json:"name"`
		Description      string `json:"description"`
		DefaultBranch    string `json:"default_branch"`
		TargetFolderPath string `json:"target_folder_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.WriteError(w, http.StatusBadRequest, "bad_request", "invalid JSON")
		return
	}
	item, err := h.service.Update(r.Context(), command.CodeRepositoryUpdateInput{
		RepositoryID:     r.PathValue("id"),
		Name:             body.Name,
		Description:      body.Description,
		DefaultBranch:    body.DefaultBranch,
		TargetFolderPath: body.TargetFolderPath,
		ActorID:          middleware.UserIDFromContext(r.Context()),
		ActorRole:        middleware.UserRoleFromContext(r.Context()),
	})
	if err != nil {
		writeServiceError(w, err, "code repository not found", "failed to update code repository")
		return
	}
	item.RemoteURL = h.remoteURL(r, item.Slug)
	response.WriteData(w, http.StatusOK, item)
}

func (h CodeRepositoryHandler) ListPushEvents(w http.ResponseWriter, r *http.Request) {
	actorID, actorRole := actorFrom(r)
	items, err := h.service.ListPushEvents(r.Context(), r.PathValue("id"), actorID, actorRole)
	if err != nil {
		writeServiceError(w, err, "code repository not found", "failed to list push events")
		return
	}
	response.WriteData(w, http.StatusOK, items)
}

func (h CodeRepositoryHandler) ServeGit(w http.ResponseWriter, r *http.Request) {
	rel := strings.TrimPrefix(r.URL.Path, h.prefix+"/git/")
	parts := strings.SplitN(rel, ".git", 2)
	if len(parts) != 2 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	slug, err := url.PathUnescape(parts[0])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	_, token, _ := r.BasicAuth()
	repo, ok, err := h.service.AuthenticatePush(r.Context(), slug, token)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to authenticate git repository")
		return
	}
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="DigiDocs Code"`)
		response.WriteError(w, http.StatusUnauthorized, "unauthorized", "invalid code repository token")
		return
	}

	pathInfo := "/" + slug + ".git" + parts[1]
	if err := h.runGitHTTPBackend(w, r, repo.RepoStoragePath, pathInfo); err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "git backend failed")
		return
	}
	if r.Method == http.MethodPost && strings.HasSuffix(pathInfo, "/git-receive-pack") {
		if _, err := h.service.RecordPush(r.Context(), repo, ""); err != nil {
			// Git already accepted the pack; keep the client response successful and rely on logs.
			return
		}
	}
}

// runGitHTTPBackend proxies the request to `git http-backend` (CGI) and streams
// the response straight through.
//
// This used to buffer git's entire stdout with cmd.Output() before writing a
// byte, which meant a clone allocated the whole pack in memory and lost
// git's incremental progress reporting. Only the CGI header block is read
// eagerly; the body is copied as it arrives.
func (h CodeRepositoryHandler) runGitHTTPBackend(w http.ResponseWriter, r *http.Request, repoPath string, pathInfo string) error {
	cmd := exec.CommandContext(r.Context(), "git", "http-backend")
	cmd.Env = append(os.Environ(),
		"GIT_PROJECT_ROOT="+filepath.Dir(repoPath),
		"GIT_HTTP_EXPORT_ALL=1",
		"REMOTE_USER=digidocs",
		"REQUEST_METHOD="+r.Method,
		"QUERY_STRING="+r.URL.RawQuery,
		"PATH_INFO="+pathInfo,
		"CONTENT_TYPE="+r.Header.Get("Content-Type"),
		"CONTENT_LENGTH="+r.Header.Get("Content-Length"),
	)
	if r.Body != nil {
		defer r.Body.Close()
		cmd.Stdin = r.Body
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	reader := bufio.NewReader(stdout)
	status, err := writeCGIHeaders(w, reader)
	if err != nil {
		_ = cmd.Wait()
		return err
	}
	w.WriteHeader(status)

	// Flush per chunk so git's sideband progress reaches the client live
	// instead of arriving all at once when the process exits.
	flusher, _ := w.(http.Flusher)
	buf := make([]byte, 32*1024)
	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				_ = cmd.Wait()
				return writeErr
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			_ = cmd.Wait()
			return readErr
		}
	}

	if err := cmd.Wait(); err != nil {
		log.Printf("[code-repo] git http-backend failed path=%s stderr=%s err=%v",
			pathInfo, strings.TrimSpace(stderr.String()), err)
		// The response is already on the wire; surfacing a second status here
		// would corrupt it, so the failure is only logged.
		return nil
	}
	return nil
}

// writeCGIHeaders consumes the CGI header block from reader, copies the headers
// onto w and returns the status code to send. The reader is left positioned at
// the first byte of the body.
func writeCGIHeaders(w http.ResponseWriter, reader *bufio.Reader) (int, error) {
	status := http.StatusOK
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && strings.TrimSpace(line) == "" {
				return status, nil
			}
			return status, err
		}
		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "" {
			return status, nil
		}
		key, value, found := strings.Cut(trimmed, ":")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if strings.EqualFold(key, "Status") {
			if fields := strings.Fields(value); len(fields) > 0 {
				if code, convErr := strconv.Atoi(fields[0]); convErr == nil {
					status = code
				}
			}
			continue
		}
		w.Header().Add(key, value)
	}
}

func (h CodeRepositoryHandler) remoteURL(r *http.Request, slug string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if xf := r.Header.Get("X-Forwarded-Proto"); xf != "" {
		scheme = xf
	}
	return scheme + "://" + r.Host + h.prefix + "/git/" + slug + ".git"
}

func (h CodeRepositoryHandler) RotatePushToken(w http.ResponseWriter, r *http.Request) {
	actorID, actorRole := actorFrom(r)
	token, err := h.service.RotatePushToken(r.Context(), r.PathValue("id"), actorID, actorRole)
	if err != nil {
		writeServiceError(w, err, "code repository not found", "failed to rotate repository token")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	response.WriteData(w, http.StatusOK, map[string]string{"push_token": token})
}
