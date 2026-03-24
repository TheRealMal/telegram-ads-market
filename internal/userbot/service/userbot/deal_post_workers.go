package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"strconv"
	"time"

	helpertelegram "ads-mrkt/internal/helpers/telegram"
	marketentity "ads-mrkt/internal/market/domain/entity"
	marketerrors "ads-mrkt/internal/market/domain/errors"
	"ads-mrkt/internal/worker"

	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
)

const (
	postSenderInterval   = 1 * time.Minute
	postCheckerInterval  = 5 * time.Minute
	postCheckAdvanceHour = time.Hour
)

func (s *service) RunDealPostSenderWorker(ctx context.Context) {
	worker.RunTicker(ctx, "deal_post_sender", postSenderInterval, false, s.runDealPostSenderOnce)
}

func (s *service) runDealPostSenderOnce(ctx context.Context, logger *slog.Logger) {
	deals, err := s.dealRepo.ListDealsEscrowDepositConfirmedWithoutPostMessage(ctx)
	if err != nil {
		logger.Error("list deals", "error", err)
		return
	}
	for _, deal := range deals {
		s.processPostDeal(ctx, logger, deal)
	}
}

func (s *service) processPostDeal(ctx context.Context, logger *slog.Logger, deal *marketentity.Deal) {
	listing, err := s.listingRepo.GetListingByID(ctx, deal.ListingID)
	if err != nil {
		logger.Error("skip deal, listing error", "deal_id", deal.ID, "error", err)
		return
	}
	if listing.ChannelID == nil {
		logger.Error("skip deal, no channel on listing", "deal_id", deal.ID)
		return
	}
	channel, err := s.channelRepo.GetChannelByID(ctx, *listing.ChannelID)
	if err != nil {
		logger.Error("skip deal, channel not found", "deal_id", deal.ID, "channel_id", *listing.ChannelID, "error", err)
		return
	}
	channel, err = s.ensureChannelAccess(ctx, channel)
	if err != nil {
		logger.Error("ensure channel access", "deal_id", deal.ID, "channel_id", *listing.ChannelID, "error", err)
		return
	}
	if deal.IsStory() {
		if !channel.AdminRights.Userbot.PostStories {
			logger.Error("skip story deal, userbot lacks post_stories right", "deal_id", deal.ID, "channel_id", *listing.ChannelID)
			return
		}
		s.processStoryDeal(ctx, logger, deal, listing, channel)
		return
	}
	if !channel.AdminRights.Userbot.HasEnoughRights() {
		logger.Error("skip deal, userbot lacks post_messages right", "deal_id", deal.ID, "channel_id", *listing.ChannelID)
		return
	}
	text := marketentity.GetMessageFromDetails(deal.Details)
	if text == "" {
		logger.Error("skip deal, no message in details", "deal_id", deal.ID)
		return
	}
	if postedAt, ok := marketentity.GetPostedAtFromDetails(deal.Details); ok && time.Now().Before(postedAt) {
		logger.Debug("skip deal, posted_at in future", "deal_id", deal.ID, "posted_at", postedAt)
		return
	}

	var botEntities []helpertelegram.MessageEntity
	if raw := marketentity.GetBotAPIEntitiesFromDetails(deal.Details); len(raw) > 0 {
		if err := json.Unmarshal(raw, &botEntities); err != nil {
			logger.Error("unmarshal entities from deal details", "deal_id", deal.ID, "error", err)
		}
	}
	mtprotoEntities := toMTProtoEntities(botEntities)

	// If the last lock for post_message is in status Locked and expired, previous run may have posted then crashed: try to find the message in the channel.
	lastLock, err := s.dealActionLockRepo.GetLastDealActionLock(ctx, deal.ID, marketentity.DealActionTypePostMessage)
	if err != nil && !errors.Is(err, marketerrors.ErrNotFound) {
		logger.Error("get last lock", "deal_id", deal.ID, "error", err)
		return
	}
	if err == nil && lastLock.Status == marketentity.DealActionLockStatusLocked && !lastLock.ExpireAt.After(time.Now()) {
		if foundMsgID, found := s.tryRecoverPostFromChannel(ctx, *listing.ChannelID, channel.AccessHash, text); found {
			m := s.makeDealPostMessage(deal, *listing.ChannelID, foundMsgID, text)
			if err := s.dealPostMessageRepo.CreateDealPostMessageAndSetDealInProgress(ctx, m); err != nil {
				logger.Error("recover create deal_post_message", "deal_id", deal.ID, "error", err)
				_ = s.dealActionLockRepo.ReleaseDealActionLock(ctx, lastLock.ID, marketentity.DealActionLockStatusFailed)
				return
			}
			_ = s.dealActionLockRepo.ReleaseDealActionLock(ctx, lastLock.ID, marketentity.DealActionLockStatusCompleted)
			logger.Info("recovered post from channel", "deal_id", deal.ID, "channel_id", *listing.ChannelID, "message_id", foundMsgID)
			return
		}
		_ = s.dealActionLockRepo.ReleaseDealActionLock(ctx, lastLock.ID, marketentity.DealActionLockStatusFailed)
	}

	lockID, err := s.dealActionLockRepo.TakeDealActionLock(ctx, deal.ID, marketentity.DealActionTypePostMessage)
	if err != nil {
		logger.Debug("skip deal, lock not acquired", "deal_id", deal.ID, "error", err)
		return
	}
	releaseLock := func(status marketentity.DealActionLockStatus) {
		_ = s.dealActionLockRepo.ReleaseDealActionLock(ctx, lockID, status)
	}

	msgID, err := s.sendChannelMessage(ctx, *listing.ChannelID, channel.AccessHash, text, mtprotoEntities)
	if err != nil {
		logger.Error("send message", "deal_id", deal.ID, "error", err)
		releaseLock(marketentity.DealActionLockStatusFailed)
		return
	}
	m := s.makeDealPostMessage(deal, *listing.ChannelID, msgID, text)
	if err := s.dealPostMessageRepo.CreateDealPostMessageAndSetDealInProgress(ctx, m); err != nil {
		logger.Error("create deal_post_message", "deal_id", deal.ID, "error", err)
		releaseLock(marketentity.DealActionLockStatusFailed)
		return
	}
	releaseLock(marketentity.DealActionLockStatusCompleted)
	logger.Info("sent and saved", "deal_id", deal.ID, "channel_id", *listing.ChannelID, "message_id", msgID)
}

// handleExpiredLock checks the last lock for a deal+action combo. If it's expired and
// still in locked status, releases it as failed. Returns error only on unexpected failures.
func (s *service) handleExpiredLock(ctx context.Context, logger *slog.Logger, dealID int64, actionType marketentity.DealActionType) error {
	lastLock, err := s.dealActionLockRepo.GetLastDealActionLock(ctx, dealID, actionType)
	if err != nil {
		if errors.Is(err, marketerrors.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("get last lock: %w", err)
	}
	if lastLock.Status == marketentity.DealActionLockStatusLocked && !lastLock.ExpireAt.After(time.Now()) {
		_ = s.dealActionLockRepo.ReleaseDealActionLock(ctx, lastLock.ID, marketentity.DealActionLockStatusFailed)
		logger.Warn("released expired lock as failed", "deal_id", dealID, "lock_id", lastLock.ID)
	}
	return nil
}

func (s *service) makeDealPostMessage(deal *marketentity.Deal, channelID int64, msgID int64, text string) *marketentity.DealPostMessage {
	return &marketentity.DealPostMessage{
		DealID:      deal.ID,
		ChannelID:   channelID,
		MessageID:   msgID,
		PostMessage: text,
		Status:      marketentity.DealPostMessageStatusExists,
		NextCheck:   time.Now().Add(postCheckAdvanceHour),
		UntilTs:     time.Now().Add(time.Duration(deal.Duration) * time.Hour),
	}
}

const lastMessagesRecoveryLimit = 20

// toMTProtoEntities converts Bot API MessageEntity slice to MTProto entity slice.
func toMTProtoEntities(entities []helpertelegram.MessageEntity) []tg.MessageEntityClass {
	if len(entities) == 0 {
		return nil
	}
	result := make([]tg.MessageEntityClass, 0, len(entities))
	for _, e := range entities {
		var me tg.MessageEntityClass
		switch e.Type {
		case "bold":
			me = &tg.MessageEntityBold{Offset: e.Offset, Length: e.Length}
		case "italic":
			me = &tg.MessageEntityItalic{Offset: e.Offset, Length: e.Length}
		case "underline":
			me = &tg.MessageEntityUnderline{Offset: e.Offset, Length: e.Length}
		case "strikethrough":
			me = &tg.MessageEntityStrike{Offset: e.Offset, Length: e.Length}
		case "code":
			me = &tg.MessageEntityCode{Offset: e.Offset, Length: e.Length}
		case "pre":
			me = &tg.MessageEntityPre{Offset: e.Offset, Length: e.Length, Language: e.Language}
		case "text_link":
			me = &tg.MessageEntityTextURL{Offset: e.Offset, Length: e.Length, URL: e.URL}
		case "text_mention":
			if e.User != nil {
				me = &tg.MessageEntityMentionName{Offset: e.Offset, Length: e.Length, UserID: e.User.ID}
			}
		case "spoiler":
			me = &tg.MessageEntitySpoiler{Offset: e.Offset, Length: e.Length}
		case "blockquote":
			me = &tg.MessageEntityBlockquote{Offset: e.Offset, Length: e.Length}
		case "expandable_blockquote":
			me = &tg.MessageEntityBlockquote{Offset: e.Offset, Length: e.Length, Collapsed: true}
		case "url":
			me = &tg.MessageEntityURL{Offset: e.Offset, Length: e.Length}
		case "mention":
			me = &tg.MessageEntityMention{Offset: e.Offset, Length: e.Length}
		case "hashtag":
			me = &tg.MessageEntityHashtag{Offset: e.Offset, Length: e.Length}
		case "bot_command":
			me = &tg.MessageEntityBotCommand{Offset: e.Offset, Length: e.Length}
		case "email":
			me = &tg.MessageEntityEmail{Offset: e.Offset, Length: e.Length}
		case "phone_number":
			me = &tg.MessageEntityPhone{Offset: e.Offset, Length: e.Length}
		case "custom_emoji":
			if e.CustomEmojiID != "" {
				docID, _ := strconv.ParseInt(e.CustomEmojiID, 10, 64)
				me = &tg.MessageEntityCustomEmoji{Offset: e.Offset, Length: e.Length, DocumentID: docID}
			}
		default:
			continue // unknown entity type, skip
		}
		if me != nil {
			result = append(result, me)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func (s *service) sendChannelMessage(ctx context.Context, channelID int64, accessHash int64, text string, entities []tg.MessageEntityClass) (int64, error) {
	peer := &tg.InputPeerChannel{ChannelID: helpertelegram.ToMTProtoChannelID(channelID), AccessHash: accessHash}
	req := &tg.MessagesSendMessageRequest{
		Peer:     peer,
		Message:  text,
		RandomID: rand.Int63(),
	}
	if len(entities) > 0 {
		req.Entities = entities
	}
	result, err := s.telegramClient.API().MessagesSendMessage(ctx, req)
	if err != nil {
		return 0, err
	}
	upd, ok := result.(*tg.Updates)
	if !ok {
		return 0, nil
	}
	for _, u := range upd.Updates {
		if msg, ok := u.(*tg.UpdateMessageID); ok {
			return int64(msg.ID), nil
		}
	}
	return 0, nil
}

func (s *service) RunDealPostCheckerWorker(ctx context.Context) {
	worker.RunTicker(ctx, "deal_post_checker", postCheckerInterval, false, s.runDealPostCheckerOnce)
}

func (s *service) runDealPostCheckerOnce(ctx context.Context, logger *slog.Logger) {
	list, err := s.dealPostMessageRepo.ListDealPostMessageExistsWithNextCheckBefore(ctx, time.Now())
	if err != nil {
		logger.Error("list", "error", err)
		return
	}

	for _, m := range list {
		channel, err := s.channelRepo.GetChannelByID(ctx, m.ChannelID)
		if err != nil {
			continue
		}
		channel, err = s.ensureChannelAccess(ctx, channel)
		if err != nil {
			logger.Error("ensure channel access", "channel_id", m.ChannelID, "error", err)
			continue
		}

		deal, err := s.dealRepo.GetDealByID(ctx, m.DealID)
		if err != nil {
			logger.Error("get deal for checker", "deal_id", m.DealID, "error", err)
			continue
		}

		var exists bool
		if deal.IsStory() {
			exists, err = s.getChannelStoryExists(ctx, m.ChannelID, channel.AccessHash, m.MessageID)
		} else {
			exists, err = s.getChannelMessageExists(ctx, m.ChannelID, channel.AccessHash, m.MessageID)
		}
		if err != nil {
			logger.Error("check existence", "id", m.ID, "is_story", deal.IsStory(), "error", err)
			continue
		}
		if !exists {
			_ = s.dealPostMessageRepo.UpdateDealPostMessageStatus(ctx, m.ID, marketentity.DealPostMessageStatusDeleted)
			logger.Info("deleted", "id", m.ID, "deal_id", m.DealID, "is_story", deal.IsStory())
			continue
		}
		nextCheck := m.NextCheck.Add(postCheckAdvanceHour)
		if nextCheck.After(m.UntilTs) {
			_ = s.dealPostMessageRepo.UpdateDealPostMessageStatus(ctx, m.ID, marketentity.DealPostMessageStatusPassed)
			logger.Info("passed", "id", m.ID, "deal_id", m.DealID)
		} else {
			_ = s.dealPostMessageRepo.UpdateDealPostMessageStatusAndNextCheck(ctx, m.ID, marketentity.DealPostMessageStatusExists, nextCheck)
		}
	}
}

type channelMessage struct {
	ID   int64
	Text string
}

// See https://core.telegram.org/api/offsets: offset = offsetFromID(offsetID) + addOffset; results are reverse chronological (newest first).
// For "most recent N": offsetID=0, addOffset=0, limit=N.
// For "around message ID": offsetID=messageID, addOffset=-halfWindow, limit=windowSize.
func (s *service) getChannelHistory(ctx context.Context, channelID int64, accessHash int64, offsetID int, addOffset int, limit int) ([]channelMessage, error) {
	result, err := s.telegramClient.API().MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer: &tg.InputPeerChannel{
			ChannelID:  helpertelegram.ToMTProtoChannelID(channelID),
			AccessHash: accessHash,
		},
		OffsetID:   offsetID,
		OffsetDate: 0,
		AddOffset:  addOffset,
		Limit:      limit,
		MaxID:      0,
		MinID:      0,
		Hash:       0,
	})
	if err != nil {
		return nil, err
	}
	var messages []tg.MessageClass
	switch r := result.(type) {
	case *tg.MessagesMessages:
		messages = r.Messages
	case *tg.MessagesChannelMessages:
		messages = r.Messages
	default:
		return nil, nil
	}
	out := make([]channelMessage, 0, len(messages))
	for _, msg := range messages {
		m, ok := msg.(*tg.Message)
		if !ok {
			continue
		}
		out = append(out, channelMessage{ID: int64(m.ID), Text: m.Message})
	}
	return out, nil
}

func (s *service) getChannelMessageExists(ctx context.Context, channelID int64, accessHash int64, messageID int64) (bool, error) {
	const windowAround = 20
	msgs, err := s.getChannelHistory(ctx, channelID, accessHash, int(messageID), -windowAround/2, windowAround)
	if err != nil {
		return false, err
	}
	for _, m := range msgs {
		if m.ID == messageID {
			return true, nil
		}
	}
	return false, nil
}

func (s *service) tryRecoverPostFromChannel(ctx context.Context, channelID int64, accessHash int64, text string) (int64, bool) {
	msgs, err := s.getChannelHistory(ctx, channelID, accessHash, 0, 0, lastMessagesRecoveryLimit)
	if err != nil {
		return 0, false
	}
	for _, m := range msgs {
		if m.Text == text {
			return m.ID, true
		}
	}
	return 0, false
}

func (s *service) processStoryDeal(ctx context.Context, logger *slog.Logger, deal *marketentity.Deal, listing *marketentity.Listing, channel *marketentity.Channel) {
	mediaType, mediaFileID := marketentity.GetMediaFromDetails(deal.Details)
	if mediaType == "" || mediaFileID == "" {
		logger.Error("skip story deal, no media in details", "deal_id", deal.ID)
		return
	}
	if postedAt, ok := marketentity.GetPostedAtFromDetails(deal.Details); ok && time.Now().Before(postedAt) {
		logger.Debug("skip story deal, posted_at in future", "deal_id", deal.ID, "posted_at", postedAt)
		return
	}

	// Stories cannot be recovered by text matching if the lock expired while locked.
	// Release the expired lock as failed and let the next cycle retry from scratch.
	if err := s.handleExpiredLock(ctx, logger, deal.ID, marketentity.DealActionTypePostMessage); err != nil {
		logger.Error("handle expired lock for story", "deal_id", deal.ID, "error", err)
		return
	}

	lockID, err := s.dealActionLockRepo.TakeDealActionLock(ctx, deal.ID, marketentity.DealActionTypePostMessage)
	if err != nil {
		logger.Debug("skip story deal, lock not acquired", "deal_id", deal.ID, "error", err)
		return
	}
	releaseLock := func(status marketentity.DealActionLockStatus) {
		_ = s.dealActionLockRepo.ReleaseDealActionLock(ctx, lockID, status)
	}

	inputMedia, err := s.uploadMediaForStory(ctx, mediaType, mediaFileID)
	if err != nil {
		logger.Error("upload media for story", "deal_id", deal.ID, "error", err)
		releaseLock(marketentity.DealActionLockStatusFailed)
		return
	}

	caption := marketentity.GetMessageFromDetails(deal.Details)
	var botEntities []helpertelegram.MessageEntity
	if raw := marketentity.GetBotAPIEntitiesFromDetails(deal.Details); len(raw) > 0 {
		if err := json.Unmarshal(raw, &botEntities); err != nil {
			logger.Error("unmarshal entities from deal details", "deal_id", deal.ID, "error", err)
		}
	}
	mtprotoEntities := toMTProtoEntities(botEntities)

	storyID, err := s.sendChannelStory(ctx, *listing.ChannelID, channel.AccessHash, inputMedia, caption, mtprotoEntities)
	if err != nil {
		logger.Error("send story", "deal_id", deal.ID, "error", err)
		releaseLock(marketentity.DealActionLockStatusFailed)
		return
	}

	m := s.makeDealPostMessage(deal, *listing.ChannelID, storyID, caption)
	if err := s.dealPostMessageRepo.CreateDealPostMessageAndSetDealInProgress(ctx, m); err != nil {
		logger.Error("create deal_post_message for story", "deal_id", deal.ID, "error", err)
		releaseLock(marketentity.DealActionLockStatusFailed)
		return
	}
	releaseLock(marketentity.DealActionLockStatusCompleted)
	logger.Info("story sent and saved", "deal_id", deal.ID, "channel_id", *listing.ChannelID, "story_id", storyID)
}

func (s *service) uploadMediaForStory(ctx context.Context, mediaType string, mediaFileID string) (tg.InputMediaClass, error) {
	fileURL, err := s.botAPIClient.GetFileURL(ctx, mediaFileID)
	if err != nil {
		return nil, fmt.Errorf("get file URL: %w", err)
	}

	u := uploader.NewUploader(s.telegramClient.API())
	inputFile, err := u.FromURL(ctx, fileURL)
	if err != nil {
		return nil, fmt.Errorf("upload from URL: %w", err)
	}

	switch mediaType {
	case "photo":
		return &tg.InputMediaUploadedPhoto{
			File: inputFile,
		}, nil
	case "video":
		return &tg.InputMediaUploadedDocument{
			File:     inputFile,
			MimeType: "video/mp4",
			Attributes: []tg.DocumentAttributeClass{
				&tg.DocumentAttributeVideo{
					SupportsStreaming: true,
				},
			},
		}, nil
	case "animation":
		return &tg.InputMediaUploadedDocument{
			File:         inputFile,
			MimeType:     "video/mp4",
			NosoundVideo: true,
			Attributes: []tg.DocumentAttributeClass{
				&tg.DocumentAttributeAnimated{},
				&tg.DocumentAttributeVideo{
					SupportsStreaming: true,
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported media type for story: %s", mediaType)
	}
}

func (s *service) sendChannelStory(ctx context.Context, channelID int64, accessHash int64, media tg.InputMediaClass, caption string, entities []tg.MessageEntityClass) (int64, error) {
	peer := &tg.InputPeerChannel{
		ChannelID:  helpertelegram.ToMTProtoChannelID(channelID),
		AccessHash: accessHash,
	}

	req := &tg.StoriesSendStoryRequest{
		Peer:     peer,
		Media:    media,
		RandomID: rand.Int63(),
		PrivacyRules: []tg.InputPrivacyRuleClass{
			&tg.InputPrivacyValueAllowAll{},
		},
	}
	req.SetPinned(true)
	if caption != "" {
		req.SetCaption(caption)
	}
	if len(entities) > 0 {
		req.SetEntities(entities)
	}

	result, err := s.telegramClient.API().StoriesSendStory(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("stories.sendStory: %w", err)
	}

	upd, ok := result.(*tg.Updates)
	if !ok {
		return 0, fmt.Errorf("unexpected response type from stories.sendStory: %T", result)
	}
	for _, u := range upd.Updates {
		if storyUpdate, ok := u.(*tg.UpdateStory); ok {
			if storyItem, ok := storyUpdate.Story.(*tg.StoryItem); ok {
				return int64(storyItem.ID), nil
			}
		}
	}
	return 0, fmt.Errorf("no UpdateStory found in stories.sendStory response")
}

func (s *service) getChannelStoryExists(ctx context.Context, channelID int64, accessHash int64, storyID int64) (bool, error) {
	peer := &tg.InputPeerChannel{
		ChannelID:  helpertelegram.ToMTProtoChannelID(channelID),
		AccessHash: accessHash,
	}
	result, err := s.telegramClient.API().StoriesGetStoriesByID(ctx, &tg.StoriesGetStoriesByIDRequest{
		Peer: peer,
		ID:   []int{int(storyID)},
	})
	if err != nil {
		return false, fmt.Errorf("stories.getStoriesByID: %w", err)
	}
	for _, story := range result.Stories {
		if _, ok := story.(*tg.StoryItem); ok && story.GetID() == int(storyID) {
			return true, nil
		}
	}
	return false, nil
}
