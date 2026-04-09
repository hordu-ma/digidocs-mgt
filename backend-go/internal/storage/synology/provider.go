package synology

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"digidocs-mgt/backend-go/internal/storage"
)

// Provider implements storage.Provider using Synology DSM FileStation API.
type Provider struct {
	baseURL   string // e.g. "http://192.168.1.100:5000"
	account   string
	password  string
	sharePath string // Synology share root, e.g. "/DigiDocs"

	mu  sync.RWMutex
	sid string // DSM session ID; guarded by mu

	client *http.Client
}

// Config holds Synology connection parameters.
type Config struct {
	Host               string // IP or hostname
	Port               int    // DSM port (5000 HTTP / 5001 HTTPS)
	HTTPS              bool
	InsecureSkipVerify bool
	Account            string
	Password           string
	SharePath          string // top-level shared folder path, e.g. "/DigiDocs"
}

func NewProvider(cfg Config) *Provider {
	scheme := "http"
	if cfg.HTTPS {
		scheme = "https"
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.HTTPS {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: cfg.InsecureSkipVerify,
		}
	}

	return &Provider{
		baseURL:   fmt.Sprintf("%s://%s:%d", scheme, cfg.Host, cfg.Port),
		account:   cfg.Account,
		password:  cfg.Password,
		sharePath: strings.TrimRight(cfg.SharePath, "/"),
		client:    &http.Client{Timeout: 120 * time.Second, Transport: transport},
	}
}

// --- DSM API response envelope ---

type dsmResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   *dsmError       `json:"error,omitempty"`
}

type dsmError struct {
	Code int `json:"code"`
}

func (e *dsmError) Error() string {
	return fmt.Sprintf("Synology API error code %d", e.Code)
}

// --- Session management ---

func (p *Provider) login(ctx context.Context) error {
	params := url.Values{
		"api":     {"SYNO.API.Auth"},
		"version": {"3"},
		"method":  {"login"},
		"account": {p.account},
		"passwd":  {p.password},
		"session": {"FileStation"},
		"format":  {"sid"},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		p.baseURL+"/webapi/auth.cgi?"+params.Encode(), nil)
	if err != nil {
		return fmt.Errorf("synology login request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("synology login: %w", err)
	}
	defer resp.Body.Close()

	var dr dsmResponse
	if err := json.NewDecoder(resp.Body).Decode(&dr); err != nil {
		return fmt.Errorf("synology login decode: %w", err)
	}
	if !dr.Success {
		return fmt.Errorf("synology login failed: %v", dr.Error)
	}

	var authData struct {
		SID string `json:"sid"`
	}
	if err := json.Unmarshal(dr.Data, &authData); err != nil {
		return fmt.Errorf("synology login unmarshal sid: %w", err)
	}

	p.mu.Lock()
	p.sid = authData.SID
	p.mu.Unlock()
	return nil
}

func (p *Provider) getSID() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.sid
}

// ensureSession performs login if sid is empty.
func (p *Provider) ensureSession(ctx context.Context) error {
	if p.getSID() != "" {
		return nil
	}
	return p.login(ctx)
}

// callAPI executes a FileStation API call and returns parsed data.
// On auth error (code 105/106), it re-authenticates once and retries.
func (p *Provider) callAPI(ctx context.Context, params url.Values) (*dsmResponse, error) {
	if err := p.ensureSession(ctx); err != nil {
		return nil, err
	}

	params.Set("_sid", p.getSID())
	reqURL := p.baseURL + "/webapi/entry.cgi?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var dr dsmResponse
	if err := json.NewDecoder(resp.Body).Decode(&dr); err != nil {
		return nil, fmt.Errorf("synology decode: %w", err)
	}

	// re-auth on session expired
	if !dr.Success && dr.Error != nil && (dr.Error.Code == 105 || dr.Error.Code == 106) {
		if err := p.login(ctx); err != nil {
			return nil, err
		}
		params.Set("_sid", p.getSID())
		reqURL = p.baseURL + "/webapi/entry.cgi?" + params.Encode()
		req2, _ := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		resp2, err := p.client.Do(req2)
		if err != nil {
			return nil, err
		}
		defer resp2.Body.Close()
		if err := json.NewDecoder(resp2.Body).Decode(&dr); err != nil {
			return nil, fmt.Errorf("synology decode retry: %w", err)
		}
	}

	if !dr.Success {
		if dr.Error != nil {
			return nil, dr.Error
		}
		return nil, fmt.Errorf("synology API call failed")
	}
	return &dr, nil
}

// absPath converts an objectKey to the absolute Synology path.
func (p *Provider) absPath(objectKey string) string {
	return p.sharePath + "/" + objectKey
}

// --- Provider interface implementation ---

func (p *Provider) PutObject(ctx context.Context, input storage.PutObjectInput) (storage.PutObjectResult, error) {
	if err := p.ensureSession(ctx); err != nil {
		return storage.PutObjectResult{}, err
	}

	destFolder := p.absPath(path.Dir(input.ObjectKey))
	fileName := path.Base(input.ObjectKey)

	// auto-create parent folder
	if input.CreatePaths {
		if err := p.CreateFolder(ctx, path.Dir(input.ObjectKey)); err != nil {
			return storage.PutObjectResult{}, fmt.Errorf("create parent folder: %w", err)
		}
	}

	overwrite := "false"
	if input.Overwrite {
		overwrite = "true"
	}

	// build multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("api", "SYNO.FileStation.Upload")
	_ = writer.WriteField("version", "2")
	_ = writer.WriteField("method", "upload")
	_ = writer.WriteField("path", destFolder)
	_ = writer.WriteField("create_parents", "true")
	_ = writer.WriteField("overwrite", overwrite)

	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return storage.PutObjectResult{}, err
	}
	if input.Reader != nil {
		if _, err := io.Copy(part, input.Reader); err != nil {
			return storage.PutObjectResult{}, err
		}
	}
	writer.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.baseURL+"/webapi/entry.cgi?_sid="+url.QueryEscape(p.getSID()), &body)
	if err != nil {
		return storage.PutObjectResult{}, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := p.client.Do(req)
	if err != nil {
		return storage.PutObjectResult{}, fmt.Errorf("synology upload: %w", err)
	}
	defer resp.Body.Close()

	var dr dsmResponse
	if err := json.NewDecoder(resp.Body).Decode(&dr); err != nil {
		return storage.PutObjectResult{}, fmt.Errorf("synology upload decode: %w", err)
	}
	if !dr.Success {
		return storage.PutObjectResult{}, fmt.Errorf("synology upload failed: %v", dr.Error)
	}

	return storage.PutObjectResult{
		ObjectKey: input.ObjectKey,
		Provider:  "synology",
	}, nil
}

func (p *Provider) GetObject(ctx context.Context, objectKey string) (*storage.GetObjectOutput, error) {
	if err := p.ensureSession(ctx); err != nil {
		return nil, err
	}

	params := url.Values{
		"api":     {"SYNO.FileStation.Download"},
		"version": {"2"},
		"method":  {"download"},
		"path":    {p.absPath(objectKey)},
		"mode":    {"download"},
		"_sid":    {p.getSID()},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		p.baseURL+"/webapi/entry.cgi?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("synology download: %w", err)
	}

	// if response is JSON, it's an error
	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "application/json") {
		defer resp.Body.Close()
		var dr dsmResponse
		_ = json.NewDecoder(resp.Body).Decode(&dr)
		return nil, fmt.Errorf("synology download failed: %v", dr.Error)
	}

	return &storage.GetObjectOutput{
		Reader:      resp.Body,
		ContentType: ct,
		Size:        resp.ContentLength,
	}, nil
}

func (p *Provider) DeleteObject(ctx context.Context, objectKey string) error {
	_, err := p.callAPI(ctx, url.Values{
		"api":       {"SYNO.FileStation.Delete"},
		"version":   {"2"},
		"method":    {"delete"},
		"path":      {p.absPath(objectKey)},
		"recursive": {"true"},
	})
	return err
}

func (p *Provider) Stat(ctx context.Context, objectKey string) (*storage.FileInfo, error) {
	dr, err := p.callAPI(ctx, url.Values{
		"api":     {"SYNO.FileStation.List"},
		"version": {"2"},
		"method":  {"getinfo"},
		"path":    {p.absPath(objectKey)},
		"additional": {"[\"size\",\"time\"]"},
	})
	if err != nil {
		return nil, err
	}

	var info struct {
		Files []struct {
			Path     string `json:"path"`
			Name     string `json:"name"`
			IsDir    bool   `json:"isdir"`
			Additional struct {
				Size int64 `json:"size"`
				Time struct {
					Mtime int64 `json:"mtime"`
				} `json:"time"`
			} `json:"additional"`
		} `json:"files"`
	}
	if err := json.Unmarshal(dr.Data, &info); err != nil {
		return nil, fmt.Errorf("synology stat unmarshal: %w", err)
	}
	if len(info.Files) == 0 {
		return nil, fmt.Errorf("object not found: %s", objectKey)
	}

	f := info.Files[0]
	return &storage.FileInfo{
		Name:       f.Name,
		Path:       objectKey,
		IsDir:      f.IsDir,
		Size:       f.Additional.Size,
		ModifiedAt: time.Unix(f.Additional.Time.Mtime, 0),
	}, nil
}

func (p *Provider) ListDir(ctx context.Context, folderPath string) ([]storage.FileInfo, error) {
	dr, err := p.callAPI(ctx, url.Values{
		"api":         {"SYNO.FileStation.List"},
		"version":     {"2"},
		"method":      {"list"},
		"folder_path": {p.absPath(folderPath)},
		"additional":  {"[\"size\",\"time\"]"},
	})
	if err != nil {
		return nil, err
	}

	var listData struct {
		Files []struct {
			Path     string `json:"path"`
			Name     string `json:"name"`
			IsDir    bool   `json:"isdir"`
			Additional struct {
				Size int64 `json:"size"`
				Time struct {
					Mtime int64 `json:"mtime"`
				} `json:"time"`
			} `json:"additional"`
		} `json:"files"`
	}
	if err := json.Unmarshal(dr.Data, &listData); err != nil {
		return nil, fmt.Errorf("synology list unmarshal: %w", err)
	}

	items := make([]storage.FileInfo, 0, len(listData.Files))
	for _, f := range listData.Files {
		// convert Synology absolute path back to relative objectKey
		relPath := strings.TrimPrefix(f.Path, p.sharePath+"/")
		items = append(items, storage.FileInfo{
			Name:       f.Name,
			Path:       relPath,
			IsDir:      f.IsDir,
			Size:       f.Additional.Size,
			ModifiedAt: time.Unix(f.Additional.Time.Mtime, 0),
		})
	}
	return items, nil
}

func (p *Provider) CreateFolder(ctx context.Context, folderPath string) error {
	absFolder := p.absPath(folderPath)
	parentDir := path.Dir(absFolder)
	folderName := path.Base(absFolder)

	_, err := p.callAPI(ctx, url.Values{
		"api":     {"SYNO.FileStation.CreateFolder"},
		"version": {"2"},
		"method":  {"create"},
		"folder_path": {parentDir},
		"name":        {folderName},
		"force_parent": {"true"},
	})
	return err
}

func (p *Provider) CreateShareLink(ctx context.Context, objectKey string, expireDays int) (*storage.ShareLinkResult, error) {
	params := url.Values{
		"api":     {"SYNO.FileStation.Sharing"},
		"version": {"3"},
		"method":  {"create"},
		"path":    {p.absPath(objectKey)},
	}
	if expireDays > 0 {
		expDate := time.Now().Add(time.Duration(expireDays) * 24 * time.Hour).Format("2006-01-02")
		params.Set("date_expired", expDate)
	}

	dr, err := p.callAPI(ctx, params)
	if err != nil {
		return nil, err
	}

	var shareData struct {
		Links []struct {
			URL        string `json:"url"`
			ID         string `json:"id"`
			ExpireDate string `json:"date_expired"`
		} `json:"links"`
	}
	if err := json.Unmarshal(dr.Data, &shareData); err != nil {
		return nil, fmt.Errorf("synology share unmarshal: %w", err)
	}
	if len(shareData.Links) == 0 {
		return nil, fmt.Errorf("synology share: no link returned")
	}

	result := &storage.ShareLinkResult{
		URL: shareData.Links[0].URL,
		ID:  shareData.Links[0].ID,
	}
	if shareData.Links[0].ExpireDate != "" {
		if t, err := time.Parse("2006-01-02", shareData.Links[0].ExpireDate); err == nil {
			result.ExpiresAt = &t
		}
	}
	return result, nil
}
