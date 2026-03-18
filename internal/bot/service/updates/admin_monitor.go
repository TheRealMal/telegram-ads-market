package updates

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	marketentity "ads-mrkt/internal/market/domain/entity"
)

const (
	adminMonitorInterval = 2 * time.Hour
)

func (s *service) StartAdminMonitorWorker(ctx context.Context) {
	ticker := time.NewTicker(adminMonitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("admin monitor worker stopped")
			return
		case <-ticker.C:
			s.runAdminMonitorOnce(ctx)
		}
	}
}

func (s *service) runAdminMonitorOnce(ctx context.Context) {
	channels, err := s.channelRepo.ListChannelsWithBotAccess(ctx)
	if err != nil {
		slog.Error("admin monitor: list channels", "error", err)
		return
	}

	for _, ch := range channels {
		if err := s.syncChannelAdmins(ctx, ch.ID); err != nil {
			slog.Error("admin monitor: sync admins", "channel_id", ch.ID, "error", err)
			continue
		}
		slog.Debug("admin monitor: synced", "channel_id", ch.ID)

		s.syncChannelPhoto(ctx, ch.ID)
	}
}

func (s *service) syncChannelPhoto(ctx context.Context, channelID int64) {
	photo, err := s.telegramClient.GetChannelPhotoBase64(ctx, channelID)
	if err != nil {
		slog.Error("sync channel photo: fetch", "channel_id", channelID, "error", err)
		return
	}
	if photo == "" {
		return
	}
	if err := s.channelRepo.UpdateChannelPhoto(ctx, channelID, photo); err != nil {
		slog.Error("sync channel photo: update", "channel_id", channelID, "error", err)
	}
}

func (s *service) syncChannelAdmins(ctx context.Context, channelID int64) error {
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
		if admin.Status == marketentity.BotMemberStatusCreator {
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
