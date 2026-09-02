package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPAddr       string
	ServiceName    string
	DatabaseURL    string
	RedisURL       string
	RabbitURL      string
	CatalogURL     string
	MigrationsPath string
}

func Load() Config {
	return Config{
		HTTPAddr:       env("HTTP_ADDR", ":8003"),
		ServiceName:    env("SERVICE_NAME", "order"),
		DatabaseURL:    must("DATABASE_URL"),
		RedisURL:       must("REDIS_URL"),
		RabbitURL:      must("RABBITMQ_URL"),
		CatalogURL:     must("CATALOG_URL"),
		MigrationsPath: env("MIGRATIONS_PATH", "migrations"),
	}
}

func env(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func must(k string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		panic("missing required env " + k)
	}
	return v
}
