// Package embedding calls a local Ollama server for text embeddings (bge-m3)
// and formats vectors for pgvector. It deliberately has no pgvector or Ollama
// library dependency: one HTTP endpoint and a text literal are all it needs.
package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Embedder turns texts into vectors.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	Model() string
}

// OllamaClient calls POST {baseURL}/api/embed.
type OllamaClient struct {
	BaseURL string
	ModelID string
	Dim     int
	HTTP    *http.Client
}

func NewOllamaClient(baseURL, model string, dim int) OllamaClient {
	return OllamaClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		ModelID: model,
		Dim:     dim,
		// Generous: the first call after the model was unloaded includes a cold
		// load (~12 s for bge-m3 on this host). Query-time callers wrap their
		// own shorter context deadline.
		HTTP: &http.Client{Timeout: 60 * time.Second},
	}
}

func (c OllamaClient) Model() string { return c.ModelID }

func (c OllamaClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	// keep_alive keeps the model resident: an unloaded bge-m3 takes ~12 s to
	// come back, longer than a search is willing to wait for its query vector.
	body, err := json.Marshal(map[string]any{"model": c.ModelID, "input": texts, "keep_alive": "24h"})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/embed", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embedding server returned %d", resp.StatusCode)
	}
	var out struct {
		Embeddings [][]float32 `json:"embeddings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Embeddings) != len(texts) {
		return nil, fmt.Errorf("embedding server returned %d vectors for %d inputs", len(out.Embeddings), len(texts))
	}
	for _, v := range out.Embeddings {
		if c.Dim > 0 && len(v) != c.Dim {
			return nil, fmt.Errorf("embedding dimension %d, expected %d (does the column match the model?)", len(v), c.Dim)
		}
	}
	return out.Embeddings, nil
}

// Literal renders a vector as a pgvector text literal, e.g. "[0.1,-0.2]",
// to be bound as a parameter and cast with ::vector.
func Literal(v []float32) (string, error) {
	if len(v) == 0 {
		return "", errors.New("empty vector")
	}
	var b strings.Builder
	b.Grow(len(v) * 10)
	b.WriteByte('[')
	for i, x := range v {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(float64(x), 'g', 7, 32))
	}
	b.WriteByte(']')
	return b.String(), nil
}
