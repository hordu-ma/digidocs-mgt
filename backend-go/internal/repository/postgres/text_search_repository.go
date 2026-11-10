package postgres

import (
	"context"
	"database/sql"
	"errors"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/textsearch"

	"github.com/lib/pq"
)

// storeDocumentTextChunks replaces the searchable chunks of a document with
// the text extracted from versionID. Only the document's *current* version is
// indexed: an extraction of an older version that finishes late is ignored,
// and chunks of superseded versions are dropped, so search never answers from
// stale text.
func storeDocumentTextChunks(ctx context.Context, db DBTX, versionID string, text string) error {
	if versionID == "" || text == "" {
		return nil
	}
	var documentID string
	err := db.QueryRowContext(ctx, `
		SELECT d.id::text
		FROM document_versions v
		JOIN documents d ON d.id = v.document_id AND d.current_version_id = v.id
		WHERE v.id::text = $1
	`, versionID).Scan(&documentID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM document_text_chunks WHERE document_id::text = $1`, documentID); err != nil {
		return err
	}
	chunks := textsearch.Chunk(text)
	if len(chunks) == 0 {
		return nil
	}
	numbers := make([]int64, len(chunks))
	for i := range chunks {
		numbers[i] = int64(i)
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO document_text_chunks (document_id, version_id, chunk_no, content)
		SELECT $1::uuid, $2::uuid, n, c
		FROM unnest($3::int[], $4::text[]) AS t(n, c)
	`, documentID, versionID, pq.Array(numbers), pq.Array(chunks))
	return err
}

type TextSearchRepository struct {
	db DBTX
}

func NewTextSearchRepository(db DBTX) TextSearchRepository {
	return TextSearchRepository{db: db}
}

// SearchChunks ranks current-version chunks by how many distinct query terms
// they contain. Rows the caller may not see are filtered in SQL.
func (r TextSearchRepository) SearchChunks(ctx context.Context, filter query.TextSearchFilter) ([]query.TextSearchHit, error) {
	terms := textsearch.Terms(filter.Query)
	if len(terms) == 0 {
		return []query.TextSearchHit{}, nil
	}
	limit := filter.Limit
	if limit <= 0 || limit > 50 {
		limit = 8
	}
	isAdmin, actorID := scopeArgs(filter.Scope)
	rows, err := r.db.QueryContext(ctx, `
		SELECT d.id::text, d.title, d.project_id::text, COALESCE(p.name, ''),
			v.id::text, v.version_no, c.chunk_no, c.content,
			(SELECT COUNT(*) FROM unnest($1::text[]) AS pat WHERE c.content ILIKE pat)::int AS score
		FROM document_text_chunks c
		JOIN documents d ON d.id = c.document_id AND d.current_version_id = c.version_id
		JOIN document_versions v ON v.id = c.version_id
		LEFT JOIN projects p ON p.id = d.project_id
		WHERE d.is_deleted = false
		  AND c.content ILIKE ANY($1::text[])
		  AND ($2 = '' OR d.project_id::text = $2)
		  AND ($3 = '' OR d.id::text = $3)
		  AND `+projectScopeSQL("d.project_id", "$4", "$5")+`
		ORDER BY score DESC, d.updated_at DESC, c.chunk_no ASC
		LIMIT $6
	`, pq.Array(textsearch.LikePatterns(terms)), filter.ProjectID, filter.DocumentID, isAdmin, actorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	hits := make([]query.TextSearchHit, 0, limit)
	for rows.Next() {
		var h query.TextSearchHit
		if err := rows.Scan(&h.DocumentID, &h.Title, &h.ProjectID, &h.ProjectName, &h.VersionID, &h.VersionNo, &h.ChunkNo, &h.Content, &h.Score); err != nil {
			return nil, err
		}
		h.Snippet = textsearch.Snippet(h.Content, terms, 160)
		hits = append(hits, h)
	}
	return hits, rows.Err()
}
