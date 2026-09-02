package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPAddr       string
	ServiceName    string
	DatabaseURL    string
	RabbitURL      string
	MigrationsPath string
}

func Load() Config {
	return Config{
		HTTPAddr:       env("HTTP_ADDR", ":8004"),
		ServiceName:    env("SERVICE_NAME", "payment"),
		DatabaseURL:    must("DATABASE_URL"),
		RabbitURL:      must("RABBITMQ_URL"),
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
