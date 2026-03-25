package myhandlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/codemarked/go-lab/api/redisx"
	"github.com/redis/go-redis/v9"
)

// In-memory per-email lockout after repeated failed password attempts (same process only).
// With redisx.Client, lockout is shared across replicas. Redis errors fail open (no lock).
// Does not apply to unknown-email failures (avoid account DoS).

const (
	loginMaxFailuresPerEmail = 5
	loginLockoutDuration     = 15 * time.Minute
	loginFailKeyTTL          = time.Hour
)

type loginEmailState struct {
	mu          sync.Mutex
	failures    int
	lockedUntil time.Time
}

var loginEmailThrottle sync.Map // email -> *loginEmailState

func loginEmailKey(email string) string {
	h := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return hex.EncodeToString(h[:16])
}

func loginEmailLocked(ctx context.Context, email string) (locked bool, retryAfter time.Duration) {
	if rdb := redisx.Client; rdb != nil {
		return redisLoginEmailLocked(ctx, rdb, email)
	}
	v, ok := loginEmailThrottle.Load(email)
	if !ok {
		return false, 0
	}
	st := v.(*loginEmailState)
	st.mu.Lock()
	defer st.mu.Unlock()
	now := time.Now()
	if now.Before(st.lockedUntil) {
		return true, st.lockedUntil.Sub(now)
	}
	return false, 0
}

func redisLoginEmailLocked(ctx context.Context, rdb *redis.Client, email string) (bool, time.Duration) {
	lockKey := "login:lock:" + loginEmailKey(email)
	ttl, err := rdb.TTL(ctx, lockKey).Result()
	if err != nil {
		slog.Warn("redis_login_lock_ttl_failed_open", "error", err.Error())
		return false, 0
	}
	if ttl > 0 {
		return true, ttl
	}
	return false, 0
}

func recordLoginFailureKnownUser(ctx context.Context, email string) {
	if rdb := redisx.Client; rdb != nil {
		redisRecordLoginFailure(ctx, rdb, email)
		return
	}
	v, _ := loginEmailThrottle.LoadOrStore(email, &loginEmailState{})
	st := v.(*loginEmailState)
	st.mu.Lock()
	defer st.mu.Unlock()
	now := time.Now()
	if now.Before(st.lockedUntil) {
		return
	}
	st.failures++
	if st.failures >= loginMaxFailuresPerEmail {
		st.lockedUntil = now.Add(loginLockoutDuration)
		st.failures = 0
	}
}

func redisRecordLoginFailure(ctx context.Context, rdb *redis.Client, email string) {
	k := loginEmailKey(email)
	failKey := "login:fail:" + k
	lockKey := "login:lock:" + k
	n, err := rdb.Incr(ctx, failKey).Result()
	if err != nil {
		slog.Warn("redis_login_fail_incr_failed_open", "error", err.Error())
		return
	}
	if n == 1 {
		_ = rdb.Expire(ctx, failKey, loginFailKeyTTL).Err()
	}
	if n >= loginMaxFailuresPerEmail {
		_ = rdb.Set(ctx, lockKey, "1", loginLockoutDuration).Err()
		_, _ = rdb.Del(ctx, failKey).Result()
	}
}

func recordLoginSuccessClearThrottle(ctx context.Context, email string) {
	if rdb := redisx.Client; rdb != nil {
		k := loginEmailKey(email)
		_, _ = rdb.Del(ctx, "login:fail:"+k, "login:lock:"+k).Result()
		return
	}
	loginEmailThrottle.Delete(email)
}
