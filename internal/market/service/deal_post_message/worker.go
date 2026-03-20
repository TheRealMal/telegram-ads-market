package deal_post_message

import (
	"context"
	"log/slog"
	"time"

	"ads-mrkt/internal/market/domain/entity"
	"ads-mrkt/internal/worker"
)

const workerInterval = 5 * time.Minute

type repository interface {
	ListDealPostMessageByStatus(ctx context.Context, status entity.DealPostMessageStatus) ([]*entity.DealPostMessage, error)
	CompleteDealPostMessagesAndSetDealsWaitingEscrowRelease(ctx context.Context, ids []int64) error
	FailDealPostMessagesAndSetDealsWaitingEscrowRefund(ctx context.Context, ids []int64) error
}

type dealChatService interface {
	UpdateDealForumTopicEmoji(ctx context.Context, dealID int64, status entity.DealStatus)
}

type service struct {
	repository  repository
	dealChatSvc dealChatService
}

func NewService(repository repository, dealChatSvc dealChatService) *service {
	return &service{
		repository:  repository,
		dealChatSvc: dealChatSvc,
	}
}

func (s *service) RunDealPostMessageWorker(ctx context.Context) {
	worker.RunTicker(ctx, "deal_post_message_worker", workerInterval, false, s.runPassedOnce)
}

func (s *service) runPassedOnce(ctx context.Context, logger *slog.Logger) {
	passedList, err := s.repository.ListDealPostMessageByStatus(ctx, entity.DealPostMessageStatusPassed)
	if err != nil {
		logger.Error("list passed", "error", err)
	} else if len(passedList) > 0 {
		ids := make([]int64, 0, len(passedList))
		for _, m := range passedList {
			ids = append(ids, m.ID)
		}
		if err := s.repository.CompleteDealPostMessagesAndSetDealsWaitingEscrowRelease(ctx, ids); err != nil {
			logger.Error("complete passed", "error", err)
		} else {
			logger.Info("completed passed", "count", len(ids), "ids", ids)
			for _, m := range passedList {
				s.dealChatSvc.UpdateDealForumTopicEmoji(ctx, m.DealID, entity.DealStatusWaitingEscrowRelease)
			}
		}
	}
	deletedList, err := s.repository.ListDealPostMessageByStatus(ctx, entity.DealPostMessageStatusDeleted)
	if err != nil {
		logger.Error("list deleted", "error", err)
	} else if len(deletedList) > 0 {
		ids := make([]int64, 0, len(deletedList))
		for _, m := range deletedList {
			ids = append(ids, m.ID)
		}
		if err := s.repository.FailDealPostMessagesAndSetDealsWaitingEscrowRefund(ctx, ids); err != nil {
			logger.Error("fail deleted", "error", err)
		} else {
			logger.Info("failed deleted", "count", len(ids), "ids", ids)
			for _, m := range deletedList {
				s.dealChatSvc.UpdateDealForumTopicEmoji(ctx, m.DealID, entity.DealStatusWaitingEscrowRefund)
			}
		}
	}
}
