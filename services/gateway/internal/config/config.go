package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPAddr         string
	ServiceName      string
	JWTSecret        string
	RedisURL         string
	IdentityURL      string
	CatalogURL       string
	OrderURL         string
	PaymentURL       string
	CourierURL       string
	NotificationURL  string
}

func Load() Config {
	return Config{
		HTTPAddr:        env("HTTP_ADDR", ":8000"),
		ServiceName:     env("SERVICE_NAME", "gateway"),
		JWTSecret:       must("JWT_SECRET"),
		RedisURL:        env("REDIS_URL", "redis://redis:6379"),
		IdentityURL:     must("IDENTITY_URL"),
		CatalogURL:      must("CATALOG_URL"),
		OrderURL:        must("ORDER_URL"),
		PaymentURL:      must("PAYMENT_URL"),
		CourierURL:      must("COURIER_URL"),
		NotificationURL: must("NOTIFICATION_URL"),
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
