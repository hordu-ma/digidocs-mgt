package db

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
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

func writeMigration(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("write migration %s: %v", name, err)
	}
}
