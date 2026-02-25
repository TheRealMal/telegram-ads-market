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

type service struct {
	repository repository
}

func NewService(repository repository) *service {
	return &service{
		repository: repository,
	}
}

func (s *service) RunPassedWorker(ctx context.Context) {
	ticker := time.NewTicker(workerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			passedList, err := s.repository.ListDealPostMessageByStatus(ctx, entity.DealPostMessageStatusPassed)
			if err != nil {
				slog.Error("deal_post_message worker: list passed", "error", err)
			} else if len(passedList) > 0 {
				ids := make([]int64, 0, len(passedList))
				for _, m := range passedList {
					ids = append(ids, m.ID)
				}
				if err := s.repository.CompleteDealPostMessagesAndSetDealsWaitingEscrowRelease(ctx, ids); err != nil {
					slog.Error("deal_post_message worker: complete (passed)", "error", err)
				} else {
					slog.Info("deal_post_message worker: completed (passed)", "count", len(ids), "ids", ids)
				}
			}
			deletedList, err := s.repository.ListDealPostMessageByStatus(ctx, entity.DealPostMessageStatusDeleted)
			if err != nil {
				slog.Error("deal_post_message worker: list deleted", "error", err)
			} else if len(deletedList) > 0 {
				ids := make([]int64, 0, len(deletedList))
				for _, m := range deletedList {
					ids = append(ids, m.ID)
				}
				if err := s.repository.FailDealPostMessagesAndSetDealsWaitingEscrowRefund(ctx, ids); err != nil {
					slog.Error("deal_post_message worker: fail (deleted)", "error", err)
				} else {
					slog.Info("deal_post_message worker: failed (deleted)", "count", len(ids), "ids", ids)
				}
			}
		}
	}
}
