package service

import (
	"context"
	"encoding/json"

	"ads-mrkt/internal/market/domain/entity"
)

type marketRepository interface {
	GetChannelByID(ctx context.Context, id int64) (*entity.Channel, error)
	ListChannelsByAdminUserID(ctx context.Context, userID int64) ([]*entity.Channel, error)
	GetChannelStats(ctx context.Context, channelID int64) (json.RawMessage, error)
	IsChannelHasActiveListing(ctx context.Context, channelID int64) (bool, error)
	IsChannelAdmin(ctx context.Context, userID, channelID int64) (bool, error)
}

type channelService struct {
	marketRepo marketRepository
}

func NewChannelService(marketRepo marketRepository) *channelService {
	return &channelService{marketRepo: marketRepo}
}
