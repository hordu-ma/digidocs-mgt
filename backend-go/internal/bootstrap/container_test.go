package bootstrap

import (
	"strings"
	"testing"

	"digidocs-mgt/backend-go/internal/config"
	"digidocs-mgt/backend-go/internal/domain/auth"
	memstorage "digidocs-mgt/backend-go/internal/storage/memory"
	synostorage "digidocs-mgt/backend-go/internal/storage/synology"
)

func TestBuildContainerMemoryBackend(t *testing.T) {
	cfg := config.Config{
		DataBackend:         "memory",
		StorageBackend:      "memory",
		JWTSecret:           "secret",
		WorkerCallbackToken: "worker",
		CodeRepoRoot:        t.TempDir(),
		APIV1Prefix:         "/api/v1",
	}

	container, err := BuildContainer(cfg)
	if err != nil {
		t.Fatalf("build container: %v", err)
	}
	if container.QueueConsumer == nil {
		t.Fatal("expected queue consumer")
	}
	if _, err := container.TokenService.Generate(testClaims()); err != nil {
		t.Fatalf("token service not wired: %v", err)
	}
}

func TestBuildContainerPostgresBackendWiresRepositories(t *testing.T) {
	cfg := config.Config{
		DataBackend:         "postgres",
		StorageBackend:      "memory",
		DatabaseURL:         "postgres://bad-host.invalid:5432/db?sslmode=disable",
		JWTSecret:           "secret",
		WorkerCallbackToken: "worker",
	}

	// Migrations are now a hard startup dependency: an unreachable database
	// fails the build instead of yielding a container that would serve traffic
	// against an unmigrated schema. sql.Open itself is lazy, so this error
	// surfaces from the migration step, which is exactly the point.
	container, err := BuildContainer(cfg)
	if err == nil {
		if container.DB != nil {
			_ = container.DB.Close()
		}
		t.Fatal("expected BuildContainer to fail when migrations cannot be applied")
	}
	if !strings.Contains(err.Error(), "run migrations") {
		t.Fatalf("err = %v, want it to name the migration step", err)
	}
}

func TestFindMigrationsDir(t *testing.T) {
	dir := findMigrationsDir()
	if dir == "" {
		t.Fatal("expected migrations dir to be found")
	}
	if !strings.HasSuffix(dir, "migrations") {
		t.Fatalf("unexpected migrations dir: %s", dir)
	}
}

func TestBuildStorageProvider(t *testing.T) {
	mem := buildStorageProvider(config.Config{StorageBackend: "memory"})
	if _, ok := mem.(*memstorage.Provider); !ok {
		t.Fatalf("expected memory provider, got %T", mem)
	}

	syno := buildStorageProvider(config.Config{
		StorageBackend:    "synology",
		SynologyHost:      "nas.local",
		SynologyPort:      5001,
		SynologyHTTPS:     true,
		SynologySharePath: "/DigiDocs",
	})
	if _, ok := syno.(*synostorage.Provider); !ok {
		t.Fatalf("expected synology provider, got %T", syno)
	}
}

func testClaims() auth.Claims {
	return auth.Claims{UserID: "user-1", Username: "user", Role: "admin"}
}
