package service

import (
	"context"
	"encoding/json"

	"ads-mrkt/internal/market/domain/entity"
	marketerrors "ads-mrkt/internal/market/domain/errors"
)

// ListMyChannels returns all channels where the user is admin.
func (s *channelService) ListMyChannels(ctx context.Context, userID int64) ([]*entity.Channel, error) {
	return s.marketRepo.ListChannelsByAdminUserID(ctx, userID)
}

// RefreshChannel returns the channel by id after verifying the user is admin of that channel.
// Caller can use this to "refresh" (get latest from DB); actual Telegram re-fetch can be wired separately.
func (s *channelService) RefreshChannel(ctx context.Context, channelID int64, userID int64) (*entity.Channel, error) {
	ok, err := s.marketRepo.IsChannelAdmin(ctx, userID, channelID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, marketerrors.ErrNotChannelAdmin
	}
	return s.marketRepo.GetChannelByID(ctx, channelID)
}

// GetChannelStats returns stats for the channel. Allowed only if user is channel admin or has an active listing with this channel_id.
func (s *channelService) GetChannelStats(ctx context.Context, channelID int64, userID int64) (interface{}, error) {
	admin, err := s.marketRepo.IsChannelAdmin(ctx, userID, channelID)
	if err != nil {
		return nil, err
	}
	if !admin {
		listed, err := s.marketRepo.IsChannelHasActiveListing(ctx, channelID)
		if err != nil {
			return nil, err
		}
		if !listed {
			return nil, marketerrors.ErrChannelStatsDenied
		}
	}

	raw, err := s.marketRepo.GetChannelStats(ctx, channelID)
	if err != nil || raw == nil {
		return nil, err
	}

	// Decode and re-encode so the response is a JSON object, not a raw string.
	var out interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}
