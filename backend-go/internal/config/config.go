package config

import "os"

type Config struct {
	AppName             string
	HTTPAddr            string
	AppEnv              string
	APIV1Prefix         string
	DatabaseURL         string
	DataBackend         string
	WorkerCallbackToken string
	JWTSecret           string
}

func Load() Config {
	return Config{
		AppName:             getEnv("APP_NAME", "DigiDocs Mgt Go API"),
		HTTPAddr:            getEnv("HTTP_ADDR", ":8080"),
		AppEnv:              getEnv("APP_ENV", "development"),
		APIV1Prefix:         getEnv("API_V1_PREFIX", "/api/v1"),
		DatabaseURL:         getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:15432/digidocs_mgt?sslmode=disable"),
		DataBackend:         getEnv("DATA_BACKEND", "memory"),
		WorkerCallbackToken: getEnv("WORKER_CALLBACK_TOKEN", "replace-me"),
		JWTSecret:           getEnv("JWT_SECRET", "dev-secret"),
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
