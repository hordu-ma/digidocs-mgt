package postgres

import (
	"context"
	"strings"
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
	if limit <= 0 || limit > 200 {
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

// SearchDocuments is the user-facing full-text search: one row per document,
// represented by its best-matching chunk, with how many chunks matched.
func (r TextSearchRepository) SearchDocuments(ctx context.Context, filter query.TextSearchFilter) ([]query.TextSearchHit, error) {
	terms := textsearch.Terms(filter.Query)
	if len(terms) == 0 {
		return []query.TextSearchHit{}, nil
	}
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	isAdmin, actorID := scopeArgs(filter.Scope)
	rows, err := r.db.QueryContext(ctx, `
		WITH hits AS (
			SELECT d.id AS document_id, d.title, d.project_id, COALESCE(p.name, '') AS project_name,
				v.id AS version_id, v.version_no, c.chunk_no, c.content, d.updated_at,
				(SELECT COUNT(*) FROM unnest($1::text[]) AS pat WHERE c.content ILIKE pat)::int AS score
			FROM document_text_chunks c
			JOIN documents d ON d.id = c.document_id AND d.current_version_id = c.version_id
			JOIN document_versions v ON v.id = c.version_id
			LEFT JOIN projects p ON p.id = d.project_id
			WHERE d.is_deleted = false
			  AND c.content ILIKE ANY($1::text[])
			  AND ($2 = '' OR d.project_id::text = $2)
			  AND `+projectScopeSQL("d.project_id", "$3", "$4")+`
		),
		best AS (
			SELECT DISTINCT ON (document_id) *, COUNT(*) OVER (PARTITION BY document_id) AS matched
			FROM hits
			ORDER BY document_id, score DESC, chunk_no ASC
		)
		SELECT document_id::text, title, project_id::text, project_name, version_id::text,
			version_no, chunk_no, content, score, matched::int
		FROM best
		ORDER BY score DESC, matched DESC, updated_at DESC
		LIMIT $5
	`, pq.Array(textsearch.LikePatterns(terms)), filter.ProjectID, isAdmin, actorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	hits := make([]query.TextSearchHit, 0, limit)
	for rows.Next() {
		var h query.TextSearchHit
		var content string
		if err := rows.Scan(&h.DocumentID, &h.Title, &h.ProjectID, &h.ProjectName, &h.VersionID, &h.VersionNo, &h.ChunkNo, &content, &h.Score, &h.MatchedChunks); err != nil {
			return nil, err
		}
		h.Snippet = textsearch.Snippet(content, terms, 120)
		hits = append(hits, h)
	}
	return hits, rows.Err()
}

// GetDocumentText reassembles the current version's extracted text from its
// chunks, dropping each chunk's overlap with the previous one.
func (r TextSearchRepository) GetDocumentText(ctx context.Context, documentID string) (*query.DocumentText, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT v.id::text, v.version_no, c.chunk_no, c.content
		FROM documents d
		JOIN document_text_chunks c ON c.document_id = d.id AND c.version_id = d.current_version_id
		JOIN document_versions v ON v.id = c.version_id
		WHERE d.id::text = $1 AND d.is_deleted = false
		ORDER BY c.chunk_no
	`, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := &query.DocumentText{DocumentID: documentID}
	var b strings.Builder
	for rows.Next() {
		var chunkNo int
		var content string
		if err := rows.Scan(&out.VersionID, &out.VersionNo, &chunkNo, &content); err != nil {
			return nil, err
		}
		runes := []rune(content)
		if chunkNo > 0 {
			overlap := textsearch.ChunkSize - textsearch.ChunkStep
			if len(runes) > overlap {
				runes = runes[overlap:]
			} else {
				runes = nil
			}
		}
		b.WriteString(string(runes))
		out.Chunks++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out.Text = b.String()
	out.Available = out.Chunks > 0
	out.Truncated = out.Chunks >= textsearch.MaxChunks
	return out, nil
}

// VectorEnabled reports whether migration 016 created the embedding column,
// i.e. whether the database has pgvector.
func (r TextSearchRepository) VectorEnabled(ctx context.Context) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'document_text_chunks' AND column_name = 'embedding'
		)
	`).Scan(&ok)
	return ok, err
}

// ChunksNeedingEmbedding returns current-version chunks without a vector from
// the given model, oldest first.
func (r TextSearchRepository) ChunksNeedingEmbedding(ctx context.Context, model string, limit int) ([]query.ChunkForEmbedding, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.id::text, d.title, c.content
		FROM document_text_chunks c
		JOIN documents d ON d.id = c.document_id AND d.current_version_id = c.version_id
		WHERE d.is_deleted = false
		  AND (c.embedding IS NULL OR c.embedding_model IS DISTINCT FROM $1)
		ORDER BY c.created_at, c.chunk_no
		LIMIT $2
	`, model, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]query.ChunkForEmbedding, 0, limit)
	for rows.Next() {
		var c query.ChunkForEmbedding
		if err := rows.Scan(&c.ID, &c.Title, &c.Content); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SetChunkEmbedding stores a vector given as a pgvector text literal.
func (r TextSearchRepository) SetChunkEmbedding(ctx context.Context, chunkID string, vectorLiteral string, model string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE document_text_chunks SET embedding = $2::vector, embedding_model = $3 WHERE id::text = $1
	`, chunkID, vectorLiteral, model)
	return err
}

// VectorChunks returns the chunks nearest to the query vector (cosine), with
// the same visibility filters as keyword search, keeping only hits at or above
// minSimilarity.
func (r TextSearchRepository) VectorChunks(ctx context.Context, filter query.TextSearchFilter, vectorLiteral string, model string, limit int, minSimilarity float64) ([]query.TextSearchHit, error) {
	isAdmin, actorID := scopeArgs(filter.Scope)
	rows, err := r.db.QueryContext(ctx, `
		SELECT d.id::text, d.title, d.project_id::text, COALESCE(p.name, ''),
			v.id::text, v.version_no, c.chunk_no, c.content,
			1 - (c.embedding <=> $1::vector) AS similarity
		FROM document_text_chunks c
		JOIN documents d ON d.id = c.document_id AND d.current_version_id = c.version_id
		JOIN document_versions v ON v.id = c.version_id
		LEFT JOIN projects p ON p.id = d.project_id
		WHERE d.is_deleted = false
		  AND c.embedding IS NOT NULL
		  AND c.embedding_model = $2
		  AND ($3 = '' OR d.project_id::text = $3)
		  AND ($4 = '' OR d.id::text = $4)
		  AND `+projectScopeSQL("d.project_id", "$5", "$6")+`
		ORDER BY c.embedding <=> $1::vector
		LIMIT $7
	`, vectorLiteral, model, filter.ProjectID, filter.DocumentID, isAdmin, actorID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	hits := make([]query.TextSearchHit, 0, limit)
	for rows.Next() {
		var h query.TextSearchHit
		if err := rows.Scan(&h.DocumentID, &h.Title, &h.ProjectID, &h.ProjectName, &h.VersionID, &h.VersionNo, &h.ChunkNo, &h.Content, &h.Similarity); err != nil {
			return nil, err
		}
		if h.Similarity < minSimilarity {
			continue
		}
		hits = append(hits, h)
	}
	return hits, rows.Err()
}
