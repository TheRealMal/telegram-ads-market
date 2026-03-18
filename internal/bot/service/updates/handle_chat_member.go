package updates

import (
	"context"
	"log/slog"
	"strconv"

	"ads-mrkt/internal/helpers/telegram"
	marketentity "ads-mrkt/internal/market/domain/entity"
	"ads-mrkt/internal/market/repository/deal/model"
)

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
		if newStatus == marketentity.BotMemberStatusLeft ||
			newStatus == marketentity.BotMemberStatusKicked ||
			newStatus == marketentity.BotMemberStatusMember {
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
	wasAdmin := oldStatus == marketentity.BotMemberStatusAdministrator || oldStatus == marketentity.BotMemberStatusCreator
	isAdmin := newStatus == marketentity.BotMemberStatusAdministrator || newStatus == marketentity.BotMemberStatusCreator

	if wasAdmin && !isAdmin {
		s.handleAdminRemoved(ctx, channelID, userID)
	}

	return nil
}

func (s *service) handleAdminRemoved(ctx context.Context, channelID, userID int64) {
	if err := s.channelAdminRepo.DeleteChannelAdmin(ctx, userID, channelID); err != nil {
		slog.Error("delete channel admin on removal", "error", err, "channel_id", channelID, "user_id", userID)
	}

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
