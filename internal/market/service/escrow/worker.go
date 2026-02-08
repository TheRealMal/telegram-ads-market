package escrow

import (
	"context"
	"log/slog"
	"time"
)

const (
	escrowWorkerInterval = 30 * time.Second
)

// Worker runs a loop that fetches all deals in status approved without escrow and creates escrow for each.
// Interval is the delay between runs. Run blocks until ctx is cancelled.
func (s *service) Worker(ctx context.Context) {
	ticker := time.NewTicker(escrowWorkerInterval)
	defer ticker.Stop()

	run := func(ctx context.Context) {
		deals, err := s.marketRepository.ListDealsApprovedWithoutEscrow(ctx)
		if err != nil {
			slog.Error("escrow worker: list approved deals without escrow", "error", err)
			return
		}
		for _, d := range deals {
			if ctx.Err() != nil {
				return
			}
			if err := s.CreateEscrow(ctx, d.ID); err != nil {
				slog.Error("escrow worker: create escrow for deal", "deal_id", d.ID, "error", err)
				continue
			}
			slog.Info("escrow worker: created escrow for deal", "deal_id", d.ID)
		}
	}

	// Run once immediately, then on ticker
	run(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run(ctx)
		}
	}
}
