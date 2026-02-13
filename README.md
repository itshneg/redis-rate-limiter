# redis-rate-limiter

A high-performance, Redis-based rate limiting library for Go applications. This library provides a simple interface for implementing rate limiting with configurable rates, time periods, and support for composite rate limits.

## Features

- 🚀 **Simple API**: Clean and intuitive interface for rate limiting
- ⚡ **High Performance**: Built on top of Redis with efficient algorithms
- 🔀 **Composite Limiters**: Apply multiple rate limits simultaneously (e.g., per-second and per-minute)
- 🛡️ **Thread-Safe**: Safe for concurrent use in goroutines
- 🧪 **Well-Tested**: Comprehensive test suite with race condition detection
- 📦 **Production-Ready**: Used in production environments

## Installation

```bash
go get github.com/itshneg/redis-rate-limiter
```

## Prerequisites

- Go 1.23 or higher
- Redis server (version 6.0+ recommended)

## Quick Start

### Basic Usage Limiter by Sliding window

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/itshneg/redis-rate-limiter"
    "github.com/redis/go-redis/v9"
)

func main() {
    // Create Redis client
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })

    // Create a rate limiter: 10 requests per second
    srl, err := redisratelimiter.NewSlidingRateLimiter(client, "test", 10, time.Second)
	if err != nil {
      fmt.Printf("Error: %v\n", err)
      return
    }
	
	ctx := context.Background()

    // Use the limiter
    ok, err := srl.AllowN(ctx, 5)
	if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    // Your code here - 5 requests is allowed
	if ok {
      fmt.Println("Request allowed!")
    }
}
```

### Basic Usage Limiter by Token bucket

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/itshneg/redis-rate-limiter"
    "github.com/redis/go-redis/v9"
)

func main() {
    // Create Redis client
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })

    // Create a rate limiter: 10 requests per second
    srl, err := redisratelimiter.NewTokenRateLimiter(client, "test", 2, 4)
	if err != nil {
      fmt.Printf("Error: %v\n", err)
      return
    }
	
	ctx := context.Background()

    // Use the limiter
    if err := srl.WaitN(ctx, 5); err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    // Your code here - 5 requests is allowed
    fmt.Println("Request allowed!")
}
```

### Custom Time Period

Custom Time Period available for SlidingRateLimiter only.

```go
// 50 requests per minute
redisratelimiter.NewSlidingRateLimiter(client, "test", 50, time.Minute)

// 5 requests per 3 second
redisratelimiter.NewSlidingRateLimiter(client, "test", 50, 3*time.Second)

// 60 requests per hour
redisratelimiter.NewSlidingRateLimiter(client, "test", 60, time.Hour)
```


## API Reference

### Types

#### `RateLimiter` Interface

```go
type RateLimiter interface {
    Allow(ctx context.Context) (bool, error)
    AllowN(ctx context.Context, n int) (bool, error)
    Wait(ctx context.Context) error
    WaitN(ctx context.Context, n int) error
}
```

- **Take()**: Blocks until the request is allowed under the rate limit, then returns the time when the request was allowed and any error that occurred.

### Functions

#### `func NewSlidingRateLimiter(client *redis.Client, key string, r int, w time.Duration) (RateLimiter, error) `
#### `func NewTokenRateLimiter(client *redis.Client, key string, r Limit, burst int) (RateLimiter, error)`
Creates a new rate limiter instance.

**Parameters:**

- `rdb`: Redis client (supports `*redis.Client`, `*redis.ClusterClient`, etc.)
- `key`: Unique key for this rate limit (e.g., user ID, API endpoint)
- `rate`: Maximum number of requests allowed in the time period
- `opts`: Optional configuration:
    - `Per(duration)`: Time period for the rate limit (default: 1 second)
    - `WithContext(ctx)`: Context for cancellation support

**Returns:** A `RateLimiter` interface

## Examples

### HTTP Middleware

```go
func rateLimitMiddleware(rrl redisratelimiter.RateLimiter) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ok, err := rrl.Allow(r.Context)
        if err != nil {
            http.Error(w, "Rate limit error", http.StatusInternalServerError)
            return
        }
		
		if !ok {
          http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
          return
        }   
		
        // Continue with request
    }
}

// Usage
limiter := ratelimiter.New(rdb, "api:endpoint", 100, ratelimiter.Per(time.Minute))
http.HandleFunc("/api", rateLimitMiddleware(limiter))
```

## Testing

Run the test suite:

```bash
# Run all tests
go test ./...

# Run tests with race detection
go test -race ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...

# Run benchmarks
go test -bench=. -benchmem ./...
```

Or use the Makefile:

```bash
make test        # Run tests with race detection and coverage
make lint        # Run linters
make coverage    # Generate coverage report
make all         # Run all checks
```

## Thread Safety

All limiter implementations are thread-safe and can be safely used from multiple goroutines concurrently.

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on how to contribute to this project.

## License

See [LICENSE](LICENSE) file for details.

## Author

Created and maintained by [itshneg](https://github.com/itshneg)
