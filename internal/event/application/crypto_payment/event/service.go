package event

import (
	"context"
	"log"
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
	RemoveConsumer(ctx context.Context, stream, group, consumer string) error
	TrimStreamByAge(ctx context.Context, group string, maxAge time.Duration) error
}

type Service struct {
	repository repository
}

const (
	groupName = "master"
)

func NewService(repository repository) *Service {
	s := &Service{
		repository: repository,
	}

	err := s.repository.CreateGroup(context.Background(), (*entity.EventCryptoPayment)(nil).StreamKey(), groupName, "$")
	if err != nil {
		if !strings.Contains(err.Error(), "BUSYGROUP") {
			log.Fatalf("failed to create group: %v", err)
		}
	}

	return s
}
