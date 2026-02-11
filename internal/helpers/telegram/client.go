package telegram

import (
	"bytes"
	"context"
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
func (c *APIClient) SendMessage(ctx context.Context, chatID int64, text string) (*SentMessage, error) {
	allow, err := c.rateLimiter.CheckLimits(ctx)
	if err != nil {
		return nil, fmt.Errorf("check rate limiting failed: %w", err)
	}
	if !allow {
		return nil, fmt.Errorf("too many requests to send message in telegram")
	}

	url := c.buildTelegramURL(telegramPathSendMessage)
	payload := MessagePayload{
		Payload: Payload{ChatID: chatID},
		Text:    text,
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

// ForceReplyMarkup is the reply_markup for "Reply to this message" (Bot API ForceReply).
// See https://core.telegram.org/bots/api#forcereply and https://core.telegram.org/constructor/replyKeyboardForceReply.
type ForceReplyMarkup struct {
	ForceReply            bool   `json:"force_reply"`
	InputFieldPlaceholder string `json:"input_field_placeholder,omitempty"` // 1-64 characters
	Selective             bool   `json:"selective,omitempty"`
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
		ChatID      int64             `json:"chat_id"`
		Text        string            `json:"text"`
		ReplyMarkup ForceReplyMarkup  `json:"reply_markup"`
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
