package middleware

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func fixedWindowRedisAllow(ctx context.Context, rdb *redis.Client, scope, ip string, limit int) (allowed bool, err error) {
	slot := time.Now().Unix() / 60
	safeIP := strings.ReplaceAll(strings.ReplaceAll(ip, ":", "_"), "%", "_")
	key := fmt.Sprintf("ratelimit:%s:%s:%d", scope, safeIP, slot)
	n, err := rdb.Incr(ctx, key).Result()
	if err != nil {
		return true, err
	}
	if n == 1 {
		_ = rdb.Expire(ctx, key, 75*time.Second).Err()
	}
	return n <= int64(limit), nil
}
