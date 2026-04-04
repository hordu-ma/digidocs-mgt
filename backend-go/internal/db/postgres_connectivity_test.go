package db

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

func TestPostgresConnectivity(t *testing.T) {
	t.Helper()

	if os.Getenv("ENABLE_PG_INTEGRATION") != "1" {
		t.Skip("set ENABLE_PG_INTEGRATION=1 to run postgres connectivity test")
	}

	db, err := sql.Open("postgres", "postgres://postgres:postgres@127.0.0.1:15432/digidocs_mgt?sslmode=disable")
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	defer db.Close()

	var n int
	if err := db.QueryRow("select 1").Scan(&n); err != nil {
		t.Fatalf("query postgres: %v", err)
	}

	if n != 1 {
		t.Fatalf("unexpected result: %d", n)
	}
}
