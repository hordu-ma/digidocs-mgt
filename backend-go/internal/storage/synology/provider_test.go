package synology

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"digidocs-mgt/backend-go/internal/storage"
)

func newTestProvider(t *testing.T, handler http.HandlerFunc) *Provider {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil {
		t.Fatalf("parse server port: %v", err)
	}

	provider := NewProvider(Config{
		Host:      parsed.Hostname(),
		Port:      port,
		HTTPS:     parsed.Scheme == "https",
		Account:   "worker",
		Password:  "secret",
		SharePath: "/DigiDocs",
	})
	provider.client = server.Client()
	return provider
}

func TestProviderPutObjectUploadsMultipartFile(t *testing.T) {
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/webapi/query.cgi":
			_, _ = w.Write([]byte(`{"success":true,"data":{"SYNO.API.Auth":{"path":"entry.cgi"}}}`))
		case r.URL.Path == "/webapi/entry.cgi" && r.Method == http.MethodGet && r.URL.Query().Get("api") == "SYNO.API.Auth":
			_, _ = w.Write([]byte(`{"success":true,"data":{"sid":"sid-1"}}`))
		case r.URL.Path == "/webapi/entry.cgi" && r.Method == http.MethodPost:
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatalf("parse multipart form: %v", err)
			}
			if got := r.URL.Query().Get("_sid"); got != "sid-1" {
				t.Fatalf("query _sid = %s, want sid-1", got)
			}
			if got := r.FormValue("api"); got != "SYNO.FileStation.Upload" {
				t.Fatalf("api = %s, want SYNO.FileStation.Upload", got)
			}
			if got := r.FormValue("path"); got != "/DigiDocs/nested" {
				t.Fatalf("path = %s, want /DigiDocs/nested", got)
			}
			if got := r.FormValue("overwrite"); got != "true" {
				t.Fatalf("overwrite = %s, want true", got)
			}
			if got := strings.Join(r.MultipartForm.Value["_sid"], ","); got != "" {
				t.Fatalf("multipart _sid = %s, want empty", got)
			}
			file, _, err := r.FormFile("file")
			if err != nil {
				t.Fatalf("form file: %v", err)
			}
			defer file.Close()
			content, err := io.ReadAll(file)
			if err != nil {
				t.Fatalf("read upload file: %v", err)
			}
			if string(content) != "hello synology" {
				t.Fatalf("upload content = %q, want hello synology", string(content))
			}
			_, _ = w.Write([]byte(`{"success":true}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	})

	result, err := provider.PutObject(context.Background(), storage.PutObjectInput{
		ObjectKey: "nested/report.txt",
		Reader:    strings.NewReader("hello synology"),
		Overwrite: true,
	})
	if err != nil {
		t.Fatalf("PutObject error: %v", err)
	}
	if result.Provider != "synology" {
		t.Fatalf("provider = %s, want synology", result.Provider)
	}
	if result.ObjectKey != "nested/report.txt" {
		t.Fatalf("object key = %s, want nested/report.txt", result.ObjectKey)
	}
}

func TestProviderListDirReauthsOnExpiredSession(t *testing.T) {
	loginCount := 0
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/webapi/query.cgi":
			_, _ = w.Write([]byte(`{"success":true,"data":{"SYNO.API.Auth":{"path":"webapi/entry.cgi"}}}`))
		case r.URL.Path == "/webapi/entry.cgi" && r.URL.Query().Get("api") == "SYNO.API.Auth":
			loginCount++
			sid := "sid-1"
			if loginCount > 1 {
				sid = "sid-2"
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{"sid":"` + sid + `"}}`))
		case r.URL.Path == "/webapi/entry.cgi":
			if got := r.URL.Query().Get("method"); got != "list" {
				t.Fatalf("method = %s, want list", got)
			}
			sid := r.URL.Query().Get("_sid")
			if sid == "sid-1" {
				_, _ = w.Write([]byte(`{"success":false,"error":{"code":105}}`))
				return
			}
			if sid != "sid-2" {
				t.Fatalf("retry sid = %s, want sid-2", sid)
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{"files":[{"path":"/DigiDocs/folder/report.txt","name":"report.txt","isdir":false,"additional":{"size":12,"time":{"mtime":1712448000}}}]}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	})

	items, err := provider.ListDir(context.Background(), "folder")
	if err != nil {
		t.Fatalf("ListDir error: %v", err)
	}
	if loginCount != 2 {
		t.Fatalf("login count = %d, want 2", loginCount)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Path != "folder/report.txt" {
		t.Fatalf("path = %s, want folder/report.txt", items[0].Path)
	}
}

func TestProviderGetObjectReturnsStream(t *testing.T) {
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/webapi/query.cgi":
			_, _ = w.Write([]byte(`{"success":true,"data":{"SYNO.API.Auth":{"path":"webapi/entry.cgi"}}}`))
		case r.URL.Path == "/webapi/entry.cgi" && r.URL.Query().Get("api") == "SYNO.API.Auth":
			_, _ = w.Write([]byte(`{"success":true,"data":{"sid":"sid-1"}}`))
		case r.URL.Path == "/webapi/entry.cgi":
			if got := r.URL.Query().Get("api"); got != "SYNO.FileStation.Download" {
				t.Fatalf("api = %s, want SYNO.FileStation.Download", got)
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte("downloaded payload"))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	})

	obj, err := provider.GetObject(context.Background(), "folder/report.txt")
	if err != nil {
		t.Fatalf("GetObject error: %v", err)
	}
	defer obj.Reader.Close()

	body, err := io.ReadAll(obj.Reader)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != "downloaded payload" {
		t.Fatalf("body = %q, want downloaded payload", string(body))
	}
	if obj.ContentType != "text/plain; charset=utf-8" {
		t.Fatalf("content type = %s, want text/plain; charset=utf-8", obj.ContentType)
	}
}

func TestProviderGetObjectReauthsOnExpiredSession(t *testing.T) {
	loginCount := 0
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/webapi/query.cgi":
			_, _ = w.Write([]byte(`{"success":true,"data":{"SYNO.API.Auth":{"path":"webapi/entry.cgi"}}}`))
		case r.URL.Path == "/webapi/entry.cgi" && r.URL.Query().Get("api") == "SYNO.API.Auth":
			loginCount++
			sid := "sid-1"
			if loginCount > 1 {
				sid = "sid-2"
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{"sid":"` + sid + `"}}`))
		case r.URL.Path == "/webapi/entry.cgi" && r.URL.Query().Get("api") == "SYNO.FileStation.Download":
			sid := r.URL.Query().Get("_sid")
			if sid == "sid-1" {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"success":false,"error":{"code":106}}`))
				return
			}
			if sid != "sid-2" {
				t.Fatalf("retry sid = %s, want sid-2", sid)
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte("downloaded payload"))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	})

	// Prime the cached sid so ensureSession() short-circuits and the only way
	// GetObject can recover is by detecting the 106 response and re-logging in.
	if err := provider.login(context.Background()); err != nil {
		t.Fatalf("prime login: %v", err)
	}
	loginCount = 1

	obj, err := provider.GetObject(context.Background(), "folder/report.txt")
	if err != nil {
		t.Fatalf("GetObject error: %v", err)
	}
	defer obj.Reader.Close()

	body, err := io.ReadAll(obj.Reader)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if string(body) != "downloaded payload" {
		t.Fatalf("body = %q, want downloaded payload", string(body))
	}
	if loginCount != 2 {
		t.Fatalf("login count = %d, want 2", loginCount)
	}
}

func TestProviderPutObjectReauthsOnExpiredSession(t *testing.T) {
	loginCount := 0
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/webapi/query.cgi":
			_, _ = w.Write([]byte(`{"success":true,"data":{"SYNO.API.Auth":{"path":"webapi/entry.cgi"}}}`))
		case r.URL.Path == "/webapi/entry.cgi" && r.Method == http.MethodGet && r.URL.Query().Get("api") == "SYNO.API.Auth":
			loginCount++
			sid := "sid-1"
			if loginCount > 1 {
				sid = "sid-2"
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{"sid":"` + sid + `"}}`))
		case r.URL.Path == "/webapi/entry.cgi" && r.Method == http.MethodPost:
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatalf("parse multipart form: %v", err)
			}
			sid := r.URL.Query().Get("_sid")
			if sid == "sid-1" {
				_, _ = w.Write([]byte(`{"success":false,"error":{"code":105}}`))
				return
			}
			if sid != "sid-2" {
				t.Fatalf("retry sid = %s, want sid-2", sid)
			}
			file, _, err := r.FormFile("file")
			if err != nil {
				t.Fatalf("form file: %v", err)
			}
			defer file.Close()
			content, err := io.ReadAll(file)
			if err != nil {
				t.Fatalf("read upload file: %v", err)
			}
			if string(content) != "hello synology" {
				t.Fatalf("upload content = %q, want hello synology (body must survive the retry)", string(content))
			}
			_, _ = w.Write([]byte(`{"success":true}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	})

	if err := provider.login(context.Background()); err != nil {
		t.Fatalf("prime login: %v", err)
	}
	loginCount = 1

	result, err := provider.PutObject(context.Background(), storage.PutObjectInput{
		ObjectKey: "nested/report.txt",
		Reader:    strings.NewReader("hello synology"),
	})
	if err != nil {
		t.Fatalf("PutObject error: %v", err)
	}
	if result.ObjectKey != "nested/report.txt" {
		t.Fatalf("object key = %s, want nested/report.txt", result.ObjectKey)
	}
	if loginCount != 2 {
		t.Fatalf("login count = %d, want 2", loginCount)
	}
}

func TestProviderCreateShareLinkParsesExpireDate(t *testing.T) {
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/webapi/query.cgi":
			_, _ = w.Write([]byte(`{"success":true,"data":{"SYNO.API.Auth":{"path":"webapi/entry.cgi"}}}`))
		case r.URL.Path == "/webapi/entry.cgi" && r.URL.Query().Get("api") == "SYNO.API.Auth":
			_, _ = w.Write([]byte(`{"success":true,"data":{"sid":"sid-1"}}`))
		case r.URL.Path == "/webapi/entry.cgi":
			if got := r.URL.Query().Get("method"); got != "create" {
				t.Fatalf("method = %s, want create", got)
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{"links":[{"url":"https://nas.example/s/abc","id":"share-1","date_expired":"2026-04-30"}]}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	})

	result, err := provider.CreateShareLink(context.Background(), "folder/report.txt", 7)
	if err != nil {
		t.Fatalf("CreateShareLink error: %v", err)
	}
	if result.URL != "https://nas.example/s/abc" {
		t.Fatalf("url = %s, want https://nas.example/s/abc", result.URL)
	}
	if result.ID != "share-1" {
		t.Fatalf("id = %s, want share-1", result.ID)
	}
	if result.ExpiresAt == nil {
		t.Fatal("expected expires_at")
	}
	if got := result.ExpiresAt.Format("2006-01-02"); got != "2026-04-30" {
		t.Fatalf("expires_at = %s, want 2026-04-30", got)
	}
	if result.ExpiresAt.Before(time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("expires_at parsing failed")
	}
}

func TestProviderLoginSupportsInsecureSkipVerify(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/webapi/query.cgi":
			_, _ = w.Write([]byte(`{"success":true,"data":{"SYNO.API.Auth":{"path":"webapi/entry.cgi"}}}`))
		case r.URL.Path == "/webapi/entry.cgi" && r.URL.Query().Get("api") == "SYNO.API.Auth":
			_, _ = w.Write([]byte(`{"success":true,"data":{"sid":"sid-1"}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	t.Cleanup(server.Close)

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	port, err := strconv.Atoi(parsed.Port())
	if err != nil {
		t.Fatalf("parse server port: %v", err)
	}

	provider := NewProvider(Config{
		Host:      parsed.Hostname(),
		Port:      port,
		HTTPS:     true,
		Account:   "worker",
		Password:  "secret",
		SharePath: "/DigiDocs",
	})
	if err := provider.login(context.Background()); err == nil {
		t.Fatal("expected TLS verification error")
	}

	provider = NewProvider(Config{
		Host:               parsed.Hostname(),
		Port:               port,
		HTTPS:              true,
		InsecureSkipVerify: true,
		Account:            "worker",
		Password:           "secret",
		SharePath:          "/DigiDocs",
	})
	if err := provider.login(context.Background()); err != nil {
		t.Fatalf("expected login success with insecure skip verify: %v", err)
	}
}

func TestProviderLoginFallsBackToAuthCGIWhenInfoPathUnavailable(t *testing.T) {
	entryAttempts := 0
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/webapi/query.cgi":
			_, _ = w.Write([]byte(`{"success":false,"error":{"code":999}}`))
		case r.URL.Path == "/webapi/entry.cgi":
			entryAttempts++
			_, _ = w.Write([]byte(`{"success":false,"error":{"code":400}}`))
		case r.URL.Path == "/webapi/auth.cgi":
			_, _ = w.Write([]byte(`{"success":true,"data":{"sid":"sid-fallback"}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	})

	if err := provider.login(context.Background()); err != nil {
		t.Fatalf("expected fallback login success: %v", err)
	}
	if provider.getSID() != "sid-fallback" {
		t.Fatalf("sid = %s, want sid-fallback", provider.getSID())
	}
	if entryAttempts == 0 {
		t.Fatal("expected entry.cgi to be tried before auth.cgi fallback")
	}
}

func TestProviderCreateFolderTreatsExistingDirectoryAsSuccess(t *testing.T) {
	createCalls := 0
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/webapi/query.cgi":
			_, _ = w.Write([]byte(`{"success":true,"data":{"SYNO.API.Auth":{"path":"webapi/entry.cgi"}}}`))
		case r.URL.Path == "/webapi/entry.cgi" && r.URL.Query().Get("api") == "SYNO.API.Auth":
			_, _ = w.Write([]byte(`{"success":true,"data":{"sid":"sid-1"}}`))
		case r.URL.Path == "/webapi/entry.cgi" && r.URL.Query().Get("api") == "SYNO.FileStation.CreateFolder":
			createCalls++
			_, _ = w.Write([]byte(`{"success":false,"error":{"code":400}}`))
		case r.URL.Path == "/webapi/entry.cgi" && r.URL.Query().Get("api") == "SYNO.FileStation.List":
			if got := r.URL.Query().Get("method"); got != "getinfo" {
				t.Fatalf("method = %s, want getinfo", got)
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{"files":[{"path":"/DigiDocs/documents/doc-1","name":"doc-1","isdir":true,"additional":{"size":0,"time":{"mtime":1712448000}}}]}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	})

	if err := provider.CreateFolder(context.Background(), "documents/doc-1"); err != nil {
		t.Fatalf("expected existing folder to be treated as success: %v", err)
	}
	if createCalls != 1 {
		t.Fatalf("create calls = %d, want 1", createCalls)
	}
}

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, context.DeadlineExceeded
}

func TestTransportErrorsDoNotLeakCredentialsOrSessions(t *testing.T) {
	for _, operation := range []string{"login", "list", "upload", "download"} {
		t.Run(operation, func(t *testing.T) {
			p := NewProvider(Config{Host: "nas.invalid", Port: 5001, HTTPS: true, Account: "private-user", Password: "password-with-&-secrets", SharePath: "/DigiDocs"})
			p.client = &http.Client{Transport: failingTransport{}}
			if operation != "login" {
				p.sid = "secret-session-id"
			}
			var logs bytes.Buffer
			previous := log.Writer()
			log.SetOutput(&logs)
			defer log.SetOutput(previous)
			var err error
			switch operation {
			case "login":
				err = p.login(context.Background())
			case "list":
				_, err = p.ListDir(context.Background(), "ops-backups")
			case "upload":
				_, err = p.PutObject(context.Background(), storage.PutObjectInput{ObjectKey: "backup.gz", Reader: strings.NewReader("test")})
			case "download":
				_, err = p.GetObject(context.Background(), "backup.gz")
			}
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("lost timeout cause: %v", err)
			}
			var urlErr *url.Error
			if !errors.As(err, &urlErr) || !urlErr.Timeout() {
				t.Fatalf("lost timeout type: %v", err)
			}
			output := err.Error() + logs.String()
			for _, secret := range []string{p.account, p.password, url.QueryEscape(p.password), "secret-session-id", "passwd=", "_sid="} {
				if strings.Contains(output, secret) {
					t.Fatalf("sensitive value leaked for %s", operation)
				}
			}
			if !strings.Contains(output, "nas.invalid:5001") {
				t.Fatal("lost destination context")
			}
		})
	}
}

func TestRedactURLErrorRemovesUserInfoAndNestedQuery(t *testing.T) {
	err := &url.Error{Op: "Get", URL: "https://user:secret@nas/path?passwd=secret#token", Err: &url.Error{Op: "Get", URL: "https://nas/path?_sid=secret", Err: context.DeadlineExceeded}}
	safe := redactURLError(err)
	if strings.Contains(safe.Error(), "secret") || strings.Contains(safe.Error(), "user") || strings.Contains(safe.Error(), "token") {
		t.Fatal("nested URL credentials leaked")
	}
	if !errors.Is(safe, context.DeadlineExceeded) {
		t.Fatal("lost cause")
	}
	if !strings.Contains(err.URL, "passwd=secret") {
		t.Fatal("mutated original error")
	}
}
