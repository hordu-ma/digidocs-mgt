package service

import (
	"context"
	"strings"

	"digidocs-mgt/backend-go/internal/domain/query"
)

// TextSearchReader is implemented by the postgres text search repository.
type TextSearchReader interface {
	SearchChunks(ctx context.Context, filter query.TextSearchFilter) ([]query.TextSearchHit, error)
}

// SearchService serves lexical search over extracted document text, for the
// assistant's retrieval and for users. Callers pass the scope; the repository
// filters by it in SQL.
type SearchService struct {
	reader TextSearchReader
}

func NewSearchService(reader TextSearchReader) SearchService {
	return SearchService{reader: reader}
}

func (s SearchService) SearchChunks(ctx context.Context, filter query.TextSearchFilter) ([]query.TextSearchHit, error) {
	if s.reader == nil || strings.TrimSpace(filter.Query) == "" {
		return []query.TextSearchHit{}, nil
	}
	return s.reader.SearchChunks(ctx, filter)
}
