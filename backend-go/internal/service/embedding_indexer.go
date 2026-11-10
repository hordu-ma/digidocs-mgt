package service

import (
	"context"
	"log"
	"time"

	"digidocs-mgt/backend-go/internal/embedding"
)

// EmbeddingIndexer fills document_text_chunks.embedding in the background.
//
// Embedding is kept out of the result-callback transaction on purpose: chunks
// become keyword-searchable the moment text is extracted, and a slow or
// unavailable embedding server can never fail or delay that write. The indexer
// polls for chunks without a vector from the configured model, so it also
// backfills history and re-embeds after a model change.
type EmbeddingIndexer struct {
	store    VectorStore
	embedder embedding.Embedder
	batch    int
	idle     time.Duration
	backoff  time.Duration
}

func NewEmbeddingIndexer(store VectorStore, embedder embedding.Embedder) EmbeddingIndexer {
	return EmbeddingIndexer{store: store, embedder: embedder, batch: 16, idle: 20 * time.Second, backoff: time.Minute}
}

// Run loops until ctx is cancelled.
func (x EmbeddingIndexer) Run(ctx context.Context) {
	log.Printf("[embedding] indexer started model=%s", x.embedder.Model())
	failing := false
	for {
		n, err := x.RunOnce(ctx)
		wait := x.idle
		switch {
		case err != nil:
			if !failing {
				log.Printf("[embedding] indexing paused, will retry: %v", err)
			}
			failing = true
			wait = x.backoff
		case n > 0:
			if failing {
				log.Printf("[embedding] indexing resumed")
			}
			failing = false
			if n == x.batch {
				wait = 0 // more backlog: keep going
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

// RunOnce embeds one batch and returns how many chunks were stored.
func (x EmbeddingIndexer) RunOnce(ctx context.Context) (int, error) {
	chunks, err := x.store.ChunksNeedingEmbedding(ctx, x.embedder.Model(), x.batch)
	if err != nil || len(chunks) == 0 {
		return 0, err
	}
	texts := make([]string, len(chunks))
	for i, c := range chunks {
		// The title gives every chunk its document's topic; a chunk from the
		// middle of a report often never names what it is about.
		texts[i] = "标题：" + c.Title + "\n" + c.Content
	}
	vecs, err := x.embedder.Embed(ctx, texts)
	if err != nil {
		return 0, err
	}
	for i, c := range chunks {
		lit, err := embedding.Literal(vecs[i])
		if err != nil {
			return i, err
		}
		if err := x.store.SetChunkEmbedding(ctx, c.ID, lit, x.embedder.Model()); err != nil {
			return i, err
		}
	}
	return len(chunks), nil
}
