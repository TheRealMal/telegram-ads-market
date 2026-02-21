package service

import (
	"context"
	"log/slog"
	"time"

	evententity "ads-mrkt/internal/event/domain/entity"
)

const (
	channelUpdateStatsGroup    = "userbot"
	channelUpdateStatsConsumer = "channel-update-stats"
	channelUpdateStatsLimit    = 10
	channelUpdateStatsInterval = 2 * time.Second

	channelUpdateStatsPendingPeriod  = 15 * time.Second
	channelUpdateStatsPendingMinIdle = 30 * time.Second
	channelUpdateStatsStreamMaxAge   = 7 * 24 * time.Hour
)

// RunChannelUpdateStatsWorker runs the main event loop and starts the stream cleaner and pending processor in the background.
func (s *service) RunChannelUpdateStatsWorker(ctx context.Context) {
	logger := slog.With("component", "channel_update_stats_worker")

	go s.channelUpdateStatsStreamCleaner(ctx, logger)
	go s.processPendingChannelUpdateStats(ctx, logger)
	s.runChannelUpdateStatsWorker(ctx, logger)
}

func (s *service) runChannelUpdateStatsWorker(ctx context.Context, logger *slog.Logger) {
	ticker := time.NewTicker(channelUpdateStatsInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Info("channel update stats worker stopped")
			return
		case <-ticker.C:
			events, err := s.channelUpdateStatsEventSvc.ReadChannelUpdateStatsEvents(ctx, channelUpdateStatsGroup, channelUpdateStatsConsumer, channelUpdateStatsLimit)
			if err != nil || len(events) == 0 {
				continue
			}
			s.processChannelUpdateStatsEvents(ctx, logger, events)
		}
	}
}

func (s *service) channelUpdateStatsStreamCleaner(ctx context.Context, logger *slog.Logger) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Info("channel update stats stream cleaner stopped")
			return
		case <-ticker.C:
			if err := s.channelUpdateStatsEventSvc.TrimStreamByAge(ctx, channelUpdateStatsStreamMaxAge); err != nil {
				logger.Error("trim channel_update_stats stream by age", "err", err)
			}
			ticker.Reset(24 * time.Hour)
		}
	}
}

func (s *service) processPendingChannelUpdateStats(ctx context.Context, logger *slog.Logger) {
	ticker := time.NewTicker(channelUpdateStatsPendingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Info("pending channel update stats processor stopped")
			return
		case <-ticker.C:
			events, err := s.channelUpdateStatsEventSvc.PendingChannelUpdateStatsEvents(ctx, channelUpdateStatsGroup, channelUpdateStatsConsumer, channelUpdateStatsLimit, channelUpdateStatsPendingMinIdle)
			if err != nil {
				logger.Error("read pending channel_update_stats events", "error", err)
				ticker.Reset(channelUpdateStatsPendingPeriod)
				continue
			}
			if len(events) == 0 {
				ticker.Reset(channelUpdateStatsPendingPeriod)
				continue
			}
			s.processChannelUpdateStatsEvents(ctx, logger, events)
			ticker.Reset(channelUpdateStatsPendingPeriod)
		}
	}
}

func (s *service) processChannelUpdateStatsEvents(ctx context.Context, logger *slog.Logger, events []*evententity.EventChannelUpdateStats) {
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
		if ch.AdminRights.CanViewStats {
			slog.Info("updating channel stats", "channel_id", ev.ChannelID)
			if err := s.UpdateChannelStats(ctx, ev.ChannelID, ch.AccessHash, 0); err != nil {
				logger.Error("update channel stats", "channel_id", ev.ChannelID, "error", err)
				continue
			}
		}
		s.updateChannelPhotoFromTelegram(ctx, ev.ChannelID, ch.AccessHash)
		ids = append(ids, ev.ID)
	}
	if len(ids) > 0 {
		_ = s.channelUpdateStatsEventSvc.AckChannelUpdateStatsMessages(ctx, channelUpdateStatsGroup, ids)
	}
}
