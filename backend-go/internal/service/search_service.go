package service

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/embedding"
	"digidocs-mgt/backend-go/internal/textsearch"
)

// TextSearchReader is implemented by the postgres text search repository.
type TextSearchReader interface {
	SearchChunks(ctx context.Context, filter query.TextSearchFilter) ([]query.TextSearchHit, error)
	SearchDocuments(ctx context.Context, filter query.TextSearchFilter) ([]query.TextSearchHit, error)
	GetDocumentText(ctx context.Context, documentID string) (*query.DocumentText, error)
}

// VectorStore is the pgvector side of the text search repository.
type VectorStore interface {
	VectorChunks(ctx context.Context, filter query.TextSearchFilter, vectorLiteral string, model string, limit int, minSimilarity float64) ([]query.TextSearchHit, error)
	ChunksNeedingEmbedding(ctx context.Context, model string, limit int) ([]query.ChunkForEmbedding, error)
	SetChunkEmbedding(ctx context.Context, chunkID string, vectorLiteral string, model string) error
}

const (
	// hybridCandidates is how many chunks each retriever contributes to fusion.
	hybridCandidates = 100
	// rrfK is the usual reciprocal-rank-fusion constant: it damps the weight of
	// the very top ranks so neither retriever dominates.
	rrfK = 60
	// queryEmbedTimeout bounds how long a search waits for the query vector
	// before falling back to keyword-only results.
	queryEmbedTimeout = 4 * time.Second
)

// SearchService serves lexical + semantic search over extracted document
// text, for the assistant's retrieval and for users. Callers pass the scope;
// the repository filters by it in SQL.
type SearchService struct {
	reader   TextSearchReader
	vectors  VectorStore
	embedder embedding.Embedder
	minSim   float64
	cache    *queryVectorCache
}

func NewSearchService(reader TextSearchReader) SearchService {
	return SearchService{reader: reader}
}

// WithSemantic enables hybrid retrieval. Without it (no pgvector, no
// embedding server) search is keyword-only.
func (s SearchService) WithSemantic(vectors VectorStore, embedder embedding.Embedder, minSimilarity float64) SearchService {
	s.vectors = vectors
	s.embedder = embedder
	s.minSim = minSimilarity
	s.cache = &queryVectorCache{items: map[string][]float32{}}
	return s
}

// Semantic reports whether hybrid retrieval is configured.
func (s SearchService) Semantic() bool { return s.vectors != nil && s.embedder != nil }

func (s SearchService) SearchChunks(ctx context.Context, filter query.TextSearchFilter) ([]query.TextSearchHit, error) {
	if s.reader == nil || strings.TrimSpace(filter.Query) == "" {
		return []query.TextSearchHit{}, nil
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 8
	}
	if !s.Semantic() {
		return s.reader.SearchChunks(ctx, filter)
	}
	fused, err := s.hybridChunks(ctx, filter)
	if err != nil {
		return nil, err
	}
	if len(fused) > limit {
		fused = fused[:limit]
	}
	return fused, nil
}

// SearchDocuments is the user-facing search; the scope must come from the
// authenticated caller (handlers use scopeFrom).
func (s SearchService) SearchDocuments(ctx context.Context, filter query.TextSearchFilter) ([]query.TextSearchHit, error) {
	if s.reader == nil || strings.TrimSpace(filter.Query) == "" {
		return []query.TextSearchHit{}, nil
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	if !s.Semantic() {
		return s.reader.SearchDocuments(ctx, filter)
	}
	chunks, err := s.hybridChunks(ctx, filter)
	if err != nil {
		return nil, err
	}
	// Group fused chunks by document: the best-ranked chunk represents it.
	terms := textsearch.Terms(filter.Query)
	order := make([]string, 0)
	byDoc := map[string]*query.TextSearchHit{}
	for _, c := range chunks {
		if d, ok := byDoc[c.DocumentID]; ok {
			d.MatchedChunks++
			if d.Match != c.Match {
				d.Match = "both"
			}
			continue
		}
		hit := c
		hit.MatchedChunks = 1
		hit.Snippet = textsearch.Snippet(c.Content, terms, 120)
		hit.Content = ""
		byDoc[c.DocumentID] = &hit
		order = append(order, c.DocumentID)
	}
	out := make([]query.TextSearchHit, 0, len(order))
	for _, id := range order {
		out = append(out, *byDoc[id])
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

// hybridChunks runs keyword and vector retrieval and fuses them with RRF.
// Any failure on the semantic side degrades to keyword-only, never to an error.
func (s SearchService) hybridChunks(ctx context.Context, filter query.TextSearchFilter) ([]query.TextSearchHit, error) {
	kwFilter := filter
	kwFilter.Limit = hybridCandidates
	keyword, err := s.reader.SearchChunks(ctx, kwFilter)
	if err != nil {
		return nil, err
	}
	for i := range keyword {
		keyword[i].Match = "keyword"
	}
	vec, err := s.queryVector(ctx, filter.Query)
	if err != nil {
		log.Printf("[search] semantic retrieval skipped: %v", err)
		return keyword, nil
	}
	literal, err := embedding.Literal(vec)
	if err != nil {
		return keyword, nil
	}
	semantic, err := s.vectors.VectorChunks(ctx, filter, literal, s.embedder.Model(), hybridCandidates, s.minSim)
	if err != nil {
		log.Printf("[search] vector query failed, keyword only: %v", err)
		return keyword, nil
	}
	terms := textsearch.Terms(filter.Query)
	for i := range semantic {
		semantic[i].Match = "semantic"
		semantic[i].Snippet = textsearch.Snippet(semantic[i].Content, terms, 160)
	}
	return fuseRRF(keyword, semantic), nil
}

// fuseRRF merges two ranked lists by reciprocal rank fusion: each chunk scores
// sum(1/(rrfK+rank)) over the lists it appears in. Chunks found by both rank
// highest; the keyword copy is kept for its term-centred snippet.
func fuseRRF(keyword, semantic []query.TextSearchHit) []query.TextSearchHit {
	type entry struct {
		hit   query.TextSearchHit
		score float64
		first int
	}
	key := func(h query.TextSearchHit) string { return h.VersionID + "#" + itoa(h.ChunkNo) }
	entries := map[string]*entry{}
	order := 0
	add := func(list []query.TextSearchHit) {
		for rank, h := range list {
			k := key(h)
			e, ok := entries[k]
			if !ok {
				e = &entry{hit: h, first: order}
				order++
				entries[k] = e
			} else {
				e.hit.Match = "both"
				if h.Similarity > 0 {
					e.hit.Similarity = h.Similarity
				}
			}
			e.score += 1 / float64(rrfK+rank+1)
		}
	}
	add(keyword)
	add(semantic)
	out := make([]*entry, 0, len(entries))
	for _, e := range entries {
		out = append(out, e)
	}
	sortEntries(out, func(a, b *entry) bool {
		if a.score != b.score {
			return a.score > b.score
		}
		return a.first < b.first
	})
	hits := make([]query.TextSearchHit, len(out))
	for i, e := range out {
		hits[i] = e.hit
	}
	return hits
}

func (s SearchService) queryVector(ctx context.Context, q string) ([]float32, error) {
	q = strings.TrimSpace(q)
	if v, ok := s.cache.get(q); ok {
		return v, nil
	}
	ctx, cancel := context.WithTimeout(ctx, queryEmbedTimeout)
	defer cancel()
	vecs, err := s.embedder.Embed(ctx, []string{q})
	if err != nil {
		return nil, err
	}
	s.cache.put(q, vecs[0])
	return vecs[0], nil
}

// queryVectorCache keeps recent query vectors; repeated searches (paging,
// the assistant re-asking in a conversation) skip the embedding call.
type queryVectorCache struct {
	mu    sync.Mutex
	items map[string][]float32
	order []string
}

const queryVectorCacheSize = 256

func (c *queryVectorCache) get(q string) ([]float32, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.items[q]
	return v, ok
}

func (c *queryVectorCache) put(q string, v []float32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.items[q]; ok {
		return
	}
	if len(c.order) >= queryVectorCacheSize {
		delete(c.items, c.order[0])
		c.order = c.order[1:]
	}
	c.items[q] = v
	c.order = append(c.order, q)
}

// DocumentText returns the stored text; callers authorize the document first.
func (s SearchService) DocumentText(ctx context.Context, documentID string) (*query.DocumentText, error) {
	if s.reader == nil {
		return &query.DocumentText{DocumentID: documentID}, nil
	}
	return s.reader.GetDocumentText(ctx, documentID)
}
