package main

import (
	"context"
	"encoding/json"
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
	"github.com/redis/go-redis/v9"
	"github.com/savor/order/internal/catalog"
	"github.com/savor/order/internal/config"
	"github.com/savor/order/internal/httpapi"
	"github.com/savor/order/internal/live"
	"github.com/savor/order/internal/mq"
	"github.com/savor/order/internal/store"
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

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Error("redis", "err", err)
		os.Exit(1)
	}
	rdb := redis.NewClient(opt)
	bus := live.New(rdb)
	st := store.New(pool)
	cat := catalog.New(cfg.CatalogURL)

	broker, err := mq.Dial(cfg.RabbitURL, log)
	if err != nil {
		log.Error("rabbit", "err", err)
		os.Exit(1)
	}

	go outboxLoop(ctx, st, broker, log)
	_ = broker.Consume("order.payment.events", "payment.*", func(body []byte) error {
		return onPayment(ctx, st, bus, body, log)
	})
	_ = broker.Consume("order.courier.assigned", "courier.assigned", func(body []byte) error {
		return onCourierAssigned(ctx, st, bus, body, log)
	})
	_ = broker.Consume("order.courier.progress", "courier.picked_up", func(body []byte) error {
		return onCourierProgress(ctx, st, bus, body, "picked_up", []string{"assigned"}, "order.picked_up", log)
	})
	_ = broker.Consume("order.courier.delivered", "courier.delivered", func(body []byte) error {
		return onCourierProgress(ctx, st, bus, body, "delivered", []string{"picked_up"}, "order.delivered", log)
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.New(st, cat, bus, log),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		log.Info("order listening", "addr", cfg.HTTPAddr)
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

func outboxLoop(ctx context.Context, st *store.Store, broker *mq.Conn, log *slog.Logger) {
	t := time.NewTicker(300 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			events, err := st.Unpublished(ctx, 50)
			if err != nil {
				log.Error("outbox read", "err", err)
				continue
			}
			for _, e := range events {
				var payload any
				_ = json.Unmarshal(e.Payload, &payload)
				if err := broker.Publish(ctx, e.EventType, payload); err != nil {
					log.Error("outbox publish", "err", err, "type", e.EventType)
					continue
				}
				_ = st.MarkPublished(ctx, e.ID)
			}
		}
	}
}

func onPayment(ctx context.Context, st *store.Store, bus *live.Bus, body []byte, log *slog.Logger) error {
	var ev struct {
		OrderID string `json:"orderId"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(body, &ev); err != nil {
		return err
	}
	id, err := uuid.Parse(ev.OrderID)
	if err != nil {
		return err
	}
	var o store.Order
	if ev.Status == "succeeded" || ev.Status == "payment.succeeded" {
		o, err = st.Transition(ctx, id, []string{"pending_payment"}, "paid", nil, "order.paid", nil)
	} else {
		o, err = st.Transition(ctx, id, []string{"pending_payment"}, "payment_failed", nil, "order.payment_failed", nil)
	}
	if err != nil {
		if errors.Is(err, store.ErrConflict) || errors.Is(err, store.ErrNotFound) {
			return nil
		}
		return err
	}
	publish(bus, o)
	return nil
}

func onCourierAssigned(ctx context.Context, st *store.Store, bus *live.Bus, body []byte, log *slog.Logger) error {
	var ev struct {
		OrderID     string `json:"orderId"`
		CourierID   string `json:"courierId"`
		CourierName string `json:"courierName"`
	}
	if err := json.Unmarshal(body, &ev); err != nil {
		return err
	}
	oid, err := uuid.Parse(ev.OrderID)
	if err != nil {
		return err
	}
	cid, err := uuid.Parse(ev.CourierID)
	if err != nil {
		return err
	}
	o, err := st.AssignCourier(ctx, oid, cid, ev.CourierName)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			return nil
		}
		return err
	}
	publish(bus, o)
	return nil
}

func onCourierProgress(ctx context.Context, st *store.Store, bus *live.Bus, body []byte, to string, from []string, event string, log *slog.Logger) error {
	var ev struct {
		OrderID string `json:"orderId"`
	}
	if err := json.Unmarshal(body, &ev); err != nil {
		return err
	}
	id, err := uuid.Parse(ev.OrderID)
	if err != nil {
		return err
	}
	o, err := st.Transition(ctx, id, from, to, nil, event, nil)
	if err != nil {
		if errors.Is(err, store.ErrConflict) || errors.Is(err, store.ErrNotFound) {
			return nil
		}
		return err
	}
	publish(bus, o)
	return nil
}

func publish(bus *live.Bus, o store.Order) {
	ids := []string{o.CustomerID.String(), o.RestaurantOwnerID.String()}
	if o.CourierID != nil {
		ids = append(ids, o.CourierID.String())
	}
	bus.Publish(context.Background(), ids, map[string]any{
		"type": "order.updated", "orderId": o.ID.String(), "status": o.Status,
	})
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
