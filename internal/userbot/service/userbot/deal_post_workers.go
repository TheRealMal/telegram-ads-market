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
	postSenderInterval   = 2 * time.Minute
	postCheckerInterval  = 5 * time.Minute
	postCheckAdvanceHour = time.Hour
)

// RunDealPostSenderWorker lists deals escrow_deposit_confirmed without a post message, sends the post text to the listing's channel, and creates deal_post_message.
func (s *service) RunDealPostSenderWorker(ctx context.Context, repo marketRepository) {
	ticker := time.NewTicker(postSenderInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runDealPostSenderOnce(ctx, repo)
		}
	}
}

func (s *service) runDealPostSenderOnce(ctx context.Context, repo marketRepository) {
	deals, err := repo.ListDealsEscrowDepositConfirmedWithoutPostMessage(ctx)
	if err != nil {
		slog.Error("deal post sender: list deals", "error", err)
		return
	}
	for _, deal := range deals {
		listing, err := repo.GetListingByID(ctx, deal.ListingID)
		if err != nil || listing == nil || listing.ChannelID == nil {
			slog.Debug("deal post sender: skip deal, no listing or channel", "deal_id", deal.ID)
			continue
		}
		channel, err := repo.GetChannelByID(ctx, *listing.ChannelID)
		if err != nil || channel == nil {
			slog.Debug("deal post sender: skip deal, channel not found", "deal_id", deal.ID, "channel_id", *listing.ChannelID)
			continue
		}
		text := domain.GetMessageFromDetails(deal.Details)
		if text == "" {
			slog.Debug("deal post sender: skip deal, no message in details", "deal_id", deal.ID)
			continue
		}

		lockID, err := repo.TakeDealActionLock(ctx, deal.ID, marketentity.DealActionTypePostMessage)
		if err != nil {
			slog.Debug("deal post sender: skip deal, lock not acquired", "deal_id", deal.ID, "error", err)
			continue
		}
		releaseLock := func(status marketentity.DealActionLockStatus) {
			_ = repo.ReleaseDealActionLock(ctx, lockID, status)
		}

		msgID, err := s.sendChannelMessage(ctx, *listing.ChannelID, channel.AccessHash, text)
		if err != nil {
			slog.Error("deal post sender: send message", "deal_id", deal.ID, "error", err)
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
		if err := repo.CreateDealPostMessage(ctx, m); err != nil {
			slog.Error("deal post sender: create deal_post_message", "deal_id", deal.ID, "error", err)
			releaseLock(marketentity.DealActionLockStatusFailed)
		} else {
			releaseLock(marketentity.DealActionLockStatusCompleted)
			slog.Info("deal post sender: sent and saved", "deal_id", deal.ID, "channel_id", *listing.ChannelID, "message_id", msgID)
		}
	}
}

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
func (s *service) RunDealPostCheckerWorker(ctx context.Context, repo marketRepository) {
	ticker := time.NewTicker(postCheckerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runDealPostCheckerOnce(ctx, repo)
		}
	}
}

func (s *service) runDealPostCheckerOnce(ctx context.Context, repo marketRepository) {
	list, err := repo.ListDealPostMessageExistsWithNextCheckBefore(ctx, time.Now())
	if err != nil {
		slog.Error("deal post checker: list", "error", err)
		return
	}
	for _, m := range list {
		channel, err := repo.GetChannelByID(ctx, m.ChannelID)
		if err != nil || channel == nil {
			continue
		}
		exists, err := s.getChannelMessageExists(ctx, m.ChannelID, channel.AccessHash, m.MessageID)
		if err != nil {
			slog.Error("deal post checker: get message", "id", m.ID, "error", err)
			continue
		}
		if !exists {
			_ = repo.UpdateDealPostMessageStatus(ctx, m.ID, marketentity.DealPostMessageStatusDeleted)
			slog.Info("deal post checker: message deleted", "id", m.ID, "deal_id", m.DealID)
			continue
		}
		nextCheck := m.NextCheck.Add(postCheckAdvanceHour)
		if nextCheck.After(m.UntilTs) {
			_ = repo.UpdateDealPostMessageStatus(ctx, m.ID, marketentity.DealPostMessageStatusPassed)
			slog.Info("deal post checker: passed", "id", m.ID, "deal_id", m.DealID)
		} else {
			_ = repo.UpdateDealPostMessageStatusAndNextCheck(ctx, m.ID, marketentity.DealPostMessageStatusExists, nextCheck)
		}
	}
}

// getChannelMessageExists returns true if the message exists in the channel.
func (s *service) getChannelMessageExists(ctx context.Context, channelID int64, accessHash int64, messageID int64) (bool, error) {
	channel := &tg.InputChannel{ChannelID: channelID, AccessHash: accessHash}
	req := &tg.ChannelsGetMessagesRequest{
		Channel: channel,
		ID:      []tg.InputMessageClass{&tg.InputMessageID{ID: int(messageID)}},
	}
	result, err := s.telegramClient.API().ChannelsGetMessages(ctx, req)
	if err != nil {
		return false, err
	}
	messages, ok := result.(*tg.MessagesMessages)
	if !ok {
		return false, nil
	}
	return len(messages.Messages) > 0, nil
}
