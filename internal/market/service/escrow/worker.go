package escrow

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"ads-mrkt/internal/market/domain/entity"
)

const (
	escrowWorkerInterval = 30 * time.Second
)

func (s *service) Worker(ctx context.Context) {
	logger := slog.With("component", "escrow_creator_worker")
	ticker := time.NewTicker(escrowWorkerInterval)
	defer ticker.Stop()

	run := func(ctx context.Context) {
		deals, err := s.dealRepo.ListDealsApprovedWithoutEscrow(ctx)
		if err != nil {
			logger.Error("escrow worker: list approved deals without escrow", "error", err)
			return
		}
		for _, d := range deals {
			if ctx.Err() != nil {
				return
			}
			if err := s.CreateEscrow(ctx, d.ID); err != nil {
				logger.Error("escrow worker: create escrow for deal", "deal_id", d.ID, "error", err)
				continue
			}
			logger.Info("escrow worker: created escrow for deal", "deal_id", d.ID)
		}
	}

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

const releaseRefundWorkerInterval = 1 * time.Minute

func (s *service) ReleaseRefundWorker(ctx context.Context) {
	logger := slog.With("component", "escrow_manager_worker")
	ticker := time.NewTicker(releaseRefundWorkerInterval)
	defer ticker.Stop()
	run := func(ctx context.Context) {
		for _, release := range []bool{true, false} {
			var deals []*entity.Deal
			var err error
			if release {
				deals, err = s.dealRepo.ListDealsWaitingEscrowRelease(ctx)
			} else {
				deals, err = s.dealRepo.ListDealsWaitingEscrowRefund(ctx)
			}
			if err != nil {
				logger.Error("list deals", "release", release, "error", err)
				continue
			}
			for _, d := range deals {
				if ctx.Err() != nil {
					return
				}
				if err := s.ReleaseOrRefundEscrow(ctx, logger, d.ID, release); err != nil {
					if errors.Is(err, ErrPayoutAddressNotSet) {
						logger.Debug("skip deal, payout address not set", "deal_id", d.ID, "release", release)
					} else {
						logger.Error("release/refund failed", "deal_id", d.ID, "release", release, "error", err)
					}
					continue
				}
			}
		}
	}
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
