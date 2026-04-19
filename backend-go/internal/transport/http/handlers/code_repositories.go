package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
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
	})
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to list code repositories")
		return
	}
	response.WriteData(w, http.StatusOK, map[string]any{"items": items, "total": total, "page": page})
}

func (h CodeRepositoryHandler) Get(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			response.WriteError(w, http.StatusNotFound, "not_found", "code repository not found")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to get code repository")
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
		h.writeServiceError(w, err, "failed to create code repository")
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
		h.writeServiceError(w, err, "failed to update code repository")
		return
	}
	item.RemoteURL = h.remoteURL(r, item.Slug)
	response.WriteData(w, http.StatusOK, item)
}

func (h CodeRepositoryHandler) ListPushEvents(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListPushEvents(r.Context(), r.PathValue("id"))
	if err != nil {
		h.writeServiceError(w, err, "failed to list push events")
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
	out, err := cmd.Output()
	if err != nil {
		return err
	}
	header, body, ok := bytes.Cut(out, []byte("\r\n\r\n"))
	if !ok {
		header, body, ok = bytes.Cut(out, []byte("\n\n"))
	}
	if !ok {
		_, _ = w.Write(out)
		return nil
	}
	status := http.StatusOK
	for _, line := range strings.Split(strings.ReplaceAll(string(header), "\r\n", "\n"), "\n") {
		if line == "" {
			continue
		}
		key, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if strings.EqualFold(key, "Status") {
			fields := strings.Fields(value)
			if len(fields) > 0 {
				if code, err := strconv.Atoi(fields[0]); err == nil {
					status = code
				}
			}
			continue
		}
		w.Header().Add(key, value)
	}
	w.WriteHeader(status)
	_, _ = io.Copy(w, bytes.NewReader(body))
	return nil
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

func (h CodeRepositoryHandler) writeServiceError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, service.ErrValidation):
		response.WriteError(w, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, service.ErrForbidden):
		response.WriteError(w, http.StatusForbidden, "forbidden", "permission denied")
	case errors.Is(err, service.ErrNotFound):
		response.WriteError(w, http.StatusNotFound, "not_found", "code repository not found")
	case errors.Is(err, service.ErrConflict):
		response.WriteError(w, http.StatusConflict, "conflict", "code repository already exists")
	default:
		response.WriteError(w, http.StatusInternalServerError, "internal_error", fallback)
	}
}
