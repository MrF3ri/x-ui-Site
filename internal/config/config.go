package config

import "os"

type Config struct {
	AppPort   string
	ServerMode string
	DatabaseDSN string
	JWTSecret string
}

func Load() Config {
	return Config{
		AppPort: env("APP_PORT", "8080"),
		ServerMode: env("SERVER_MODE", "nethttp"),
		DatabaseDSN: env("DATABASE_DSN", "postgres://garuda:garuda@postgres:5432/garudapanel?sslmode=disable"),
		JWTSecret: env("JWT_SECRET", "change-me"),
	}
}

func env(key, fallback string) string { if v := os.Getenv(key); v != "" { return v }; return fallback }
