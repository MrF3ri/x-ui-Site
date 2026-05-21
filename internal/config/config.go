package config

import "os"

type Config struct {
	AppPort    string
	ServerMode string
}

func Load() Config {
	return Config{
		AppPort:    env("APP_PORT", "8080"),
		ServerMode: env("SERVER_MODE", "nethttp"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
