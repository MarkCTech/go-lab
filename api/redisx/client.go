// Package redisx holds an optional shared Redis client for rate limits and login lockout.
// When REDIS_URL is unset or connection fails, Client stays nil and callers use in-memory paths.
package redisx

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client is non-nil after successful Init; read-only for consumers.
var Client *redis.Client

// Init parses url, pings once, and sets Client. Invalid URL or ping failure logs a warning and leaves Client nil (fail open).
func Init(url string) {
	url = strings.TrimSpace(url)
	if url == "" {
		return
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		slog.Warn("redis_url_invalid_failing_open", "error", err.Error())
		return
	}
	c := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Ping(ctx).Err(); err != nil {
		slog.Warn("redis_ping_failed_failing_open", "error", err.Error())
		_ = c.Close()
		return
	}
	Client = c
	slog.Info("redis_connected", "addr", opts.Addr)
}

// Close releases the client if set.
func Close() {
	if Client != nil {
		_ = Client.Close()
		Client = nil
	}
}
