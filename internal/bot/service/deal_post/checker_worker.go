package deal_post

import (
	"context"
	"log/slog"
	"time"

	"ads-mrkt/internal/market/domain/entity"
	"ads-mrkt/internal/worker"
)

const postCheckerInterval = 5 * time.Minute

func (s *service) RunDealPostCheckerWorker(ctx context.Context) {
	worker.RunTicker(ctx, "bot_deal_post_checker", postCheckerInterval, false, s.runDealPostCheckerOnce)
}

func (s *service) runDealPostCheckerOnce(ctx context.Context, logger *slog.Logger) {
	list, err := s.dealPostMessageRepo.ListDealPostMessageExistsWithNextCheckBefore(ctx, time.Now())
	if err != nil {
		logger.Error("list", "error", err)
		return
	}
	for _, m := range list {
		exists, err := s.telegramClient.CheckMessageExists(ctx, m.ChannelID, m.MessageID, s.serviceChatID)
		if err != nil {
			logger.Error("check message exists", "id", m.ID, "error", err)
			continue
		}
		if !exists {
			_ = s.dealPostMessageRepo.UpdateDealPostMessageStatus(ctx, m.ID, entity.DealPostMessageStatusDeleted)
			logger.Info("message deleted", "id", m.ID, "deal_id", m.DealID)
			continue
		}
		nextCheck := m.NextCheck.Add(postCheckAdvanceHour)
		if nextCheck.After(m.UntilTs) {
			_ = s.dealPostMessageRepo.UpdateDealPostMessageStatus(ctx, m.ID, entity.DealPostMessageStatusPassed)
			logger.Info("passed", "id", m.ID, "deal_id", m.DealID)
		} else {
			_ = s.dealPostMessageRepo.UpdateDealPostMessageStatusAndNextCheck(ctx, m.ID, entity.DealPostMessageStatusExists, nextCheck)
		}
	}
}
