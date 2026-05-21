package config

import (
	"errors"
	"os"
)

type Config struct {
	AppPort       string
	AppEnv        string
	DatabaseDSN   string
	JWTSecret     string
	RedisAddr     string
	MinIOEndpoint string
}

func Load() (Config, error) {
	cfg := Config{
		AppPort:       env("APP_PORT", "8080"),
		AppEnv:        env("APP_ENV", "development"),
		DatabaseDSN:   os.Getenv("DATABASE_DSN"),
		JWTSecret:     os.Getenv("JWT_SECRET"),
		RedisAddr:     env("REDIS_ADDR", "redis:6379"),
		MinIOEndpoint: env("MINIO_ENDPOINT", "minio:9000"),
	}
	if cfg.DatabaseDSN == "" {
		return Config{}, errors.New("missing required env DATABASE_DSN")
	}
	if cfg.JWTSecret == "" {
		return Config{}, errors.New("missing required env JWT_SECRET")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
