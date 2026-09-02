package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/savor/payment/internal/config"
	"github.com/savor/payment/internal/mq"
	"github.com/savor/payment/internal/store"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := config.Load()
	if err := runMigrations(cfg.DatabaseURL, cfg.MigrationsPath); err != nil {
		log.Error("migrate", "err", err)
		os.Exit(1)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()
	st := store.New(pool)
	broker, err := mq.Dial(cfg.RabbitURL, log)
	if err != nil {
		log.Error("rabbit", "err", err)
		os.Exit(1)
	}

	_ = broker.Consume("payment.order.created", "order.created", func(body []byte) error {
		var ev struct {
			OrderID    string `json:"orderId"`
			TotalCents int    `json:"totalCents"`
			Address    string `json:"address"`
		}
		if err := json.Unmarshal(body, &ev); err != nil {
			return err
		}
		oid, err := uuid.Parse(ev.OrderID)
		if err != nil {
			return err
		}
		time.Sleep(600 * time.Millisecond)
		fail := strings.Contains(strings.ToUpper(ev.Address), "FAIL")
		p, created, err := st.Charge(ctx, oid, ev.TotalCents, fail)
		if err != nil {
			return err
		}
		if !created && p.Status != "processing" {
			return nil
		}
		etype := "payment.succeeded"
		status := "succeeded"
		if p.Status == "failed" {
			etype = "payment.failed"
			status = "failed"
		}
		return broker.Publish(ctx, etype, map[string]any{
			"paymentId":   p.ID.String(),
			"orderId":     p.OrderID.String(),
			"amountCents": p.AmountCents,
			"status":      status,
		})
	})

	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	r.Get("/ready", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})
	r.Handle("/metrics", promhttp.Handler())
	r.Get("/payments/{id}", func(w http.ResponseWriter, r *http.Request) {
		p, err := st.Get(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, `{"error":{"code":"not_found","message":"payment not found"}}`, 404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": p.ID.String(), "orderId": p.OrderID.String(), "amountCents": p.AmountCents,
			"status": p.Status, "createdAt": p.CreatedAt.UTC().Format(time.RFC3339),
		})
	})

	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: r, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		log.Info("payment listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("listen", "err", err)
			os.Exit(1)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
}

func runMigrations(dbURL, path string) error {
	m, err := migrate.New("file://"+path, dbURL)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}
