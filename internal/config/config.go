package config

import "os"

type Config struct {
	AppPort, DBDSN, RedisAddr, RedisPass, JWTSecret, MinioEndpoint, MinioKey, MinioSecret string
	RedisDB                                                                                int
}

func Load() Config {
	return Config{
		AppPort:       env("APP_PORT", "8080"),
		DBDSN:         env("DATABASE_DSN", "postgres://garuda:garuda@postgres:5432/garudapanel?sslmode=disable"),
		RedisAddr:     env("REDIS_ADDR", "redis:6379"),
		RedisPass:     env("REDIS_PASSWORD", ""),
		JWTSecret:     env("JWT_SECRET", "change-me"),
		MinioEndpoint: env("MINIO_ENDPOINT", "minio:9000"),
		MinioKey:      env("MINIO_ROOT_USER", "minioadmin"),
		MinioSecret:   env("MINIO_ROOT_PASSWORD", "minioadmin"),
	}
}

func env(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }
