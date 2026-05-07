package postgres_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"digidocs-mgt/backend-go/internal/db"
	"digidocs-mgt/backend-go/internal/domain/command"
	"digidocs-mgt/backend-go/internal/domain/query"
	pgrepo "digidocs-mgt/backend-go/internal/repository/postgres"
)

const (
	integrationUserID      = "00000000-0000-0000-0000-000000000101"
	integrationTeamSpaceID = "10000000-0000-0000-0000-000000000101"
	integrationProjectID   = "20000000-0000-0000-0000-000000000101"
)

func TestPostgresIntegrationMigrationsAndRepositoryFlows(t *testing.T) {
	ctx := context.Background()
	database := openIntegrationDB(t)
	seedIntegrationBaseRows(t, ctx, database)

	documents := pgrepo.NewDocumentRepository(database)
	createdDoc, err := documents.CreateDocument(ctx, command.DocumentCreateInput{
		TeamSpaceID:    integrationTeamSpaceID,
		ProjectID:      integrationProjectID,
		Title:          "集成测试文档",
		Description:    "真实 PostgreSQL 集成测试",
		CurrentOwnerID: integrationUserID,
		ActorID:        integrationUserID,
	})
	if err != nil {
		t.Fatalf("CreateDocument: %v", err)
	}
	documentID, _ := createdDoc["id"].(string)
	if documentID == "" {
		t.Fatalf("created document id is empty: %#v", createdDoc)
	}

	versionWorkflow := pgrepo.NewVersionWorkflow(database)
	createdVersion, err := versionWorkflow.CreateUploadedVersion(ctx, command.VersionCreateInput{
		DocumentID:       documentID,
		FileName:         "integration.txt",
		FileSize:         128,
		CommitMessage:    "integration upload",
		StorageObjectKey: "integration/documents/integration.txt",
		StorageProvider:  "memory",
		ActorID:          integrationUserID,
	})
	if err != nil {
		t.Fatalf("CreateUploadedVersion: %v", err)
	}
	if createdVersion["version_no"] != 1 || createdVersion["current_status"] != "in_progress" {
		t.Fatalf("created version = %#v, want version_no=1 current_status=in_progress", createdVersion)
	}

	listedDocs, total, err := documents.ListDocuments(ctx, query.DocumentListFilter{
		ProjectID: integrationProjectID,
		Page:      1,
		PageSize:  10,
	})
	if err != nil {
		t.Fatalf("ListDocuments: %v", err)
	}
	if total != 1 || len(listedDocs) != 1 || listedDocs[0].ID != documentID || listedDocs[0].CurrentVersionNo == nil || *listedDocs[0].CurrentVersionNo != 1 {
		t.Fatalf("ListDocuments items=%#v total=%d, want created doc with version 1", listedDocs, total)
	}

	versionRepo := pgrepo.NewVersionRepository(database)
	versions, err := versionRepo.ListVersions(ctx, documentID)
	if err != nil {
		t.Fatalf("ListVersions: %v", err)
	}
	if len(versions) != 1 || versions[0].FileName != "integration.txt" {
		t.Fatalf("ListVersions = %#v, want uploaded version", versions)
	}
	versionDetail, err := versionRepo.GetVersion(ctx, createdVersion["id"].(string))
	if err != nil {
		t.Fatalf("GetVersion: %v", err)
	}
	if versionDetail.DocumentID != documentID || versionDetail.StorageObjectKey != "integration/documents/integration.txt" {
		t.Fatalf("GetVersion = %#v, want document and object key", versionDetail)
	}

	dataAssets := pgrepo.NewDataAssetRepository(database)
	folder, err := dataAssets.CreateDataFolder(ctx, command.DataFolderCreateInput{
		ProjectID: integrationProjectID,
		Name:      "集成数据",
		ActorID:   integrationUserID,
	})
	if err != nil {
		t.Fatalf("CreateDataFolder: %v", err)
	}
	asset, err := dataAssets.CreateDataAsset(ctx, command.DataAssetCreateInput{
		TeamSpaceID:      integrationTeamSpaceID,
		ProjectID:        integrationProjectID,
		FolderID:         folder.ID,
		DisplayName:      "集成数据集",
		FileName:         "dataset.csv",
		FileSize:         64,
		StorageProvider:  "memory",
		StorageObjectKey: "integration/data/dataset.csv",
		ActorID:          integrationUserID,
	})
	if err != nil {
		t.Fatalf("CreateDataAsset: %v", err)
	}
	assets, assetTotal, err := dataAssets.ListDataAssets(ctx, query.DataAssetListFilter{
		ProjectID: integrationProjectID,
		FolderID:  folder.ID,
		Page:      1,
		PageSize:  10,
	})
	if err != nil {
		t.Fatalf("ListDataAssets: %v", err)
	}
	if assetTotal != 1 || len(assets) != 1 || assets[0].ID != asset["id"] {
		t.Fatalf("ListDataAssets items=%#v total=%d, want created asset", assets, assetTotal)
	}
}

func openIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("DIGIDOCS_POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("set DIGIDOCS_POSTGRES_TEST_DSN to run PostgreSQL integration tests")
	}

	database, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := database.PingContext(ctx); err != nil {
		t.Fatalf("PingContext: %v", err)
	}

	schema := fmt.Sprintf("digidocs_it_%d", time.Now().UnixNano())
	if _, err := database.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = database.Exec(`DROP SCHEMA IF EXISTS ` + schema + ` CASCADE`)
	})
	if _, err := database.ExecContext(ctx, `SET search_path TO `+schema+`, public`); err != nil {
		t.Fatalf("set search_path: %v", err)
	}

	migrationsDir, err := filepath.Abs("../../../migrations")
	if err != nil {
		t.Fatalf("resolve migrations dir: %v", err)
	}
	if err := db.RunMigrations(database, migrationsDir); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	var migrationCount int
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations`).Scan(&migrationCount); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	if migrationCount < 8 {
		t.Fatalf("schema_migrations count = %d, want at least 8", migrationCount)
	}
	return database
}

func seedIntegrationBaseRows(t *testing.T, ctx context.Context, database *sql.DB) {
	t.Helper()

	if _, err := database.ExecContext(ctx, `
		INSERT INTO users (id, username, password_hash, display_name, role, status)
		VALUES ($1::uuid, 'integration-user', 'hash', '集成测试用户', 'admin'::user_role, 'active')
	`, integrationUserID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO team_spaces (id, name, code, description, created_by)
		VALUES ($1::uuid, '集成测试团队', 'integration-team', 'integration', $2::uuid)
	`, integrationTeamSpaceID, integrationUserID); err != nil {
		t.Fatalf("seed team space: %v", err)
	}
	if _, err := database.ExecContext(ctx, `
		INSERT INTO projects (id, team_space_id, name, code, owner_id)
		VALUES ($1::uuid, $2::uuid, '集成测试课题', 'integration-project', $3::uuid)
	`, integrationProjectID, integrationTeamSpaceID, integrationUserID); err != nil {
		t.Fatalf("seed project: %v", err)
	}
}
