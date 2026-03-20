package deal_post

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	telegram "ads-mrkt/internal/helpers/telegram"
	"ads-mrkt/internal/market/domain/entity"
	marketerrors "ads-mrkt/internal/market/domain/errors"
	"ads-mrkt/internal/worker"
)

const (
	postSenderInterval   = 1 * time.Minute
	postCheckAdvanceHour = time.Hour
)

func (s *service) RunDealPostSenderWorker(ctx context.Context) {
	worker.RunTicker(ctx, "bot_deal_post_sender", postSenderInterval, false, s.runDealPostSenderOnce)
}

func (s *service) runDealPostSenderOnce(ctx context.Context, logger *slog.Logger) {
	deals, err := s.dealRepo.ListDealsEscrowDepositConfirmedWithoutPostMessage(ctx)
	if err != nil {
		logger.Error("list deals", "error", err)
		return
	}
	for _, deal := range deals {
		listing, err := s.listingRepo.GetListingByID(ctx, deal.ListingID)
		if err != nil {
			logger.Error("skip deal, listing error", "deal_id", deal.ID, "error", err)
			continue
		}
		if listing.ChannelID == nil {
			logger.Error("skip deal, no channel on listing", "deal_id", deal.ID)
			continue
		}
		channelID := *listing.ChannelID

		ch, err := s.channelRepo.GetChannelByID(ctx, channelID)
		if err != nil {
			logger.Error("skip deal, channel not found", "deal_id", deal.ID, "channel_id", channelID, "error", err)
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
		if raw := entity.GetBotAPIEntitiesFromDetails(deal.Details); len(raw) > 0 {
			if err := json.Unmarshal(raw, &entities); err != nil {
				logger.Error("unmarshal entities from deal details", "deal_id", deal.ID, "error", err)
			}
		}

		// Crash recovery: if last lock is expired and still locked, release it as failed and let next tick retry.
		lastLock, err := s.dealActionLockRepo.GetLastDealActionLock(ctx, deal.ID, entity.DealActionTypePostMessage)
		if err != nil && !errors.Is(err, marketerrors.ErrNotFound) {
			logger.Error("get last lock", "deal_id", deal.ID, "error", err)
			continue
		}
		if err == nil && lastLock.Status == entity.DealActionLockStatusLocked && !lastLock.ExpireAt.After(time.Now()) {
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
