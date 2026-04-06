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
)

type AssistantRepository struct {
	db DBTX
}

func NewAssistantRepository(db DBTX) repository.AssistantRepository {
	return AssistantRepository{db: db}
}

func (r AssistantRepository) CreateAssistantRequest(
	ctx context.Context,
	message task.Message,
	actorID string,
) error {
	payload, err := json.Marshal(message.Payload)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(
		ctx,
		`
		INSERT INTO assistant_requests (
			id,
			request_type,
			related_type,
			related_id,
			payload,
			status,
			created_by,
			created_at
		)
		VALUES (
			$1::uuid,
			$2,
			NULLIF($3, ''),
			NULLIF($4, '')::uuid,
			$5::jsonb,
			'pending',
			NULLIF($6, '')::uuid,
			$7
		)
		`,
		message.RequestID,
		string(message.TaskType),
		message.RelatedType,
		message.RelatedID,
		string(payload),
		actorID,
		time.Now().UTC(),
	)
	return err
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
	var payloadText string
	if err := tx.QueryRowContext(
		ctx,
		`
		SELECT
			request_type,
			COALESCE(related_type, ''),
			COALESCE(related_id::text, ''),
			COALESCE(payload::text, '{}')
		FROM assistant_requests
		WHERE id::text = $1
		FOR UPDATE
		`,
		result.RequestID,
	).Scan(&reqType, &relatedType, &relatedID, &payloadText); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return service.ErrNotFound
		}
		return err
	}

	if _, err := tx.ExecContext(
		ctx,
		`
		UPDATE assistant_requests
		SET status = $2,
		    error_message = NULLIF($3, ''),
		    completed_at = $4
		WHERE id::text = $1
		`,
		result.RequestID,
		result.Status,
		result.ErrorMessage,
		time.Now().UTC(),
	); err != nil {
		return err
	}

	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM assistant_suggestions WHERE request_id = $1`,
		result.RequestID,
	); err != nil {
		return err
	}

	payload := map[string]any{}
	_ = json.Unmarshal([]byte(payloadText), &payload)
	request := assistantRequestRecord{
		RequestID:   result.RequestID,
		RequestType: reqType,
		RelatedType: relatedType,
		RelatedID:   relatedID,
		Payload:     payload,
		Status:      result.Status,
		Error:       result.ErrorMessage,
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
		  AND ($3 = '' OR status = $3)
		  AND ($4 = '' OR suggestion_type = $4)
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

type assistantRequestRecord struct {
	RequestID   string
	RequestType string
	RelatedType string
	RelatedID   string
	Payload     map[string]any
	Status      string
	Error       string
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

func clonePayload(payload map[string]any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(payload))
	for k, v := range payload {
		cloned[k] = v
	}
	return cloned
}

func stringValue(value any) string {
	raw, _ := value.(string)
	return raw
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
