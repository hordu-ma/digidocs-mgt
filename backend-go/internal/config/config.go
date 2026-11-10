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
	StorageBackend             string
	SynologyHost               string
	SynologyPort               int
	SynologyHTTPS              bool
	SynologyInsecureSkipVerify bool
	SynologyAccount            string
	SynologyPassword           string
	SynologySharePath          string // shared folder path, e.g. "/DigiDocs"

	CodeRepoRoot string

	// LoginRateLimitPerMin caps login attempts per client IP per minute (0 disables).
	LoginRateLimitPerMin int
	// RegisterRateLimitPerMin caps registration attempts per client IP per minute (0 disables).
	RegisterRateLimitPerMin int
	// TrustedProxyDepth is how many trailing X-Forwarded-For entries were added
	// by reverse proxies under our control, counted from the end. 1 means only
	// the frontend nginx. Raising it trusts one more entry of client-supplied
	// data, so it must match the real proxy chain — see middleware.clientIP.
	TrustedProxyDepth int

	// Notifications (2026-11-10). In-app notifications are always on; the
	// external channels below are optional and off when left empty.
	PublicBaseURL     string // e.g. https://docs.example.org, used for links in external messages
	NotifyWebhookURL  string // group robot URL (contains its key; never log it)
	NotifyWebhookKind string // wecom | dingtalk | generic
	SMTPHost          string
	SMTPPort          int
	SMTPUsername      string
	SMTPPassword      string
	SMTPFrom          string

	// AutoSummarizeOnUpload chains an AI summary after upload-time text
	// extraction succeeds (2026-11-10, default on).
	AutoSummarizeOnUpload bool

	// SelfRegistration controls POST /auth/register: open (default),
	// invite_only (a valid invite token is required) or closed.
	SelfRegistration string
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

		StorageBackend:             getEnv("STORAGE_BACKEND", "memory"),
		SynologyHost:               getEnv("SYNOLOGY_HOST", ""),
		SynologyPort:               synoPort,
		SynologyHTTPS:              getEnv("SYNOLOGY_HTTPS", "false") == "true",
		SynologyInsecureSkipVerify: getEnv("SYNOLOGY_INSECURE_SKIP_VERIFY", "false") == "true",
		SynologyAccount:            getEnv("SYNOLOGY_ACCOUNT", ""),
		SynologyPassword:           getEnv("SYNOLOGY_PASSWORD", ""),
		SynologySharePath:          getEnv("SYNOLOGY_SHARE_PATH", "/DigiDocs"),
		CodeRepoRoot:               getEnv("CODE_REPO_ROOT", "/tmp/digidocs-code-repos"),
		LoginRateLimitPerMin:       getEnvInt("LOGIN_RATE_LIMIT_PER_MIN", 10),
		TrustedProxyDepth:          getEnvInt("TRUSTED_PROXY_DEPTH", 1),
		RegisterRateLimitPerMin:    getEnvInt("REGISTER_RATE_LIMIT_PER_MIN", 5),
		PublicBaseURL:              getEnv("PUBLIC_BASE_URL", ""),
		NotifyWebhookURL:           getEnv("NOTIFY_WEBHOOK_URL", ""),
		NotifyWebhookKind:          getEnv("NOTIFY_WEBHOOK_KIND", "generic"),
		SMTPHost:                   getEnv("SMTP_HOST", ""),
		SMTPPort:                   getEnvInt("SMTP_PORT", 587),
		SMTPUsername:               getEnv("SMTP_USERNAME", ""),
		SMTPPassword:               getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:                   getEnv("SMTP_FROM", ""),
		AutoSummarizeOnUpload:      getEnv("AUTO_SUMMARIZE_ON_UPLOAD", "true") == "true",
		SelfRegistration:           getEnv("SELF_REGISTRATION", "open"),
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

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
