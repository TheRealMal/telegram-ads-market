package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"ads-mrkt/internal/postgres/config"
)

var (
	pingTimeout  = 5 * time.Second
	pingAttempts = 3
)

type postgres struct {
	pg          *pgxpool.Pool
	pingTimeout time.Duration
}

// New creates and initializes a PostgreSQL connection pool using the provided configuration.
// It sets a session-level lock timeout on each new connection and returns a postgres instance with the configured pool and ping timeout.
// Returns an error if the connection string cannot be parsed or the pool cannot be created.
func New(ctx context.Context, cfg config.Config) (*postgres, error) {
	dbConnString := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?connect_timeout=%d&sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.ConnectTimeout,
		cfg.SSLMode)

	pgConf, err := pgxpool.ParseConfig(dbConnString)
	if err != nil {
		slog.Error("db connection string parsing", "error", err)

		return nil, err
	}

	pgConf.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx,
			fmt.Sprintf("SET lock_timeout = '%s'", cfg.LockTimeout),
		)
		return err
	}

	pgPool, err := pgxpool.NewWithConfig(ctx, pgConf)
	if err != nil {
		slog.Error("connect to postgres", "error", err)

		return nil, err
	}

	return &postgres{
		pg:          pgPool,
		pingTimeout: cfg.PingTimeout,
	}, nil
}

func (p *postgres) Ping(ctx context.Context) error {
	ticker := time.NewTicker(pingTimeout)
	defer ticker.Stop()

	var err error
	// Ping 3 times with a specified time interval.
	for i := 1; i <= pingAttempts; i++ {
		// Maximum allowed context lifetime for ping, slightly less than the duration of the test cycle.
		// If the ping process is not successful, it hangs indefinitely, so it's important to limit the context with a timeout.
		pingCtx, cancel := context.WithTimeout(ctx, pingTimeout-time.Millisecond*10)
		err = p.pg.Ping(pingCtx)
		if err == nil {
			cancel()

			return nil
		}

		slog.Info("ping attempt was not successful", "attempt", i, "component", "postgres", "address", fmt.Sprintf("%s:%d", p.pg.Config().ConnConfig.Host, p.pg.Config().ConnConfig.Port), "database", p.pg.Config().ConnConfig.Database)
		cancel()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
	return err
}

func (p *postgres) Close() {
	p.pg.Close()
}
