package event

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"ads-mrkt/internal/event/domain/entity"

	"github.com/redis/go-redis/v9"
)

type repository interface {
	PushEvent(ctx context.Context, event entity.Event) error
	ReadEvents(ctx context.Context, args *redis.XReadGroupArgs) ([]redis.XMessage, error)
	CreateGroup(ctx context.Context, stream, group, id string) error
	AckMessages(ctx context.Context, stream, group string, messageIDs []string) error
	AutoClaimPendingEvents(ctx context.Context, args *redis.XAutoClaimArgs) ([]redis.XMessage, string, error)
	TrimStreamByAge(ctx context.Context, stream string, maxAge time.Duration) error
}

type Service struct {
	repository repository
}

const (
	groupName = "bot"
)

func NewService(ctx context.Context, repository repository) (*Service, error) {
	s := &Service{
		repository: repository,
	}

	streamKey := (*entity.EventTelegramNotification)(nil).StreamKey()
	err := s.repository.CreateGroup(ctx, streamKey, groupName, "0")
	if err != nil {
		if strings.Contains(err.Error(), "BUSYGROUP") {
			slog.Info("consumer group already exists", "group", groupName)
		} else {
			return nil, fmt.Errorf("failed to create telegram_notification event group: %w", err)
		}
	}

	return s, nil
}
