package redis

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"time"

	"ads-mrkt/internal/redis/config"

	"github.com/redis/go-redis/v9"
)

var (
	pingTimeout  = 5 * time.Second
	pingAttempts = 3
)

type Client struct {
	client *redis.Client
}

func optsFromConfig(cfg config.Config) *redis.Options {
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	opts := &redis.Options{
		Addr:     addr,
		Username: cfg.User,
		Password: cfg.Password,
		DB:       cfg.DB,
	}
	if cfg.EnableTLS {
		opts.TLSConfig = &tls.Config{
			MinVersion: cfg.MinTLSVersion,
		}
	}
	return opts
}

func New(ctx context.Context, cfg config.Config) (*Client, error) {
	client := redis.NewClient(optsFromConfig(cfg))

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{
		client: client,
	}, nil
}

func NewOptional(ctx context.Context, cfg config.Config) *Client {
	client := redis.NewClient(optsFromConfig(cfg))
	return &Client{
		client: client,
	}
}

func (c *Client) Ping(ctx context.Context) error {
	ticker := time.NewTicker(pingTimeout)
	defer ticker.Stop()

	var err error
	for i := 1; i <= pingAttempts; i++ {
		pingCtx, cancel := context.WithTimeout(ctx, pingTimeout-time.Millisecond*10)
		err = c.client.Ping(pingCtx).Err()
		if err == nil {
			cancel()
			return nil
		}
		slog.Info("redis ping attempt failed", "attempt", i, "addr", c.client.Options().Addr)
		cancel()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
	return err
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) Client() *redis.Client {
	return c.client
}
