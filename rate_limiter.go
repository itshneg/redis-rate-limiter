package redisratelimiter

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
)

type Limit float64

// Inf is the infinite rate limit; it allows all events (even if burst is zero).
const Inf = Limit(math.MaxFloat64)

const (
	keyPrefix         = "ratelimiter"
	keyLastUpdateTime = "lasttime"
)

var (
	errRateIsZero   error = errors.New("rate is zero")
	errWindowIsZero error = errors.New("time interval is zero")
	errBurstIsZero  error = errors.New("burst is zero")
	errKeyIsZero    error = errors.New("key is zero")
)

// RateLimiter
//
// Allow is an alias for AllowN when n=1.
// AllowN reports whether n events may happen at time t.
// Use this method if you intend to drop / skip events that exceed the rate limit.
// Otherwise use WaitN or Wait
//
// Wait is an alias for WaitN when n=1.
// WaitN blocks until lim permits n events to happen.
// Or until the associated context.Context is canceled.
type RateLimiter interface {
	Allow(ctx context.Context) (bool, error)
	AllowN(ctx context.Context, n int) (bool, error)
	Wait(ctx context.Context) error
	WaitN(ctx context.Context, n int) error
}

type rateLimiter struct {
	client *redis.Client
	mu     sync.Mutex
	key    string
}

func newRateLimiter(client *redis.Client, key string) rateLimiter {
	return rateLimiter{
		client: client,
		mu:     sync.Mutex{},
		key:    genKey(key),
	}
}

func genKey(key ...string) string {
	s := append([]string{keyPrefix}, key...)
	return strings.Join(s, ":")
}
