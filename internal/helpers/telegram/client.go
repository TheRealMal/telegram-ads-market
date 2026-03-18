package telegram

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"ads-mrkt/internal/helpers/ratelimiter"
	"ads-mrkt/internal/helpers/telegram/config"
)

type TelegramPath string

const (
	telegramBasePath                                = "https://api.telegram.org/bot"
	telegramFileBasePath                            = "https://api.telegram.org/file/bot"
	telegramPathSendMessage            TelegramPath = "/sendMessage"
	telegramPathSendVideo              TelegramPath = "/sendVideo"
	telegramPathGetFile                TelegramPath = "/getFile"
	telegramPathCreateInvoiceLink      TelegramPath = "/createInvoiceLink"
	telegramPathAnswerPreCheckoutQuery TelegramPath = "/answerPreCheckoutQuery"
	telegramPathRefundStarPayment      TelegramPath = "/refundStarPayment"
	telegramPathSetMessageReaction     TelegramPath = "/setMessageReaction"
	telegramPathCreateForumTopic       TelegramPath = "/createForumTopic"
	telegramPathDeleteForumTopic       TelegramPath = "/deleteForumTopic"
	telegramPathCopyMessage            TelegramPath = "/copyMessage"
	telegramPathDeleteMessage          TelegramPath = "/deleteMessage"
	telegramPathGetChatAdministrators  TelegramPath = "/getChatAdministrators"
	telegramPathPromoteChatMember      TelegramPath = "/promoteChatMember"
	telegramPathGetChat                TelegramPath = "/getChat"
	telegramPathSendPhoto              TelegramPath = "/sendPhoto"
	telegramPathAnswerCallbackQuery    TelegramPath = "/answerCallbackQuery"
	telegramPathEditMessageReplyMarkup TelegramPath = "/editMessageReplyMarkup"

	messageWelcome = "Start message"
	openAppURL     = "https://t.me/%s?startapp="
)

var (
	errorMarshallingJson = "error marshaling JSON: %w"
)

type redisClient interface {
	Decr(ctx context.Context, key string) (int64, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
}

type APIClient struct {
	token         string
	botUsername   string
	botWebAppName string
	secretToken   string
	httpClient    *http.Client
	rateLimiter   *ratelimiter.RateLimiter
}

func NewAPIClient(ctx context.Context, cfg config.Config, redisClient redisClient) *APIClient {
	return &APIClient{
		token:         cfg.Token,
		botUsername:   cfg.BotUsername,
		botWebAppName: cfg.BotWebAppName,
		secretToken:   cfg.SecretToken,
		httpClient:    &http.Client{},
		rateLimiter:   ratelimiter.NewRateLimiter(ctx, redisClient, cfg.RateLimit, time.Second),
	}
}

func (c *APIClient) SendWelcomeMessage(ctx context.Context, chatID int64) error {
	allow, err := c.rateLimiter.CheckLimits(ctx)
	if err != nil {
		return fmt.Errorf("check rate limiting failed: %w", err)
	}

	if !allow {
		return fmt.Errorf("too many requests to send welcome message in telegram")
	}

	url := c.buildTelegramURL(telegramPathSendMessage)
	payload := MessagePayload{
		Payload: Payload{
			ChatID:    chatID,
			ParseMode: ModeMarkdownV2,
			ReplyMarkup: ReplyMarkup{
				InlineKeyboardMarkup: InlineKeyboardMarkup{
					InlineKeyboard: [][]InlineKeyboardButton{
						{
							{Text: "Open", URL: fmt.Sprintf(openAppURL, c.botUsername)},
						},
					},
				},
			},
		},
		Text: messageWelcome,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf(errorMarshallingJson, err)
	}

	_, err = c.sendRequest(ctx, url, jsonData)
	return err
}

func (c *APIClient) CreateInvoiceLink(ctx context.Context, payload InvoiceLinkPayload) (string, error) {
	allow, err := c.rateLimiter.CheckLimits(ctx)
	if err != nil {
		return "", fmt.Errorf("check rate limiting failed: %w", err)
	}

	if !allow {
		return "", fmt.Errorf("too many requests to create invoice link in telegram")
	}

	uri := c.buildTelegramURL(telegramPathCreateInvoiceLink)
	slog.Debug("Creating invoice link", "uri", uri)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshaling JSON: %w", err)
	}

	slog.Debug("The payload is", "payload", payload, "serialized", jsonData)

	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		httpError := errors.Unwrap(err)
		return "", fmt.Errorf("error sending request: %w", httpError)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error parsing response body: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request failed with status %d, response body: %s", resp.StatusCode, string(bodyBytes))
	}

	var result InvoiceLinkResponse

	err = json.Unmarshal(bodyBytes, &result)
	if err != nil {
		return "", fmt.Errorf("couldn't unmarshal response: %w", err)
	}

	return result.Result, nil
}

func (c *APIClient) AnswerPreCheckoutQuery(ctx context.Context, queryID string, ok bool, errorMessage string) error {
	if queryID == "" {
		return nil
	}

	uri := c.buildTelegramURL(telegramPathAnswerPreCheckoutQuery)

	message := PreCheckoutAnswerPayload{
		PreCheckoutQueryID: queryID,
		OK:                 ok,
		ErrorMessage:       errorMessage,
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %w", err)
	}

	slog.Debug("The payload is", "payload", message, "serialized", jsonData)

	_, err = c.sendRequest(ctx, uri, jsonData)
	return err
}

func (c *APIClient) RefundStarPayment(ctx context.Context, userID int64, telegramPaymentChargeID string) error {
	if userID == 0 || telegramPaymentChargeID == "" {
		return fmt.Errorf("invalid refund parameters %d, %s", userID, telegramPaymentChargeID)
	}

	uri := c.buildTelegramURL(telegramPathRefundStarPayment)

	message := RefundStarPaymentPayload{
		UserID:                  userID,
		TelegramPaymentChargeID: telegramPaymentChargeID,
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %w", err)
	}

	slog.Debug("The payload is", "payload", message, "serialized", jsonData)

	_, err = c.sendRequest(ctx, uri, jsonData)
	return err
}

func (c *APIClient) buildTelegramURL(path TelegramPath) string { //nolint:unparam
	b := strings.Builder{}
	b.Grow(len(telegramBasePath) + len(c.token) + len(path))
	b.WriteString(telegramBasePath)
	b.WriteString(c.token)
	b.WriteString(string(path))
	return b.String()
}

// sendRequest sends a POST request and returns the response body on success.
func (c *APIClient) sendRequest(ctx context.Context, uri string, payload []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", uri, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		httpError := errors.Unwrap(err)
		return nil, fmt.Errorf("error sending request: %w", httpError)
	}

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var telegramResponse TelegramErrorResponse
		if jsonErr := json.Unmarshal(bodyBytes, &telegramResponse); jsonErr != nil {
			slog.Error("Failed to unmarshal Telegram response", "body", string(bodyBytes))
		}
		if mappedErr := telegramResponse.GetMappedError(); mappedErr != nil {
			return nil, mappedErr
		}
		return nil, fmt.Errorf("request failed with status %d, response body: %s", resp.StatusCode, string(bodyBytes))
	}

	return bodyBytes, nil
}

// ReactionTypeEmoji is used in setMessageReaction (Bot API: type "emoji").
type ReactionTypeEmoji struct {
	Type  string `json:"type"`  // "emoji"
	Emoji string `json:"emoji"` // e.g. "👍"
}

// SetMessageReaction sets the chosen reactions on a message. See https://core.telegram.org/bots/api#setmessagereaction.
// Pass empty emoji to clear reactions. Bots can set up to one reaction per message (non-premium).
func (c *APIClient) SetMessageReaction(ctx context.Context, chatID, messageID int64, emoji string) error {
	uri := c.buildTelegramURL(telegramPathSetMessageReaction)
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
	}
	if emoji != "" {
		payload["reaction"] = []ReactionTypeEmoji{{Type: "emoji", Emoji: emoji}}
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal setMessageReaction payload: %w", err)
	}
	_, err = c.sendRequest(ctx, uri, jsonData)
	return err
}

// SendMessage sends a text message to the chat and returns the sent message (chat_id, message_id) on success.
func (c *APIClient) SendMessage(ctx context.Context, chatID int64, text string, entities []MessageEntity) (*SentMessage, error) {
	allow, err := c.rateLimiter.CheckLimits(ctx)
	if err != nil {
		return nil, fmt.Errorf("check rate limiting failed: %w", err)
	}
	if !allow {
		return nil, fmt.Errorf("too many requests to send message in telegram")
	}

	url := c.buildTelegramURL(telegramPathSendMessage)
	payload := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	}
	if len(entities) > 0 {
		payload["entities"] = entities
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf(errorMarshallingJson, err)
	}

	body, err := c.sendRequest(ctx, url, jsonData)
	if err != nil {
		return nil, err
	}

	var result SendMessageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal sendMessage response: %w", err)
	}
	if !result.OK || result.Result == nil {
		return nil, fmt.Errorf("sendMessage returned ok=false or empty result")
	}
	return result.Result, nil
}

// SendMessageSimple sends a text message to the chat and returns only an error (for use by notification workers).
func (c *APIClient) SendMessageSimple(ctx context.Context, chatID int64, text string) error {
	_, err := c.SendMessage(ctx, chatID, text, nil)
	return err
}

// ForceReplyMarkup is the reply_markup for "Reply to this message" (Bot API ForceReply).
// See https://core.telegram.org/bots/api#forcereply and https://core.telegram.org/constructor/replyKeyboardForceReply.
type ForceReplyMarkup struct {
	ForceReply            bool   `json:"force_reply"`
	InputFieldPlaceholder string `json:"input_field_placeholder,omitempty"` // 1-64 characters
	Selective             bool   `json:"selective,omitempty"`
}

// CreateForumTopic creates a forum topic in a supergroup. Returns the message_thread_id of the new topic.
// See https://core.telegram.org/bots/api#createforumtopic
func (c *APIClient) CreateForumTopic(ctx context.Context, chatID int64, name string) (messageThreadID int64, err error) {
	uri := c.buildTelegramURL(telegramPathCreateForumTopic)
	payload := map[string]interface{}{
		"chat_id":              chatID,
		"name":                 name,
		"icon_custom_emoji_id": 5373251851074415873, // Custom emoji ID for initial topic state
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("marshal createForumTopic payload: %w", err)
	}
	body, err := c.sendRequest(ctx, uri, jsonData)
	if err != nil {
		return 0, err
	}
	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageThreadID int64 `json:"message_thread_id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("unmarshal createForumTopic response: %w", err)
	}
	if !result.OK {
		return 0, fmt.Errorf("createForumTopic returned ok=false")
	}
	return result.Result.MessageThreadID, nil
}

// DeleteForumTopic deletes a forum topic. See https://core.telegram.org/bots/api#deleteforumtopic
func (c *APIClient) DeleteForumTopic(ctx context.Context, chatID int64, messageThreadID int64) error {
	uri := c.buildTelegramURL(telegramPathDeleteForumTopic)
	payload := map[string]interface{}{
		"chat_id":           chatID,
		"message_thread_id": messageThreadID,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal deleteForumTopic payload: %w", err)
	}
	_, err = c.sendRequest(ctx, uri, jsonData)
	return err
}

// CopyMessage copies a message to a chat (optionally to a forum topic). See https://core.telegram.org/bots/api#copymessage
func (c *APIClient) CopyMessage(ctx context.Context, fromChatID int64, messageID int64, toChatID int64, toMessageThreadID *int64) (copiedMessageID int64, err error) {
	uri := c.buildTelegramURL(telegramPathCopyMessage)
	payload := map[string]interface{}{
		"chat_id":      toChatID,
		"from_chat_id": fromChatID,
		"message_id":   messageID,
	}
	if toMessageThreadID != nil {
		payload["message_thread_id"] = *toMessageThreadID
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("marshal copyMessage payload: %w", err)
	}
	body, err := c.sendRequest(ctx, uri, jsonData)
	if err != nil {
		return 0, err
	}
	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int64 `json:"message_id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("unmarshal copyMessage response: %w", err)
	}
	if !result.OK {
		return 0, fmt.Errorf("copyMessage returned ok=false")
	}
	return result.Result.MessageID, nil
}

// GetChatAdministrators returns a list of administrators in a chat.
// See https://core.telegram.org/bots/api#getchatadministrators
func (c *APIClient) GetChatAdministrators(ctx context.Context, chatID int64) ([]*ChatMember, error) {
	uri := c.buildTelegramURL(telegramPathGetChatAdministrators)
	payload := map[string]interface{}{"chat_id": chatID}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal getChatAdministrators: %w", err)
	}
	body, err := c.sendRequest(ctx, uri, jsonData)
	if err != nil {
		return nil, err
	}
	var result ChatMemberResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal getChatAdministrators: %w", err)
	}
	if !result.OK {
		return nil, fmt.Errorf("getChatAdministrators returned ok=false")
	}
	return result.Result, nil
}

// PromoteChatMember promotes a user in a channel/group.
// See https://core.telegram.org/bots/api#promotechatmember
func (c *APIClient) PromoteChatMember(ctx context.Context, chatID int64, userID int64, rights AdminPromoteRights) error {
	uri := c.buildTelegramURL(telegramPathPromoteChatMember)
	payload := map[string]interface{}{
		"chat_id":             chatID,
		"user_id":             userID,
		"can_post_messages":   rights.CanPostMessages,
		"can_edit_messages":   rights.CanEditMessages,
		"can_delete_messages": rights.CanDeleteMessages,
		"can_post_stories":    rights.CanPostStories,
		"can_delete_stories":  rights.CanDeleteStories,
		"can_manage_chat":     true,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal promoteChatMember: %w", err)
	}
	_, err = c.sendRequest(ctx, uri, jsonData)
	return err
}

// GetChat returns full information about a chat.
// See https://core.telegram.org/bots/api#getchat
func (c *APIClient) GetChat(ctx context.Context, chatID int64) (*ChatFullInfo, error) {
	uri := c.buildTelegramURL(telegramPathGetChat)
	payload := map[string]interface{}{"chat_id": chatID}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal getChat payload: %w", err)
	}
	body, err := c.sendRequest(ctx, uri, jsonData)
	if err != nil {
		return nil, err
	}
	var result GetChatResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal getChat response: %w", err)
	}
	if !result.OK || result.Result == nil {
		return nil, fmt.Errorf("getChat returned ok=false or empty result")
	}
	return result.Result, nil
}

// GetChannelPhotoBase64 fetches a channel's photo via getChat, downloads it,
// and returns the base64-encoded image. Returns empty string if no photo.
func (c *APIClient) GetChannelPhotoBase64(ctx context.Context, chatID int64) (string, error) {
	chat, err := c.GetChat(ctx, chatID)
	if err != nil {
		return "", fmt.Errorf("getChat for photo: %w", err)
	}
	if chat.Photo == nil || chat.Photo.BigFileID == "" {
		return "", nil
	}

	fileURL, err := c.GetFileURL(chat.Photo.SmallFileID)
	if err != nil {
		return "", fmt.Errorf("get file URL for photo: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return "", fmt.Errorf("create photo download request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download photo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("photo download failed with status %d", resp.StatusCode)
	}

	photoBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read photo body: %w", err)
	}

	return base64.StdEncoding.EncodeToString(photoBytes), nil
}

// SendChannelMessage sends a text message to a channel and returns the message ID.
// chatID must be in Bot API format (negative -100 prefix).
func (c *APIClient) SendChannelMessage(ctx context.Context, chatID int64, text string, entities []MessageEntity) (int64, error) {
	sent, err := c.SendMessage(ctx, chatID, text, entities)
	if err != nil {
		return 0, err
	}
	return sent.MessageID, nil
}

// DeleteMessage deletes a message in a chat.
func (c *APIClient) DeleteMessage(ctx context.Context, chatID int64, messageID int64) error {
	uri := c.buildTelegramURL(telegramPathDeleteMessage)
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal deleteMessage payload: %w", err)
	}
	_, err = c.sendRequest(ctx, uri, jsonData)
	return err
}

// CheckMessageExists checks if a message exists in a chat by copying it to probeChatID
// (a service account chat). If the copy succeeds, the message exists. The copied message
// stays in the service chat (no deletion needed — it serves as an audit trail).
// Returns false if the message does not exist.
func (c *APIClient) CheckMessageExists(ctx context.Context, chatID int64, messageID int64, probeChatID int64) (bool, error) {
	_, err := c.CopyMessage(ctx, chatID, messageID, probeChatID, nil)
	if err != nil {
		if strings.Contains(err.Error(), "message to copy not found") ||
			strings.Contains(err.Error(), "message not found") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// SendMessageToThread sends a text message to a specific forum topic (message_thread_id).
func (c *APIClient) SendMessageToThread(ctx context.Context, chatID int64, messageThreadID int64, text string, entities []MessageEntity) error {
	allow, err := c.rateLimiter.CheckLimits(ctx)
	if err != nil {
		return fmt.Errorf("check rate limiting failed: %w", err)
	}
	if !allow {
		return fmt.Errorf("too many requests to send message in telegram")
	}
	url := c.buildTelegramURL(telegramPathSendMessage)
	payload := map[string]interface{}{
		"chat_id":           chatID,
		"message_thread_id": messageThreadID,
		"text":              text,
	}
	if len(entities) > 0 {
		payload["entities"] = entities
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal sendMessage payload: %w", err)
	}
	_, err = c.sendRequest(ctx, url, jsonData)
	return err
}

// SendMessageToThreadWithMarkup sends a text message to a forum topic with optional inline keyboard.
func (c *APIClient) SendMessageToThreadWithMarkup(ctx context.Context, chatID int64, messageThreadID int64, text string, markup *InlineKeyboardMarkup, entities []MessageEntity) (*SentMessage, error) {
	allow, err := c.rateLimiter.CheckLimits(ctx)
	if err != nil {
		return nil, fmt.Errorf("check rate limiting failed: %w", err)
	}
	if !allow {
		return nil, fmt.Errorf("too many requests to send message in telegram")
	}
	url := c.buildTelegramURL(telegramPathSendMessage)
	payload := map[string]interface{}{
		"chat_id":           chatID,
		"message_thread_id": messageThreadID,
		"text":              text,
	}
	if markup != nil {
		payload["reply_markup"] = markup
	}
	if len(entities) > 0 {
		payload["entities"] = entities
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal sendMessage payload: %w", err)
	}
	body, err := c.sendRequest(ctx, url, jsonData)
	if err != nil {
		return nil, err
	}
	var result SendMessageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal sendMessage response: %w", err)
	}
	if !result.OK || result.Result == nil {
		return nil, fmt.Errorf("sendMessage returned ok=false or empty result")
	}
	return result.Result, nil
}

// SendPhotoToThread sends a photo (by file_id) to a forum topic with optional caption and inline keyboard.
func (c *APIClient) SendPhotoToThread(ctx context.Context, chatID int64, messageThreadID int64, photoFileID string, caption string, markup *InlineKeyboardMarkup, captionEntities []MessageEntity) error {
	allow, err := c.rateLimiter.CheckLimits(ctx)
	if err != nil {
		return fmt.Errorf("check rate limiting failed: %w", err)
	}
	if !allow {
		return fmt.Errorf("too many requests to send message in telegram")
	}
	url := c.buildTelegramURL(telegramPathSendPhoto)
	payload := map[string]interface{}{
		"chat_id":           chatID,
		"message_thread_id": messageThreadID,
		"photo":             photoFileID,
	}
	if caption != "" {
		payload["caption"] = caption
	}
	if markup != nil {
		payload["reply_markup"] = markup
	}
	if len(captionEntities) > 0 {
		payload["caption_entities"] = captionEntities
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal sendPhoto payload: %w", err)
	}
	_, err = c.sendRequest(ctx, url, jsonData)
	return err
}

// SendVideoToThread sends a video (by file_id) to a forum topic with optional caption and inline keyboard.
func (c *APIClient) SendVideoToThread(ctx context.Context, chatID int64, messageThreadID int64, videoFileID string, caption string, markup *InlineKeyboardMarkup, captionEntities []MessageEntity) error {
	allow, err := c.rateLimiter.CheckLimits(ctx)
	if err != nil {
		return fmt.Errorf("check rate limiting failed: %w", err)
	}
	if !allow {
		return fmt.Errorf("too many requests to send message in telegram")
	}
	url := c.buildTelegramURL(telegramPathSendVideo)
	payload := map[string]interface{}{
		"chat_id":           chatID,
		"message_thread_id": messageThreadID,
		"video":             videoFileID,
	}
	if caption != "" {
		payload["caption"] = caption
	}
	if markup != nil {
		payload["reply_markup"] = markup
	}
	if len(captionEntities) > 0 {
		payload["caption_entities"] = captionEntities
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal sendVideo payload: %w", err)
	}
	_, err = c.sendRequest(ctx, url, jsonData)
	return err
}

// AnswerCallbackQuery sends a response to a callback query from an inline keyboard button.
func (c *APIClient) AnswerCallbackQuery(ctx context.Context, callbackQueryID string, text string, showAlert bool) error {
	uri := c.buildTelegramURL(telegramPathAnswerCallbackQuery)
	payload := map[string]interface{}{
		"callback_query_id": callbackQueryID,
	}
	if text != "" {
		payload["text"] = text
	}
	if showAlert {
		payload["show_alert"] = true
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal answerCallbackQuery payload: %w", err)
	}
	_, err = c.sendRequest(ctx, uri, jsonData)
	return err
}

// EditMessageReplyMarkup edits the reply markup of a message. Pass nil to remove markup.
func (c *APIClient) EditMessageReplyMarkup(ctx context.Context, chatID int64, messageID int64, markup *InlineKeyboardMarkup) error {
	uri := c.buildTelegramURL(telegramPathEditMessageReplyMarkup)
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
	}
	if markup != nil {
		payload["reply_markup"] = markup
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal editMessageReplyMarkup payload: %w", err)
	}
	_, err = c.sendRequest(ctx, uri, jsonData)
	return err
}

// SendMessageWithForceReply sends a text message with reply_markup force_reply so the user is asked to reply.
// placeholder is optional (1-64 chars); use "" to omit.
func (c *APIClient) SendMessageWithForceReply(ctx context.Context, chatID int64, text string, placeholder string) (*SentMessage, error) {
	allow, err := c.rateLimiter.CheckLimits(ctx)
	if err != nil {
		return nil, fmt.Errorf("check rate limiting failed: %w", err)
	}
	if !allow {
		return nil, fmt.Errorf("too many requests to send message in telegram")
	}

	url := c.buildTelegramURL(telegramPathSendMessage)
	payload := struct {
		ChatID      int64            `json:"chat_id"`
		Text        string           `json:"text"`
		ReplyMarkup ForceReplyMarkup `json:"reply_markup"`
	}{
		ChatID: chatID,
		Text:   text,
		ReplyMarkup: ForceReplyMarkup{
			ForceReply:            true,
			InputFieldPlaceholder: placeholder,
		},
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf(errorMarshallingJson, err)
	}

	body, err := c.sendRequest(ctx, url, jsonData)
	if err != nil {
		return nil, err
	}

	var result SendMessageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal sendMessage response: %w", err)
	}
	if !result.OK || result.Result == nil {
		return nil, fmt.Errorf("sendMessage returned ok=false or empty result")
	}
	return result.Result, nil
}
