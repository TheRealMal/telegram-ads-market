package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	helpertelegram "ads-mrkt/internal/helpers/telegram"
	marketentity "ads-mrkt/internal/market/domain/entity"
)

var ErrSkipThatIteration = errors.New("skip that iteration to restart process")

// ensureChannelAccess ensures the userbot has MTProto access to the channel.
// If access_hash is 0 (bot-only access), it promotes the userbot via Bot API,
// then resolves the channel to get the access hash.
func (s *service) ensureChannelAccess(ctx context.Context, channel *marketentity.Channel) (*marketentity.Channel, error) {
	if channel.AccessHash != 0 {
		return channel, nil
	}

	if channel.BotMemberStatus != marketentity.BotMemberStatusAdministrator {
		return nil, fmt.Errorf("channel %d: bot is not administrator (status=%s), cannot add userbot", channel.ID, channel.BotMemberStatus)
	}

	slog.Info("promoting userbot via bot api", "channel_id", channel.ID)
	err := s.botAPIClient.PromoteChatMember(ctx, channel.ID, s.userID, helpertelegram.AdminPromoteRights{
		CanPostMessages:   true,
		CanEditMessages:   true,
		CanDeleteMessages: true,
		CanPostStories:    true,
		CanDeleteStories:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("promote userbot in channel %d: %w", channel.ID, err)
	}

	return nil, ErrSkipThatIteration
}
