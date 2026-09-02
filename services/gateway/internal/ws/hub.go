package ws

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/savor/gateway/internal/auth"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func Handler(rdb *redis.Client, secret string, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			token = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		}
		claims, err := auth.Parse(secret, token)
		if err != nil {
			http.Error(w, `{"error":{"code":"unauthorized","message":"invalid token"}}`, http.StatusUnauthorized)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		ctx := r.Context()
		channel := "user:" + claims.Subject
		sub := rdb.Subscribe(ctx, channel)
		defer sub.Close()
		ch := sub.Channel()

		_ = conn.WriteJSON(map[string]string{"type": "connected", "userId": claims.Subject})

		go func() {
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				if err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
					return
				}
			}
		}
	}
}
