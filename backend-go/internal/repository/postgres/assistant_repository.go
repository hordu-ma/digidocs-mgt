package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/domain/task"
	"digidocs-mgt/backend-go/internal/repository"
	"digidocs-mgt/backend-go/internal/service"
	"github.com/lib/pq"
)

type AssistantRepository struct {
	db DBTX
}

func NewAssistantRepository(db DBTX) repository.AssistantRepository {
	return AssistantRepository{db: db}
}

func (r AssistantRepository) CreateConversation(
	ctx context.Context,
	scopeType string,
	scopeID string,
	sourceScope map[string]any,
	title string,
	actorID string,
) (*query.AssistantConversationItem, error) {
	now := time.Now().UTC()
	id := newID()
	_, err := r.db.ExecContext(
		ctx,
		`
		INSERT INTO assistant_conversations (
			id,
			scope_type,
			scope_id,
			source_scope,
			title,
			created_by,
			created_at,
			last_message_at
		)
		VALUES (
			$1::uuid,
			$2,
			$3::uuid,
			$4::jsonb,
			NULLIF($5, ''),
			NULLIF($6, '')::uuid,
			$7,
			$7
		)
		`,
		id,
		scopeType,
		scopeID,
		mustJSON(sourceScope),
		title,
		actorID,
		now,
	)
	if err != nil {
		return nil, err
	}
	return &query.AssistantConversationItem{
		ID:            id,
		ScopeType:     scopeType,
		ScopeID:       scopeID,
		SourceScope:   clonePayload(sourceScope),
		Title:         title,
		CreatedBy:     actorID,
		CreatedAt:     now.Format(time.RFC3339),
		LastMessageAt: now.Format(time.RFC3339),
	}, nil
}

func (r AssistantRepository) GetConversation(
	ctx context.Context,
	conversationID string,
) (*query.AssistantConversationItem, error) {
	row := r.db.QueryRowContext(
		ctx,
		`
		SELECT
			c.id::text,
			c.scope_type,
			c.scope_id::text,
			COALESCE(c.source_scope::text, '{}'),
			COALESCE(
				CASE c.scope_type
					WHEN 'document' THEN (SELECT d.title FROM documents d WHERE d.id = c.scope_id)
					WHEN 'project'  THEN (SELECT p.name  FROM projects p  WHERE p.id = c.scope_id)
				END,
				''
			),
			COALESCE(c.title, ''),
			COALESCE(c.created_by::text, ''),
			TO_CHAR(c.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			TO_CHAR(c.last_message_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			COALESCE(TO_CHAR(c.archived_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')
		FROM assistant_conversations c
		WHERE c.id::text = $1
		`,
		conversationID,
	)
	item, err := scanAssistantConversationItem(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (r AssistantRepository) ListConversations(
	ctx context.Context,
	filter query.AssistantConversationFilter,
) ([]query.AssistantConversationItem, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT
			c.id::text,
			c.scope_type,
			c.scope_id::text,
			COALESCE(c.source_scope::text, '{}'),
			COALESCE(
				CASE c.scope_type
					WHEN 'document' THEN (SELECT d.title FROM documents d WHERE d.id = c.scope_id)
					WHEN 'project'  THEN (SELECT p.name  FROM projects p  WHERE p.id = c.scope_id)
				END,
				''
			),
			COALESCE(c.title, ''),
			COALESCE(c.created_by::text, ''),
			TO_CHAR(c.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			TO_CHAR(c.last_message_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			COALESCE(TO_CHAR(c.archived_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')
		FROM assistant_conversations c
		WHERE ($1 = '' OR c.scope_type = $1)
		  AND ($2 = '' OR c.scope_id::text = $2)
		  AND ($3 = '' OR COALESCE(c.created_by::text, '') = $3)
		  AND ($4 OR c.archived_at IS NULL)
		ORDER BY c.last_message_at DESC, c.created_at DESC
		`,
		filter.ScopeType,
		filter.ScopeID,
		filter.CreatedBy,
		filter.IncludeArchived,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]query.AssistantConversationItem, 0)
	for rows.Next() {
		item, err := scanAssistantConversationItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r AssistantRepository) ListConversationMessages(
	ctx context.Context,
	conversationID string,
) ([]query.AssistantConversationMessageItem, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT
			id::text,
			conversation_id::text,
			role,
			content,
			COALESCE(request_id::text, ''),
			COALESCE(metadata::text, '{}'),
			COALESCE(created_by::text, ''),
			TO_CHAR(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM assistant_conversation_messages
		WHERE conversation_id::text = $1
		ORDER BY created_at ASC, id ASC
		`,
		conversationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]query.AssistantConversationMessageItem, 0)
	for rows.Next() {
		item, err := scanAssistantConversationMessageItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		if _, err := r.GetConversation(ctx, conversationID); err != nil {
			return nil, err
		}
	}
	return items, rows.Err()
}

func (r AssistantRepository) CreateAssistantRequest(
	ctx context.Context,
	message task.Message,
	actorID string,
) error {
	payload := clonePayload(message.Payload)
	payloadText, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	tx, err := asTx(ctx, r.db)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	conversationID := stringValue(payload["conversation_id"])
	_, err = tx.ExecContext(
		ctx,
		`
		INSERT INTO assistant_requests (
			id,
			request_type,
			related_type,
			related_id,
			conversation_id,
			payload,
			output,
			status,
			created_by,
			created_at
		)
		VALUES (
			$1::uuid,
			$2,
			NULLIF($3, ''),
			NULLIF($4, '')::uuid,
			NULLIF($5, '')::uuid,
			$6::jsonb,
			'{}'::jsonb,
			'pending',
			NULLIF($7, '')::uuid,
			$8
		)
		`,
		message.RequestID,
		string(message.TaskType),
		message.RelatedType,
		message.RelatedID,
		conversationID,
		string(payloadText),
		actorID,
		now,
	)
	if err != nil {
		return err
	}

	if conversationID != "" && stringValue(payload["question"]) != "" {
		if err := insertConversationMessage(
			ctx,
			tx,
			conversationMessageRecord{
				ID:             newID(),
				ConversationID: conversationID,
				Role:           "user",
				Content:        stringValue(payload["question"]),
				RequestID:      message.RequestID,
				Metadata: map[string]any{
					"source_scope":   extractScopeFromPayload(payload),
					"memory_sources": extractMemorySources(payload),
				},
				CreatedBy: actorID,
				CreatedAt: now,
			},
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r AssistantRepository) CompleteAssistantRequest(
	ctx context.Context,
	result task.Result,
) error {
	tx, err := asTx(ctx, r.db)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var reqType string
	var relatedType string
	var relatedID string
	var conversationID string
	var payloadText string
	var createdAt time.Time
	if err := tx.QueryRowContext(
		ctx,
		`
		SELECT
			request_type,
			COALESCE(related_type, ''),
			COALESCE(related_id::text, ''),
			COALESCE(conversation_id::text, ''),
			COALESCE(payload::text, '{}'),
			created_at
		FROM assistant_requests
		WHERE id::text = $1
		FOR UPDATE
		`,
		result.RequestID,
	).Scan(&reqType, &relatedType, &relatedID, &conversationID, &payloadText, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrNotFound
		}
		return err
	}

	completedAt := time.Now().UTC()
	if _, err := tx.ExecContext(
		ctx,
		`
		UPDATE assistant_requests
		SET status = $2,
		    error_message = NULLIF($3, ''),
		    output = $4::jsonb,
		    completed_at = $5
		WHERE id::text = $1
		`,
		result.RequestID,
		result.Status,
		result.ErrorMessage,
		mustJSON(result.Output),
		completedAt,
	); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM assistant_suggestions WHERE request_id = $1`, result.RequestID); err != nil {
		return err
	}

	payload := map[string]any{}
	_ = json.Unmarshal([]byte(payloadText), &payload)
	request := assistantRequestRecord{
		RequestID:      result.RequestID,
		RequestType:    reqType,
		RelatedType:    relatedType,
		RelatedID:      relatedID,
		ConversationID: conversationID,
		Payload:        payload,
		Status:         result.Status,
		Error:          result.ErrorMessage,
		CreatedAt:      createdAt.Format(time.RFC3339),
		CompletedAt:    completedAt.Format(time.RFC3339),
	}

	if err := updateAssistantProjection(ctx, tx, request, result); err != nil {
		return err
	}
	for _, item := range buildSuggestionItems(request, result) {
		var confidence any
		if item.Confidence != nil {
			confidence = *item.Confidence
		}
		_, err := tx.ExecContext(
			ctx,
			`
			INSERT INTO assistant_suggestions (
				id,
				related_type,
				related_id,
				suggestion_type,
				status,
				title,
				content,
				source_scope,
				confidence,
				request_id,
				generated_at
			)
			VALUES (
				$1::uuid,
				$2,
				$3::uuid,
				$4,
				$5,
				NULLIF($6, ''),
				$7,
				NULLIF($8, ''),
				$9,
				$10,
				$11
			)
			`,
			item.ID,
			item.RelatedType,
			item.RelatedID,
			item.SuggestionType,
			item.Status,
			item.Title,
			item.Content,
			item.SourceScope,
			confidence,
			item.RequestID,
			parseRFC3339(item.GeneratedAt),
		)
		if err != nil {
			return err
		}
	}

	if conversationID != "" {
		content := stringValue(result.Output["answer"])
		if content == "" && result.Status == "failed" {
			content = result.ErrorMessage
		}
		if content != "" {
			if err := insertConversationMessage(
				ctx,
				tx,
				conversationMessageRecord{
					ID:             newID(),
					ConversationID: conversationID,
					Role:           "assistant",
					Content:        content,
					RequestID:      result.RequestID,
					Metadata: map[string]any{
						"status":                 result.Status,
						"model":                  stringValue(result.Output["model"]),
						"skill_name":             stringValue(result.Output["skill_name"]),
						"skill_version":          stringValue(result.Output["skill_version"]),
						"upstream_request_id":    stringValue(result.Output["request_id"]),
						"source_scope":           extractScope(payload, result.Output),
						"memory_sources":         extractMemorySources(payload),
						"processing_duration_ms": completedAt.Sub(createdAt).Milliseconds(),
					},
					CreatedAt: completedAt,
				},
			); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func updateAssistantProjection(
	ctx context.Context,
	tx *sql.Tx,
	request assistantRequestRecord,
	result task.Result,
) error {
	if result.Status != "completed" {
		return nil
	}

	switch request.RequestType {
	case string(task.TaskTypeDocumentExtractText):
		versionID := stringValue(request.Payload["version_id"])
		if versionID == "" {
			return nil
		}
		status := "failed"
		if stringValue(result.Output["extracted_text"]) != "" {
			status = "completed"
		}
		_, err := tx.ExecContext(
			ctx,
			`
			UPDATE document_versions
			SET extracted_text_status = $2
			WHERE id::text = $1
			`,
			versionID,
			status,
		)
		return err
	case string(task.TaskTypeDocumentSummarize):
		versionID := stringValue(request.Payload["version_id"])
		if versionID == "" {
			return nil
		}
		_, err := tx.ExecContext(
			ctx,
			`
			UPDATE document_versions
			SET summary_status = 'completed',
			    summary_text = NULLIF($2, '')
			WHERE id::text = $1
			`,
			versionID,
			stringValue(result.Output["summary_text"]),
		)
		return err
	case string(task.TaskTypeHandoverSummarize):
		if request.RelatedID == "" {
			return nil
		}
		_, err := tx.ExecContext(
			ctx,
			`
			UPDATE graduation_handovers
			SET ai_summary = NULLIF($2, '')
			WHERE id::text = $1
			`,
			request.RelatedID,
			stringValue(result.Output["summary_text"]),
		)
		return err
	default:
		return nil
	}
}

func (r AssistantRepository) ListSuggestions(
	ctx context.Context,
	filter query.AssistantSuggestionFilter,
) ([]query.AssistantSuggestionItem, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT
			id::text,
			related_type,
			related_id::text,
			suggestion_type,
			status,
			COALESCE(title, ''),
			content,
			COALESCE(source_scope, ''),
			confidence,
			COALESCE(request_id, ''),
			TO_CHAR(generated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM assistant_suggestions
		WHERE ($1 = '' OR related_type = $1)
		  AND ($2 = '' OR related_id::text = $2)
		  AND ($3 = '' OR status::text = $3)
		  AND ($4 = '' OR suggestion_type::text = $4)
		ORDER BY generated_at DESC
		`,
		filter.RelatedType,
		filter.RelatedID,
		filter.Status,
		filter.SuggestionType,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]query.AssistantSuggestionItem, 0)
	for rows.Next() {
		var item query.AssistantSuggestionItem
		var confidence sql.NullFloat64
		if err := rows.Scan(
			&item.ID,
			&item.RelatedType,
			&item.RelatedID,
			&item.SuggestionType,
			&item.Status,
			&item.Title,
			&item.Content,
			&item.SourceScope,
			&confidence,
			&item.RequestID,
			&item.GeneratedAt,
		); err != nil {
			return nil, err
		}
		if confidence.Valid {
			item.Confidence = &confidence.Float64
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r AssistantRepository) ListAssistantRequests(
	ctx context.Context,
	filter query.AssistantRequestFilter,
) ([]query.AssistantRequestItem, int, error) {
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int
	err := r.db.QueryRowContext(
		ctx,
		`
		SELECT COUNT(*)
		FROM assistant_requests
		WHERE ($1 = '' OR request_type = $1)
		  AND ($2 = '' OR COALESCE(related_type, '') = $2)
		  AND ($3 = '' OR COALESCE(related_id::text, '') = $3)
		  AND ($4 = '' OR COALESCE(conversation_id::text, '') = $4)
		  AND ($5 = '' OR status = $5)
		  AND ($6 = '' OR COALESCE(payload->>'question', '') ILIKE '%' || $6 || '%')
		`,
		filter.RequestType,
		filter.RelatedType,
		filter.RelatedID,
		filter.ConversationID,
		filter.Status,
		filter.Keyword,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT
			id::text,
			request_type,
			COALESCE(related_type, ''),
			COALESCE(related_id::text, ''),
			COALESCE(conversation_id::text, ''),
			status,
			COALESCE(error_message, ''),
			COALESCE(payload::text, '{}'),
			COALESCE(output::text, '{}'),
			TO_CHAR(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			COALESCE(TO_CHAR(completed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')
		FROM assistant_requests
		WHERE ($1 = '' OR request_type = $1)
		  AND ($2 = '' OR COALESCE(related_type, '') = $2)
		  AND ($3 = '' OR COALESCE(related_id::text, '') = $3)
		  AND ($4 = '' OR COALESCE(conversation_id::text, '') = $4)
		  AND ($5 = '' OR status = $5)
		  AND ($6 = '' OR COALESCE(payload->>'question', '') ILIKE '%' || $6 || '%')
		ORDER BY created_at DESC
		LIMIT $7 OFFSET $8
		`,
		filter.RequestType,
		filter.RelatedType,
		filter.RelatedID,
		filter.ConversationID,
		filter.Status,
		filter.Keyword,
		pageSize,
		offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]query.AssistantRequestItem, 0)
	for rows.Next() {
		item, err := scanAssistantRequestItem(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

func (r AssistantRepository) GetAssistantRequest(
	ctx context.Context,
	requestID string,
) (*query.AssistantRequestItem, error) {
	row := r.db.QueryRowContext(
		ctx,
		`
		SELECT
			id::text,
			request_type,
			COALESCE(related_type, ''),
			COALESCE(related_id::text, ''),
			COALESCE(conversation_id::text, ''),
			status,
			COALESCE(error_message, ''),
			COALESCE(payload::text, '{}'),
			COALESCE(output::text, '{}'),
			TO_CHAR(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
			COALESCE(TO_CHAR(completed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')
		FROM assistant_requests
		WHERE id::text = $1
		`,
		requestID,
	)
	item, err := scanAssistantRequestItem(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (r AssistantRepository) GetLatestDocumentExtractedText(
	ctx context.Context,
	documentID string,
) (string, error) {
	var outputText string
	err := r.db.QueryRowContext(
		ctx,
		`
		SELECT COALESCE(output::text, '{}')
		FROM assistant_requests
		WHERE request_type = ANY($1)
		  AND related_type = 'document'
		  AND related_id::text = $2
		  AND status = 'completed'
		ORDER BY completed_at DESC NULLS LAST, created_at DESC
		LIMIT 1
		`,
		pq.Array([]string{
			string(task.TaskTypeDocumentExtractText),
			string(task.TaskTypeDocumentSummarize),
		}),
		documentID,
	).Scan(&outputText)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}

	payload := map[string]any{}
	_ = json.Unmarshal([]byte(outputText), &payload)
	return stringValue(payload["extracted_text"]), nil
}

func (r AssistantRepository) ConfirmSuggestion(
	ctx context.Context,
	suggestionID string,
	actorID string,
	note string,
) (map[string]any, error) {
	return r.updateSuggestionStatus(ctx, suggestionID, actorID, note, "confirmed")
}

func (r AssistantRepository) DismissSuggestion(
	ctx context.Context,
	suggestionID string,
	actorID string,
	reason string,
) (map[string]any, error) {
	return r.updateSuggestionStatus(ctx, suggestionID, actorID, reason, "dismissed")
}

func (r AssistantRepository) updateSuggestionStatus(
	ctx context.Context,
	suggestionID string,
	actorID string,
	text string,
	status string,
) (map[string]any, error) {
	fieldByStatus := map[string]string{
		"confirmed": "confirmed",
		"dismissed": "dismissed",
	}
	prefix, ok := fieldByStatus[status]
	if !ok {
		return nil, fmt.Errorf("%w: invalid suggestion status %s", service.ErrValidation, status)
	}

	queryText := fmt.Sprintf(
		`
		UPDATE assistant_suggestions
		SET status = '%s',
		    %s_by = NULLIF($2, '')::uuid,
		    %s_at = $3
		WHERE id::text = $1
		RETURNING id::text
		`,
		status,
		prefix,
		prefix,
	)

	var updatedID string
	if err := r.db.QueryRowContext(ctx, queryText, suggestionID, actorID, time.Now().UTC()).Scan(&updatedID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrNotFound
		}
		return nil, err
	}

	result := map[string]any{
		"id":     updatedID,
		"status": status,
	}
	if status == "confirmed" {
		result["confirmed_by"] = actorID
		result["note"] = text
	} else {
		result["dismissed_by"] = actorID
		result["reason"] = text
	}
	return result, nil
}

type assistantRequestRecord struct {
	RequestID      string
	RequestType    string
	RelatedType    string
	RelatedID      string
	ConversationID string
	Payload        map[string]any
	Status         string
	Error          string
	CreatedAt      string
	CompletedAt    string
}

type conversationMessageRecord struct {
	ID             string
	ConversationID string
	Role           string
	Content        string
	RequestID      string
	Metadata       map[string]any
	CreatedBy      string
	CreatedAt      time.Time
}

func insertConversationMessage(
	ctx context.Context,
	tx *sql.Tx,
	record conversationMessageRecord,
) error {
	if _, err := tx.ExecContext(
		ctx,
		`
		INSERT INTO assistant_conversation_messages (
			id,
			conversation_id,
			role,
			content,
			request_id,
			metadata,
			created_by,
			created_at
		)
		VALUES (
			$1::uuid,
			$2::uuid,
			$3,
			$4,
			NULLIF($5, '')::uuid,
			$6::jsonb,
			NULLIF($7, '')::uuid,
			$8
		)
		`,
		record.ID,
		record.ConversationID,
		record.Role,
		record.Content,
		record.RequestID,
		mustJSON(record.Metadata),
		record.CreatedBy,
		record.CreatedAt,
	); err != nil {
		return err
	}
	_, err := tx.ExecContext(
		ctx,
		`
		UPDATE assistant_conversations
		SET last_message_at = $2
		WHERE id::text = $1
		`,
		record.ConversationID,
		record.CreatedAt,
	)
	return err
}

func buildSuggestionItems(
	req assistantRequestRecord,
	result task.Result,
) []query.AssistantSuggestionItem {
	if result.Status != "completed" {
		return nil
	}

	items := make([]query.AssistantSuggestionItem, 0)
	now := time.Now().UTC().Format(time.RFC3339)
	sourceScope := stringifyScope(req.Payload)
	if outputScope, ok := result.Output["source_scope"].(map[string]any); ok {
		if raw, err := json.Marshal(outputScope); err == nil {
			sourceScope = string(raw)
		}
	}

	if text := stringValue(result.Output["summary_text"]); text != "" {
		items = append(items, query.AssistantSuggestionItem{
			ID:             newID(),
			RelatedType:    req.RelatedType,
			RelatedID:      req.RelatedID,
			SuggestionType: suggestionTypeForRequest(req.RequestType),
			Status:         "pending",
			Title:          defaultSuggestionTitle(req.RequestType),
			Content:        text,
			SourceScope:    sourceScope,
			RequestID:      req.RequestID,
			GeneratedAt:    now,
		})
	}

	if rawSuggestions, ok := result.Output["suggestions"].([]any); ok {
		for i, raw := range rawSuggestions {
			m, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			content := stringValue(m["content"])
			if content == "" {
				continue
			}
			item := query.AssistantSuggestionItem{
				ID:             newID(),
				RelatedType:    req.RelatedType,
				RelatedID:      req.RelatedID,
				SuggestionType: fallbackString(stringValue(m["suggestion_type"]), suggestionTypeForRequest(req.RequestType)),
				Status:         "pending",
				Title:          fallbackString(stringValue(m["title"]), defaultSuggestionTitle(req.RequestType)+"-"+strconv.Itoa(i+1)),
				Content:        content,
				SourceScope:    fallbackString(stringValue(m["source_scope"]), sourceScope),
				RequestID:      req.RequestID,
				GeneratedAt:    now,
			}
			if confidence, ok := floatValue(m["confidence"]); ok {
				item.Confidence = &confidence
			}
			items = append(items, item)
		}
	}

	return items
}

func suggestionTypeForRequest(requestType string) string {
	switch requestType {
	case string(task.TaskTypeDocumentSummarize):
		return "document_summary"
	case string(task.TaskTypeHandoverSummarize):
		return "handover_summary"
	case string(task.TaskTypeAssistantGenerateSuggestion):
		return "structure_recommendation"
	default:
		return "document_summary"
	}
}

func defaultSuggestionTitle(requestType string) string {
	switch requestType {
	case string(task.TaskTypeDocumentSummarize):
		return "文档摘要"
	case string(task.TaskTypeHandoverSummarize):
		return "交接摘要"
	default:
		return "AI 建议"
	}
}

func stringifyScope(payload map[string]any) string {
	scopePayload := map[string]any{}
	if scope, ok := payload["scope"].(map[string]any); ok {
		scopePayload = clonePayload(scope)
	} else {
		if projectID := stringValue(payload["project_id"]); projectID != "" {
			scopePayload["project_id"] = projectID
		}
		if documentID := stringValue(payload["document_id"]); documentID != "" {
			scopePayload["document_id"] = documentID
		}
	}
	if len(scopePayload) == 0 {
		return ""
	}
	raw, err := json.Marshal(scopePayload)
	if err != nil {
		return ""
	}
	return string(raw)
}

func asTx(ctx context.Context, db DBTX) (*sql.Tx, error) {
	if tx, ok := db.(*sql.Tx); ok {
		return tx, nil
	}
	rawDB, ok := db.(*sql.DB)
	if !ok {
		return nil, fmt.Errorf("assistant repository requires *sql.DB for transactions")
	}
	return rawDB.BeginTx(ctx, nil)
}

func parseRFC3339(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Now().UTC()
	}
	return parsed
}

type assistantRequestScanner interface {
	Scan(dest ...any) error
}

func scanAssistantRequestItem(scanner assistantRequestScanner) (query.AssistantRequestItem, error) {
	var item query.AssistantRequestItem
	var payloadText string
	var outputText string
	if err := scanner.Scan(
		&item.ID,
		&item.RequestType,
		&item.RelatedType,
		&item.RelatedID,
		&item.ConversationID,
		&item.Status,
		&item.ErrorMessage,
		&payloadText,
		&outputText,
		&item.CreatedAt,
		&item.CompletedAt,
	); err != nil {
		return query.AssistantRequestItem{}, err
	}

	payload := map[string]any{}
	_ = json.Unmarshal([]byte(payloadText), &payload)
	item.Question = stringValue(payload["question"])
	item.SourceScope = extractScopeFromPayload(payload)
	item.MemorySources = extractMemorySources(payload)

	item.Output = map[string]any{}
	_ = json.Unmarshal([]byte(outputText), &item.Output)
	if scope, ok := item.Output["source_scope"].(map[string]any); ok {
		item.SourceScope = clonePayload(scope)
	}
	item.SkillName = stringValue(item.Output["skill_name"])
	item.SkillVersion = stringValue(item.Output["skill_version"])
	item.Model = stringValue(item.Output["model"])
	item.UpstreamRequestID = stringValue(item.Output["request_id"])
	if usage, ok := item.Output["usage"].(map[string]any); ok {
		item.Usage = clonePayload(usage)
	}
	if item.CreatedAt != "" && item.CompletedAt != "" {
		createdAt, createdErr := time.Parse(time.RFC3339, item.CreatedAt)
		completedAt, completedErr := time.Parse(time.RFC3339, item.CompletedAt)
		if createdErr == nil && completedErr == nil {
			item.ProcessingDurationMs = completedAt.Sub(createdAt).Milliseconds()
		}
	}
	return item, nil
}

type assistantConversationScanner interface {
	Scan(dest ...any) error
}

func scanAssistantConversationItem(scanner assistantConversationScanner) (query.AssistantConversationItem, error) {
	var item query.AssistantConversationItem
	var scopeText string
	if err := scanner.Scan(
		&item.ID,
		&item.ScopeType,
		&item.ScopeID,
		&scopeText,
		&item.ScopeDisplayName,
		&item.Title,
		&item.CreatedBy,
		&item.CreatedAt,
		&item.LastMessageAt,
		&item.ArchivedAt,
	); err != nil {
		return query.AssistantConversationItem{}, err
	}
	_ = json.Unmarshal([]byte(scopeText), &item.SourceScope)
	return item, nil
}

type assistantConversationMessageScanner interface {
	Scan(dest ...any) error
}

func scanAssistantConversationMessageItem(scanner assistantConversationMessageScanner) (query.AssistantConversationMessageItem, error) {
	var item query.AssistantConversationMessageItem
	var metadataText string
	if err := scanner.Scan(
		&item.ID,
		&item.ConversationID,
		&item.Role,
		&item.Content,
		&item.RequestID,
		&metadataText,
		&item.CreatedBy,
		&item.CreatedAt,
	); err != nil {
		return query.AssistantConversationMessageItem{}, err
	}
	item.Metadata = map[string]any{}
	_ = json.Unmarshal([]byte(metadataText), &item.Metadata)
	return item, nil
}

func extractScopeFromPayload(payload map[string]any) map[string]any {
	if scope, ok := payload["scope"].(map[string]any); ok {
		return clonePayload(scope)
	}
	result := map[string]any{}
	if projectID := stringValue(payload["project_id"]); projectID != "" {
		result["project_id"] = projectID
	}
	if documentID := stringValue(payload["document_id"]); documentID != "" {
		result["document_id"] = documentID
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func extractScope(payload map[string]any, output map[string]any) map[string]any {
	if scope, ok := output["source_scope"].(map[string]any); ok {
		return clonePayload(scope)
	}
	return extractScopeFromPayload(payload)
}

func extractMemorySources(payload map[string]any) []map[string]any {
	raw, ok := payload["memory_sources"].([]any)
	if !ok {
		return nil
	}
	items := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		items = append(items, clonePayload(m))
	}
	return items
}

func clonePayload(payload map[string]any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(payload))
	for key, value := range payload {
		cloned[key] = value
	}
	return cloned
}

func mustJSON(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func floatValue(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	default:
		return 0, false
	}
}

func fallbackString(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func stringValue(value any) string {
	raw, _ := value.(string)
	return raw
}
