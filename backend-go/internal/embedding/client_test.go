package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaClientEmbed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		_ = json.NewDecoder(r.Body).Decode(&in)
		if r.URL.Path != "/api/embed" || in.Model != "bge-m3" {
			w.WriteHeader(400)
			return
		}
		out := make([][]float32, len(in.Input))
		for i := range out {
			out[i] = []float32{1, 0, 0}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"embeddings": out})
	}))
	defer srv.Close()
	c := NewOllamaClient(srv.URL+"/", "bge-m3", 3)
	vecs, err := c.Embed(context.Background(), []string{"a", "b"})
	if err != nil || len(vecs) != 2 || vecs[1][0] != 1 {
		t.Fatalf("got %v %v", vecs, err)
	}
	if _, err := NewOllamaClient(srv.URL, "bge-m3", 1024).Embed(context.Background(), []string{"a"}); err == nil {
		t.Fatal("dimension mismatch must be an error")
	}
}

func TestLiteral(t *testing.T) {
	s, err := Literal([]float32{0.5, -0.25, 1})
	if err != nil || s != "[0.5,-0.25,1]" {
		t.Fatalf("got %q %v", s, err)
	}
	if _, err := Literal(nil); err == nil {
		t.Fatal("empty vector must be rejected")
	}
}
