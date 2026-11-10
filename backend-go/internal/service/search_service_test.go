package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"digidocs-mgt/backend-go/internal/domain/query"
)

type fakeTextReader struct{ keyword []query.TextSearchHit }

func (f fakeTextReader) SearchChunks(context.Context, query.TextSearchFilter) ([]query.TextSearchHit, error) {
	return append([]query.TextSearchHit(nil), f.keyword...), nil
}
func (f fakeTextReader) SearchDocuments(context.Context, query.TextSearchFilter) ([]query.TextSearchHit, error) {
	return []query.TextSearchHit{{DocumentID: "kw-only-path"}}, nil
}
func (f fakeTextReader) GetDocumentText(context.Context, string) (*query.DocumentText, error) {
	return nil, nil
}

type fakeVectorStore struct {
	semantic []query.TextSearchHit
	err      error
	pending  []query.ChunkForEmbedding
	stored   map[string]string
}

func (f *fakeVectorStore) VectorChunks(context.Context, query.TextSearchFilter, string, string, int, float64) ([]query.TextSearchHit, error) {
	return append([]query.TextSearchHit(nil), f.semantic...), f.err
}
func (f *fakeVectorStore) ChunksNeedingEmbedding(context.Context, string, int) ([]query.ChunkForEmbedding, error) {
	return f.pending, nil
}
func (f *fakeVectorStore) SetChunkEmbedding(_ context.Context, id, lit, _ string) error {
	if f.stored == nil {
		f.stored = map[string]string{}
	}
	f.stored[id] = lit
	return nil
}

type fakeEmbedder struct {
	err    error
	inputs []string
}

func (f *fakeEmbedder) Model() string { return "fake" }
func (f *fakeEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	f.inputs = append(f.inputs, texts...)
	if f.err != nil {
		return nil, f.err
	}
	out := make([][]float32, len(texts))
	for i := range out {
		out[i] = []float32{1, 0}
	}
	return out, nil
}

func hit(doc string, chunk int) query.TextSearchHit {
	return query.TextSearchHit{DocumentID: doc, VersionID: "v-" + doc, ChunkNo: chunk, Content: "内容" + doc}
}

func TestFuseRRFRanksAgreementFirst(t *testing.T) {
	keyword := []query.TextSearchHit{hit("a", 0), hit("b", 0)}
	semantic := []query.TextSearchHit{hit("c", 0), hit("b", 0)}
	semantic[1].Similarity = 0.7
	got := fuseRRF(keyword, semantic)
	if len(got) != 3 || got[0].DocumentID != "b" || got[0].Match != "both" || got[0].Similarity != 0.7 {
		t.Fatalf("chunk found by both retrievers must rank first: %#v", got)
	}
}

func TestHybridFallsBackToKeywordWhenEmbeddingFails(t *testing.T) {
	svc := NewSearchService(fakeTextReader{keyword: []query.TextSearchHit{hit("a", 0)}}).
		WithSemantic(&fakeVectorStore{semantic: []query.TextSearchHit{hit("z", 0)}}, &fakeEmbedder{err: errors.New("down")}, 0.4)
	got, err := svc.SearchChunks(context.Background(), query.TextSearchFilter{Query: "奖励函数"})
	if err != nil || len(got) != 1 || got[0].DocumentID != "a" || got[0].Match != "keyword" {
		t.Fatalf("got %#v %v", got, err)
	}
	svc = NewSearchService(fakeTextReader{keyword: []query.TextSearchHit{hit("a", 0)}}).
		WithSemantic(&fakeVectorStore{err: errors.New("no vector")}, &fakeEmbedder{}, 0.4)
	if got, err := svc.SearchChunks(context.Background(), query.TextSearchFilter{Query: "q"}); err != nil || len(got) != 1 {
		t.Fatalf("vector query failure must degrade, got %#v %v", got, err)
	}
}

func TestHybridDocumentsGroupAndLimit(t *testing.T) {
	semantic := []query.TextSearchHit{hit("d1", 3), hit("d2", 0)}
	svc := NewSearchService(fakeTextReader{keyword: []query.TextSearchHit{hit("d1", 1)}}).
		WithSemantic(&fakeVectorStore{semantic: semantic}, &fakeEmbedder{}, 0.4)
	got, err := svc.SearchDocuments(context.Background(), query.TextSearchFilter{Query: "q", Limit: 5})
	if err != nil || len(got) != 2 || got[0].DocumentID != "d1" || got[0].MatchedChunks != 2 || got[0].Match != "both" || got[0].Content != "" {
		t.Fatalf("got %#v %v", got, err)
	}
	if got[1].Match != "semantic" || got[1].Snippet == "" {
		t.Fatalf("semantic-only hit needs a snippet: %#v", got[1])
	}
	// Keyword-only service keeps using the repository's document search.
	plain, _ := NewSearchService(fakeTextReader{}).SearchDocuments(context.Background(), query.TextSearchFilter{Query: "q"})
	if len(plain) != 1 || plain[0].DocumentID != "kw-only-path" {
		t.Fatalf("keyword-only path: %#v", plain)
	}
}

func TestQueryVectorIsCached(t *testing.T) {
	emb := &fakeEmbedder{}
	svc := NewSearchService(fakeTextReader{}).WithSemantic(&fakeVectorStore{}, emb, 0.4)
	for i := 0; i < 3; i++ {
		_, _ = svc.SearchChunks(context.Background(), query.TextSearchFilter{Query: "同一个问题"})
	}
	if len(emb.inputs) != 1 {
		t.Fatalf("query vector should be embedded once, got %d", len(emb.inputs))
	}
}

func TestIndexerPrefixesTitleAndStores(t *testing.T) {
	store := &fakeVectorStore{pending: []query.ChunkForEmbedding{{ID: "c1", Title: "开题报告", Content: "第一章"}}}
	emb := &fakeEmbedder{}
	n, err := NewEmbeddingIndexer(store, emb).RunOnce(context.Background())
	if err != nil || n != 1 || store.stored["c1"] != "[1,0]" {
		t.Fatalf("n=%d err=%v stored=%v", n, err, store.stored)
	}
	if !strings.HasPrefix(emb.inputs[0], "标题：开题报告\n") {
		t.Fatalf("chunk must be embedded with its title: %q", emb.inputs[0])
	}
	if _, err := NewEmbeddingIndexer(store, &fakeEmbedder{err: errors.New("down")}).RunOnce(context.Background()); err == nil {
		t.Fatal("embedding failure must surface so the loop backs off")
	}
}
