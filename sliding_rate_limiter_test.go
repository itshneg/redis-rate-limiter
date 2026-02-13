package redisratelimiter_test

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	"github.com/itshneg/redis-rate-limiter"
)

func TestSlidingRateLimiterAllowN(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	const key = "testSRL"

	srl, err := redisratelimiter.NewSlidingRateLimiter(client, key, 2, time.Second)
	assert.NoError(t, err)

	ctx := context.Background()

	defer func() {
		_ = client.Del(ctx, key)
		_ = client.Close()
	}()

	t.Run("AllowN", func(t *testing.T) {
		result, err := srl.AllowN(ctx, 1)

		assert.NoError(t, err)
		assert.Equal(t, true, result)
	})

	t.Run("NotAllowN", func(t *testing.T) {
		result, err := srl.AllowN(ctx, 4)

		assert.NoError(t, err)
		assert.Equal(t, false, result)
	})

	t.Run("WaitN", func(t *testing.T) {
		now := time.Now()

		err := srl.WaitN(ctx, 4)

		wait := time.Since(now)

		assert.NoError(t, err)
		assert.Greater(t, wait, time.Second)
	})
}
