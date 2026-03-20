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
	ForumCommandPreview = "/preview"
	ForumCommandEdit    = "/edit"
)

// IsForumCommand checks whether the message text or caption starts with a known command prefix,
// or is a wizard control command (/start, /empty).
func IsForumCommand(message *telegram.UpdateMessage) bool {
	text := message.Text
	if text == "" {
		text = message.Caption
	}

	return strings.HasPrefix(text, "/preview") ||
		strings.HasPrefix(text, "/edit") ||
		text == "/start" ||
		text == "/empty"
}

// HandleForumCommand processes a forum command message. Returns nil if handled successfully.
func (s *service) HandleForumCommand(ctx context.Context, message *telegram.UpdateMessage) error {
	chatID := message.Chat.ID
	threadID := message.MessageThreadID

	// Parse command from text or caption
	text := message.Text
	if text == "" {
		text = message.Caption
	}

	// /start cancels interface{} active wizard — no deal lookup needed
	if text == "/start" {
		if message.From != nil {
			return s.handleStartInForum(ctx, chatID, threadID, message.From.ID)
		}
		return nil
	}

	// /empty is only meaningful during an active wizard
	if text == "/empty" {
		if message.From != nil {
			handled, err := s.HandleWizardStep(ctx, chatID, threadID, message)
			if !handled {
				return nil
			}
			return err
		}
		return nil
	}

	// Look up the deal forum topic
	topic, _, err := s.forumTopicRepo.GetDealForumTopicByChatAndThread(ctx, chatID, threadID)
	if err != nil {
		if errors.Is(err, marketerrors.ErrNotFound) {
			return nil // Not a deal topic, ignore
		}
		return fmt.Errorf("get forum topic: %w", err)
	}

	// Load the deal
	deal, err := s.dealRepo.GetDealByID(ctx, topic.DealID)
	if err != nil {
		if errors.Is(err, marketerrors.ErrNotFound) {
			s.sendToThread(ctx, chatID, threadID, "Deal not found.")
			return nil
		}
		return fmt.Errorf("get deal: %w", err)
	}

	// Check deal status is draft
	if deal.Status != entity.DealStatusDraft {
		s.sendToThread(ctx, chatID, threadID, "Commands are only available in draft status.")
		return nil
	}

	switch {
	case strings.HasPrefix(text, "/preview"):
		return s.handlePreview(ctx, deal, chatID, threadID)
	case strings.HasPrefix(text, "/edit"):
		return s.handleEdit(ctx, deal, chatID, threadID, message)
	}

	return nil
}

func (s *service) handlePreview(ctx context.Context, deal *entity.Deal, chatID int64, threadID int64) error {
	msg := entity.GetMessageFromDetails(deal.Details)
	mediaType, _ := entity.GetMediaFromDetails(deal.Details)

	if deal.IsStory() {
		if mediaType == "" {
			s.sendToThread(ctx, chatID, threadID, "No media set yet. Use /edit to set the story media.")
			return nil
		}
	} else {
		if strings.TrimSpace(msg) == "" && mediaType == "" {
			s.sendToThread(ctx, chatID, threadID, "No message set yet. Use /edit to craft the ad message.")
			return nil
		}
	}

	buttons := entity.GetButtonsFromDetails(deal.Details)
	var markup *telegram.InlineKeyboardMarkup
	if len(buttons) > 0 {
		markup = buildURLButtons(buttons)
	}

	s.sendAdPreview(ctx, chatID, threadID, deal.Details, markup)
	return nil
}

// handleEdit starts the multi-step wizard by storing an empty wizard state in Redis.
func (s *service) handleEdit(ctx context.Context, deal *entity.Deal, chatID int64, threadID int64, message *telegram.UpdateMessage) error {
	if message.From == nil {
		return nil
	}
	userID := message.From.ID

	// Verify the sender is a deal participant
	if userID != deal.LessorID && userID != deal.LesseeID {
		return nil
	}

	state := wizardState{
		DealID:  deal.ID,
		AdType:  string(deal.Type),
		Details: json.RawMessage("{}"),
	}

	if deal.IsStory() {
		// Stories skip text and buttons -- go directly to media
		state.Step = wizardStepMedia
	} else {
		state.Step = wizardStepText
	}

	stateBytes, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal wizard state: %w", err)
	}
	key := wizardKey(userID, threadID)
	if err := s.wizardStore.Set(ctx, key, string(stateBytes), wizardTTL); err != nil {
		return fmt.Errorf("set wizard state: %w", err)
	}

	if deal.IsStory() {
		s.sendToThread(ctx, chatID, threadID, "Send a photo, video, or GIF for the story (9:16 portrait aspect ratio required)")
	} else {
		s.sendToThread(ctx, chatID, threadID, "Type post text or leave it /empty")
	}
	return nil
}

// handleStartInForum cancels interface{} active wizard for this user+thread.
func (s *service) handleStartInForum(ctx context.Context, chatID, threadID int64, userID int64) error {
	key := wizardKey(userID, threadID)
	_ = s.wizardStore.Del(ctx, key)
	s.sendToThread(ctx, chatID, threadID, "Post crafting cancelled.")
	return nil
}

// IsWizardActive checks if there's an active wizard for this user+thread.
func (s *service) IsWizardActive(ctx context.Context, userID, threadID int64) bool {
	key := wizardKey(userID, threadID)
	val, err := s.wizardStore.Get(ctx, key)
	return err == nil && val != ""
}

// HandleWizardStep processes the next step in the wizard. Returns (true, nil) if handled.
func (s *service) HandleWizardStep(ctx context.Context, chatID, threadID int64, message *telegram.UpdateMessage) (bool, error) {
	userID := message.From.ID
	key := wizardKey(userID, threadID)
	val, err := s.wizardStore.Get(ctx, key)
	if err != nil || val == "" {
		return false, nil // No active wizard
	}

	var state wizardState
	if err := json.Unmarshal([]byte(val), &state); err != nil {
		_ = s.wizardStore.Del(ctx, key)
		return false, nil
	}

	switch state.Step {
	case wizardStepText:
		return true, s.handleWizardText(ctx, chatID, threadID, message, &state, key)
	case wizardStepMedia:
		return true, s.handleWizardMedia(ctx, chatID, threadID, message, &state, key)
	case wizardStepButtons:
		return true, s.handleWizardButtons(ctx, chatID, threadID, message, &state, key)
	}
	return false, nil
}

func (s *service) handleWizardText(ctx context.Context, chatID, threadID int64, message *telegram.UpdateMessage, state *wizardState, key string) error {
	text := message.Text
	if text == "" {
		text = message.Caption
	}

	if text != "/empty" {
		rawEntities := message.Entities
		if message.Text == "" {
			rawEntities = message.CaptionEntities
		}

		// Set message text
		var err error
		state.Details, err = entity.SetMessageInDetails(state.Details, text)
		if err != nil {
			return fmt.Errorf("set message: %w", err)
		}

		// Handle entities
		if len(rawEntities) > 0 {
			entJSON, err := json.Marshal(rawEntities)
			if err != nil {
				return fmt.Errorf("marshal entities: %w", err)
			}
			state.Details, err = entity.SetRawEntitiesInDetails(state.Details, entJSON)
			if err != nil {
				return fmt.Errorf("set entities: %w", err)
			}
		}

		// If message has media attached (e.g., photo with caption), detect and store it
		mediaType, mediaFileID := detectMediaFromMessage(message)
		if mediaType != "" {
			state.Details, _ = entity.SetMediaInDetails(state.Details, mediaType, mediaFileID)
			// Move entities to caption_entities for media messages
			if len(rawEntities) > 0 {
				entJSON, _ := json.Marshal(rawEntities)
				state.Details, _ = entity.SetRawCaptionEntitiesInDetails(state.Details, entJSON)
				state.Details, _ = entity.SetRawEntitiesInDetails(state.Details, nil)
			}
		}
	}

	// If media was already set with the text message, skip the media step
	mediaType, _ := entity.GetMediaFromDetails(state.Details)
	if mediaType != "" {
		state.Step = wizardStepButtons
		s.saveWizardState(ctx, key, state)
		s.sendToThread(ctx, chatID, threadID, "Send a list of buttons for your post. Please use this format:\n\nButton text 1 - http://example.com\nButton text 2 - http://example.com - danger\nButton text 3 - http://example.com - danger - 5348227245599105972\n\nSend /empty to not add buttons")
		return nil
	}

	state.Step = wizardStepMedia
	s.saveWizardState(ctx, key, state)
	s.sendToThread(ctx, chatID, threadID, "Send image/video/gif that must be added to post or leave it /empty")
	return nil
}

func (s *service) handleWizardMedia(ctx context.Context, chatID, threadID int64, message *telegram.UpdateMessage, state *wizardState, key string) error {
	text := message.Text
	if text == "" {
		text = message.Caption
	}

	isStory := state.AdType == string(entity.AdTypeStory)

	if text == "/empty" {
		if isStory {
			// Stories require media -- cannot skip
			s.sendToThread(ctx, chatID, threadID, "Stories require media. Please send a photo, video, or GIF with 9:16 aspect ratio.")
			return nil
		}
	} else {
		if isStory {
			// For stories, validate dimensions
			mediaType, mediaFileID, width, height := detectMediaDimensionsFromMessage(message)
			if mediaType == "" {
				s.sendToThread(ctx, chatID, threadID, "Please send a photo, video, or GIF with 9:16 aspect ratio.")
				return nil
			}
			if !isStoryAspectRatio(width, height) {
				s.sendToThread(ctx, chatID, threadID, fmt.Sprintf("Invalid aspect ratio (%dx%d). Stories require 9:16 portrait orientation. Please send media with the correct ratio.", width, height))
				return nil
			}
			if message.Caption != "" {
				s.sendToThread(ctx, chatID, threadID, "Note: caption text is not supported for stories and has been ignored. Only the media will be used.")
			}
			var err error
			state.Details, err = entity.SetMediaInDetails(state.Details, mediaType, mediaFileID)
			if err != nil {
				return fmt.Errorf("set media: %w", err)
			}
		} else {
			mediaType, mediaFileID := detectMediaFromMessage(message)
			if mediaType == "" {
				s.sendToThread(ctx, chatID, threadID, "Please send a photo, video, or GIF. Or send /empty to skip.")
				return nil
			}
			var err error
			state.Details, err = entity.SetMediaInDetails(state.Details, mediaType, mediaFileID)
			if err != nil {
				return fmt.Errorf("set media: %w", err)
			}

			// Move entities to caption_entities for media messages (text was set in the previous step)
			rawEntities := entity.GetRawEntitiesFromDetails(state.Details)
			if len(rawEntities) > 0 {
				state.Details, _ = entity.SetRawCaptionEntitiesInDetails(state.Details, rawEntities)
				state.Details, _ = entity.SetRawEntitiesInDetails(state.Details, nil)
			}
		}
	}

	if isStory {
		// Stories skip buttons -- go directly to preview
		s.saveWizardState(ctx, key, state)
		confirmMarkup := buildConfirmSignMarkup(state.DealID, nil)
		s.sendAdPreview(ctx, chatID, threadID, state.Details, confirmMarkup)
		return nil
	}

	state.Step = wizardStepButtons
	s.saveWizardState(ctx, key, state)
	s.sendToThread(ctx, chatID, threadID, "Send a list of buttons for your post. Please use this format:\n\nButton text 1 - http://example.com\nButton text 2 - http://example.com - danger\nButton text 3 - http://example.com - danger - 5348227245599105972\n\nSend /empty to not add buttons")
	return nil
}

func (s *service) handleWizardButtons(ctx context.Context, chatID, threadID int64, message *telegram.UpdateMessage, state *wizardState, key string) error {
	text := message.Text
	if text == "" {
		text = message.Caption
	}

	if text != "/empty" {
		btns, err := parseButtonLines(text)
		if err != nil {
			s.sendToThread(ctx, chatID, threadID, fmt.Sprintf("Invalid button format: %s\n\nPlease use:\nButton text - http://example.com [style] [emoji_id]\n\nSend /empty to skip.", err.Error()))
			return nil
		}
		state.Details, err = entity.SetButtonsInDetails(state.Details, btns)
		if err != nil {
			return fmt.Errorf("set buttons: %w", err)
		}
	}

	// Save final state to Redis (before preview)
	s.saveWizardState(ctx, key, state)

	// Send ad preview with "Confirm & Sign" button
	btns := entity.GetButtonsFromDetails(state.Details)
	confirmMarkup := buildConfirmSignMarkup(state.DealID, btns)
	s.sendAdPreview(ctx, chatID, threadID, state.Details, confirmMarkup)
	return nil
}

const maxButtons = 10

// parseButtonLines parses a multi-line button definition string.
// Each line format: "<text> - <url> - <style> - <custom_emoji_id>"
// Only text and url are required; style defaults to "default", emoji is optional.
func parseButtonLines(text string) ([]entity.DealDetailsButton, error) {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	var buttons []entity.DealDetailsButton
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, " - ")
		if len(parts) < 2 {
			return nil, fmt.Errorf("missing ' - ' separator in: %s", line)
		}
		btnText := strings.TrimSpace(parts[0])
		if btnText == "" {
			return nil, fmt.Errorf("empty button text")
		}

		btnURL := strings.TrimSpace(parts[1])
		parsed, err := url.ParseRequestURI(btnURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return nil, fmt.Errorf("invalid URL: %s", btnURL)
		}

		style := "default"
		if len(parts) > 2 {
			s := strings.TrimSpace(parts[2])
			if s != "" {
				style = s
			}
		}
		if !entity.ValidButtonStyles[style] {
			return nil, fmt.Errorf("invalid style: %s", style)
		}

		emoji := ""
		if len(parts) > 3 {
			e := strings.TrimSpace(parts[3])
			if e != "" {
				if _, err := strconv.ParseInt(e, 10, 64); err != nil {
					return nil, fmt.Errorf("invalid emoji ID (must be numeric): %s", e)
				}
				emoji = e
			}
		}

		buttons = append(buttons, entity.DealDetailsButton{
			Text:  btnText,
			URL:   btnURL,
			Style: style,
			Emoji: emoji,
		})
	}
	if len(buttons) == 0 {
		return nil, fmt.Errorf("no buttons provided")
	}
	if len(buttons) > maxButtons {
		return nil, fmt.Errorf("too many buttons (max %d)", maxButtons)
	}
	return buttons, nil
}

// saveWizardState marshals and stores the wizard state in Redis, logging errors.
func (s *service) saveWizardState(ctx context.Context, key string, state *wizardState) {
	stateBytes, err := json.Marshal(state)
	if err != nil {
		slog.Error("marshal wizard state", "error", err)
		return
	}
	if err := s.wizardStore.Set(ctx, key, string(stateBytes), wizardTTL); err != nil {
		slog.Error("save wizard state", "error", err, "key", key)
	}
}

// HandleApproveCallback handles the "Approve" button callback query.
// It signs the deal for the pressing user and preserves interface{} URL buttons on the message.
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

	// Success: answer callback and remove Approve button (preserve URL buttons)
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
	case errors.Is(err, marketerrors.ErrDealDetailsMediaRequired):
		return "Set the story media first."
	case errors.Is(err, marketerrors.ErrDealNotDraft):
		return "Deal is no longer in draft."
	case errors.Is(err, marketerrors.ErrNotFound):
		return "Deal not found."
	default:
		return "Signing failed. Try again."
	}
}

// buildMarkupWithoutApprove returns markup with only the URL buttons (if interface{}), without the Approve button.
func (s *service) buildMarkupWithoutApprove(ctx context.Context, dealID int64) *telegram.InlineKeyboardMarkup {
	deal, err := s.dealRepo.GetDealByID(ctx, dealID)
	if err != nil {
		return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{}}
	}
	buttons := entity.GetButtonsFromDetails(deal.Details)
	if len(buttons) > 0 {
		return buildURLButtons(buttons)
	}
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{}}
}

// buildConfirmSignMarkup builds an InlineKeyboardMarkup with optional URL button rows and a "Confirm & Sign" callback button row.
func buildConfirmSignMarkup(dealID int64, buttons []entity.DealDetailsButton) *telegram.InlineKeyboardMarkup {
	var rows [][]telegram.InlineKeyboardButton
	for _, btn := range buttons {
		b := btn
		rows = append(rows, []telegram.InlineKeyboardButton{toInlineKeyboardButton(&b)})
	}
	rows = append(rows, []telegram.InlineKeyboardButton{
		{Text: "Confirm & Sign", CallbackData: fmt.Sprintf("confirm_sign:%d", dealID)},
	})
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// HandleConfirmSignCallback handles the "Confirm & Sign" button callback query.
// If a wizard state exists for this user+thread, it saves the wizard details to DB first.
// Then it signs the deal, removes the button, and notifies the other side.
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

	// Check for wizard state — if exists, verify ownership and save details to DB before signing
	if callbackQuery.Message != nil && callbackQuery.Message.Chat != nil {
		threadID := callbackQuery.Message.MessageThreadID
		key := wizardKey(userID, threadID)
		if val, _ := s.wizardStore.Get(ctx, key); val != "" {
			var state wizardState
			if err := json.Unmarshal([]byte(val), &state); err == nil && state.DealID == dealID {
				// Verify the user is a party to the deal before writing details
				deal, err := s.dealRepo.GetDealByID(ctx, dealID)
				if err != nil {
					if errors.Is(err, marketerrors.ErrNotFound) {
						_ = s.wizardStore.Del(ctx, key)
						return nil
					}
					return fmt.Errorf("get deal for wizard confirm: %w", err)
				}
				if userID != deal.LessorID && userID != deal.LesseeID {
					_ = s.wizardStore.Del(ctx, key)
					return nil
				}
				canonDetails, valErr := entity.ValidateDealDetails(state.Details)
				if valErr != nil {
					s.sendToThread(ctx, callbackQuery.Message.Chat.ID, threadID, "Invalid deal details. Please try /edit again.")
					_ = s.wizardStore.Del(ctx, key)
					return nil
				}
				if deal.IsStory() {
					canonDetails = entity.StripNonMediaFields(canonDetails)
				}
				if err := s.dealRepo.UpdateDealDetailsAndClearSignatures(ctx, dealID, canonDetails); err != nil {
					s.sendToThread(ctx, callbackQuery.Message.Chat.ID, threadID, "Failed to save changes.")
					return fmt.Errorf("save wizard details to db: %w", err)
				}
				_ = s.wizardStore.Del(ctx, key)
			}
		}
	}

	deal, signErr := s.dealSigner.SignDeal(ctx, userID, dealID)
	if signErr != nil {
		alertText := mapSignError(signErr)
		if err := s.telegramForum.AnswerCallbackQuery(ctx, callbackQuery.ID, alertText, true); err != nil {
			slog.Error("answer confirm sign callback query with alert", "error", err)
		}
		return nil
	}

	// Success: answer callback and remove "Confirm & Sign" button (preserve URL buttons)
	if err := s.telegramForum.AnswerCallbackQuery(ctx, callbackQuery.ID, "Signed!", false); err != nil {
		slog.Error("answer confirm sign callback query", "error", err)
	}

	buttons := entity.GetButtonsFromDetails(deal.Details)

	if callbackQuery.Message != nil && callbackQuery.Message.Chat != nil {
		var markup *telegram.InlineKeyboardMarkup
		if len(buttons) > 0 {
			markup = buildURLButtons(buttons)
		} else {
			markup = &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{}}
		}
		if err := s.telegramForum.EditMessageReplyMarkup(ctx, callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID, markup); err != nil {
			slog.Error("edit message reply markup after confirm sign", "error", err)
		}
	}

	topic, err := s.forumTopicRepo.GetDealForumTopicByDealID(ctx, dealID)
	if err != nil {
		return nil
	}

	// Determine editor's side
	side := "lessee"
	if userID == deal.LessorID {
		side = "lessor"
	}

	// Notify the other side with the actual ad preview + Approve button
	otherChatID, otherThreadID := s.otherSide(topic, side)
	approveMarkup := buildApproveMarkup(dealID, buttons)
	s.sendAdPreview(ctx, otherChatID, otherThreadID, deal.Details, approveMarkup)

	return nil
}

// buildApproveMarkup builds an InlineKeyboardMarkup with optional URL button rows and an Approve callback button row.
func buildApproveMarkup(dealID int64, buttons []entity.DealDetailsButton) *telegram.InlineKeyboardMarkup {
	var rows [][]telegram.InlineKeyboardButton
	for _, btn := range buttons {
		b := btn
		rows = append(rows, []telegram.InlineKeyboardButton{toInlineKeyboardButton(&b)})
	}
	rows = append(rows, []telegram.InlineKeyboardButton{
		{Text: "Approve", CallbackData: fmt.Sprintf("approve_edit:%d", dealID)},
	})
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// buildURLButtons creates an InlineKeyboardMarkup with one row per button.
func buildURLButtons(buttons []entity.DealDetailsButton) *telegram.InlineKeyboardMarkup {
	var rows [][]telegram.InlineKeyboardButton
	for _, btn := range buttons {
		b := btn
		rows = append(rows, []telegram.InlineKeyboardButton{toInlineKeyboardButton(&b)})
	}
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
	raw := entity.GetBotAPIEntitiesFromDetails(details)
	if len(raw) == 0 {
		return nil
	}
	var entities []telegram.MessageEntity
	if err := json.Unmarshal(raw, &entities); err != nil {
		slog.Error("unmarshal entities from deal details", "error", err)
		return nil
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
	case "animation":
		event.Animation = mediaFileID
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

// detectMediaFromMessage extracts the media type and file ID from a message, if interface{}.
func detectMediaFromMessage(message *telegram.UpdateMessage) (mediaType, mediaFileID string) {
	if len(message.Photo) > 0 {
		// Pick the largest photo (last in array per Telegram docs)
		return "photo", message.Photo[len(message.Photo)-1].FileID
	}
	if message.Video != nil {
		return "video", message.Video.FileID
	}
	if message.Animation != nil {
		return "animation", message.Animation.FileID
	}
	return "", ""
}

// detectMediaDimensionsFromMessage extracts the media type, file ID, width, and height from a message, if interface{}.
func detectMediaDimensionsFromMessage(message *telegram.UpdateMessage) (mediaType, mediaFileID string, width, height int) {
	if len(message.Photo) > 0 {
		// Pick the largest photo (last in array per Telegram docs)
		p := message.Photo[len(message.Photo)-1]
		return "photo", p.FileID, p.Width, p.Height
	}
	if message.Video != nil {
		return "video", message.Video.FileID, message.Video.Width, message.Video.Height
	}
	if message.Animation != nil {
		return "animation", message.Animation.FileID, message.Animation.Width, message.Animation.Height
	}
	return "", "", 0, 0
}

// isStoryAspectRatio checks whether the given dimensions have a 9:16 portrait aspect ratio
// within a +/-5% tolerance. The ideal ratio is 9/16 = 0.5625; accepted range is [0.53, 0.60].
func isStoryAspectRatio(width, height int) bool {
	if width <= 0 || height <= 0 {
		return false
	}
	ratio := float64(width) / float64(height)
	return ratio >= 0.53 && ratio <= 0.60
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
