package event

import (
	"context"
	"log/slog"
	"time"

	"ads-mrkt/internal/event/domain/entity"

	"github.com/redis/go-redis/v9"
)

const (
	maxReadLimit int64 = 100
	minReadLimit int64 = 1
)

var eventChannelUpdateStatsStream = (*entity.EventChannelUpdateStats)(nil).StreamKey()

func (s *Service) AddChannelUpdateStatsEvent(ctx context.Context, channelID int64) error {
	return s.repository.PushEvent(ctx, &entity.EventChannelUpdateStats{ChannelID: channelID})
}

func (s *Service) ReadChannelUpdateStatsEvents(ctx context.Context, group string, consumer string, limit int64) ([]*entity.EventChannelUpdateStats, error) {
	args := &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{eventChannelUpdateStatsStream, ">"},
		Block:    time.Millisecond * 100, //nolint:revive
		Count:    max(minReadLimit, min(maxReadLimit, limit)),
	}

	streamMessages, err := s.repository.ReadEvents(ctx, args)
	if err != nil {
		return nil, err
	}

	events := make([]*entity.EventChannelUpdateStats, 0, len(streamMessages))
	for _, msg := range streamMessages {
		event := createEvent(msg)
		if event != nil {
			events = append(events, event)
		} else {
			slog.Info("can't parse event channel_update_stats", "event", msg)
			if errAck := s.AckChannelUpdateStatsMessages(ctx, group, []string{msg.ID}); errAck != nil {
				slog.Error("cannot ack message channel_update_stats", "err", errAck)
			}
		}
	}

	return events, nil
}

func (s *Service) AckChannelUpdateStatsMessages(ctx context.Context, group string, messageIDs []string) error {
	return s.repository.AckMessages(ctx, eventChannelUpdateStatsStream, group, messageIDs)
}

func createEvent(streamMessage redis.XMessage) *entity.EventChannelUpdateStats {
	e := &entity.EventChannelUpdateStats{
		ID: streamMessage.ID,
	}
	e.FromMap(streamMessage.Values)
	return e
}
