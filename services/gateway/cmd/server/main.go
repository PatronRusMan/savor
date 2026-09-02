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
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/savor/gateway/internal/auth"
	"github.com/savor/gateway/internal/config"
	"github.com/savor/gateway/internal/metrics"
	"github.com/savor/gateway/internal/proxy"
	"github.com/savor/gateway/internal/ws"
	"golang.org/x/time/rate"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := config.Load()

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Error("redis url", "err", err)
		os.Exit(1)
	}
	rdb := redis.NewClient(opt)

	identity := proxy.StripAPI(proxy.New(cfg.IdentityURL))
	catalog := proxy.StripAPI(proxy.New(cfg.CatalogURL))
	order := proxy.StripAPI(proxy.New(cfg.OrderURL))
	payment := proxy.StripAPI(proxy.New(cfg.PaymentURL))
	courier := proxy.StripAPI(proxy.New(cfg.CourierURL))
	notification := proxy.StripAPI(proxy.New(cfg.NotificationURL))

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(metrics.Middleware(cfg.ServiceName))
	r.Use(cors.Handler(cors.Options{
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-Id"},
		AllowCredentials: true,
		AllowOriginFunc: func(r *http.Request, origin string) bool {
			return strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "http://127.0.0.1")
		},
	}))
	r.Use(rateLimit())

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"gateway"}`))
	})
	r.Get("/ready", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})
	r.Handle("/metrics", promhttp.Handler())
	r.Get("/ws", ws.Handler(rdb, cfg.JWTSecret, log))

	r.Group(func(r chi.Router) {
		r.Use(optionalAuth(cfg.JWTSecret))
		r.Handle("/api/v1/auth/*", identity)
		r.Handle("/api/v1/auth", identity)
		r.Handle("/api/v1/restaurants/*", catalog)
		r.Handle("/api/v1/restaurants", catalog)
		r.Handle("/api/v1/dishes/*", catalog)
		r.Handle("/api/v1/dishes", catalog)
	})

	r.Group(func(r chi.Router) {
		r.Use(requireAuth(cfg.JWTSecret))
		r.Handle("/api/v1/cart/*", order)
		r.Handle("/api/v1/cart", order)
		r.Handle("/api/v1/orders/*", order)
		r.Handle("/api/v1/orders", order)
		r.Handle("/api/v1/payments/*", payment)
		r.Handle("/api/v1/payments", payment)
		r.Handle("/api/v1/courier/*", courier)
		r.Handle("/api/v1/courier", courier)
		r.Handle("/api/v1/notifications/*", notification)
		r.Handle("/api/v1/notifications", notification)
	})

	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: r, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		log.Info("gateway listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("listen", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

func optionalAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			injectAuth(secret, r)
			if isProtectedCatalogWrite(r) && r.Header.Get("X-User-Id") == "" {
				writeUnauth(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func requireAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !injectAuth(secret, r) {
				writeUnauth(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func injectAuth(secret string, r *http.Request) bool {
	raw := auth.Bearer(r.Header.Get("Authorization"))
	if raw == "" || raw == r.Header.Get("Authorization") {
		return false
	}
	claims, err := auth.Parse(secret, raw)
	if err != nil {
		return false
	}
	r.Header.Set("X-User-Id", claims.Subject)
	r.Header.Set("X-User-Role", claims.Role)
	r.Header.Set("X-User-Email", claims.Email)
	r.Header.Set("X-User-Name", claims.Name)
	return true
}

func isProtectedCatalogWrite(r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodOptions {
		return false
	}
	return strings.HasPrefix(r.URL.Path, "/api/v1/restaurants") || strings.HasPrefix(r.URL.Path, "/api/v1/dishes")
}

func writeUnauth(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "unauthorized", "message": "login required"}})
}

func rateLimit() func(http.Handler) http.Handler {
	type client struct {
		lim  *rate.Limiter
		last time.Time
	}
	var mu sync.Mutex
	clients := map[string]*client{}
	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, c := range clients {
				if time.Since(c.last) > 5*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			mu.Lock()
			c, ok := clients[ip]
			if !ok {
				c = &client{lim: rate.NewLimiter(rate.Limit(30), 60)}
				clients[ip] = c
			}
			c.last = time.Now()
			lim := c.lim
			mu.Unlock()
			if !lim.Allow() {
				http.Error(w, `{"error":{"code":"rate_limited","message":"slow down"}}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
