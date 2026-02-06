package service

import (
	"context"
	"fmt"
	"log/slog"

	marketentity "ads-mrkt/internal/market/domain/entity"

	"github.com/gotd/td/tg"
)

func (s *service) handleChannelUpdate(ctx context.Context, e tg.Entities, update *tg.UpdateChannel) error {
	slog.Info("channel update received", "channel_id", update.ChannelID, "title", e.Channels[update.ChannelID].Title)

	fullChannel, err := s.telegramClient.API().ChannelsGetFullChannel(ctx, &tg.InputChannel{
		ChannelID:  update.ChannelID,
		AccessHash: e.Channels[update.ChannelID].AccessHash,
	})
	if err != nil {
		return fmt.Errorf("failed to get full channel: %w", err)
	}

	channel := mapChannel(fullChannel)
	if channel == nil {
		slog.Error("failed to map channel", "channel_id", update.ChannelID, "full_channel", fullChannel)
		return nil
	}
	channel.AccessHash = e.Channels[update.ChannelID].AccessHash

	if err = s.marketRepository.UpsertChannel(ctx, channel); err != nil {
		return fmt.Errorf("failed to upsert channel id=%d: %w", update.ChannelID, err)
	}

	if !channel.AdminRights.CanViewStats {
		slog.Info("user does not have stats access", "channel_id", update.ChannelID)
		return nil
	}

	slog.Info("updating channel stats", "channel_id", update.ChannelID)
	if err = s.UpdateChannelStats(ctx, update.ChannelID, e.Channels[update.ChannelID].AccessHash); err != nil {
		return fmt.Errorf("failed to update channel stats: %w", err)
	}

	return s.syncChannelAdmins(ctx, update.ChannelID, e.Channels[update.ChannelID].AccessHash)
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

// handleChannelParticipant handles updateChannelParticipant (admin added/removed, participant join/leave).
// Resyncs channel admins from Telegram so DB stays in sync.
func (s *service) handleChannelParticipant(ctx context.Context, e tg.Entities, update *tg.UpdateChannelParticipant) error {
	slog.Info("channel participant update", "channel_id", update.ChannelID, "user_id", update.UserID)

	var accessHash int64
	if ch, ok := e.Channels[update.ChannelID]; ok {
		accessHash = ch.AccessHash
	} else {
		// Channel may not be in this update batch; try DB (we store it on channel join/update).
		channel, err := s.marketRepository.GetChannelByID(ctx, update.ChannelID)
		if err != nil || channel == nil {
			slog.Info("channel not in entities and not in DB, skip admin sync", "channel_id", update.ChannelID)
			return nil
		}
		accessHash = channel.AccessHash
	}

	if err := s.syncChannelAdmins(ctx, update.ChannelID, accessHash); err != nil {
		return fmt.Errorf("sync channel admins after participant update: %w", err)
	}
	return nil
}

func mapChannel(rawChannel *tg.MessagesChatFull) *marketentity.Channel {
	if len(rawChannel.Chats) == 0 {
		return nil
	}

	channel, ok := rawChannel.GetChats()[0].(*tg.Channel)
	if !ok {
		return nil
	}

	channelFull, ok := rawChannel.GetFullChat().(*tg.ChannelFull)
	if !ok {
		return nil
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
	}
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
