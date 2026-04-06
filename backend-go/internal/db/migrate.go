package db

import (
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
		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
		log.Printf("[migrate] applied %s", name)
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
