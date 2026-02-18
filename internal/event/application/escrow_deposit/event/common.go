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

var eventEscrowDepositStream = (*entity.EventEscrowDeposit)(nil).StreamKey()

func (s *Service) AddEscrowDepositEvent(ctx context.Context, event *entity.EventEscrowDeposit) error {
	return s.repository.PushEvent(ctx, event)
}

func (s *Service) ReadEscrowDepositEvents(ctx context.Context, group string, consumer string, limit int64) ([]*entity.EventEscrowDeposit, error) {
	args := &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{eventEscrowDepositStream, ">"},
		Block:    time.Millisecond * 100, //nolint:revive
		Count:    max(minReadLimit, min(maxReadLimit, limit)),
	}

	streamMessages, err := s.repository.ReadEvents(ctx, args)
	if err != nil {
		return nil, err
	}

	events := make([]*entity.EventEscrowDeposit, 0, len(streamMessages))
	for _, msg := range streamMessages {
		event := createEvent(msg)
		if event != nil {
			events = append(events, event)
		} else {
			slog.Info("can't parse event escrow_deposit", "event", msg)
			if errAck := s.AckEscrowDepositMessages(ctx, group, []string{msg.ID}); errAck != nil {
				slog.Error("cannot ack message escrow_deposit", "err", errAck)
			}
		}
	}

	return events, nil
}

func (s *Service) AckEscrowDepositMessages(ctx context.Context, group string, messageIDs []string) error {
	return s.repository.AckMessages(ctx, eventEscrowDepositStream, group, messageIDs)
}

func createEvent(streamMessage redis.XMessage) *entity.EventEscrowDeposit {
	e := &entity.EventEscrowDeposit{
		ID: streamMessage.ID,
	}
	e.FromMap(streamMessage.Values)
	return e
}
