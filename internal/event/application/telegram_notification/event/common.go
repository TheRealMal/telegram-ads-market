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

var eventTelegramNotificationStream = (*entity.EventTelegramNotification)(nil).StreamKey()

func (s *Service) AddTelegramNotificationEvent(ctx context.Context, chatID int64, message string) error {
	return s.repository.PushEvent(ctx, &entity.EventTelegramNotification{ChatID: chatID, Message: message})
}

func (s *Service) ReadTelegramNotificationEvents(ctx context.Context, group string, consumer string, limit int64) ([]*entity.EventTelegramNotification, error) {
	args := &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{eventTelegramNotificationStream, ">"},
		Block:    time.Millisecond * 100, //nolint:revive
		Count:    max(minReadLimit, min(maxReadLimit, limit)),
	}

	streamMessages, err := s.repository.ReadEvents(ctx, args)
	if err != nil {
		return nil, err
	}

	events := make([]*entity.EventTelegramNotification, 0, len(streamMessages))
	for _, msg := range streamMessages {
		event := createEvent(msg)
		if event != nil {
			events = append(events, event)
		} else {
			slog.Info("can't parse event telegram_notification", "event", msg)
			if errAck := s.AckTelegramNotificationMessages(ctx, group, []string{msg.ID}); errAck != nil {
				slog.Error("cannot ack message telegram_notification", "err", errAck)
			}
		}
	}

	return events, nil
}

func (s *Service) AckTelegramNotificationMessages(ctx context.Context, group string, messageIDs []string) error {
	return s.repository.AckMessages(ctx, eventTelegramNotificationStream, group, messageIDs)
}

func (s *Service) PendingTelegramNotificationEvents(ctx context.Context, group string, consumer string, limit int64, minIdle time.Duration) ([]*entity.EventTelegramNotification, error) {
	data, _, err := s.repository.AutoClaimPendingEvents(ctx, &redis.XAutoClaimArgs{
		Stream:   eventTelegramNotificationStream,
		Group:    group,
		MinIdle:  minIdle,
		Count:    max(minReadLimit, min(maxReadLimit, limit)),
		Consumer: consumer,
		Start:    "0-0",
	})
	if err != nil {
		return nil, err
	}
	events := make([]*entity.EventTelegramNotification, 0, len(data))
	for _, item := range data {
		event := createEvent(item)
		if event != nil {
			events = append(events, event)
		} else {
			slog.Info("can't parse event.pending telegram_notification", "event", item)
			if errAck := s.AckTelegramNotificationMessages(ctx, group, []string{item.ID}); errAck != nil {
				slog.Error("cannot ack pending telegram_notification", "err", errAck)
			}
		}
	}
	return events, nil
}

func (s *Service) TrimStreamByAge(ctx context.Context, age time.Duration) error {
	return s.repository.TrimStreamByAge(ctx, eventTelegramNotificationStream, age)
}

func createEvent(streamMessage redis.XMessage) *entity.EventTelegramNotification {
	e := &entity.EventTelegramNotification{
		ID: streamMessage.ID,
	}
	e.FromMap(streamMessage.Values)
	return e
}
