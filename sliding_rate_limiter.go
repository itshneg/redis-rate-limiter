package redisratelimiter

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewSlidingRateLimiter returns a new RateLimiter that allows events up to rate r
// per time interval w
//
// A SlidingRateLimiter controls how frequently events are allowed to happen.
// It implements a "sliding window" at rate r requests per time interval window.
//
// Request timestamps are stored in the "ratelimiter:key" key
//
// If the limit is exhausted, Allow returns false.
// If the limit is exhausted, Wait blocks until a request is acquired.
// Or until the associated context.Context is canceled.
//
// The methods AllowN and WaitN consume n requests.
//
// SlidingRateLimiter is safe for simultaneous use by multiple goroutines.
func NewSlidingRateLimiter(client *redis.Client, key string, r int, w time.Duration) (RateLimiter, error) {
	if key == "" {
		return nil, errKeyIsZero
	}

	if r == 0 {
		return nil, errRateIsZero
	}

	if w == 0 {
		return nil, errWindowIsZero
	}

	return &slidingRateLimiter{
		rateLimiter: newRateLimiter(client, key),
		rate:        r,
		window:      w,
	}, nil
}

// A SlidingRateLimiter controls how frequently events are allowed to happen.
// It implements a "sliding window" at rate r tokens per time interval window.
type slidingRateLimiter struct {
	rateLimiter rateLimiter
	rate        int
	window      time.Duration
}

func (srl *slidingRateLimiter) Allow(ctx context.Context) (bool, error) {
	return srl.AllowN(ctx, 1)
}

func (srl *slidingRateLimiter) AllowN(ctx context.Context, n int) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	srl.rateLimiter.mu.Lock()
	defer srl.rateLimiter.mu.Unlock()

	// Lua script for atomic rate limiting
	script := `
        local last = '[' .. ARGV[1]
		redis.call('ZREMRANGEBYLEX', KEYS[1], '[0', last)
        local tokens = tonumber(redis.call('ZCOUNT', KEYS[1], 0, '+inf'))

		local rate = tonumber(ARGV[2])
		local allow = tonumber(ARGV[3])
        local now = tonumber(ARGV[4])

        if tokens + allow > rate then
            return 0
        end

		for i = 0, allow-1 do 
			now = now + i
			redis.call('ZADD', KEYS[1], 0, now)
		end
        
        return 1
    `

	now := time.Now().UnixNano()
	window := srl.window.Nanoseconds()

	// ARGV
	// 1 - time window in nanoseconds
	// 2 - rate limit
	// 3 - count of request
	// 4 - current time in nanoseconds
	result, err := srl.rateLimiter.client.Eval(ctx, script,
		[]string{srl.rateLimiter.key},
		now-window, srl.rate, n, now).Int()

	return result == 1, err
}

func (srl *slidingRateLimiter) Wait(ctx context.Context) error {
	return srl.WaitN(ctx, 1)
}

func (srl *slidingRateLimiter) WaitN(ctx context.Context, n int) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	srl.rateLimiter.mu.Lock()
	defer srl.rateLimiter.mu.Unlock()

	// Lua script for atomic rate limiting
	script := `
        local last = '[' .. ARGV[1]
		redis.call('ZREMRANGEBYLEX', KEYS[1], '[0', last)
        local tokens = tonumber(redis.call('ZCOUNT', KEYS[1], 0, '+inf'))

		local rate = tonumber(ARGV[2])
		local allow = tonumber(ARGV[3])
        local now = tonumber(ARGV[4])

		for i = 0, allow-1 do 
			now = now + i
			redis.call('ZADD', KEYS[1], 0, now)
		end

        local wait = rate - allow - tokens
		if wait < 0 then
            return -wait
        end
        
        return 0
    `

	now := time.Now().UnixNano()
	window := srl.window.Nanoseconds()

	// ARGV
	// 1 - time window in nanoseconds
	// 2 - rate limit
	// 3 - count of request
	// 4 - current time in nanoseconds
	result, err := srl.rateLimiter.client.Eval(ctx, script,
		[]string{srl.rateLimiter.key},
		now-window, srl.rate, n, now).Int()
	if err != nil {
		return err
	}

	if result == 0 {
		return nil
	}

	waitTime := time.Duration(result) * srl.window / time.Duration(srl.rate)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitTime):
	}

	return nil
}
