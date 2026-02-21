package service

import (
	"context"
	"fmt"
	"log/slog"

	marketentity "ads-mrkt/internal/market/domain/entity"

	"github.com/gotd/td/tg"
)

func (s *service) handleChannelUpdate(ctx context.Context, e tg.Entities, update *tg.UpdateChannel) error {
	channelEnt, ok := e.Channels[update.ChannelID]
	if !ok || channelEnt == nil {
		slog.Info("channel update skipped: channel not in entities", "channel_id", update.ChannelID)
		return nil
	}
	slog.Info("channel update received", "channel_id", update.ChannelID, "title", channelEnt.Title)

	fullChannel, err := s.telegramClient.API().ChannelsGetFullChannel(ctx, &tg.InputChannel{
		ChannelID:  update.ChannelID,
		AccessHash: channelEnt.AccessHash,
	})
	if err != nil {
		return fmt.Errorf("failed to get full channel: %w", err)
	}

	channel, statsDC := mapChannel(fullChannel)
	if channel == nil {
		slog.Error("failed to map channel", "channel_id", update.ChannelID, "full_channel", fullChannel)
		return nil
	}
	channel.AccessHash = channelEnt.AccessHash

	if err = s.marketRepository.UpsertChannel(ctx, channel); err != nil {
		return fmt.Errorf("failed to upsert channel id=%d: %w", update.ChannelID, err)
	}

	if channel.AdminRights.CanViewStats {
		slog.Info("updating channel stats", "channel_id", update.ChannelID)
		if err = s.UpdateChannelStats(ctx, update.ChannelID, channelEnt.AccessHash, statsDC); err != nil {
			slog.Error("failed to update channel stats", "channel_id", update.ChannelID, "error", err)
			return fmt.Errorf("failed to update channel stats: %w", err)
		}
	}

	if err := s.syncChannelAdmins(ctx, update.ChannelID, channelEnt.AccessHash); err != nil {
		return err
	}

	s.updateChannelPhotoFromTelegram(ctx, update.ChannelID, channelEnt.AccessHash)
	return nil
}

// syncChannelAdmins fetches current admins from Telegram and replaces channel_admin rows for the channel.
func (s *service) syncChannelAdmins(ctx context.Context, channelID, accessHash int64) error {
	participantsResp, err := s.telegramClient.API().ChannelsGetParticipants(ctx, &tg.ChannelsGetParticipantsRequest{
		Channel: &tg.InputChannel{
			ChannelID:  channelID,
			AccessHash: accessHash,
		},
		Filter: &tg.ChannelParticipantsAdmins{},
		Offset: 0,
		Limit:  100,
	})
	if err != nil {
		return fmt.Errorf("failed to get participants: %w", err)
	}

	modified, ok := participantsResp.(*tg.ChannelsChannelParticipants)
	if !ok {
		slog.Info("channel participants not modified or empty", "channel_id", channelID)
		return nil
	}

	if err = s.marketRepository.DeleteChannelAdmins(ctx, channelID); err != nil {
		return fmt.Errorf("delete channel admins: %w", err)
	}

	for _, p := range modified.Participants {
		var userID int64
		var role string
		switch v := p.(type) {
		case *tg.ChannelParticipantCreator:
			userID = v.UserID
			role = "owner"
		case *tg.ChannelParticipantAdmin:
			userID = v.UserID
			role = "admin"
		default:
			continue
		}
		if err = s.marketRepository.UpsertChannelAdmin(ctx, userID, channelID, role); err != nil {
			return fmt.Errorf("upsert channel admin user_id=%d: %w", userID, err)
		}
		slog.Info("synced channel admin", "channel_id", channelID, "user_id", userID, "role", role)
	}
	return nil
}

func mapChannel(rawChannel *tg.MessagesChatFull) (*marketentity.Channel, int) {
	if len(rawChannel.Chats) == 0 {
		return nil, 0
	}

	channel, ok := rawChannel.GetChats()[0].(*tg.Channel)
	if !ok {
		return nil, 0
	}

	channelFull, ok := rawChannel.GetFullChat().(*tg.ChannelFull)
	if !ok {
		return nil, 0
	}

	username, ok := channel.GetUsername()
	if !ok {
		username = ""
	}

	return &marketentity.Channel{
		ID:          channel.GetID(),
		Title:       channel.GetTitle(),
		Username:    username,
		Photo:       "",
		AdminRights: mapAdminRights(channel.AdminRights, channelFull.CanViewStats),
	}, channelFull.StatsDC
}

func mapAdminRights(adminRights tg.ChatAdminRights, canViewStats bool) marketentity.AdminRights {
	return marketentity.AdminRights{
		DeleteMessages: adminRights.DeleteMessages,
		EditMessages:   adminRights.EditMessages,
		PostMessages:   adminRights.PostMessages,
		DeleteStories:  adminRights.DeleteStories,
		PostStories:    adminRights.PostStories,
		CanViewStats:   canViewStats,
	}
}
