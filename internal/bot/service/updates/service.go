package updates

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	evententity "ads-mrkt/internal/event/domain/entity"
	"ads-mrkt/internal/helpers/telegram"
)

type UpdateType string

const (
	UpdateCommandStart UpdateType = "start"
	UpdateCallback     UpdateType = "callback"
	UpdateUnknown      UpdateType = "unknown"

	groupName                     = "master"
	consumerName                  = "updates"
	readTelegramUpdateEventsLimit = 100
	telegramUpdateEventsAge       = 48 * time.Hour
	pendingPeriod                 = 15 * time.Second
	pendingMinIdle                = 30 * time.Second
)

type eventService interface {
	AddTelegramUpdateEvent(ctx context.Context, update *telegram.Update, createdAt time.Time) error
	ReadTelegramUpdateEvents(ctx context.Context, group string, consumer string, limit int64) ([]*evententity.EventTelegramUpdate, error)
	PendingTelegramUpdateEvents(ctx context.Context, group string, consumer string, limit int64, minIdle time.Duration) ([]*evententity.EventTelegramUpdate, error)
	AckMessages(ctx context.Context, group string, messageIDs []string) error
	TrimStreamByAge(ctx context.Context, age time.Duration) error
}

type telegramService interface {
	SendWelcomeMessage(ctx context.Context, chatID int64) error
	SetMessageReaction(ctx context.Context, chatID, messageID int64, emoji string) error
}

type marketDealChatService interface {
	SetRepliedMessageIfMatch(ctx context.Context, replyToChatID, replyToMessageID int64, repliedText string) error
}

type Service struct {
	telegramClient  telegramService
	eventService    eventService
	dealChatReplier marketDealChatService
}

func NewService(
	telegramClient telegramService,
	eventService eventService,
	dealChatReplier marketDealChatService,
) *Service {
	return &Service{
		telegramClient:  telegramClient,
		eventService:    eventService,
		dealChatReplier: dealChatReplier,
	}
}

func (s *Service) HandleUpdate(ctx context.Context, raw []byte) error {
	update, err := telegram.ParseUpdateData(raw)
	if err != nil {
		return nil
	}

	return s.eventService.AddTelegramUpdateEvent(ctx, update, time.Now())
}

func (s *Service) StartBackgroundProcessingUpdates(ctx context.Context) {
	go s.streamCleaner(ctx)
	go s.processPendingUpdates(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("context done, stopping background processing updates")
			return
		default:
			if err := s.processUpdates(ctx); err != nil {
				slog.Error("failed to process updates", "error", err)
			}
		}
	}
}

func (s *Service) processUpdates(ctx context.Context) error {
	telegramUpdateEvents, err := s.eventService.ReadTelegramUpdateEvents(ctx, groupName, consumerName, readTelegramUpdateEventsLimit)
	if err != nil {
		return fmt.Errorf("failed to get pending updates: %w", err)
	}

	if len(telegramUpdateEvents) == 0 {
		return nil
	}

	messageIDs := make([]string, 0, len(telegramUpdateEvents))
	for _, updateEvent := range telegramUpdateEvents {
		if err := s.processUpdate(ctx, updateEvent); err != nil {
			slog.Error("failed to process update", "error", err, "event_id", updateEvent.ID)
			continue
		}
		messageIDs = append(messageIDs, updateEvent.ID)
	}

	if len(messageIDs) > 0 {
		if err := s.eventService.AckMessages(ctx, groupName, messageIDs); err != nil {
			slog.Error("failed to ack telegram update messages", "error", err)
		}
	}

	return nil
}

func (s *Service) processUpdate(ctx context.Context, updateEvent *evententity.EventTelegramUpdate) error {
	update := updateEvent.Update
	updateType := s.getUpdateType(update)
	switch updateType {
	case UpdateCommandStart:
		err := s.telegramClient.SendWelcomeMessage(ctx, update.Message.Chat.ID)
		if err != nil {
			if strings.Contains(err.Error(), "USER_FORBIDDEN") {
				return nil
			}
			return fmt.Errorf("failed to send welcome message: %w", err)
		}
	}

	// If user replied to a message, check if it's a deal-chat invite and save the reply; then set reaction on the user's message.
	if update.Message != nil && update.Message.ReplyToMessage != nil && update.Message.ReplyToMessage.Chat != nil && s.dealChatReplier != nil {
		replyTo := update.Message.ReplyToMessage
		if err := s.dealChatReplier.SetRepliedMessageIfMatch(ctx, replyTo.Chat.ID, replyTo.MessageID, update.Message.Text); err != nil {
			slog.Error("failed to set deal chat replied message", "error", err, "chat_id", replyTo.Chat.ID, "message_id", replyTo.MessageID)
		}
		if update.Message.Chat == nil {
			return nil
		}

		// Reply was saved; set a reaction on the user's message to acknowledge.
		if err := s.telegramClient.SetMessageReaction(ctx, update.Message.Chat.ID, update.Message.MessageID, "👍"); err != nil {
			slog.Debug("failed to set message reaction on saved reply", "error", err, "chat_id", update.Message.Chat.ID, "message_id", update.Message.MessageID)
		}

	}
	return nil
}

func (s *Service) getUpdateType(update *telegram.Update) UpdateType {
	if update.Message != nil {
		if update.Message.Text == "/start" {
			return UpdateCommandStart
		}
	}
	return UpdateUnknown
}

func (s *Service) processPendingUpdates(ctx context.Context) {
	tickerPending := time.NewTicker(pendingPeriod)
	defer tickerPending.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("pending telegram update events processor stopped")
			return
		case <-tickerPending.C:
			events, err := s.eventService.PendingTelegramUpdateEvents(ctx, groupName, consumerName, readTelegramUpdateEventsLimit, pendingMinIdle)
			if err != nil {
				slog.Error("failed to read pending telegram update events", "error", err)
				continue
			}

			if len(events) == 0 {
				tickerPending.Reset(pendingPeriod)
				continue
			}

			messageIDs := make([]string, 0, len(events))
			for _, updateEvent := range events {
				if err := s.processUpdate(ctx, updateEvent); err != nil {
					slog.Error("failed to process pending update", "error", err, "event_id", updateEvent.ID)
					continue
				}
				messageIDs = append(messageIDs, updateEvent.ID)
			}

			if len(messageIDs) > 0 {
				if err := s.eventService.AckMessages(ctx, groupName, messageIDs); err != nil {
					slog.Error("failed to ack pending telegram update messages", "error", err)
				}
			}

			tickerPending.Reset(pendingPeriod)
		}
	}
}

func (s *Service) streamCleaner(ctx context.Context) {
	tickerPeriod := 24 * time.Hour
	ticker := time.NewTicker(tickerPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("telegram update events stream cleaner stopped")
			return
		case <-ticker.C:
			if err := s.eventService.TrimStreamByAge(ctx, telegramUpdateEventsAge); err != nil {
				slog.Error("failed to trim telegram update events stream by age", "err", err)
			}
			ticker.Reset(tickerPeriod)
		}
	}
}
