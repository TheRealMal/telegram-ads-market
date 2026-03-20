package worker

import (
	"context"
	"log/slog"
	"time"
)

// RunTicker runs fn on a ticker interval, structured logging,
// and graceful shutdown via context cancellation.
//
// If runOnStart is true, fn is called once immediately before the ticker starts.
// The logger passed to fn includes a "component" attribute set to name.
func RunTicker(ctx context.Context, name string, interval time.Duration, runOnStart bool, fn func(ctx context.Context, logger *slog.Logger)) {
	logger := slog.With("component", name)
	logger.Info("started")
	defer logger.Info("stopped")

	if runOnStart {
		fn(ctx, logger)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fn(ctx, logger)
		}
	}
}
