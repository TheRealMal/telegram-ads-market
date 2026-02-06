package service

import (
	"context"

	"ads-mrkt/internal/market/domain/entity"
)

type ChannelRepository interface {
	GetChannelByID(ctx context.Context, id int64) (*entity.Channel, error)
	ListChannelsByAdminUserID(ctx context.Context, userID int64) ([]*entity.Channel, error)
}

type ChannelAdminRepository interface {
	IsChannelAdmin(ctx context.Context, userID, channelID int64) (bool, error)
}

type ChannelService struct {
	channelRepo ChannelRepository
	adminRepo   ChannelAdminRepository
}

func NewChannelService(channelRepo ChannelRepository, adminRepo ChannelAdminRepository) *ChannelService {
	return &ChannelService{channelRepo: channelRepo, adminRepo: adminRepo}
}
