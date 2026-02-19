package service

import (
	"context"
	"encoding/json"
	"time"

	"ads-mrkt/internal/market/domain/entity"
)

const refreshStatsCooldown = time.Hour

type marketRepository interface {
	GetChannelByID(ctx context.Context, id int64) (*entity.Channel, error)
	ListChannelsByAdminUserID(ctx context.Context, userID int64) ([]*entity.Channel, error)
	GetChannelStats(ctx context.Context, channelID int64) (json.RawMessage, error)
	MergeStatsRequestedAt(ctx context.Context, channelID int64, requestedAtUnix int64) error
	IsChannelHasActiveListing(ctx context.Context, channelID int64) (bool, error)
	IsChannelAdmin(ctx context.Context, userID, channelID int64) (bool, error)
}

type channelUpdateStatsEventAdder interface {
	AddChannelUpdateStatsEvent(ctx context.Context, channelID int64) error
}

type channelService struct {
	marketRepo            marketRepository
	channelUpdateStatsAdder channelUpdateStatsEventAdder
}

func NewChannelService(marketRepo marketRepository, channelUpdateStatsAdder channelUpdateStatsEventAdder) *channelService {
	return &channelService{
		marketRepo:             marketRepo,
		channelUpdateStatsAdder: channelUpdateStatsAdder,
	}
}
