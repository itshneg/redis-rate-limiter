package redisratelimiter

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewTokenRateLimiter returns a new RateLimiter that allows events up to rate r
// per time interval w
//
// A SlidingRateLimiter controls how frequently events are allowed to happen.
// It implements a "token bucket" of size b, initially full and refilled
// at rate r tokens per second.
//
// Tokens are stored in the "ratelimiter:key" key
// Last update time are stored in the "ratelimiter:key:lasttime" key
//
// If no token is available, Allow returns false.
// If no token is available, Wait blocks until one can be obtained
//
// It returns an error if n exceeds the TokenRateLimiter's burst size, the Context is
// canceled or its associated context.Context is canceled.
// The burst limit is ignored if the rate limit is Inf.
//
// The methods AllowN and WaitN consume n tokens.
//
// TokenRateLimiter is safe for simultaneous use by multiple goroutines.
func NewTokenRateLimiter(client *redis.Client, key string, r Limit, burst int) (RateLimiter, error) {
	if key == "" {
		return nil, errKeyIsZero
	}

	if r == 0 {
		return nil, errRateIsZero
	}

	if burst == 0 {
		return nil, errBurstIsZero
	}

	return &tokenRateLimiter{
		rateLimiter:       newRateLimiter(client, key),
		keyLastUpdateTime: genKey(key, keyLastUpdateTime),
		rate:              r,
		burst:             burst,
	}, nil
}

type tokenRateLimiter struct {
	rateLimiter       rateLimiter
	keyLastUpdateTime string
	rate              Limit
	burst             int
}

func (trl *tokenRateLimiter) Allow(ctx context.Context) (bool, error) {
	return trl.AllowN(ctx, 1)
}

func (trl *tokenRateLimiter) AllowN(ctx context.Context, n int) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	trl.rateLimiter.mu.Lock()
	defer trl.rateLimiter.mu.Unlock()

	// Lua script for atomic rate limiting
	script := `
        local tokens = tonumber(redis.call('get', KEYS[1]) or ARGV[2])
        local rate = tonumber(ARGV[1])
        local burst = tonumber(ARGV[2])
		local allow = tonumber(ARGV[3])
        local now = tonumber(ARGV[4])
        local lastUpdate = tonumber(redis.call('get', KEYS[2]) or now)
        
        local elapsed = now - lastUpdate
        tokens = math.min(burst, tokens + (elapsed * rate))
        
        if tokens >= allow then
            tokens = tokens - allow
            redis.call('set', KEYS[1], tokens)
            redis.call('set', KEYS[2], now)
            redis.call('expire', KEYS[1], 60)
            redis.call('expire', KEYS[2], 60)
            return 1
        end
        
        return 0
    `

	now := time.Now().Unix()

	// ARGV
	// 1 - rate limit
	// 2 - max tokens
	// 3 - requested tokens
	// 4 - current time in seconds
	result, err := trl.rateLimiter.client.Eval(ctx, script,
		[]string{trl.rateLimiter.key, trl.keyLastUpdateTime},
		float64(trl.rate), trl.burst, n, now).Int()

	return result == 1, err
}

func (trl *tokenRateLimiter) Wait(ctx context.Context) error {
	return trl.WaitN(ctx, 1)
}

func (trl *tokenRateLimiter) WaitN(ctx context.Context, n int) error {
	if n > trl.burst && trl.rate != Inf {
		return fmt.Errorf("rate: Wait(n=%d) exceeds limiter's burst %v", n, trl.rate)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	trl.rateLimiter.mu.Lock()
	defer trl.rateLimiter.mu.Unlock()

	// Lua script for atomic rate limiting
	script := `
         local tokens = tonumber(redis.call('get', KEYS[1]) or ARGV[2])
        local rate = tonumber(ARGV[1])
        local burst = tonumber(ARGV[2])
		local allow = tonumber(ARGV[3])
        local now = tonumber(ARGV[4])
        local lastUpdate = tonumber(redis.call('get', KEYS[2]) or now)
        
        local elapsed = now - lastUpdate
        tokens = math.min(burst, tokens + (elapsed * rate))
		local result = 0
        
        if tokens >= allow then
            tokens = tokens - allow
		else
			tokens = allow
			result = allow - tokens
        end

		redis.call('set', KEYS[1], tokens)
		redis.call('set', KEYS[2], now)
        redis.call('expire', KEYS[1], 60)
        redis.call('expire', KEYS[2], 60)
        
        return result
    `

	now := time.Now().Unix()

	// ARGV
	// 1 - rate limit
	// 2 - max tokens
	// 3 - requested tokens
	// 4 - current time in seconds
	result, err := trl.rateLimiter.client.Eval(ctx, script,
		[]string{trl.rateLimiter.key, trl.keyLastUpdateTime},
		float64(trl.rate), trl.burst, n, now).Int()
	if err != nil {
		return err
	}

	if result == 0 {
		return nil
	}

	waitTime := Limit(result) / trl.rate

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Duration(waitTime) * time.Second):
	}

	return nil
}
