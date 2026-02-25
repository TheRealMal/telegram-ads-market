package service

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"ads-mrkt/internal/market/domain"
	marketentity "ads-mrkt/internal/market/domain/entity"

	"github.com/gotd/td/tg"
)

const (
	postSenderInterval   = 1 * time.Minute
	postCheckerInterval  = 5 * time.Minute
	postCheckAdvanceHour = time.Hour
)

func (s *service) RunDealPostSenderWorker(ctx context.Context) {
	logger := slog.With("component", "deal_post_sendr")
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
		channel, err := s.channelRepo.GetChannelByID(ctx, *listing.ChannelID)
		if err != nil || channel == nil {
			logger.Error("skip deal, channel not found", "deal_id", deal.ID, "channel_id", *listing.ChannelID)
			continue
		}
		text := domain.GetMessageFromDetails(deal.Details)
		if text == "" {
			logger.Error("skip deal, no message in details", "deal_id", deal.ID)
			continue
		}
		if postedAt, ok := domain.GetPostedAtFromDetails(deal.Details); ok && time.Now().Before(postedAt) {
			logger.Debug("skip deal, posted_at in future", "deal_id", deal.ID, "posted_at", postedAt)
			continue
		}

		// If the last lock for post_message is in status Locked and expired, previous run may have posted then crashed: try to find the message in the channel.
		lastLock, err := s.dealActionLockRepo.GetLastDealActionLock(ctx, deal.ID, marketentity.DealActionTypePostMessage)
		if err != nil {
			logger.Error("get last lock", "deal_id", deal.ID, "error", err)
			continue
		}
		if lastLock != nil && lastLock.Status == marketentity.DealActionLockStatusLocked && !lastLock.ExpireAt.After(time.Now()) {
			if foundMsgID, found := s.tryRecoverPostFromChannel(ctx, *listing.ChannelID, channel.AccessHash, text); found {
				untilTs := time.Now().Add(time.Duration(deal.Duration) * time.Hour)
				nextCheck := time.Now().Add(postCheckAdvanceHour)
				m := &marketentity.DealPostMessage{
					DealID:      deal.ID,
					ChannelID:   *listing.ChannelID,
					MessageID:   foundMsgID,
					PostMessage: text,
					Status:      marketentity.DealPostMessageStatusExists,
					NextCheck:   nextCheck,
					UntilTs:     untilTs,
				}
				if err := s.dealPostMessageRepo.CreateDealPostMessageAndSetDealInProgress(ctx, m); err != nil {
					logger.Error("recover create deal_post_message", "deal_id", deal.ID, "error", err)
					_ = s.dealActionLockRepo.ReleaseDealActionLock(ctx, lastLock.ID, marketentity.DealActionLockStatusFailed)
					continue
				}
				_ = s.dealActionLockRepo.ReleaseDealActionLock(ctx, lastLock.ID, marketentity.DealActionLockStatusCompleted)
				logger.Info("recovered post from channel", "deal_id", deal.ID, "channel_id", *listing.ChannelID, "message_id", foundMsgID)
				continue
			}
			_ = s.dealActionLockRepo.ReleaseDealActionLock(ctx, lastLock.ID, marketentity.DealActionLockStatusFailed)
		}

		lockID, err := s.dealActionLockRepo.TakeDealActionLock(ctx, deal.ID, marketentity.DealActionTypePostMessage)
		if err != nil {
			logger.Debug("skip deal, lock not acquired", "deal_id", deal.ID, "error", err)
			continue
		}
		releaseLock := func(status marketentity.DealActionLockStatus) {
			_ = s.dealActionLockRepo.ReleaseDealActionLock(ctx, lockID, status)
		}

		msgID, err := s.sendChannelMessage(ctx, *listing.ChannelID, channel.AccessHash, text)
		if err != nil {
			logger.Error("send message", "deal_id", deal.ID, "error", err)
			releaseLock(marketentity.DealActionLockStatusFailed)
			continue
		}
		untilTs := time.Now().Add(time.Duration(deal.Duration) * time.Hour)
		nextCheck := time.Now().Add(postCheckAdvanceHour)
		m := &marketentity.DealPostMessage{
			DealID:      deal.ID,
			ChannelID:   *listing.ChannelID,
			MessageID:   msgID,
			PostMessage: text,
			Status:      marketentity.DealPostMessageStatusExists,
			NextCheck:   nextCheck,
			UntilTs:     untilTs,
		}
		if err := s.dealPostMessageRepo.CreateDealPostMessageAndSetDealInProgress(ctx, m); err != nil {
			logger.Error("create deal_post_message", "deal_id", deal.ID, "error", err)
			releaseLock(marketentity.DealActionLockStatusFailed)
			continue
		}
		releaseLock(marketentity.DealActionLockStatusCompleted)
		logger.Info("sent and saved", "deal_id", deal.ID, "channel_id", *listing.ChannelID, "message_id", msgID)

	}
}

const lastMessagesRecoveryLimit = 20

func (s *service) sendChannelMessage(ctx context.Context, channelID int64, accessHash int64, text string) (int64, error) {
	peer := &tg.InputPeerChannel{ChannelID: channelID, AccessHash: accessHash}
	req := &tg.MessagesSendMessageRequest{
		Peer:     peer,
		Message:  text,
		RandomID: rand.Int63(),
	}
	result, err := s.telegramClient.API().MessagesSendMessage(ctx, req)
	if err != nil {
		return 0, err
	}
	upd, ok := result.(*tg.Updates)
	if !ok {
		return 0, nil
	}
	for _, u := range upd.Updates {
		if msg, ok := u.(*tg.UpdateMessageID); ok {
			return int64(msg.ID), nil
		}
	}
	return 0, nil
}

func (s *service) RunDealPostCheckerWorker(ctx context.Context) {
	logger := slog.With("component", "deal_post_checker")
	ticker := time.NewTicker(postCheckerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runDealPostCheckerOnce(ctx, logger)
		}
	}
}

func (s *service) runDealPostCheckerOnce(ctx context.Context, logger *slog.Logger) {
	list, err := s.dealPostMessageRepo.ListDealPostMessageExistsWithNextCheckBefore(ctx, time.Now())
	if err != nil {
		logger.Error("list", "error", err)
		return
	}

	for _, m := range list {
		channel, err := s.channelRepo.GetChannelByID(ctx, m.ChannelID)
		if err != nil || channel == nil {
			continue
		}
		exists, err := s.getChannelMessageExists(ctx, m.ChannelID, channel.AccessHash, m.MessageID)
		if err != nil {
			logger.Error("get message", "id", m.ID, "error", err)
			continue
		}
		if !exists {
			_ = s.dealPostMessageRepo.UpdateDealPostMessageStatus(ctx, m.ID, marketentity.DealPostMessageStatusDeleted)
			logger.Info("message deleted", "id", m.ID, "deal_id", m.DealID)
			continue
		}
		nextCheck := m.NextCheck.Add(postCheckAdvanceHour)
		if nextCheck.After(m.UntilTs) {
			_ = s.dealPostMessageRepo.UpdateDealPostMessageStatus(ctx, m.ID, marketentity.DealPostMessageStatusPassed)
			logger.Info("passed", "id", m.ID, "deal_id", m.DealID)
		} else {
			_ = s.dealPostMessageRepo.UpdateDealPostMessageStatusAndNextCheck(ctx, m.ID, marketentity.DealPostMessageStatusExists, nextCheck)
		}
	}
}

type channelMessage struct {
	ID   int64
	Text string
}

// See https://core.telegram.org/api/offsets: offset = offsetFromID(offsetID) + addOffset; results are reverse chronological (newest first).
// For "most recent N": offsetID=0, addOffset=0, limit=N.
// For "around message ID": offsetID=messageID, addOffset=-halfWindow, limit=windowSize.
func (s *service) getChannelHistory(ctx context.Context, channelID int64, accessHash int64, offsetID int, addOffset int, limit int) ([]channelMessage, error) {
	result, err := s.telegramClient.API().MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer: &tg.InputPeerChannel{
			ChannelID:  channelID,
			AccessHash: accessHash,
		},
		OffsetID:   offsetID,
		OffsetDate: 0,
		AddOffset:  addOffset,
		Limit:      limit,
		MaxID:      0,
		MinID:      0,
		Hash:       0,
	})
	if err != nil {
		return nil, err
	}
	var messages []tg.MessageClass
	switch r := result.(type) {
	case *tg.MessagesMessages:
		messages = r.Messages
	case *tg.MessagesChannelMessages:
		messages = r.Messages
	default:
		return nil, nil
	}
	out := make([]channelMessage, 0, len(messages))
	for _, msg := range messages {
		m, ok := msg.(*tg.Message)
		if !ok {
			continue
		}
		out = append(out, channelMessage{ID: int64(m.ID), Text: m.Message})
	}
	return out, nil
}

func (s *service) getChannelMessageExists(ctx context.Context, channelID int64, accessHash int64, messageID int64) (bool, error) {
	const windowAround = 20
	msgs, err := s.getChannelHistory(ctx, channelID, accessHash, int(messageID), -windowAround/2, windowAround)
	if err != nil {
		return false, err
	}
	for _, m := range msgs {
		if m.ID == messageID {
			return true, nil
		}
	}
	return false, nil
}

func (s *service) tryRecoverPostFromChannel(ctx context.Context, channelID int64, accessHash int64, text string) (int64, bool) {
	msgs, err := s.getChannelHistory(ctx, channelID, accessHash, 0, 0, lastMessagesRecoveryLimit)
	if err != nil {
		return 0, false
	}
	for _, m := range msgs {
		if m.Text == text {
			return m.ID, true
		}
	}
	return 0, false
}
