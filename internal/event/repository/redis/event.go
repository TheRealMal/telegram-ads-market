package redis

import (
	"context"
	"fmt"
	"time"

	redisclient "github.com/redis/go-redis/v9"

	"ads-mrkt/internal/event/domain/entity"
)

func (r *repository) PushEvent(ctx context.Context, event entity.Event) error {
	if cmd := r.db.XAdd(ctx, &redisclient.XAddArgs{
		Stream: event.StreamKey(),
		Values: event.ToMap(),
	}); cmd.Err() != nil {
		return fmt.Errorf("failed to add event to stream: %w", cmd.Err())
	}

	return nil
}

// ReadEvents reads events from the stream. Provide the event to be read as an argument (do not initialize it).
func (r *repository) ReadEvents(ctx context.Context, args *redisclient.XReadGroupArgs) ([]redisclient.XMessage, error) {
	cmd := r.db.XReadGroup(ctx, args)
	if cmd.Err() != nil {
		if cmd.Err() == redisclient.Nil { // no events in the stream
			return []redisclient.XMessage{}, nil
		}
		return nil, fmt.Errorf("failed to read events from stream: %w", cmd.Err())
	}

	return cmd.Val()[0].Messages, nil
}

func (r *repository) CreateGroup(ctx context.Context, stream, group, id string) error {
	if cmd := r.db.XGroupCreateMkStream(ctx, stream, group, id); cmd.Err() != nil {
		return fmt.Errorf("failed to create group: %w", cmd.Err())
	}

	return nil
}

func (r *repository) AckMessages(ctx context.Context, stream, group string, messageIDs []string) error {
	return r.db.XAck(ctx, stream, group, messageIDs).Err()
}

func (r *repository) ClaimMessage(ctx context.Context, stream, group, consumer string, messageIDs []string) ([]redisclient.XMessage, error) {
	cmd := r.db.XClaim(ctx, &redisclient.XClaimArgs{
		Stream:   stream,
		Group:    group,
		Consumer: consumer,
		MinIdle:  time.Second * 30,
		Messages: messageIDs,
	})
	if cmd.Err() != nil {
		return nil, fmt.Errorf("failed to claim message: %w", cmd.Err())
	}
	return cmd.Val(), nil
}

func (r *repository) PendingEvents(ctx context.Context, args *redisclient.XPendingExtArgs) ([]redisclient.XPendingExt, error) {
	cmd := r.db.XPendingExt(ctx, args)
	if cmd.Err() != nil {
		return nil, cmd.Err()
	}
	return cmd.Result()
}

func (r *repository) AutoClaimPendingEvents(ctx context.Context, args *redisclient.XAutoClaimArgs) ([]redisclient.XMessage, string, error) {
	cmd := r.db.XPendingAutoClaim(ctx, args)
	if cmd.Err() != nil {
		return nil, "", cmd.Err()
	}
	return cmd.Result()
}

func (r *repository) RemoveConsumer(ctx context.Context, stream, group, consumer string) error {
	_, err := r.db.XGroupDelConsumer(ctx, stream, group, consumer)
	return err
}

func (r *repository) TrimStreamByAge(ctx context.Context, group string, maxAge time.Duration) error {
	cmd := r.db.XTrim(ctx, group, fmt.Sprintf("%d-0", time.Now().Add(-maxAge).UnixMilli()), 0)
	return cmd.Err()
}
