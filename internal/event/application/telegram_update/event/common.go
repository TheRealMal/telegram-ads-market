package event

import (
	"context"
	"log/slog"
	"time"

	"ads-mrkt/internal/event/domain/entity"
	"ads-mrkt/internal/helpers/telegram"

	"github.com/redis/go-redis/v9"
)

const (
	maxReadLimit int64 = 100
	minReadLimit int64 = 1
)

var eventTelegramUpdateStream = (*entity.EventTelegramUpdate)(nil).StreamKey()

func (s *Service) AddTelegramUpdateEvent(ctx context.Context, update *telegram.Update, createdAt time.Time) error {
	event := &entity.EventTelegramUpdate{
		Update:    update,
		Timestamp: createdAt.Unix(),
	}

	return s.repository.PushEvent(ctx, event)
}

func (s *Service) ReadTelegramUpdateEvents(ctx context.Context, group string, consumer string, limit int64) ([]*entity.EventTelegramUpdate, error) {
	args := &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{eventTelegramUpdateStream, ">"},
		Block:    time.Millisecond * 100, //nolint:revive
		Count:    max(minReadLimit, min(maxReadLimit, limit)),
	}

	streamMessages, err := s.repository.ReadEvents(ctx, args)
	if err != nil {
		return nil, err
	}

	events := []*entity.EventTelegramUpdate{}
	for _, msg := range streamMessages {
		event := createEvent(msg)
		if event != nil {
			events = append(events, event)
		} else {
			slog.Info("can't parse event telegram_update", "event", msg)
			if errAck := s.AckMessages(ctx, group, []string{msg.ID}); errAck != nil {
				slog.Error("cannot ack message telegram_update", "err", errAck)
			}
		}
	}

	return events, nil
}

func (s *Service) AckMessages(ctx context.Context, group string, messageIDs []string) error {
	return s.repository.AckMessages(ctx, eventTelegramUpdateStream, group, messageIDs)
}

func (s *Service) PendingTelegramUpdateEvents(ctx context.Context, group string, consumer string, limit int64, minIdle time.Duration) ([]*entity.EventTelegramUpdate, error) {
	data, _, err := s.repository.AutoClaimPendingEvents(ctx, &redis.XAutoClaimArgs{
		Stream:   eventTelegramUpdateStream,
		Group:    group,
		MinIdle:  minIdle,
		Count:    max(minReadLimit, min(maxReadLimit, limit)),
		Consumer: consumer,
		Start:    "0-0",
	})
	if err != nil {
		return nil, err
	}
	events := []*entity.EventTelegramUpdate{}
	for _, item := range data {
		event := createEvent(item)
		if event != nil {
			events = append(events, event)
		} else {
			slog.Info("can't parse event.pending telegram_update", "event", item)
			if errAck := s.AckMessages(ctx, group, []string{item.ID}); errAck != nil {
				slog.Error("cannot ack pending telegram_update", "err", errAck)
			}
		}
	}
	return events, nil
}

func (s *Service) RemoveConsumer(ctx context.Context, group string, consumer string) error {
	return s.repository.RemoveConsumer(ctx, eventTelegramUpdateStream, group, consumer)
}

func createEvent(streamMessage redis.XMessage) *entity.EventTelegramUpdate {
	e := &entity.EventTelegramUpdate{
		ID: streamMessage.ID,
	}
	e.FromMap(streamMessage.Values)
	return e
}

func (s *Service) TrimStreamByAge(ctx context.Context, age time.Duration) error {
	return s.repository.TrimStreamByAge(ctx, eventTelegramUpdateStream, age)
}
