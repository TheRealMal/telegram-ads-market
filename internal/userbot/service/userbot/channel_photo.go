package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"

	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/tg"
)

// updateChannelPhotoFromTelegram fetches the channel profile photo via MTProto, encodes it as base64, and stores it in the DB.
func (s *service) updateChannelPhotoFromTelegram(ctx context.Context, channelID, accessHash int64) {
	photoBytes, err := s.getChannelPhotoBytes(ctx, channelID, accessHash)
	if err != nil {
		slog.Debug("channel photo not available or download failed", "channel_id", channelID, "error", err)
		return
	}
	if len(photoBytes) == 0 {
		return
	}
	photoBase64 := base64.StdEncoding.EncodeToString(photoBytes)
	if err := s.channelRepo.UpdateChannelPhoto(ctx, channelID, photoBase64); err != nil {
		slog.Error("update channel photo", "channel_id", channelID, "error", err)
		return
	}
	slog.Info("channel photo updated", "channel_id", channelID)
}

// getChannelPhotoBytes returns the channel profile picture bytes (full size) or nil if not set / on error.
func (s *service) getChannelPhotoBytes(ctx context.Context, channelID, accessHash int64) ([]byte, error) {
	fullChannel, err := s.telegramClient.API().ChannelsGetFullChannel(ctx, &tg.InputChannel{
		ChannelID:  channelID,
		AccessHash: accessHash,
	})
	if err != nil {
		return nil, fmt.Errorf("get full channel: %w", err)
	}
	if len(fullChannel.GetChats()) == 0 {
		return nil, fmt.Errorf("no chats in full channel")
	}
	channel, ok := fullChannel.GetChats()[0].(*tg.Channel)
	if !ok {
		return nil, fmt.Errorf("first chat is not channel")
	}
	photoClass := channel.GetPhoto()
	if photoClass == nil {
		return nil, nil
	}
	chatPhoto, ok := photoClass.(*tg.ChatPhoto)
	if !ok {
		return nil, nil
	}

	location := &tg.InputPeerPhotoFileLocation{
		Big:     true,
		Peer:    &tg.InputPeerChannel{ChannelID: channelID, AccessHash: accessHash},
		PhotoID: chatPhoto.PhotoID,
	}

	var buf bytes.Buffer
	dl := downloader.NewDownloader()
	_, err = dl.Download(s.telegramClient.API(), location).Stream(ctx, &buf)
	if err != nil {
		return nil, fmt.Errorf("download photo: %w", err)
	}
	return buf.Bytes(), nil
}
