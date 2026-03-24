package service

import (
	"context"
	"fmt"
	"log/slog"

	helpertelegram "ads-mrkt/internal/helpers/telegram"
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

	botAPIID := helpertelegram.ToBotAPIChannelID(update.ChannelID)

	fullChannel, err := s.telegramClient.API().ChannelsGetFullChannel(ctx, &tg.InputChannel{
		ChannelID:  update.ChannelID,
		AccessHash: channelEnt.AccessHash,
	})
	if err != nil {
		return fmt.Errorf("failed to get full channel: %w", err)
	}

	channel, userbotRights, statsDC := mapChannel(fullChannel)
	if channel == nil {
		slog.Error("failed to map channel", "channel_id", update.ChannelID, "full_channel", fullChannel)
		return nil
	}
	channel.AccessHash = channelEnt.AccessHash

	if err = s.channelRepo.UpsertChannel(ctx, channel); err != nil {
		return fmt.Errorf("failed to upsert channel id=%d: %w", botAPIID, err)
	}

	if err = s.channelRepo.UpdateUserbotAdminRights(ctx, botAPIID, userbotRights); err != nil {
		return fmt.Errorf("failed to update userbot admin rights for channel id=%d: %w", botAPIID, err)
	}

	if userbotRights.CanViewStats {
		slog.Info("updating channel stats", "channel_id", botAPIID)
		if err = s.UpdateChannelStats(ctx, botAPIID, channelEnt.AccessHash, statsDC); err != nil {
			slog.Error("failed to update channel stats", "channel_id", botAPIID, "error", err)
			return fmt.Errorf("failed to update channel stats: %w", err)
		}
	}

	return nil
}

func mapChannel(rawChannel *tg.MessagesChatFull) (*marketentity.Channel, marketentity.UserbotAdminRights, int) {
	if len(rawChannel.Chats) == 0 {
		return nil, marketentity.UserbotAdminRights{}, 0
	}

	channel, ok := rawChannel.GetChats()[0].(*tg.Channel)
	if !ok {
		return nil, marketentity.UserbotAdminRights{}, 0
	}

	channelFull, ok := rawChannel.GetFullChat().(*tg.ChannelFull)
	if !ok {
		return nil, marketentity.UserbotAdminRights{}, 0
	}

	username, ok := channel.GetUsername()
	if !ok {
		username = ""
	}

	userbotRights := mapUserbotAdminRights(channel.AdminRights, channelFull.CanViewStats)

	return &marketentity.Channel{
		ID:       helpertelegram.ToBotAPIChannelID(channel.GetID()),
		Title:    channel.GetTitle(),
		Username: username,
		Photo:    "",
	}, userbotRights, channelFull.StatsDC
}

func mapUserbotAdminRights(adminRights tg.ChatAdminRights, canViewStats bool) marketentity.UserbotAdminRights {
	return marketentity.UserbotAdminRights{
		DeleteMessages: adminRights.DeleteMessages,
		EditMessages:   adminRights.EditMessages,
		PostMessages:   adminRights.PostMessages,
		DeleteStories:  adminRights.DeleteStories,
		PostStories:    adminRights.PostStories,
		CanViewStats:   canViewStats,
	}
}
