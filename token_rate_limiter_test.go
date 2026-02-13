package redisratelimiter_test

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"github.com/itshneg/redis-rate-limiter"
)

func TestTokenRateLimiterAllowN(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	const key = "testTRL"

	trl, err := redisratelimiter.NewTokenRateLimiter(client, key, 2, 4)
	assert.NoError(t, err)

	ctx := context.Background()

	defer func() {
		_ = client.Del(ctx, key)
		_ = client.Close()
	}()

	t.Run("AllowN", func(t *testing.T) {
		result, err := trl.AllowN(ctx, 3)

		assert.NoError(t, err)
		assert.Equal(t, true, result)
	})

	t.Run("NotAllowN", func(t *testing.T) {
		result, err := trl.AllowN(ctx, 5)

		assert.NoError(t, err)
		assert.Equal(t, false, result)
	})

	t.Run("WaitN", func(t *testing.T) {
		now := time.Now()

		err := trl.WaitN(ctx, 3)

		wait := time.Since(now)

		assert.NoError(t, err)
		assert.Less(t, wait, time.Second)
	})

	t.Run("NoWaitN", func(t *testing.T) {
		err := trl.WaitN(ctx, 5)

		assert.Error(t, err)
	})
}
