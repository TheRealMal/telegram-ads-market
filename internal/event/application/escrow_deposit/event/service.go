package event

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"ads-mrkt/internal/event/domain/entity"

	"github.com/redis/go-redis/v9"
)

type repository interface {
	PushEvent(ctx context.Context, event entity.Event) error
	ReadEvents(ctx context.Context, args *redis.XReadGroupArgs) ([]redis.XMessage, error)
	CreateGroup(ctx context.Context, stream, group, id string) error
	AckMessages(ctx context.Context, stream, group string, messageIDs []string) error
}

type Service struct {
	repository repository
}

const (
	groupName = "market"
)

func NewService(ctx context.Context, repository repository) (*Service, error) {
	s := &Service{
		repository: repository,
	}

	streamKey := (*entity.EventEscrowDeposit)(nil).StreamKey()
	err := s.repository.CreateGroup(ctx, streamKey, groupName, "0")
	if err != nil {
		if strings.Contains(err.Error(), "BUSYGROUP") {
			slog.Info("consumer group already exists", "group", groupName)
		} else {
			return nil, fmt.Errorf("failed to create escrow_deposit event group: %w", err)
		}
	}

	return s, nil
}
