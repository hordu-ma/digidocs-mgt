package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RunMigrations executes all pending SQL migration files from the migrations directory.
// It uses a schema_migrations table to track which migrations have been applied.
func RunMigrations(db *sql.DB, migrationsDir string) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version  INTEGER PRIMARY KEY,
			name     VARCHAR(255) NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("glob migrations: %w", err)
	}
	sort.Strings(files)

	for _, f := range files {
		name := filepath.Base(f)
		version := extractVersion(name)
		if version < 0 {
			log.Printf("[migrate] skip %s: cannot parse version", name)
			continue
		}

		var exists bool
		if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&exists); err != nil {
			return fmt.Errorf("check migration %d: %w", version, err)
		}
		if exists {
			continue
		}

		content, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}

		log.Printf("[migrate] applying %s ...", name)
		if err := execMigration(db, string(content)); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
		log.Printf("[migrate] applied %s", name)
	}

	return nil
}

// execMigration runs a migration script (which wraps itself in an explicit
// BEGIN/COMMIT) on a single dedicated connection.
//
// This matters because of how PostgreSQL's simple query protocol handles a
// multi-statement string: if any statement fails, the server stops executing
// the rest of the string — including the migration file's own trailing
// COMMIT — and leaves the session's transaction in an aborted state. Verified
// against a real PostgreSQL 17 instance: a failed statement partway through
// such a script left the connection permanently returning "current
// transaction is aborted, commands ignored until end of transaction block"
// for every subsequent query on it. Since database/sql pools connections,
// that one poisoned connection would keep getting handed out to unrelated
// requests (intermittently, depending on pool size) until it happened to be
// cycled out by ConnMaxLifetime. Explicitly issuing ROLLBACK on the same
// connection before releasing it back to the pool prevents that.
func execMigration(db *sql.DB, script string) error {
	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, script); err != nil {
		if _, rbErr := conn.ExecContext(ctx, "ROLLBACK"); rbErr != nil {
			log.Printf("[migrate] rollback after failed migration also failed: %v", rbErr)
		}
		return err
	}
	return nil
}

// extractVersion parses the leading integer from a filename like "001_initial_schema.sql".
func extractVersion(name string) int {
	parts := strings.SplitN(name, "_", 2)
	if len(parts) == 0 {
		return -1
	}
	var v int
	if _, err := fmt.Sscanf(parts[0], "%d", &v); err != nil {
		return -1
	}
	return v
}
