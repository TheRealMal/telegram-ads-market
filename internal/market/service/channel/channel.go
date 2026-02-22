package service

import (
	"context"
	"encoding/json"
	"time"

	"ads-mrkt/internal/market/domain/entity"
	marketerrors "ads-mrkt/internal/market/domain/errors"
)

// ListMyChannels returns all channels where the user is admin.
func (s *channelService) ListMyChannels(ctx context.Context, userID int64) ([]*entity.Channel, error) {
	return s.channelRepo.ListChannelsByAdminUserID(ctx, userID)
}

// RequestStatsRefresh verifies the user is channel admin, rate-limits by requested_at (1 hour cooldown),
// then merges requested_at, pushes a channel_update_stats event, and returns the channel.
// Returns *marketerrors.ErrStatsRefreshTooSoon when within cooldown (caller should set response Data from NextAvailableAt).
func (s *channelService) RequestStatsRefresh(ctx context.Context, channelID int64, userID int64) (*entity.Channel, error) {
	ok, err := s.channelAdminRepo.IsChannelAdmin(ctx, userID, channelID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, marketerrors.ErrNotChannelAdmin
	}
	ch, err := s.channelRepo.GetChannelByID(ctx, channelID)
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	raw, err := s.channelRepo.GetChannelStats(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if err := handleRawStatsRequestedAt(raw, now); err != nil {
		return nil, err
	}

	if err := s.channelRepo.MergeStatsRequestedAt(ctx, channelID, now); err != nil {
		return nil, err
	}
	if err := s.channelUpdateStatsAdder.AddChannelUpdateStatsEvent(ctx, channelID); err != nil {
		return nil, err
	}
	return ch, nil
}

func handleRawStatsRequestedAt(raw json.RawMessage, now int64) error {
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	var requestedAt int64
	if v := m["requested_at"]; v != nil {
		switch t := v.(type) {
		case float64:
			requestedAt = int64(t)
		case int64:
			requestedAt = t
		case int:
			requestedAt = int64(t)
		}
	}

	if requestedAt == 0 {
		return nil
	}

	if now-requestedAt >= int64(refreshStatsCooldown.Seconds()) {
		return nil
	}

	return &marketerrors.ErrStatsRefreshTooSoon{
		NextAvailableAt: time.Unix(requestedAt+int64(refreshStatsCooldown.Seconds()), 0),
	}
}

// GetChannelStats returns stats for the channel. Allowed only if user is channel admin or has an active listing with this channel_id.
func (s *channelService) GetChannelStats(ctx context.Context, channelID int64, userID int64) (interface{}, error) {
	admin, err := s.channelAdminRepo.IsChannelAdmin(ctx, userID, channelID)
	if err != nil {
		return nil, err
	}
	if !admin {
		listed, err := s.listingRepo.IsChannelHasActiveListing(ctx, channelID)
		if err != nil {
			return nil, err
		}
		if !listed {
			return nil, marketerrors.ErrChannelStatsDenied
		}
	}

	raw, err := s.channelRepo.GetChannelStats(ctx, channelID)
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

// MergeStatsRequestedAt merges requested_at (unix timestamp) into the channel's stats JSON.
func (s *channelService) MergeStatsRequestedAt(ctx context.Context, channelID int64, requestedAtUnix int64) error {
	return s.channelRepo.MergeStatsRequestedAt(ctx, channelID, requestedAtUnix)
}
