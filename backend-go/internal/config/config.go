package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	AppName             string
	HTTPAddr            string
	AppEnv              string
	APIV1Prefix         string
	DatabaseURL         string
	DataBackend         string
	WorkerCallbackToken string
	JWTSecret           string
	CORSAllowOrigins    string

	// Synology DSM connection (used when STORAGE_BACKEND=synology)
	StorageBackend    string
	SynologyHost      string
	SynologyPort      int
	SynologyHTTPS     bool
	SynologyInsecureSkipVerify bool
	SynologyAccount   string
	SynologyPassword  string
	SynologySharePath string // shared folder path, e.g. "/DigiDocs"
}

func Load() Config {
	synoPort, _ := strconv.Atoi(getEnv("SYNOLOGY_PORT", "5000"))

	cfg := Config{
		AppName:             getEnv("APP_NAME", "DigiDocs Mgt Go API"),
		HTTPAddr:            getEnv("HTTP_ADDR", ":8080"),
		AppEnv:              getEnv("APP_ENV", "development"),
		APIV1Prefix:         getEnv("API_V1_PREFIX", "/api/v1"),
		DatabaseURL:         getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:15432/digidocs_mgt?sslmode=disable"),
		DataBackend:         getEnv("DATA_BACKEND", "memory"),
		WorkerCallbackToken: getEnv("WORKER_CALLBACK_TOKEN", "replace-me"),
		JWTSecret:           getEnv("JWT_SECRET", "dev-secret"),
		CORSAllowOrigins:    getEnv("CORS_ALLOW_ORIGINS", "*"),

		StorageBackend:    getEnv("STORAGE_BACKEND", "memory"),
		SynologyHost:      getEnv("SYNOLOGY_HOST", ""),
		SynologyPort:      synoPort,
		SynologyHTTPS:     getEnv("SYNOLOGY_HTTPS", "false") == "true",
		SynologyInsecureSkipVerify: getEnv("SYNOLOGY_INSECURE_SKIP_VERIFY", "false") == "true",
		SynologyAccount:   getEnv("SYNOLOGY_ACCOUNT", ""),
		SynologyPassword:  getEnv("SYNOLOGY_PASSWORD", ""),
		SynologySharePath: getEnv("SYNOLOGY_SHARE_PATH", "/DigiDocs"),
	}

	if cfg.AppEnv == "production" {
		if cfg.JWTSecret == "dev-secret" {
			log.Fatal("FATAL: JWT_SECRET must be set to a secure value in production")
		}
		if cfg.WorkerCallbackToken == "replace-me" {
			log.Fatal("FATAL: WORKER_CALLBACK_TOKEN must be set to a secure value in production")
		}
		if cfg.CORSAllowOrigins == "*" {
			log.Fatal("FATAL: CORS_ALLOW_ORIGINS must not be '*' in production")
		}
	}

	return cfg
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
