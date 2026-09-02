package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPAddr        string
	ServiceName     string
	DatabaseURL     string
	JWTSecret       string
	MigrationsPath  string
	OTLPEndpoint    string
}

func Load() Config {
	return Config{
		HTTPAddr:       env("HTTP_ADDR", ":8001"),
		ServiceName:    env("SERVICE_NAME", "identity"),
		DatabaseURL:    must("DATABASE_URL"),
		JWTSecret:      must("JWT_SECRET"),
		MigrationsPath: env("MIGRATIONS_PATH", "migrations"),
		OTLPEndpoint:   env("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
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
