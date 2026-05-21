package config

import "os"

type Config struct {
	AppPort string
	ServerMode string
	DatabaseDSN string
	JWTSecret string
	XUIBaseURL string
	XUIAPIToken string
	MinIOEndpoint string
	MinIOBucket string
	MinIOAccessKey string
	MinIOSecretKey string
}

func Load() Config {
	return Config{
		AppPort: env("APP_PORT", "8080"),
		ServerMode: env("SERVER_MODE", "nethttp"),
		DatabaseDSN: env("DATABASE_DSN", "postgres://garuda:garuda@postgres:5432/garudapanel?sslmode=disable"),
		JWTSecret: env("JWT_SECRET", "change-me"),
		XUIBaseURL: env("XUI_BASE_URL", "http://localhost:2053"),
		XUIAPIToken: env("XUI_API_TOKEN", ""),
		MinIOEndpoint: env("MINIO_ENDPOINT", "http://minio:9000"),
		MinIOBucket: env("MINIO_BUCKET", "garuda"),
		MinIOAccessKey: env("MINIO_ROOT_USER", "minioadmin"),
		MinIOSecretKey: env("MINIO_ROOT_PASSWORD", "minioadmin123"),
	}
}
func env(key, fallback string) string { if v := os.Getenv(key); v != "" { return v }; return fallback }
