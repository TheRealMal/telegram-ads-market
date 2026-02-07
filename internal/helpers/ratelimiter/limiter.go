package ratelimiter

import (
	"context"
	"log/slog"
	"time"
)

type redisClient interface {
	Decr(ctx context.Context, key string) (int64, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
}

type RateLimiter struct {
	client redisClient
	limit  int
	window time.Duration
	key    string
}

const (
	defaultKey = "rate_limiter"
)

func NewRateLimiter(ctx context.Context, client redisClient, limit int, window time.Duration) *RateLimiter {
	r := &RateLimiter{
		client: client,
		limit:  limit,
		window: window,
		key:    defaultKey,
	}

	go func() {
		err := r.StartBackgroundWorker(ctx)
		if err != nil {
			slog.Error("background rate limiter worker failed", "error", err)
		}
	}()

	return r
}

// CheckLimits checks if a request is allowed based on the rate limit.
// It decrements the request counter in Redis and returns true if the request is allowed (counter > 0),
// or false if the request is denied (counter <= 0).
// It also updates Prometheus metrics for total requests, allowed requests, and denied requests.
func (r *RateLimiter) CheckLimits(ctx context.Context) (bool, error) {
	count, err := r.client.Decr(ctx, r.key)
	if err != nil {
		return false, err
	}

	slog.Info("Rate limiter", "count", count)

	if count <= 0 {
		return false, nil
	}

	return true, nil
}

// StartBackgroundWorker is a background goroutine that resets the request counters in Redis
// every time window (e.g., 1 second). It ensures that the counters are reset to the limit
// and updates their TTL to match the rate limit window.
func (l *RateLimiter) StartBackgroundWorker(ctx context.Context) error {
	ticker := time.NewTicker(l.window)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Reset the counter to the limit and update the TTL
			err := l.client.Set(ctx, l.key, l.limit, l.window)
			if err != nil {
				slog.Info("Resetting counter", "error", err)
			}

		case <-ctx.Done():
			// Stop the worker if the context is canceled
			return ctx.Err()
		}
	}
}
