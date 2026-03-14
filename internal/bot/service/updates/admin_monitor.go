package updates

import (
	"context"
	"log/slog"
	"time"
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
		if err := s.syncChannelAdminsViaBotAPI(ctx, ch.ID); err != nil {
			slog.Error("admin monitor: sync admins", "channel_id", ch.ID, "error", err)
			continue
		}
		slog.Debug("admin monitor: synced", "channel_id", ch.ID)
	}
}
