package service

import (
	"context"
	"log/slog"
	"time"
)

const (
	channelUpdateStatsGroup    = "userbot"
	channelUpdateStatsConsumer = "channel-update-stats"
	channelUpdateStatsLimit    = 10
	channelUpdateStatsInterval = 2 * time.Second
)

// RunChannelUpdateStatsWorker reads channel_update_stats events and triggers UpdateChannelStats for each channel.
func (s *service) RunChannelUpdateStatsWorker(ctx context.Context, eventSvc channelUpdateStatsEventService) {
	logger := slog.With("component", "channel_update_stats_worker")
	ticker := time.NewTicker(channelUpdateStatsInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			events, err := eventSvc.ReadChannelUpdateStatsEvents(ctx, channelUpdateStatsGroup, channelUpdateStatsConsumer, channelUpdateStatsLimit)
			if err != nil || len(events) == 0 {
				continue
			}
			var ids []string
			for _, ev := range events {
				ch, err := s.marketRepository.GetChannelByID(ctx, ev.ChannelID)
				if err != nil {
					logger.Error("get channel", "channel_id", ev.ChannelID, "error", err)
					ids = append(ids, ev.ID)
					continue
				}
				if ch == nil {
					ids = append(ids, ev.ID)
					continue
				}
				if err := s.UpdateChannelStats(ctx, ev.ChannelID, ch.AccessHash); err != nil {
					logger.Error("update channel stats", "channel_id", ev.ChannelID, "error", err)
					continue
				}
				logger.Info("channel stats updated", "channel_id", ev.ChannelID)
				ids = append(ids, ev.ID)
			}
			if len(ids) > 0 {
				_ = eventSvc.AckChannelUpdateStatsMessages(ctx, channelUpdateStatsGroup, ids)
			}
		}
	}
}
