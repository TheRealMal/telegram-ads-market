package updates

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	marketentity "ads-mrkt/internal/market/domain/entity"
	"ads-mrkt/internal/worker"
)

const (
	adminMonitorInterval = 2 * time.Hour
)

func (s *service) RunAdminMonitorWorker(ctx context.Context) {
	worker.RunTicker(ctx, "admin_monitor_worker", adminMonitorInterval, false, s.runAdminMonitorOnce)
}

func (s *service) runAdminMonitorOnce(ctx context.Context, logger *slog.Logger) {
	channels, err := s.channelRepo.ListChannelsWithBotAccess(ctx)
	if err != nil {
		logger.Error("list channels", "error", err)
		return
	}

	for _, ch := range channels {
		if err := s.syncChannelAdmins(ctx, logger, ch.ID); err != nil {
			logger.Error("sync admins", "channel_id", ch.ID, "error", err)
			continue
		}
		logger.Debug("synced", "channel_id", ch.ID)

		s.syncChannelPhoto(ctx, logger, ch.ID)
	}
}

func (s *service) syncChannelPhoto(ctx context.Context, logger *slog.Logger, channelID int64) {
	photo, err := s.telegramClient.GetChannelPhotoBase64(ctx, channelID)
	if err != nil {
		logger.Error("sync channel photo: fetch", "channel_id", channelID, "error", err)
		return
	}
	if photo == "" {
		return
	}
	if err := s.channelRepo.UpdateChannelPhoto(ctx, channelID, photo); err != nil {
		logger.Error("sync channel photo: update", "channel_id", channelID, "error", err)
	}
}

func (s *service) syncChannelAdmins(ctx context.Context, logger *slog.Logger, channelID int64) error {
	admins, err := s.telegramClient.GetChatAdministrators(ctx, channelID)
	if err != nil {
		return fmt.Errorf("getChatAdministrators: %w", err)
	}

	if err := s.channelAdminRepo.DeleteChannelAdmins(ctx, channelID); err != nil {
		return fmt.Errorf("delete channel admins: %w", err)
	}

	for _, admin := range admins {
		if admin.User == nil {
			continue
		}

		// Extract the bot's own admin rights
		if admin.User.IsBot && admin.User.ID == s.botUserID {
			botRights := mapBotAdminRights(admin)
			if err := s.channelRepo.UpdateBotAdminRights(ctx, channelID, botRights); err != nil {
				logger.Error("update bot admin rights", "channel_id", channelID, "error", err)
			}
			continue
		}

		if admin.User.IsBot {
			continue
		}

		role := "admin"
		if marketentity.BotMemberStatus(admin.Status) == marketentity.BotMemberStatusCreator {
			role = "owner"
		}
		if err := s.channelAdminRepo.UpsertChannelAdmin(ctx, admin.User.ID, channelID, role); err != nil {
			logger.Error("upsert admin", "user_id", admin.User.ID, "channel_id", channelID, "error", err)
			continue
		}
		logger.Info("synced admin", "channel_id", channelID, "user_id", admin.User.ID, "role", role)
	}

	return nil
}
