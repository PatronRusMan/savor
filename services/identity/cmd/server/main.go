package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/savor/identity/internal/auth"
	"github.com/savor/identity/internal/config"
	"github.com/savor/identity/internal/httpapi"
	"github.com/savor/identity/internal/store"
)

var (
	demoCustomer   = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	demoRestaurant = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	demoCourier    = uuid.MustParse("33333333-3333-3333-3333-333333333333")
	demoKitchenTwo = uuid.MustParse("44444444-4444-4444-4444-444444444444")
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
	if err := seed(ctx, st, log); err != nil {
		log.Error("seed", "err", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.New(st, cfg.JWTSecret, log),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("identity listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("listen", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
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

func seed(ctx context.Context, st *store.Store, log *slog.Logger) error {
	n, err := st.Count(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	hash, err := auth.HashPassword("savor1234")
	if err != nil {
		return err
	}
	users := []store.User{
		{ID: demoCustomer, Email: "customer@savor.dev", PasswordHash: hash, Name: "Nia Beridze", Role: "customer"},
		{ID: demoRestaurant, Email: "restaurant@savor.dev", PasswordHash: hash, Name: "Nari Kitchen", Role: "restaurant"},
		{ID: demoCourier, Email: "courier@savor.dev", PasswordHash: hash, Name: "Giorgi K.", Role: "courier"},
		{ID: demoKitchenTwo, Email: "kitchen@savor.dev", PasswordHash: hash, Name: "Ember Oven", Role: "restaurant"},
	}
	for _, u := range users {
		if err := st.CreateUser(ctx, u); err != nil && !errors.Is(err, store.ErrConflict) {
			return err
		}
	}
	log.Info("seeded demo users")
	return nil
}
