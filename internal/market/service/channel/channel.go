package service

import (
	"context"

	marketerrors "ads-mrkt/internal/market/domain/errors"
	"ads-mrkt/internal/market/domain/entity"
)

// ListMyChannels returns all channels where the user is admin.
func (s *ChannelService) ListMyChannels(ctx context.Context, userID int64) ([]*entity.Channel, error) {
	return s.channelRepo.ListChannelsByAdminUserID(ctx, userID)
}

// RefreshChannel returns the channel by id after verifying the user is admin of that channel.
// Caller can use this to "refresh" (get latest from DB); actual Telegram re-fetch can be wired separately.
func (s *ChannelService) RefreshChannel(ctx context.Context, channelID int64, userID int64) (*entity.Channel, error) {
	ok, err := s.adminRepo.IsChannelAdmin(ctx, userID, channelID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, marketerrors.ErrNotChannelAdmin
	}
	return s.channelRepo.GetChannelByID(ctx, channelID)
}
