package event

import (
	"context"
	"log"
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
	groupName = "userbot"
)

func NewService(repository repository) *Service {
	s := &Service{
		repository: repository,
	}

	streamKey := (*entity.EventChannelUpdateStats)(nil).StreamKey()
	err := s.repository.CreateGroup(context.Background(), streamKey, groupName, "0")
	if err != nil {
		if !strings.Contains(err.Error(), "BUSYGROUP") {
			log.Fatalf("failed to create channel_update_stats event group: %v", err)
		}
	}

	return s
}
