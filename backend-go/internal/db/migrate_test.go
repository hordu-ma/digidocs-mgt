package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/lib/pq"
)

func TestExtractVersion(t *testing.T) {
	cases := []struct {
		name string
		want int
	}{
		{name: "001_initial_schema.sql", want: 1},
		{name: "012.sql", want: 12},
		{name: "20260430_add_code_repositories.sql", want: 20260430},
		{name: "no_version.sql", want: -1},
		{name: "_missing.sql", want: -1},
	}

	for _, tc := range cases {
		if got := extractVersion(tc.name); got != tc.want {
			t.Fatalf("extractVersion(%q) = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestRunMigrationsAppliesPendingFilesAndSkipsAppliedVersions(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "001_initial.sql", "CREATE TABLE things (id integer); INSERT INTO schema_migrations (version, name) VALUES (1, '001_initial');")
	writeMigration(t, dir, "002_skip.sql", "CREATE TABLE skipped (id integer)")
	writeMigration(t, dir, "not_a_migration.sql", "SELECT 1")

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectExec("CREATE TABLE things").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(2).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	if err := RunMigrations(db, dir); err != nil {
		t.Fatalf("RunMigrations unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestRunMigrationsReturnsContextualErrors(t *testing.T) {
	dir := t.TempDir()
	writeMigration(t, dir, "001_initial.sql", "CREATE TABLE things (id integer)")

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
		WillReturnError(errors.New("boom"))
	if err := RunMigrations(db, dir); err == nil || !strings.Contains(err.Error(), "create schema_migrations table") {
		t.Fatalf("RunMigrations create table err=%v, want contextual error", err)
	}

	db2, mock2, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { db2.Close() })
	mock2.ExpectExec("CREATE TABLE IF NOT EXISTS schema_migrations").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock2.ExpectQuery("SELECT EXISTS").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock2.ExpectExec("CREATE TABLE things").
		WillReturnError(errors.New("apply failed"))
	mock2.ExpectExec("ROLLBACK").
		WillReturnResult(sqlmock.NewResult(0, 0))
	if err := RunMigrations(db2, dir); err == nil || !strings.Contains(err.Error(), "apply 001_initial.sql") {
		t.Fatalf("RunMigrations apply err=%v, want contextual error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
	if err := mock2.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

// TestExecMigrationRollsBackConnectionOnFailure reproduces, against a real
// PostgreSQL instance, the exact failure mode that motivated execMigration's
// explicit ROLLBACK: a migration script that fails partway through leaves the
// connection's transaction aborted (PostgreSQL's simple query protocol stops
// processing the rest of the string on the first error, so the script's own
// trailing COMMIT never runs). Without the ROLLBACK, database/sql would
// return that connection to the pool still stuck in "current transaction is
// aborted, commands ignored until end of transaction block" and hand it out
// to unrelated queries later. sqlmock cannot exercise this because it does
// not model real transaction-abort semantics.
func TestExecMigrationRollsBackConnectionOnFailure(t *testing.T) {
	dsn := os.Getenv("DIGIDOCS_POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("set DIGIDOCS_POSTGRES_TEST_DSN to run this PostgreSQL integration test")
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

	schema := fmt.Sprintf("digidocs_migrate_test_%d", time.Now().UnixNano())
	if _, err := database.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = database.Exec(`DROP SCHEMA IF EXISTS ` + schema + ` CASCADE`)
	})
	if _, err := database.ExecContext(ctx, `SET search_path TO `+schema+`, public`); err != nil {
		t.Fatalf("set search_path: %v", err)
	}

	failingScript := `
BEGIN;
CREATE TABLE t1 (id int);
SELECT * FROM this_table_does_not_exist;
COMMIT;
`
	if err := execMigration(database, failingScript); err == nil {
		t.Fatal("execMigration on failing script: want error, got nil")
	}

	// The bug this guards against: without the explicit ROLLBACK, this
	// follow-up query (same pooled connection, MaxOpenConns=1 forces reuse)
	// would fail with "current transaction is aborted".
	var one int
	if err := database.QueryRowContext(ctx, "SELECT 1").Scan(&one); err != nil {
		t.Fatalf("follow-up query after failed migration: %v (connection was left poisoned)", err)
	}
	if one != 1 {
		t.Fatalf("follow-up query = %d, want 1", one)
	}

	// The transactional DDL before the failure point must not have persisted.
	var exists bool
	if err := database.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = $1 AND table_name = 't1')`,
		schema,
	).Scan(&exists); err != nil {
		t.Fatalf("check t1 exists: %v", err)
	}
	if exists {
		t.Fatal("t1 exists = true, want false (failed migration must not partially apply)")
	}
}

func writeMigration(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("write migration %s: %v", name, err)
	}
}
