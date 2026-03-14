package updates

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"ads-mrkt/internal/helpers/telegram"
	marketentity "ads-mrkt/internal/market/domain/entity"
	"ads-mrkt/internal/market/repository/deal/model"
)

func (s *service) handleMyChatMember(ctx context.Context, update *telegram.ChatMemberUpdated) error {
	if update.Chat == nil {
		return nil
	}

	if update.Chat.Type != "channel" {
		return nil
	}

	newStatus := update.NewChatMember.Status
	channelID := update.Chat.ID

	slog.Info("bot chat member status changed",
		"channel_id", channelID,
		"title", update.Chat.Title,
		"new_status", newStatus,
	)

	if newStatus == "administrator" {
		channel := &marketentity.Channel{
			ID:              channelID,
			AccessHash:      0,
			BotMemberStatus: newStatus,
			Title:           update.Chat.Title,
			Username:        update.Chat.Username,
		}
		if err := s.channelRepo.UpsertChannel(ctx, channel); err != nil {
			return fmt.Errorf("upsert channel on bot added: %w", err)
		}

		if err := s.syncChannelAdminsViaBotAPI(ctx, channelID); err != nil {
			slog.Error("sync admins after bot added", "channel_id", channelID, "error", err)
		}
		return nil
	}

	// Bot was demoted to member, left, or kicked — deactivate listings and cancel deals
	if newStatus == "member" || newStatus == "left" || newStatus == "kicked" {
		if err := s.channelRepo.UpdateChannelBotMemberStatus(ctx, channelID, newStatus); err != nil {
			slog.Error("update bot member status on removal", "channel_id", channelID, "error", err)
		}
		s.handleBotRemovedOrDemoted(ctx, channelID)
		return nil
	}

	return nil
}

func (s *service) handleChatMember(ctx context.Context, update *telegram.ChatMemberUpdated) error {
	if update.Chat == nil || update.Chat.Type != "channel" {
		return nil
	}

	if update.NewChatMember == nil || update.NewChatMember.User == nil {
		return nil
	}

	channelID := update.Chat.ID
	userID := update.NewChatMember.User.ID
	newStatus := update.NewChatMember.Status

	slog.Info("channel chat member status changed",
		"channel_id", channelID,
		"user_id", userID,
		"new_status", newStatus,
	)

	// Check if this is the userbot being removed/demoted
	userbotID, err := s.userbotStateRepo.GetUserbotUserID(ctx)
	if err == nil && userID == userbotID {
		if newStatus == "left" || newStatus == "kicked" || newStatus == "member" {
			if err := s.channelRepo.ResetChannelAccessHash(ctx, channelID); err != nil {
				slog.Error("reset access_hash on userbot removal", "channel_id", channelID, "error", err)
			}
			slog.Info("userbot removed from channel, reset access_hash", "channel_id", channelID)
		}
		return nil
	}

	// Skip bots
	if update.NewChatMember.User.IsBot {
		return nil
	}

	// Check if admin/owner was removed or demoted
	oldStatus := ""
	if update.OldChatMember != nil {
		oldStatus = update.OldChatMember.Status
	}
	wasAdmin := oldStatus == "administrator" || oldStatus == "creator"
	isAdmin := newStatus == "administrator" || newStatus == "creator"

	if wasAdmin && !isAdmin {
		s.handleAdminRemoved(ctx, channelID, userID)
	}

	return nil
}

func (s *service) handleAdminRemoved(ctx context.Context, channelID, userID int64) {
	count, err := s.listingRepo.DeactivateListingsByUserAndChannel(ctx, userID, channelID)
	if err != nil {
		slog.Error("deactivate listings on admin removal", "error", err, "channel_id", channelID, "user_id", userID)
	} else if count > 0 {
		slog.Info("deactivated listings on admin removal", "channel_id", channelID, "user_id", userID, "count", count)
	}

	rejectedDeals, err := s.dealRepo.RejectDealsByUserAndChannel(ctx, userID, channelID)
	if err != nil {
		slog.Error("reject deals on admin removal", "error", err, "channel_id", channelID, "user_id", userID)
	}

	msg := "You have been removed as admin from a channel. Your listings have been deactivated"
	if len(rejectedDeals) > 0 {
		msg += " and related deals have been canceled"
	}
	msg += "."
	_ = s.notificationAdder.AddTelegramNotificationEvent(ctx, userID, msg)

	s.notifyDealCounterparties(ctx, rejectedDeals, userID)
}

func (s *service) handleBotRemovedOrDemoted(ctx context.Context, channelID int64) {
	count, err := s.listingRepo.DeactivateListingsByChannel(ctx, channelID)
	if err != nil {
		slog.Error("deactivate listings on bot removal", "error", err, "channel_id", channelID)
	} else if count > 0 {
		slog.Info("deactivated listings on bot removal", "channel_id", channelID, "count", count)
	}

	rejectedDeals, err := s.dealRepo.RejectDealsByChannel(ctx, channelID)
	if err != nil {
		slog.Error("reject deals on bot removal", "error", err, "channel_id", channelID)
	}

	if err := s.channelRepo.ResetChannelAccessHash(ctx, channelID); err != nil {
		slog.Error("reset access_hash on bot removal", "channel_id", channelID, "error", err)
	}

	// Notify all affected deal parties
	notified := make(map[int64]bool)
	for _, d := range rejectedDeals {
		for _, uid := range []int64{d.LessorID, d.LesseeID} {
			if notified[uid] {
				continue
			}
			notified[uid] = true
			_ = s.notificationAdder.AddTelegramNotificationEvent(ctx, uid,
				"Deal #"+strconv.FormatInt(d.ID, 10)+" was canceled because the bot was removed from the channel.")
		}
	}
}

func (s *service) notifyDealCounterparties(ctx context.Context, deals []model.RejectedDealRow, removedUserID int64) {
	for _, d := range deals {
		counterpartyID := d.LesseeID
		if removedUserID == d.LesseeID {
			counterpartyID = d.LessorID
		}
		_ = s.notificationAdder.AddTelegramNotificationEvent(ctx, counterpartyID,
			"Deal #"+strconv.FormatInt(d.ID, 10)+" was canceled because the channel admin was removed.")
	}
}

func (s *service) syncChannelAdminsViaBotAPI(ctx context.Context, channelID int64) error {
	admins, err := s.telegramClient.GetChatAdministrators(ctx, channelID)
	if err != nil {
		return fmt.Errorf("getChatAdministrators: %w", err)
	}

	if err := s.channelAdminRepo.DeleteChannelAdmins(ctx, channelID); err != nil {
		return fmt.Errorf("delete channel admins: %w", err)
	}

	for _, admin := range admins {
		if admin.User == nil || admin.User.IsBot {
			continue
		}
		role := "admin"
		if admin.Status == "creator" {
			role = "owner"
		}
		if err := s.channelAdminRepo.UpsertChannelAdmin(ctx, admin.User.ID, channelID, role); err != nil {
			slog.Error("upsert admin via bot api", "user_id", admin.User.ID, "channel_id", channelID, "error", err)
			continue
		}
		slog.Info("synced admin via bot api", "channel_id", channelID, "user_id", admin.User.ID, "role", role)
	}

	return nil
}
