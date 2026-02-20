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

// RunDealPostSenderWorker lists deals escrow_deposit_confirmed without a post message, sends the post text to the listing's channel, and creates deal_post_message.
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
	deals, err := s.marketRepository.ListDealsEscrowDepositConfirmedWithoutPostMessage(ctx)
	if err != nil {
		logger.Error("list deals", "error", err)
		return
	}
	for _, deal := range deals {
		listing, err := s.marketRepository.GetListingByID(ctx, deal.ListingID)
		if err != nil || listing == nil || listing.ChannelID == nil {
			logger.Error("skip deal, no listing or channel", "deal_id", deal.ID)
			continue
		}
		channel, err := s.marketRepository.GetChannelByID(ctx, *listing.ChannelID)
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

		// If there is an expired post_message lock, previous run may have posted then crashed: try to find the message in the channel.
		expiredLockID, hasExpired, err := s.marketRepository.GetExpiredDealActionLock(ctx, deal.ID, marketentity.DealActionTypePostMessage)
		if err != nil {
			logger.Error("get expired lock", "deal_id", deal.ID, "error", err)
			continue
		}
		if hasExpired {
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
				if err := s.marketRepository.CreateDealPostMessageAndSetDealInProgress(ctx, m); err != nil {
					logger.Error("recover create deal_post_message", "deal_id", deal.ID, "error", err)
					_ = s.marketRepository.ReleaseDealActionLock(ctx, expiredLockID, marketentity.DealActionLockStatusFailed)
					continue
				}
				_ = s.marketRepository.ReleaseDealActionLock(ctx, expiredLockID, marketentity.DealActionLockStatusCompleted)
				logger.Info("recovered post from channel", "deal_id", deal.ID, "channel_id", *listing.ChannelID, "message_id", foundMsgID)
				continue
			}
			_ = s.marketRepository.ReleaseDealActionLock(ctx, expiredLockID, marketentity.DealActionLockStatusFailed)
		}

		lockID, err := s.marketRepository.TakeDealActionLock(ctx, deal.ID, marketentity.DealActionTypePostMessage)
		if err != nil {
			logger.Debug("skip deal, lock not acquired", "deal_id", deal.ID, "error", err)
			continue
		}
		releaseLock := func(status marketentity.DealActionLockStatus) {
			_ = s.marketRepository.ReleaseDealActionLock(ctx, lockID, status)
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
		if err := s.marketRepository.CreateDealPostMessageAndSetDealInProgress(ctx, m); err != nil {
			logger.Error("create deal_post_message", "deal_id", deal.ID, "error", err)
			releaseLock(marketentity.DealActionLockStatusFailed)
			continue
		}
		releaseLock(marketentity.DealActionLockStatusCompleted)
		logger.Info("sent and saved", "deal_id", deal.ID, "channel_id", *listing.ChannelID, "message_id", msgID)

	}
}

const lastMessagesRecoveryLimit = 20

// sendChannelMessage sends a text message to the channel and returns the message ID.
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

// RunDealPostCheckerWorker lists deal_post_message with status=exists and next_check <= now, checks if the message still exists, and updates status/next_check or sets passed.
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
	list, err := s.marketRepository.ListDealPostMessageExistsWithNextCheckBefore(ctx, time.Now())
	if err != nil {
		logger.Error("list", "error", err)
		return
	}

	for _, m := range list {
		channel, err := s.marketRepository.GetChannelByID(ctx, m.ChannelID)
		if err != nil || channel == nil {
			continue
		}
		exists, err := s.getChannelMessageExists(ctx, m.ChannelID, channel.AccessHash, m.MessageID)
		if err != nil {
			logger.Error("get message", "id", m.ID, "error", err)
			continue
		}
		if !exists {
			_ = s.marketRepository.UpdateDealPostMessageStatus(ctx, m.ID, marketentity.DealPostMessageStatusDeleted)
			logger.Info("message deleted", "id", m.ID, "deal_id", m.DealID)
			continue
		}
		nextCheck := m.NextCheck.Add(postCheckAdvanceHour)
		if nextCheck.After(m.UntilTs) {
			_ = s.marketRepository.UpdateDealPostMessageStatus(ctx, m.ID, marketentity.DealPostMessageStatusPassed)
			logger.Info("passed", "id", m.ID, "deal_id", m.DealID)
		} else {
			_ = s.marketRepository.UpdateDealPostMessageStatusAndNextCheck(ctx, m.ID, marketentity.DealPostMessageStatusExists, nextCheck)
		}
	}
}

// channelMessage is a message ID and text from a channel.
type channelMessage struct {
	ID   int64
	Text string
}

// getChannelHistory fetches channel history via MessagesGetHistory.
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

// getChannelMessageExists returns true if the message exists in the channel, by fetching a small window of history around that message ID (per Telegram offset semantics).
func (s *service) getChannelMessageExists(ctx context.Context, channelID int64, accessHash int64, messageID int64) (bool, error) {
	// "Around message MSGID": offset_id=MSGID, add_offset=-10, limit=20 (per core.telegram.org/api/offsets)
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

// tryRecoverPostFromChannel fetches the last messages in the channel and returns (messageID, true) if one matches text exactly.
func (s *service) tryRecoverPostFromChannel(ctx context.Context, channelID int64, accessHash int64, text string) (int64, bool) {
	// "Most recent N": offset_id=0, add_offset=0, limit=N (per core.telegram.org/api/offsets)
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
