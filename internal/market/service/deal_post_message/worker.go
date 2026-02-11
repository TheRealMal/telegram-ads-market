package deal_post_message

import (
	"context"
	"log/slog"
	"time"

	"ads-mrkt/internal/market/domain/entity"
)

const workerInterval = 5 * time.Minute

type repository interface {
	ListDealPostMessageByStatus(ctx context.Context, status entity.DealPostMessageStatus) ([]*entity.DealPostMessage, error)
	CompleteDealPostMessagesAndSetDealsWaitingEscrowRelease(ctx context.Context, ids []int64) error
	FailDealPostMessagesAndSetDealsWaitingEscrowRefund(ctx context.Context, ids []int64) error
}

// RunPassedWorker periodically lists deal_post_message with status=passed or status=deleted, then in one tx:
// - passed -> sets rows to completed and deals to waiting_escrow_release
// - deleted -> sets rows to failed and deals to waiting_escrow_refund
func RunPassedWorker(ctx context.Context, repo repository) {
	ticker := time.NewTicker(workerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Handle passed -> completed + deal waiting_escrow_release
			passedList, err := repo.ListDealPostMessageByStatus(ctx, entity.DealPostMessageStatusPassed)
			if err != nil {
				slog.Error("deal_post_message worker: list passed", "error", err)
			} else if len(passedList) > 0 {
				ids := make([]int64, 0, len(passedList))
				for _, m := range passedList {
					ids = append(ids, m.ID)
				}
				if err := repo.CompleteDealPostMessagesAndSetDealsWaitingEscrowRelease(ctx, ids); err != nil {
					slog.Error("deal_post_message worker: complete (passed)", "error", err)
				} else {
					slog.Info("deal_post_message worker: completed (passed)", "count", len(ids), "ids", ids)
				}
			}
			// Handle deleted -> failed + deal waiting_escrow_refund
			deletedList, err := repo.ListDealPostMessageByStatus(ctx, entity.DealPostMessageStatusDeleted)
			if err != nil {
				slog.Error("deal_post_message worker: list deleted", "error", err)
			} else if len(deletedList) > 0 {
				ids := make([]int64, 0, len(deletedList))
				for _, m := range deletedList {
					ids = append(ids, m.ID)
				}
				if err := repo.FailDealPostMessagesAndSetDealsWaitingEscrowRefund(ctx, ids); err != nil {
					slog.Error("deal_post_message worker: fail (deleted)", "error", err)
				} else {
					slog.Info("deal_post_message worker: failed (deleted)", "count", len(ids), "ids", ids)
				}
			}
		}
	}
}
