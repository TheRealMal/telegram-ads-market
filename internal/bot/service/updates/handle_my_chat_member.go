package updates

import (
	"context"
	"fmt"
	"log/slog"

	evententity "ads-mrkt/internal/event/domain/entity"
	"ads-mrkt/internal/helpers/telegram"
	marketentity "ads-mrkt/internal/market/domain/entity"
)

func (s *service) handleMyChatMember(ctx context.Context, update *telegram.ChatMemberUpdated) error {
	if update.Chat == nil {
		return nil
	}

	if update.Chat.Type != "channel" {
		return nil
	}

	newStatusStr := update.NewChatMember.Status
	newStatus := marketentity.BotMemberStatus(newStatusStr)
	channelID := update.Chat.ID

	slog.Info("bot chat member status changed",
		"channel_id", channelID,
		"title", update.Chat.Title,
		"new_status", newStatusStr,
	)

	if newStatus == marketentity.BotMemberStatusAdministrator {
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

		botRights := mapBotAdminRights(update.NewChatMember)
		if err := s.channelRepo.UpdateBotAdminRights(ctx, channelID, botRights); err != nil {
			slog.Error("update bot admin rights on promotion", "channel_id", channelID, "error", err)
		}

		if err := s.syncChannelAdmins(ctx, slog.Default(), channelID); err != nil {
			slog.Error("sync admins after bot added", "channel_id", channelID, "error", err)
		}

		s.syncChannelPhoto(ctx, slog.Default(), channelID)

		return nil
	}

	// Bot was demoted to member, left, or kicked — deactivate listings and cancel deals
	if newStatus == marketentity.BotMemberStatusMember ||
		newStatus == marketentity.BotMemberStatusLeft ||
		newStatus == marketentity.BotMemberStatusKicked {
		if err := s.channelRepo.UpdateChannelBotMemberStatus(ctx, channelID, newStatus); err != nil {
			slog.Error("update bot member status on removal", "channel_id", channelID, "error", err)
		}
		s.handleBotRemovedOrDemoted(ctx, channelID)
		return nil
	}

	return nil
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

	// Notify all affected deal parties in their deal topics
	for _, d := range rejectedDeals {
		topic, _ := s.dealForumTopicRepo.GetDealForumTopicByDealID(ctx, d.ID)
		for _, userID := range []int64{d.LessorID, d.LesseeID} {
			notification := &evententity.EventTelegramNotification{
				ChatID:  userID,
				Message: "Deal was canceled because the bot was removed from the channel.",
			}
			if topic != nil {
				if userID == d.LessorID {
					notification.ThreadID = topic.LessorMessageThreadID
				} else {
					notification.ThreadID = topic.LesseeMessageThreadID
				}
			}
			if err := s.notificationAdder.AddTelegramNotificationEvent(ctx, notification); err != nil {
				slog.Error("failed to add notification", "type", "bot_removed", "deal_id", d.ID, "user_id", userID, "error", err)
			}
		}
	}
}
