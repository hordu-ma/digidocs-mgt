package memory

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"digidocs-mgt/backend-go/internal/storage"
)

type Provider struct {
	mu    sync.RWMutex
	store map[string][]byte
	dirs  map[string]bool // tracks explicitly created folders
}

func NewProvider() *Provider {
	return &Provider{
		store: make(map[string][]byte),
		dirs:  make(map[string]bool),
	}
}

func (p *Provider) PutObject(_ context.Context, input storage.PutObjectInput) (storage.PutObjectResult, error) {
	var data []byte
	if input.Reader != nil {
		var err error
		data, err = io.ReadAll(input.Reader)
		if err != nil {
			return storage.PutObjectResult{}, err
		}
	}

	p.mu.Lock()
	p.store[input.ObjectKey] = data
	// auto-create parent dirs
	dir := path.Dir(input.ObjectKey)
	for dir != "" && dir != "." && dir != "/" {
		p.dirs[dir] = true
		dir = path.Dir(dir)
	}
	p.mu.Unlock()

	return storage.PutObjectResult{
		ObjectKey: input.ObjectKey,
		Provider:  "memory",
	}, nil
}

func (p *Provider) GetObject(_ context.Context, objectKey string) (*storage.GetObjectOutput, error) {
	p.mu.RLock()
	data, ok := p.store[objectKey]
	p.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("object not found: %s", objectKey)
	}

	return &storage.GetObjectOutput{
		Reader:      io.NopCloser(bytes.NewReader(data)),
		ContentType: "application/octet-stream",
		Size:        int64(len(data)),
	}, nil
}

func (p *Provider) DeleteObject(_ context.Context, objectKey string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.store[objectKey]; ok {
		delete(p.store, objectKey)
		return nil
	}
	if p.dirs[objectKey] {
		delete(p.dirs, objectKey)
		// also delete all children
		prefix := objectKey + "/"
		for k := range p.store {
			if strings.HasPrefix(k, prefix) {
				delete(p.store, k)
			}
		}
		for k := range p.dirs {
			if strings.HasPrefix(k, prefix) {
				delete(p.dirs, k)
			}
		}
		return nil
	}
	return fmt.Errorf("object not found: %s", objectKey)
}

func (p *Provider) Stat(_ context.Context, objectKey string) (*storage.FileInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if data, ok := p.store[objectKey]; ok {
		return &storage.FileInfo{
			Name:       path.Base(objectKey),
			Path:       objectKey,
			IsDir:      false,
			Size:       int64(len(data)),
			ModifiedAt: time.Now(),
		}, nil
	}
	if p.dirs[objectKey] {
		return &storage.FileInfo{
			Name:       path.Base(objectKey),
			Path:       objectKey,
			IsDir:      true,
			ModifiedAt: time.Now(),
		}, nil
	}
	return nil, fmt.Errorf("object not found: %s", objectKey)
}

func (p *Provider) ListDir(_ context.Context, folderPath string) ([]storage.FileInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	prefix := folderPath
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	seen := make(map[string]bool)
	var items []storage.FileInfo

	// list files directly under this folder
	for key, data := range p.store {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		rel := strings.TrimPrefix(key, prefix)
		if strings.Contains(rel, "/") {
			// nested — record intermediate dir
			dirName := strings.SplitN(rel, "/", 2)[0]
			dirPath := prefix + dirName
			if !seen[dirPath] {
				seen[dirPath] = true
				items = append(items, storage.FileInfo{Name: dirName, Path: dirPath, IsDir: true})
			}
		} else {
			if !seen[key] {
				seen[key] = true
				items = append(items, storage.FileInfo{
					Name:  path.Base(key),
					Path:  key,
					IsDir: false,
					Size:  int64(len(data)),
				})
			}
		}
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func (p *Provider) CreateFolder(_ context.Context, folderPath string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.dirs[folderPath] = true
	dir := path.Dir(folderPath)
	for dir != "" && dir != "." && dir != "/" {
		p.dirs[dir] = true
		dir = path.Dir(dir)
	}
	return nil
}

func (p *Provider) CreateShareLink(_ context.Context, objectKey string, expireDays int) (*storage.ShareLinkResult, error) {
	p.mu.RLock()
	_, ok := p.store[objectKey]
	p.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("object not found: %s", objectKey)
	}

	var exp *time.Time
	if expireDays > 0 {
		t := time.Now().Add(time.Duration(expireDays) * 24 * time.Hour)
		exp = &t
	}

	return &storage.ShareLinkResult{
		URL:       fmt.Sprintf("memory://share/%s", objectKey),
		ID:        fmt.Sprintf("mem-%s", objectKey),
		ExpiresAt: exp,
	}, nil
}
