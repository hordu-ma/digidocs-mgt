package postgres_test

import (
	"context"
	"testing"

	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	"digidocs-mgt/backend-go/internal/domain/task"
	noopqueue "digidocs-mgt/backend-go/internal/queue/noop"
	pgrepo "digidocs-mgt/backend-go/internal/repository/postgres"
	"digidocs-mgt/backend-go/internal/service"
)

// conceptEmbedder maps texts onto two orthogonal "concept" axes so the test
// controls semantic similarity without a real model.
type conceptEmbedder struct{}

func (conceptEmbedder) Model() string { return "test-model" }
func (conceptEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i, t := range texts {
		v := make([]float32, 1024)
		if containsAny(t, "回报", "奖励", "激励") {
			v[0] = 1
		} else {
			v[1] = 1
		}
		out[i] = v
	}
	return out, nil
}

func containsAny(s string, subs ...string) bool {
	for _, x := range subs {
		for i := 0; i+len(x) <= len(s); i++ {
			if s[i:i+len(x)] == x {
				return true
			}
		}
	}
	return false
}

func TestPostgresIntegrationHybridSemanticSearch(t *testing.T) {
	ctx := context.Background()
	database := openIntegrationDB(t)
	seedIntegrationBaseRows(t, ctx, database)
	repo := pgrepo.NewTextSearchRepository(database)
	if ok, err := repo.VectorEnabled(ctx); err != nil || !ok {
		t.Fatalf("pgvector image must enable the embedding column (ok=%v err=%v)", ok, err)
	}
	const outsider = "00000000-0000-0000-0000-000000000a01"
	if _, err := database.ExecContext(ctx, `INSERT INTO users(id,username,password_hash,display_name,role,status) VALUES($1,'o2','h','o','member','active')`, outsider); err != nil {
		t.Fatal(err)
	}
	documents := pgrepo.NewDocumentRepository(database)
	workflow := pgrepo.NewVersionWorkflow(database)
	assistant := service.NewAssistantService(noopqueue.NewPublisher(), pgrepo.NewAssistantRepository(database), documents, pgrepo.NewVersionRepository(database))
	newDoc := func(title, text string) string {
		t.Helper()
		d, err := documents.CreateDocument(ctx, command.DocumentCreateInput{TeamSpaceID: integrationTeamSpaceID, ProjectID: integrationProjectID, Title: title, CurrentOwnerID: integrationUserID, ActorID: integrationUserID})
		if err != nil {
			t.Fatal(err)
		}
		id := d["id"].(string)
		v, err := workflow.CreateUploadedVersion(ctx, command.VersionCreateInput{DocumentID: id, FileName: "a.txt", FileSize: 1, StorageProvider: "memory", StorageObjectKey: "k", ActorID: integrationUserID})
		if err != nil {
			t.Fatal(err)
		}
		msg, err := assistant.QueueTask(ctx, task.TaskTypeDocumentExtractText, "document", id, map[string]any{"version_id": v["id"]}, integrationUserID)
		if err != nil {
			t.Fatal(err)
		}
		if err := assistant.ReceiveResult(ctx, task.Result{RequestID: msg.RequestID, Status: "completed", Output: map[string]any{"extracted_text": text, "version_id": v["id"]}}); err != nil {
			t.Fatal(err)
		}
		return id
	}
	paraphrase := newDoc("策略说明", "机械臂每一步的回报信号由距离惩罚与成功加成构成。")
	unrelated := newDoc("采购清单", "本季度需采购显卡与存储硬盘。")

	embedder := conceptEmbedder{}
	indexer := service.NewEmbeddingIndexer(repo, embedder)
	if n, err := indexer.RunOnce(ctx); err != nil || n != 2 {
		t.Fatalf("indexer stored %d chunks err=%v", n, err)
	}
	if n, _ := indexer.RunOnce(ctx); n != 0 {
		t.Fatalf("second pass must find nothing left, got %d", n)
	}

	search := service.NewSearchService(repo).WithSemantic(repo, embedder, 0.45)
	member := query.ActorScope{ActorID: integrationUserID}

	// "奖励函数设计" shares no keyword with "回报信号…": keyword search alone misses it.
	keywordOnly, err := service.NewSearchService(repo).SearchDocuments(ctx, query.TextSearchFilter{Query: "奖励函数设计", Scope: member})
	if err != nil || len(keywordOnly) != 0 {
		t.Fatalf("precondition: keyword search should miss the paraphrase, got %#v %v", keywordOnly, err)
	}
	hits, err := search.SearchDocuments(ctx, query.TextSearchFilter{Query: "奖励函数设计", Scope: member})
	if err != nil || len(hits) != 1 || hits[0].DocumentID != paraphrase || hits[0].Match != "semantic" || hits[0].Similarity < 0.99 {
		t.Fatalf("hybrid search must find the paraphrase only: %#v %v", hits, err)
	}
	for _, h := range hits {
		if h.DocumentID == unrelated {
			t.Fatal("orthogonal document must be cut by the similarity floor")
		}
	}
	if out, _ := search.SearchDocuments(ctx, query.TextSearchFilter{Query: "奖励函数设计", Scope: query.ActorScope{ActorID: outsider}}); len(out) != 0 {
		t.Fatal("semantic search must respect project visibility")
	}
	chunks, err := search.SearchChunks(ctx, query.TextSearchFilter{Query: "奖励函数设计", ProjectID: integrationProjectID, Scope: query.ActorScope{IsAdmin: true}, Limit: 5})
	if err != nil || len(chunks) != 1 || chunks[0].Content == "" {
		t.Fatalf("assistant retrieval must return the semantic chunk with content: %#v %v", chunks, err)
	}

	// A model change makes every chunk stale again.
	if pending, _ := repo.ChunksNeedingEmbedding(ctx, "other-model", 10); len(pending) != 2 {
		t.Fatalf("model change must re-queue chunks, got %d", len(pending))
	}
}
