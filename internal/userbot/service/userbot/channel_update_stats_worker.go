package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	evententity "ads-mrkt/internal/event/domain/entity"
	"ads-mrkt/internal/worker"
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

func (s *service) RunChannelUpdateStatsWorker(ctx context.Context) {
	go s.channelUpdateStatsStreamCleaner(ctx)
	go s.runPendingChannelUpdateStatsWorker(ctx)
	s.runChannelUpdateStatsWorker(ctx)
}

func (s *service) runChannelUpdateStatsWorker(ctx context.Context) {
	worker.RunTicker(ctx, "channel_update_stats_reader", channelUpdateStatsInterval, false, func(ctx context.Context, logger *slog.Logger) {
		events, err := s.channelUpdateStatsEventSvc.ReadChannelUpdateStatsEvents(ctx, channelUpdateStatsGroup, channelUpdateStatsConsumer, channelUpdateStatsLimit)
		if err != nil || len(events) == 0 {
			return
		}
		s.processChannelUpdateStatsEvents(ctx, logger, events)
	})
}

func (s *service) channelUpdateStatsStreamCleaner(ctx context.Context) {
	worker.RunTicker(ctx, "channel_update_stats_stream_cleaner", 24*time.Hour, false, func(ctx context.Context, logger *slog.Logger) {
		if err := s.channelUpdateStatsEventSvc.TrimStreamByAge(ctx, channelUpdateStatsStreamMaxAge); err != nil {
			logger.Error("trim stream by age", "err", err)
		}
	})
}

func (s *service) runPendingChannelUpdateStatsWorker(ctx context.Context) {
	worker.RunTicker(ctx, "channel_update_stats_pending_processor", channelUpdateStatsPendingPeriod, false, func(ctx context.Context, logger *slog.Logger) {
		events, err := s.channelUpdateStatsEventSvc.PendingChannelUpdateStatsEvents(ctx, channelUpdateStatsGroup, channelUpdateStatsConsumer, channelUpdateStatsLimit, channelUpdateStatsPendingMinIdle)
		if err != nil {
			logger.Error("read pending events", "error", err)
			return
		}
		if len(events) == 0 {
			return
		}
		s.processChannelUpdateStatsEvents(ctx, logger, events)
	})
}

func (s *service) processChannelUpdateStatsEvents(ctx context.Context, logger *slog.Logger, events []*evententity.EventChannelUpdateStats) {
	var ids []string
	for _, ev := range events {
		ch, err := s.channelRepo.GetChannelByID(ctx, ev.ChannelID)
		if err != nil {
			logger.Error("get channel", "channel_id", ev.ChannelID, "error", err)
			ids = append(ids, ev.ID)
			continue
		}
		if ch == nil {
			ids = append(ids, ev.ID)
			continue
		}
		ch, err = s.ensureChannelAccess(ctx, ch)
		if err != nil {
			if errors.Is(err, ErrSkipThatIteration) {
				logger.Info("channel access ensured, skipping that iteration to restart process on next iteration")
				continue
			}
			logger.Error("ensure channel access for stats", "channel_id", ev.ChannelID, "error", err)
			ids = append(ids, ev.ID)
			continue
		}
		if ch.AdminRights.CanViewStats {
			logger.Info("updating channel stats", "channel_id", ev.ChannelID)
			if err := s.UpdateChannelStats(ctx, ev.ChannelID, ch.AccessHash, 0); err != nil {
				logger.Error("update channel stats", "channel_id", ev.ChannelID, "error", err)
				continue
			}
		}
		ids = append(ids, ev.ID)
	}
	if len(ids) > 0 {
		_ = s.channelUpdateStatsEventSvc.AckChannelUpdateStatsMessages(ctx, channelUpdateStatsGroup, ids)
	}
}
