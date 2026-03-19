package deal_chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"

	evententity "ads-mrkt/internal/event/domain/entity"
	"ads-mrkt/internal/helpers/telegram"
	"ads-mrkt/internal/market/domain/entity"
	marketerrors "ads-mrkt/internal/market/domain/errors"
)

const (
	ForumCommandPreview   = "/preview"
	ForumCommandEdit      = "/edit"
	ForumCommandSetButton = "/set_button"
)

// IsForumCommand checks whether the message text or caption starts with a known command prefix.
func IsForumCommand(message *telegram.UpdateMessage) bool {
	text := message.Text
	if text == "" {
		text = message.Caption
	}

	return strings.HasPrefix(text, "/preview") ||
		strings.HasPrefix(text, "/edit") ||
		strings.HasPrefix(text, "/set_button ")
}

// HandleForumCommand processes a forum command message. Returns nil if handled successfully.
func (s *service) HandleForumCommand(ctx context.Context, message *telegram.UpdateMessage) error {
	chatID := message.Chat.ID
	threadID := message.MessageThreadID

	// Look up the deal forum topic
	topic, _, err := s.forumTopicRepo.GetDealForumTopicByChatAndThread(ctx, chatID, threadID)
	if err != nil {
		return fmt.Errorf("get forum topic: %w", err)
	}
	if topic == nil {
		return nil // Not a deal topic, ignore
	}

	// Load the deal
	deal, err := s.dealRepo.GetDealByID(ctx, topic.DealID)
	if err != nil {
		return fmt.Errorf("get deal: %w", err)
	}
	if deal == nil {
		s.sendToThread(ctx, chatID, threadID, "Deal not found.")
		return nil
	}

	// Check deal status is draft
	if deal.Status != entity.DealStatusDraft {
		s.sendToThread(ctx, chatID, threadID, "Commands are only available in draft status.")
		return nil
	}

	// Parse command from text or caption
	text := message.Text
	if text == "" {
		text = message.Caption
	}

	switch {
	case strings.HasPrefix(text, "/preview"):
		return s.handlePreview(ctx, deal, chatID, threadID)
	case strings.HasPrefix(text, "/edit"):
		return s.handleEdit(ctx, deal, chatID, threadID, message)
	case strings.HasPrefix(text, "/set_button"):
		return s.handleSetButton(ctx, deal, chatID, threadID, text)
	}

	return nil
}

func (s *service) handlePreview(ctx context.Context, deal *entity.Deal, chatID int64, threadID int64) error {
	msg := entity.GetMessageFromDetails(deal.Details)
	if strings.TrimSpace(msg) == "" {
		s.sendToThread(ctx, chatID, threadID, "No message set yet. Use /edit <text> to set the ad message.")
		return nil
	}

	button := entity.GetButtonFromDetails(deal.Details)
	var markup *telegram.InlineKeyboardMarkup
	if button != nil {
		markup = buildURLButton(button)
	}

	s.sendAdPreview(ctx, chatID, threadID, deal.Details, markup)
	return nil
}

func (s *service) handleEdit(ctx context.Context, deal *entity.Deal, chatID int64, threadID int64, message *telegram.UpdateMessage) error {
	// Extract text and entities: choose Text/Entities or Caption/CaptionEntities
	rawText := message.Text
	rawEntities := message.Entities
	if rawText == "" {
		rawText = message.Caption
		rawEntities = message.CaptionEntities
	}

	// Parse the message text after "/edit" prefix
	var newMessage string
	var adjustedEntities []telegram.MessageEntity
	if after, found := strings.CutPrefix(rawText, "/edit "); found {
		// Has text after "/edit "
		newMessage = after
		// Adjust entity offsets: "/edit " is 6 UTF-16 code units (all ASCII)
		const editPrefixUTF16Len = 6
		adjustedEntities = telegram.AdjustEntitiesOffset(rawEntities, editPrefixUTF16Len)
	}
	// else: rawText is "/edit" exactly (bare command) — newMessage stays ""

	// Detect media from the message
	mediaType, mediaFileID := detectMediaFromMessage(message)
	slog.Debug("handleEdit: detected media", "media_type", mediaType, "media_file_id_len", len(mediaFileID), "new_message_len", len(newMessage))

	// Validate: must have either text or media
	if strings.TrimSpace(newMessage) == "" && mediaType == "" {
		s.sendToThread(ctx, chatID, threadID, "Usage: /edit <message text>\nYou can also send a photo or video with /edit as caption.")
		return nil
	}

	// Update deal details
	details := deal.Details
	var err error
	details, err = entity.SetMessageInDetails(details, newMessage)
	if err != nil {
		s.sendToThread(ctx, chatID, threadID, "Failed to update message.")
		return fmt.Errorf("set message in details: %w", err)
	}
	details, err = entity.SetMediaInDetails(details, mediaType, mediaFileID)
	if err != nil {
		s.sendToThread(ctx, chatID, threadID, "Failed to update media.")
		return fmt.Errorf("set media in details: %w", err)
	}

	// Marshal adjusted entities to JSON for storage — always store under "entities"
	var entitiesJSON json.RawMessage
	if len(adjustedEntities) > 0 {
		entitiesJSON, err = json.Marshal(adjustedEntities)
		if err != nil {
			return fmt.Errorf("marshal entities: %w", err)
		}
	}
	details, err = entity.SetRawEntitiesInDetails(details, entitiesJSON)
	if err != nil {
		return fmt.Errorf("set entities: %w", err)
	}
	details, err = entity.SetRawCaptionEntitiesInDetails(details, nil)
	if err != nil {
		return fmt.Errorf("clear caption_entities: %w", err)
	}

	// Save to DB (clears both signatures)
	if err := s.dealRepo.UpdateDealDetailsAndClearSignatures(ctx, deal.ID, details); err != nil {
		s.sendToThread(ctx, chatID, threadID, "Failed to save changes.")
		return fmt.Errorf("update deal details: %w", err)
	}

	// Send the ad preview with "Confirm & Sign" button to the editor
	button := entity.GetButtonFromDetails(details)
	confirmMarkup := buildConfirmSignMarkup(deal.ID, button)
	s.sendAdPreview(ctx, chatID, threadID, details, confirmMarkup)

	return nil
}

func (s *service) handleSetButton(ctx context.Context, deal *entity.Deal, chatID int64, threadID int64, text string) error {
	// Parse: /set_button <text> <url> [style] [emoji]
	// style defaults to "default", emoji defaults to "0" (no emoji).
	// URL is always required (http:// or https://).
	const usageMsg = "Usage: /set_button <text> <url> [style] [emoji]\nStyles: danger, success, primary, default (default: default)\nEmoji: any emoji or 0 for none (default: 0)\nExample: /set_button Buy Now https://example.com primary \U0001f525"
	parts := strings.Fields(text)
	if len(parts) < 3 {
		s.sendToThread(ctx, chatID, threadID, usageMsg)
		return nil
	}

	// Find the URL: scan from the end for the rightmost http/https token.
	urlIdx := -1
	for i := len(parts) - 1; i >= 1; i-- {
		if strings.HasPrefix(parts[i], "http://") || strings.HasPrefix(parts[i], "https://") {
			urlIdx = i
			break
		}
	}
	// urlIdx must be >= 2 so there is at least one text token before the URL.
	if urlIdx < 2 {
		s.sendToThread(ctx, chatID, threadID, usageMsg)
		return nil
	}

	btnText := strings.Join(parts[1:urlIdx], " ")
	btnURL := parts[urlIdx]

	// Optional style (defaults to "default")
	style := "default"
	if urlIdx+1 < len(parts) {
		style = parts[urlIdx+1]
	}

	// Optional emoji (defaults to "0" meaning no emoji)
	emoji := "0"
	if urlIdx+2 < len(parts) {
		emoji = parts[urlIdx+2]
	}

	if btnText == "" {
		s.sendToThread(ctx, chatID, threadID, "Button text cannot be empty.")
		return nil
	}

	// Validate URL
	parsed, err := url.ParseRequestURI(btnURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		s.sendToThread(ctx, chatID, threadID, "Invalid URL. Must start with http:// or https://")
		return nil
	}

	// Validate style
	if !entity.ValidButtonStyles[style] {
		s.sendToThread(ctx, chatID, threadID, "Invalid style. Must be one of: danger, success, primary, default")
		return nil
	}

	// Emoji "0" means no emoji
	if emoji == "0" {
		emoji = ""
	}

	slog.Debug("set_button parsed", "deal_id", deal.ID, "btn_text", btnText, "btn_url", btnURL, "style", style, "emoji", emoji)

	// Check that message or media is set
	msg := entity.GetMessageFromDetails(deal.Details)
	mediaType, _ := entity.GetMediaFromDetails(deal.Details)
	if strings.TrimSpace(msg) == "" && mediaType == "" {
		s.sendToThread(ctx, chatID, threadID, "Set the ad message first with /edit before adding a button.")
		return nil
	}

	button := &entity.DealDetailsButton{
		Text:  btnText,
		URL:   btnURL,
		Style: style,
		Emoji: emoji,
	}

	details, detailsErr := entity.SetButtonInDetails(deal.Details, button)
	if detailsErr != nil {
		s.sendToThread(ctx, chatID, threadID, "Failed to update button.")
		return detailsErr
	}

	if err := s.dealRepo.UpdateDealDetailsAndClearSignatures(ctx, deal.ID, details); err != nil {
		s.sendToThread(ctx, chatID, threadID, "Failed to save changes.")
		return fmt.Errorf("update deal details: %w", err)
	}

	// Send the ad preview with "Confirm & Sign" button to the editor
	confirmMarkup := buildConfirmSignMarkup(deal.ID, button)
	s.sendAdPreview(ctx, chatID, threadID, details, confirmMarkup)

	return nil
}

// HandleApproveCallback handles the "Approve" button callback query.
// It signs the deal for the pressing user and preserves any URL button on the message.
func (s *service) HandleApproveCallback(ctx context.Context, callbackQuery *telegram.CallbackQuery) error {
	if callbackQuery == nil || !strings.HasPrefix(callbackQuery.Data, "approve_edit:") {
		return nil
	}

	dealIDStr := strings.TrimPrefix(callbackQuery.Data, "approve_edit:")
	dealID, err := strconv.ParseInt(dealIDStr, 10, 64)
	if err != nil {
		return nil
	}

	// Sign the deal for the pressing user
	if _, signErr := s.dealSigner.SignDeal(ctx, callbackQuery.From.ID, dealID); signErr != nil {
		// Show error alert, keep Approve button for retry
		alertText := mapSignError(signErr)
		if err := s.telegramForum.AnswerCallbackQuery(ctx, callbackQuery.ID, alertText, true); err != nil {
			slog.Error("answer callback query with alert", "error", err)
		}
		return nil
	}

	// Success: answer callback and remove Approve button (preserve URL button)
	if err := s.telegramForum.AnswerCallbackQuery(ctx, callbackQuery.ID, "Signed!", false); err != nil {
		slog.Error("answer callback query", "error", err)
	}

	if callbackQuery.Message != nil && callbackQuery.Message.Chat != nil {
		markup := s.buildMarkupWithoutApprove(ctx, dealID)
		if err := s.telegramForum.EditMessageReplyMarkup(ctx, callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID, markup); err != nil {
			slog.Error("edit message reply markup", "error", err)
		}
	}

	return nil
}

// mapSignError maps deal signing errors to user-friendly messages.
func mapSignError(err error) string {
	switch {
	case errors.Is(err, marketerrors.ErrWalletNotSet):
		return "Connect your wallet first."
	case errors.Is(err, marketerrors.ErrPayoutNotSet):
		return "Both parties must set payout addresses."
	case errors.Is(err, marketerrors.ErrDealDetailsMessageRequired):
		return "Set the ad message first."
	case errors.Is(err, marketerrors.ErrDealNotDraft):
		return "Deal is no longer in draft."
	case errors.Is(err, marketerrors.ErrNotFound):
		return "Deal not found."
	default:
		return "Signing failed. Try again."
	}
}

// buildMarkupWithoutApprove returns markup with only the URL button (if any), without the Approve button.
func (s *service) buildMarkupWithoutApprove(ctx context.Context, dealID int64) *telegram.InlineKeyboardMarkup {
	deal, err := s.dealRepo.GetDealByID(ctx, dealID)
	if err != nil || deal == nil {
		return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{}}
	}
	button := entity.GetButtonFromDetails(deal.Details)
	if button != nil {
		return buildURLButton(button)
	}
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{}}
}

// buildConfirmSignMarkup builds an InlineKeyboardMarkup with an optional URL button row and a "Confirm & Sign" callback button row.
func buildConfirmSignMarkup(dealID int64, button *entity.DealDetailsButton) *telegram.InlineKeyboardMarkup {
	var rows [][]telegram.InlineKeyboardButton
	if button != nil {
		rows = append(rows, []telegram.InlineKeyboardButton{toInlineKeyboardButton(button)})
	}
	rows = append(rows, []telegram.InlineKeyboardButton{
		{Text: "Confirm & Sign", CallbackData: fmt.Sprintf("confirm_sign:%d", dealID)},
	})
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// HandleConfirmSignCallback handles the "Confirm & Sign" button callback query.
// It signs the deal for the pressing user, removes the button, and notifies the other side.
func (s *service) HandleConfirmSignCallback(ctx context.Context, callbackQuery *telegram.CallbackQuery) error {
	if callbackQuery == nil || !strings.HasPrefix(callbackQuery.Data, "confirm_sign:") {
		return nil
	}

	dealIDStr := strings.TrimPrefix(callbackQuery.Data, "confirm_sign:")
	dealID, err := strconv.ParseInt(dealIDStr, 10, 64)
	if err != nil {
		return nil
	}

	userID := callbackQuery.From.ID

	deal, signErr := s.dealSigner.SignDeal(ctx, userID, dealID)
	if signErr != nil {
		alertText := mapSignError(signErr)
		if err := s.telegramForum.AnswerCallbackQuery(ctx, callbackQuery.ID, alertText, true); err != nil {
			slog.Error("answer confirm sign callback query with alert", "error", err)
		}
		return nil
	}

	// Success: answer callback and remove "Confirm & Sign" button (preserve URL button)
	if err := s.telegramForum.AnswerCallbackQuery(ctx, callbackQuery.ID, "Signed!", false); err != nil {
		slog.Error("answer confirm sign callback query", "error", err)
	}

	button := entity.GetButtonFromDetails(deal.Details)

	if callbackQuery.Message != nil && callbackQuery.Message.Chat != nil {
		var markup *telegram.InlineKeyboardMarkup
		if button != nil {
			markup = buildURLButton(button)
		} else {
			markup = &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{}}
		}
		if err := s.telegramForum.EditMessageReplyMarkup(ctx, callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID, markup); err != nil {
			slog.Error("edit message reply markup after confirm sign", "error", err)
		}
	}

	topic, err := s.forumTopicRepo.GetDealForumTopicByDealID(ctx, dealID)
	if err != nil || topic == nil {
		return nil
	}

	// Determine editor's side
	side := "lessee"
	if userID == deal.LessorID {
		side = "lessor"
	}

	// Notify the other side with the actual ad preview + Approve button
	otherChatID, otherThreadID := s.otherSide(topic, side)
	approveMarkup := buildApproveMarkup(dealID, button)
	s.sendAdPreview(ctx, otherChatID, otherThreadID, deal.Details, approveMarkup)

	return nil
}

// buildApproveMarkup builds an InlineKeyboardMarkup with an optional URL button row and an Approve callback button row.
func buildApproveMarkup(dealID int64, button *entity.DealDetailsButton) *telegram.InlineKeyboardMarkup {
	var rows [][]telegram.InlineKeyboardButton
	if button != nil {
		rows = append(rows, []telegram.InlineKeyboardButton{toInlineKeyboardButton(button)})
	}
	rows = append(rows, []telegram.InlineKeyboardButton{
		{Text: "Approve", CallbackData: fmt.Sprintf("approve_edit:%d", dealID)},
	})
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// otherSide returns (chatID, threadID) for the other side of the deal.
func (s *service) otherSide(topic *entity.DealForumTopic, side string) (int64, int64) {
	if side == "lessor" {
		return topic.LesseeChatID, topic.LesseeMessageThreadID
	}
	return topic.LessorChatID, topic.LessorMessageThreadID
}

// getEntitiesFromDetails extracts entities from deal details, falling back to caption_entities for backward compat.
func getEntitiesFromDetails(details json.RawMessage) []telegram.MessageEntity {
	var entities []telegram.MessageEntity
	if raw := entity.GetRawEntitiesFromDetails(details); len(raw) > 0 {
		_ = json.Unmarshal(raw, &entities)
	}
	if len(entities) == 0 {
		if raw := entity.GetRawCaptionEntitiesFromDetails(details); len(raw) > 0 {
			_ = json.Unmarshal(raw, &entities)
		}
	}
	return entities
}

// sendAdPreview sends the ad message preview (with media, entities, and provided markup) to a forum thread via the notification event system.
func (s *service) sendAdPreview(ctx context.Context, chatID, threadID int64, details json.RawMessage, markup *telegram.InlineKeyboardMarkup) {
	if s.notificationAdder == nil {
		return
	}

	msg := entity.GetMessageFromDetails(details)
	mediaType, mediaFileID := entity.GetMediaFromDetails(details)
	entities := getEntitiesFromDetails(details)

	event := &evententity.EventTelegramNotification{
		ChatID:   chatID,
		ThreadID: threadID,
		Message:  msg,
	}

	switch mediaType {
	case "photo":
		event.Photo = mediaFileID
	case "video":
		event.Video = mediaFileID
	}

	if len(entities) > 0 {
		if b, err := json.Marshal(entities); err == nil {
			event.Entities = string(b)
		}
	}
	if markup != nil && len(markup.InlineKeyboard) > 0 {
		if b, err := json.Marshal(markup.InlineKeyboard); err == nil {
			event.Buttons = string(b)
		}
	}

	if err := s.notificationAdder.AddTelegramNotificationEvent(ctx, event); err != nil {
		slog.Error("failed to add notification", "type", "ad_preview", "chat_id", chatID, "thread_id", threadID, "error", err)
	}
}

// sendToThread is a helper that sends a plain text message to a forum topic via the notification event system.
func (s *service) sendToThread(ctx context.Context, chatID int64, threadID int64, text string) {
	if s.notificationAdder == nil {
		return
	}
	if err := s.notificationAdder.AddTelegramNotificationEvent(ctx, &evententity.EventTelegramNotification{
		ChatID:   chatID,
		ThreadID: threadID,
		Message:  text,
	}); err != nil {
		slog.Error("failed to add notification", "type", "send_to_thread_helper", "msg", text, "chat_id", chatID, "thread_id", threadID, "error", err)
	}
}

// detectMediaFromMessage extracts the media type and file ID from a message, if any.
func detectMediaFromMessage(message *telegram.UpdateMessage) (mediaType, mediaFileID string) {
	if len(message.Photo) > 0 {
		// Pick the largest photo (last in array per Telegram docs)
		return "photo", message.Photo[len(message.Photo)-1].FileID
	}
	if message.Video != nil {
		return "video", message.Video.FileID
	}
	return "", ""
}

// toInlineKeyboardButton converts a DealDetailsButton to a Telegram InlineKeyboardButton
// with Style and IconCustomEmojiID set from the deal button config.
func toInlineKeyboardButton(button *entity.DealDetailsButton) telegram.InlineKeyboardButton {
	return telegram.InlineKeyboardButton{
		Text:              button.Text,
		URL:               button.URL,
		Style:             button.Style,
		IconCustomEmojiID: button.Emoji,
	}
}

// buildURLButton creates an InlineKeyboardMarkup with a single URL button.
func buildURLButton(button *entity.DealDetailsButton) *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{toInlineKeyboardButton(button)},
		},
	}
}
