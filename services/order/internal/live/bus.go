package live

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

type Bus struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *Bus { return &Bus{rdb: rdb} }

func (b *Bus) Publish(ctx context.Context, userIDs []string, payload any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	seen := map[string]struct{}{}
	for _, id := range userIDs {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		_ = b.rdb.Publish(ctx, "user:"+id, raw).Err()
	}
}
