package deal_post

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	telegram "ads-mrkt/internal/helpers/telegram"
	"ads-mrkt/internal/market/domain/entity"
)

const (
	postSenderInterval   = 1 * time.Minute
	postCheckAdvanceHour = time.Hour
)

func (s *service) RunDealPostSenderWorker(ctx context.Context) {
	logger := slog.With("component", "bot_deal_post_sender")
	ticker := time.NewTicker(postSenderInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runDealPostSenderOnce(ctx, logger)
		}
	}
}

func (s *service) runDealPostSenderOnce(ctx context.Context, logger *slog.Logger) {
	deals, err := s.dealRepo.ListDealsEscrowDepositConfirmedWithoutPostMessage(ctx)
	if err != nil {
		logger.Error("list deals", "error", err)
		return
	}
	for _, deal := range deals {
		listing, err := s.listingRepo.GetListingByID(ctx, deal.ListingID)
		if err != nil || listing == nil || listing.ChannelID == nil {
			logger.Error("skip deal, no listing or channel", "deal_id", deal.ID)
			continue
		}
		channelID := *listing.ChannelID

		ch, err := s.channelRepo.GetChannelByID(ctx, channelID)
		if err != nil || ch == nil {
			logger.Error("skip deal, channel not found", "deal_id", deal.ID, "channel_id", channelID)
			continue
		}
		if !ch.CanPostMessages() {
			logger.Error("skip deal, bot not admin in channel", "deal_id", deal.ID, "channel_id", channelID, "bot_status", ch.BotMemberStatus)
			continue
		}

		text := entity.GetMessageFromDetails(deal.Details)
		if text == "" {
			logger.Error("skip deal, no message in details", "deal_id", deal.ID)
			continue
		}
		if postedAt, ok := entity.GetPostedAtFromDetails(deal.Details); ok && time.Now().Before(postedAt) {
			logger.Debug("skip deal, posted_at in future", "deal_id", deal.ID, "posted_at", postedAt)
			continue
		}

		// Extract entities (prefer "entities", fall back to "caption_entities")
		var entities []telegram.MessageEntity
		if raw := entity.GetRawEntitiesFromDetails(deal.Details); len(raw) > 0 {
			_ = json.Unmarshal(raw, &entities)
		}
		if len(entities) == 0 {
			if raw := entity.GetRawCaptionEntitiesFromDetails(deal.Details); len(raw) > 0 {
				_ = json.Unmarshal(raw, &entities)
			}
		}

		// Crash recovery: if last lock is expired and still locked, release it as failed and let next tick retry.
		lastLock, err := s.dealActionLockRepo.GetLastDealActionLock(ctx, deal.ID, entity.DealActionTypePostMessage)
		if err != nil {
			logger.Error("get last lock", "deal_id", deal.ID, "error", err)
			continue
		}
		if lastLock != nil && lastLock.Status == entity.DealActionLockStatusLocked && !lastLock.ExpireAt.After(time.Now()) {
			_ = s.dealActionLockRepo.ReleaseDealActionLock(ctx, lastLock.ID, entity.DealActionLockStatusFailed)
			logger.Warn("released expired lock, will retry next tick", "deal_id", deal.ID)
			continue
		}

		lockID, err := s.dealActionLockRepo.TakeDealActionLock(ctx, deal.ID, entity.DealActionTypePostMessage)
		if err != nil {
			logger.Debug("skip deal, lock not acquired", "deal_id", deal.ID, "error", err)
			continue
		}
		releaseLock := func(status entity.DealActionLockStatus) {
			_ = s.dealActionLockRepo.ReleaseDealActionLock(ctx, lockID, status)
		}

		msgID, err := s.telegramClient.SendChannelMessage(ctx, channelID, text, entities)
		if err != nil {
			logger.Error("send message", "deal_id", deal.ID, "error", err)
			releaseLock(entity.DealActionLockStatusFailed)
			continue
		}
		untilTs := time.Now().Add(time.Duration(deal.Duration) * time.Hour)
		nextCheck := time.Now().Add(postCheckAdvanceHour)
		m := &entity.DealPostMessage{
			DealID:      deal.ID,
			ChannelID:   channelID,
			MessageID:   msgID,
			PostMessage: text,
			Status:      entity.DealPostMessageStatusExists,
			NextCheck:   nextCheck,
			UntilTs:     untilTs,
		}
		if err := s.dealPostMessageRepo.CreateDealPostMessageAndSetDealInProgress(ctx, m); err != nil {
			logger.Error("create deal_post_message", "deal_id", deal.ID, "error", err)
			releaseLock(entity.DealActionLockStatusFailed)
			continue
		}
		releaseLock(entity.DealActionLockStatusCompleted)
		logger.Info("sent and saved", "deal_id", deal.ID, "channel_id", channelID, "message_id", msgID)
	}
}
