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

	"ads-mrkt/internal/helpers/telegram"
	"ads-mrkt/internal/market/domain"
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
	msg := domain.GetMessageFromDetails(deal.Details)
	if strings.TrimSpace(msg) == "" {
		s.sendToThread(ctx, chatID, threadID, "No message set yet. Use /edit <text> to set the ad message.")
		return nil
	}

	mediaType, mediaFileID := domain.GetMediaFromDetails(deal.Details)
	button := domain.GetButtonFromDetails(deal.Details)

	var markup *telegram.InlineKeyboardMarkup
	if button != nil {
		markup = buildURLButton(button)
	}

	// Extract entities from details
	var textEntities []telegram.MessageEntity
	var captionEntities []telegram.MessageEntity
	if raw := domain.GetRawEntitiesFromDetails(deal.Details); len(raw) > 0 {
		_ = json.Unmarshal(raw, &textEntities)
	}
	if raw := domain.GetRawCaptionEntitiesFromDetails(deal.Details); len(raw) > 0 {
		_ = json.Unmarshal(raw, &captionEntities)
	}

	switch mediaType {
	case "photo":
		return s.telegramForum.SendPhotoToThread(ctx, chatID, threadID, mediaFileID, msg, markup, captionEntities)
	case "video":
		return s.telegramForum.SendVideoToThread(ctx, chatID, threadID, mediaFileID, msg, markup, captionEntities)
	default:
		if markup != nil {
			_, err := s.telegramForum.SendMessageToThreadWithMarkup(ctx, chatID, threadID, msg, markup, textEntities)
			return err
		}
		return s.telegramForum.SendMessageToThread(ctx, chatID, threadID, msg, textEntities)
	}
}

func (s *service) handleEdit(ctx context.Context, deal *entity.Deal, chatID int64, threadID int64, message *telegram.UpdateMessage) error {
	// Extract text and entities: choose Text/Entities or Caption/CaptionEntities
	rawText := message.Text
	rawEntities := message.Entities
	if rawText == "" {
		rawText = message.Caption
		rawEntities = message.CaptionEntities
	}
	newMessage := ""
	if strings.HasPrefix(rawText, "/edit ") {
		newMessage = strings.TrimPrefix(rawText, "/edit ")
	}
	if strings.TrimSpace(newMessage) == "" {
		s.sendToThread(ctx, chatID, threadID, "Usage: /edit <message text>")
		return nil
	}

	// Adjust entity offsets: "/edit " is 6 UTF-16 code units (all ASCII)
	const editPrefixUTF16Len = 6
	adjustedEntities := telegram.AdjustEntitiesOffset(rawEntities, editPrefixUTF16Len)

	// Detect media from the message
	var mediaType, mediaFileID string
	if len(message.Photo) > 0 {
		// Pick the largest photo (last in array per Telegram docs)
		mediaType = "photo"
		mediaFileID = message.Photo[len(message.Photo)-1].FileID
	} else if message.Video != nil {
		mediaType = "video"
		mediaFileID = message.Video.FileID
	}

	// Determine which entities key to use:
	// If the message has media, the text came from Caption -> store as caption_entities.
	// If no media, the text came from Text -> store as entities.
	isCaption := mediaType != ""

	// Update deal details
	details := deal.Details
	var err error
	details, err = domain.SetMessageInDetails(details, newMessage)
	if err != nil {
		s.sendToThread(ctx, chatID, threadID, "Failed to update message.")
		return fmt.Errorf("set message in details: %w", err)
	}
	details, err = domain.SetMediaInDetails(details, mediaType, mediaFileID)
	if err != nil {
		s.sendToThread(ctx, chatID, threadID, "Failed to update media.")
		return fmt.Errorf("set media in details: %w", err)
	}

	// Marshal adjusted entities to JSON for storage
	var entitiesJSON json.RawMessage
	if len(adjustedEntities) > 0 {
		entitiesJSON, err = json.Marshal(adjustedEntities)
		if err != nil {
			return fmt.Errorf("marshal entities: %w", err)
		}
	}

	// Store entities in the appropriate key, clear the other
	if isCaption {
		details, err = domain.SetRawEntitiesInDetails(details, nil)
		if err != nil {
			return fmt.Errorf("clear entities: %w", err)
		}
		details, err = domain.SetRawCaptionEntitiesInDetails(details, entitiesJSON)
		if err != nil {
			return fmt.Errorf("set caption_entities: %w", err)
		}
	} else {
		details, err = domain.SetRawEntitiesInDetails(details, entitiesJSON)
		if err != nil {
			return fmt.Errorf("set entities: %w", err)
		}
		details, err = domain.SetRawCaptionEntitiesInDetails(details, nil)
		if err != nil {
			return fmt.Errorf("clear caption_entities: %w", err)
		}
	}

	// Save to DB (clears both signatures)
	if err := s.dealRepo.UpdateDealDetailsAndClearSignatures(ctx, deal.ID, details); err != nil {
		s.sendToThread(ctx, chatID, threadID, "Failed to save changes.")
		return fmt.Errorf("update deal details: %w", err)
	}

	// Send confirmation with "Confirm & Sign" button to editor
	confirmMarkup := buildConfirmSignMarkup(deal.ID)
	confirmText := "Message updated. Review and press Confirm & Sign to proceed."
	if _, err := s.telegramForum.SendMessageToThreadWithMarkup(ctx, chatID, threadID, confirmText, confirmMarkup, nil); err != nil {
		slog.Error("send confirm sign message after edit", "deal_id", deal.ID, "error", err)
	}

	return nil
}

func (s *service) handleSetButton(ctx context.Context, deal *entity.Deal, chatID int64, threadID int64, text string) error {
	// Parse: /set_button <text> <url> <style> <emoji>
	// Last 3 fields are url, style, emoji. Everything between /set_button and those 3 is the button text.
	parts := strings.Fields(text)
	if len(parts) < 5 {
		s.sendToThread(ctx, chatID, threadID, "Usage: /set_button <text> <url> <style> <emoji>\nStyles: red, green, blue, default\nEmoji: any emoji or 0 for none\nExample: /set_button Buy Now https://example.com blue \U0001f525")
		return nil
	}

	emoji := parts[len(parts)-1]
	style := parts[len(parts)-2]
	btnURL := parts[len(parts)-3]
	btnText := strings.Join(parts[1:len(parts)-3], " ")

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
	validStyles := map[string]bool{"red": true, "green": true, "blue": true, "default": true}
	if !validStyles[style] {
		s.sendToThread(ctx, chatID, threadID, "Invalid style. Must be one of: red, green, blue, default")
		return nil
	}

	// Emoji "0" means no emoji
	if emoji == "0" {
		emoji = ""
	}

	// Check that message is set
	msg := domain.GetMessageFromDetails(deal.Details)
	if strings.TrimSpace(msg) == "" {
		s.sendToThread(ctx, chatID, threadID, "Set the ad message first with /edit before adding a button.")
		return nil
	}

	button := &domain.DealDetailsButton{
		Text:  btnText,
		URL:   btnURL,
		Style: style,
		Emoji: emoji,
	}

	details, detailsErr := domain.SetButtonInDetails(deal.Details, button)
	if detailsErr != nil {
		s.sendToThread(ctx, chatID, threadID, "Failed to update button.")
		return detailsErr
	}

	if err := s.dealRepo.UpdateDealDetailsAndClearSignatures(ctx, deal.ID, details); err != nil {
		s.sendToThread(ctx, chatID, threadID, "Failed to save changes.")
		return fmt.Errorf("update deal details: %w", err)
	}

	// Send confirmation with "Confirm & Sign" button to editor
	confirmMsg := fmt.Sprintf("Button set: %s -> %s (style: %s", btnText, btnURL, style)
	if emoji != "" {
		confirmMsg += ", emoji: " + emoji
	}
	confirmMsg += ") Review and press Confirm & Sign to proceed."
	confirmMarkup := buildConfirmSignMarkup(deal.ID)
	if _, err := s.telegramForum.SendMessageToThreadWithMarkup(ctx, chatID, threadID, confirmMsg, confirmMarkup, nil); err != nil {
		slog.Error("send confirm sign message after set_button", "deal_id", deal.ID, "error", err)
	}

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
	if signErr := s.dealSigner.SignDeal(ctx, callbackQuery.From.ID, dealID); signErr != nil {
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
	button := domain.GetButtonFromDetails(deal.Details)
	if button != nil {
		return buildURLButton(button)
	}
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{}}
}

// buildConfirmSignMarkup builds an InlineKeyboardMarkup with a single "Confirm & Sign" callback button.
func buildConfirmSignMarkup(dealID int64) *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: "Confirm & Sign", CallbackData: fmt.Sprintf("confirm_sign:%d", dealID)},
			},
		},
	}
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

	if signErr := s.dealSigner.SignDeal(ctx, userID, dealID); signErr != nil {
		alertText := mapSignError(signErr)
		if err := s.telegramForum.AnswerCallbackQuery(ctx, callbackQuery.ID, alertText, true); err != nil {
			slog.Error("answer confirm sign callback query with alert", "error", err)
		}
		return nil
	}

	// Success: answer callback and remove "Confirm & Sign" button
	if err := s.telegramForum.AnswerCallbackQuery(ctx, callbackQuery.ID, "Signed!", false); err != nil {
		slog.Error("answer confirm sign callback query", "error", err)
	}

	if callbackQuery.Message != nil && callbackQuery.Message.Chat != nil {
		emptyMarkup := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{}}
		if err := s.telegramForum.EditMessageReplyMarkup(ctx, callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID, emptyMarkup); err != nil {
			slog.Error("edit message reply markup after confirm sign", "error", err)
		}
	}

	// Get deal and topic to notify the other side
	deal, err := s.dealRepo.GetDealByID(ctx, dealID)
	if err != nil || deal == nil {
		return nil
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

	// Notify the other side with Approve button + ad message preview
	otherChatID, otherThreadID := s.otherSide(topic, side)
	msg := domain.GetMessageFromDetails(deal.Details)
	mediaType, _ := domain.GetMediaFromDetails(deal.Details)
	button := domain.GetButtonFromDetails(deal.Details)

	notifyText := fmt.Sprintf("The other party proposed an ad message:\n\n%s", msg)
	if mediaType != "" {
		notifyText += fmt.Sprintf("\n\n[%s attached]", mediaType)
	}
	notifyText += "\n\nPress Approve to accept, or use /edit to propose changes."
	approveMarkup := buildApproveMarkup(dealID, button)
	if _, err := s.telegramForum.SendMessageToThreadWithMarkup(ctx, otherChatID, otherThreadID, notifyText, approveMarkup, nil); err != nil {
		slog.Error("send confirm sign notification to other side", "deal_id", dealID, "error", err)
	}

	return nil
}

// buildApproveMarkup builds an InlineKeyboardMarkup with an optional URL button row and an Approve callback button row.
func buildApproveMarkup(dealID int64, button *domain.DealDetailsButton) *telegram.InlineKeyboardMarkup {
	var rows [][]telegram.InlineKeyboardButton
	if button != nil {
		btnText := button.Text
		if button.Emoji != "" {
			btnText = button.Emoji + " " + btnText
		}
		rows = append(rows, []telegram.InlineKeyboardButton{
			{Text: btnText, URL: button.URL},
		})
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

// sendToThread is a helper that sends a plain text message to a forum topic, logging errors.
func (s *service) sendToThread(ctx context.Context, chatID int64, threadID int64, text string) {
	if err := s.telegramForum.SendMessageToThread(ctx, chatID, threadID, text, nil); err != nil {
		slog.Error("send message to thread", "chat_id", chatID, "thread_id", threadID, "error", err)
	}
}

// buildURLButton creates an InlineKeyboardMarkup with a single URL button.
func buildURLButton(button *domain.DealDetailsButton) *telegram.InlineKeyboardMarkup {
	btnText := button.Text
	if button.Emoji != "" {
		btnText = button.Emoji + " " + btnText
	}
	return &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{
					Text: btnText,
					URL:  button.URL,
				},
			},
		},
	}
}
