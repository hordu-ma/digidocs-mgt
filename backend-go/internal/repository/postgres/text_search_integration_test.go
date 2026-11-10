package postgres_test

import (
	"context"
	"strings"
	"testing"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/domain/task"
	noopqueue "digidocs-mgt/backend-go/internal/queue/noop"
	pgrepo "digidocs-mgt/backend-go/internal/repository/postgres"
	"digidocs-mgt/backend-go/internal/service"
)

func TestPostgresIntegrationTextChunksSearchProgressAndActions(t *testing.T) {
	ctx := context.Background()
	database := openIntegrationDB(t)
	seedIntegrationBaseRows(t, ctx, database)
	const outsider = "00000000-0000-0000-0000-000000000601"
	if _, err := database.ExecContext(ctx, `INSERT INTO users(id,username,password_hash,display_name,role,status) VALUES($1,'o','h','o','member','active')`, outsider); err != nil {
		t.Fatal(err)
	}
	documents := pgrepo.NewDocumentRepository(database)
	versions := pgrepo.NewVersionRepository(database)
	workflow := pgrepo.NewVersionWorkflow(database)
	permissions := service.NewPermissionService(pgrepo.NewPermissionRepository(database))
	repo := pgrepo.NewAssistantRepository(database)
	assistant := service.NewAssistantService(noopqueue.NewPublisher(), repo, documents, versions, permissions)
	search := service.NewSearchService(pgrepo.NewTextSearchRepository(database))

	doc, err := documents.CreateDocument(ctx, command.DocumentCreateInput{TeamSpaceID: integrationTeamSpaceID, ProjectID: integrationProjectID, Title: "强化学习方案", CurrentOwnerID: integrationUserID, ActorID: integrationUserID})
	if err != nil {
		t.Fatal(err)
	}
	docID := doc["id"].(string)
	newVersion := func() string {
		t.Helper()
		v, err := workflow.CreateUploadedVersion(ctx, command.VersionCreateInput{DocumentID: docID, FileName: "a.txt", FileSize: 1, StorageProvider: "memory", StorageObjectKey: "k", ActorID: integrationUserID})
		if err != nil {
			t.Fatal(err)
		}
		return v["id"].(string)
	}
	extract := func(version, text string) {
		t.Helper()
		msg, err := assistant.QueueTask(ctx, task.TaskTypeDocumentExtractText, "document", docID, map[string]any{"version_id": version}, integrationUserID)
		if err != nil {
			t.Fatal(err)
		}
		if err := assistant.ReceiveResult(ctx, task.Result{RequestID: msg.RequestID, Status: "completed", Output: map[string]any{"extracted_text": text, "version_id": version}}); err != nil {
			t.Fatal(err)
		}
	}
	chunkCount := func() int {
		t.Helper()
		var n int
		if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM document_text_chunks WHERE document_id=$1`, docID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	v1 := newVersion()
	long := strings.Repeat("背景介绍。", 300) + "本方案采用 PPO 算法，奖励函数由距离惩罚构成。" + strings.Repeat("实验细节。", 300)
	extract(v1, long)
	if n := chunkCount(); n < 3 {
		t.Fatalf("long text should be chunked, got %d", n)
	}
	hits, err := search.SearchChunks(ctx, query.TextSearchFilter{Query: "PPO 的奖励函数是什么", ProjectID: integrationProjectID, Scope: query.ActorScope{ActorID: integrationUserID}})
	if err != nil || len(hits) == 0 || !strings.Contains(hits[0].Content, "奖励函数") || hits[0].Title != "强化学习方案" {
		t.Fatalf("top hit should contain the answer: %#v %v", hits, err)
	}
	if !strings.Contains(hits[0].Snippet, "奖励") {
		t.Fatalf("snippet should center on the match: %q", hits[0].Snippet)
	}
	if out, _ := search.SearchChunks(ctx, query.TextSearchFilter{Query: "奖励函数", Scope: query.ActorScope{ActorID: outsider}}); len(out) != 0 {
		t.Fatal("non-member must not see project text")
	}
	docs, err := search.SearchDocuments(ctx, query.TextSearchFilter{Query: "PPO 奖励函数", Scope: query.ActorScope{ActorID: integrationUserID}})
	if err != nil || len(docs) != 1 || docs[0].DocumentID != docID || docs[0].MatchedChunks < 1 || !strings.Contains(docs[0].Snippet, "奖励") {
		t.Fatalf("document search should return one row per document: %#v %v", docs, err)
	}
	if out, _ := search.SearchDocuments(ctx, query.TextSearchFilter{Query: "奖励函数", Scope: query.ActorScope{ActorID: outsider}}); len(out) != 0 {
		t.Fatal("non-member must not find documents by content")
	}

	// A new version replaces the index; an old version's late extraction is ignored.
	v2 := newVersion()
	extract(v2, "第二版只讨论数据集划分。")
	extract(v1, long)
	if hits, _ := search.SearchChunks(ctx, query.TextSearchFilter{Query: "奖励函数", Scope: query.ActorScope{IsAdmin: true}}); len(hits) != 0 {
		t.Fatalf("superseded version text must not be searchable: %#v", hits)
	}
	if hits, _ := search.SearchChunks(ctx, query.TextSearchFilter{Query: "数据集划分", DocumentID: docID, Scope: query.ActorScope{IsAdmin: true}}); len(hits) != 1 || hits[0].VersionID != v2 {
		t.Fatalf("current version should be indexed: %#v", hits)
	}

	// Progress only lands on a running request.
	ask, err := assistant.QueueTask(ctx, task.TaskTypeAssistantAsk, "document", docID, map[string]any{"question": "q"}, integrationUserID)
	if err != nil {
		t.Fatal(err)
	}
	if err := assistant.ReceiveProgress(ctx, ask.RequestID, "早到的片段"); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `UPDATE assistant_requests SET status='running' WHERE id=$1`, ask.RequestID); err != nil {
		t.Fatal(err)
	}
	if err := assistant.ReceiveProgress(ctx, ask.RequestID, "生成中的回答"); err != nil {
		t.Fatal(err)
	}
	var partial string
	if err := database.QueryRowContext(ctx, `SELECT COALESCE(output->>'partial_answer','') FROM assistant_requests WHERE id=$1`, ask.RequestID).Scan(&partial); err != nil || partial != "生成中的回答" {
		t.Fatalf("partial answer %q err=%v", partial, err)
	}

	// Suggestions: only whitelisted actions on documents survive.
	sum, err := assistant.QueueTask(ctx, task.TaskTypeDocumentSummarize, "document", docID, map[string]any{}, integrationUserID)
	if err != nil {
		t.Fatal(err)
	}
	if err := assistant.ReceiveResult(ctx, task.Result{RequestID: sum.RequestID, Status: "completed", Output: map[string]any{
		"summary_text": "摘要", "version_id": v2,
		"suggestions": []any{
			map[string]any{"title": "可以定稿", "content": "内容完整", "suggested_action": "finalize"},
			map[string]any{"title": "转给别人", "content": "x", "suggested_action": "transfer"},
		},
	}}); err != nil {
		t.Fatal(err)
	}
	items, err := repo.ListSuggestions(ctx, query.AssistantSuggestionFilter{RelatedType: "document", RelatedID: docID, Scope: query.ActorScope{IsAdmin: true}})
	if err != nil {
		t.Fatal(err)
	}
	actions := map[string]string{}
	for _, it := range items {
		actions[it.Title] = it.SuggestedAction
	}
	if actions["可以定稿"] != "finalize" || actions["转给别人"] != "" {
		t.Fatalf("suggested actions %#v", actions)
	}
	var summary string
	if err := database.QueryRowContext(ctx, `SELECT COALESCE(summary_text,'') FROM document_versions WHERE id=$1`, v2).Scan(&summary); err != nil || summary != "摘要" {
		t.Fatalf("summary without payload version should land on output version: %q %v", summary, err)
	}
}

func TestPostgresIntegrationDocumentListFilters(t *testing.T) {
	ctx := context.Background()
	database := openIntegrationDB(t)
	seedIntegrationBaseRows(t, ctx, database)
	const member = "00000000-0000-0000-0000-000000000701"
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := database.ExecContext(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO users(id,username,password_hash,display_name,role,status) VALUES($1,'m','h','成员','member','active')`, member)
	exec(`INSERT INTO project_members(project_id,user_id,project_role) VALUES($1,$2,'contributor')`, integrationProjectID, member)
	documents := pgrepo.NewDocumentRepository(database)
	newDoc := func(title, owner string) string {
		t.Helper()
		d, err := documents.CreateDocument(ctx, command.DocumentCreateInput{TeamSpaceID: integrationTeamSpaceID, ProjectID: integrationProjectID, Title: title, CurrentOwnerID: owner, ActorID: owner})
		if err != nil {
			t.Fatal(err)
		}
		return d["id"].(string)
	}
	newDoc("b 报告", integrationUserID)
	mine := newDoc("a 方案", member)
	old := newDoc("100% 完成", member)
	gone := newDoc("已删除", member)
	exec(`UPDATE documents SET updated_at = NOW() - INTERVAL '20 days' WHERE id=$1`, old)
	exec(`UPDATE documents SET is_deleted = true WHERE id=$1`, gone)

	list := func(f query.DocumentListFilter) ([]query.DocumentListItem, int) {
		t.Helper()
		items, total, err := documents.ListDocuments(ctx, f)
		if err != nil {
			t.Fatal(err)
		}
		return items, total
	}
	if items, total := list(query.DocumentListFilter{ActorID: member, Sort: "title_asc", PageSize: 2}); total != 3 || len(items) != 2 || items[0].Title != "100% 完成" || items[1].ID != mine {
		t.Fatalf("title sort + paging: total=%d %#v", total, items)
	}
	if _, total := list(query.DocumentListFilter{ActorID: member, OwnerID: member}); total != 2 {
		t.Fatalf("mine: %d", total)
	}
	if _, total := list(query.DocumentListFilter{ActorID: member, UpdatedWithinDays: 7}); total != 2 {
		t.Fatalf("recent: %d", total)
	}
	if items, total := list(query.DocumentListFilter{ActorID: member, Keyword: "%"}); total != 1 || items[0].ID != old {
		t.Fatalf("keyword %% must be literal: total=%d", total)
	}
	if _, total := list(query.DocumentListFilter{ActorID: member, DeletedOnly: true}); total != 0 {
		t.Fatal("a contributor must not see the recycle bin")
	}
	if items, total := list(query.DocumentListFilter{ActorID: integrationUserID, DeletedOnly: true}); total != 1 || items[0].ID != gone || !items[0].IsDeleted {
		t.Fatalf("project owner sees the recycle bin: total=%d", total)
	}
	counts, err := documents.CountDocumentsByProject(ctx, query.DocumentListFilter{ActorID: member, ProjectID: "ignored-for-counts"})
	if err != nil || len(counts) != 1 || counts[0].Count != 3 {
		t.Fatalf("project counts: %#v %v", counts, err)
	}
}
