package escrow

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"ads-mrkt/internal/market/domain/entity"
	"ads-mrkt/internal/worker"
)

const (
	escrowWorkerInterval = 30 * time.Second
)

func (s *service) RunEscrowCreatorWorker(ctx context.Context) {
	worker.RunTicker(ctx, "escrow_creator_worker", escrowWorkerInterval, true, s.runEscrowCreatorOnce)
}

func (s *service) runEscrowCreatorOnce(ctx context.Context, logger *slog.Logger) {
	deals, err := s.dealRepo.ListDealsApprovedWithoutEscrow(ctx)
	if err != nil {
		logger.Error("list approved deals without escrow", "error", err)
		return
	}
	for _, d := range deals {
		if ctx.Err() != nil {
			return
		}
		if err := s.CreateEscrow(ctx, d.ID); err != nil {
			logger.Error("create escrow for deal", "deal_id", d.ID, "error", err)
			continue
		}
		logger.Info("created escrow for deal", "deal_id", d.ID)
	}
}

const releaseRefundWorkerInterval = 1 * time.Minute

func (s *service) RunReleaseRefundWorker(ctx context.Context) {
	worker.RunTicker(ctx, "escrow_release_refund_worker", releaseRefundWorkerInterval, true, s.runReleaseRefundOnce)
}

func (s *service) runReleaseRefundOnce(ctx context.Context, logger *slog.Logger) {
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
