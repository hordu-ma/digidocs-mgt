package config

import "testing"

func TestLoadUsesDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("SYNOLOGY_PORT", "")

	cfg := Load()

	if cfg.AppName != "DigiDocs Mgt Go API" {
		t.Fatalf("unexpected app name: %s", cfg.AppName)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Fatalf("unexpected http addr: %s", cfg.HTTPAddr)
	}
	if cfg.SynologyPort != 5000 {
		t.Fatalf("unexpected synology port: %d", cfg.SynologyPort)
	}
	if cfg.SynologyHTTPS {
		t.Fatal("expected synology https default to be false")
	}
	if cfg.StorageBackend != "memory" {
		t.Fatalf("unexpected storage backend: %s", cfg.StorageBackend)
	}
}

func TestLoadUsesEnvironment(t *testing.T) {
	t.Setenv("APP_NAME", "custom")
	t.Setenv("HTTP_ADDR", ":18081")
	t.Setenv("APP_ENV", "development")
	t.Setenv("API_V1_PREFIX", "/api")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("DATA_BACKEND", "postgres")
	t.Setenv("WORKER_CALLBACK_TOKEN", "worker-token")
	t.Setenv("JWT_SECRET", "jwt-secret")
	t.Setenv("CORS_ALLOW_ORIGINS", "https://example.test")
	t.Setenv("STORAGE_BACKEND", "synology")
	t.Setenv("SYNOLOGY_HOST", "nas.local")
	t.Setenv("SYNOLOGY_PORT", "5001")
	t.Setenv("SYNOLOGY_HTTPS", "true")
	t.Setenv("SYNOLOGY_INSECURE_SKIP_VERIFY", "true")
	t.Setenv("SYNOLOGY_ACCOUNT", "user")
	t.Setenv("SYNOLOGY_PASSWORD", "pass")
	t.Setenv("SYNOLOGY_SHARE_PATH", "/Research")
	t.Setenv("CODE_REPO_ROOT", "/srv/repos")

	cfg := Load()

	if cfg.AppName != "custom" || cfg.HTTPAddr != ":18081" || cfg.APIV1Prefix != "/api" {
		t.Fatalf("env values not applied: %+v", cfg)
	}
	if cfg.DatabaseURL != "postgres://example" || cfg.DataBackend != "postgres" {
		t.Fatalf("database env values not applied: %+v", cfg)
	}
	if cfg.WorkerCallbackToken != "worker-token" || cfg.JWTSecret != "jwt-secret" {
		t.Fatalf("secret env values not applied: %+v", cfg)
	}
	if cfg.CORSAllowOrigins != "https://example.test" {
		t.Fatalf("unexpected cors origins: %s", cfg.CORSAllowOrigins)
	}
	if cfg.StorageBackend != "synology" || cfg.SynologyHost != "nas.local" || cfg.SynologyPort != 5001 {
		t.Fatalf("synology env values not applied: %+v", cfg)
	}
	if !cfg.SynologyHTTPS || !cfg.SynologyInsecureSkipVerify {
		t.Fatalf("synology bool env values not applied: %+v", cfg)
	}
	if cfg.SynologyAccount != "user" || cfg.SynologyPassword != "pass" || cfg.SynologySharePath != "/Research" {
		t.Fatalf("synology credential env values not applied: %+v", cfg)
	}
	if cfg.CodeRepoRoot != "/srv/repos" {
		t.Fatalf("unexpected code repo root: %s", cfg.CodeRepoRoot)
	}
}

func TestLoadInvalidSynologyPortFallsBackToZero(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("SYNOLOGY_PORT", "not-a-number")

	cfg := Load()

	if cfg.SynologyPort != 0 {
		t.Fatalf("expected invalid port to parse as zero, got %d", cfg.SynologyPort)
	}
}
