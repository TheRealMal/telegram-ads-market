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

type telegramNotificationEventService interface {
	ReadTelegramNotificationEvents(ctx context.Context, group, consumer string, limit int64) ([]*evententity.EventTelegramNotification, error)
	PendingTelegramNotificationEvents(ctx context.Context, group, consumer string, limit int64, minIdle time.Duration) ([]*evententity.EventTelegramNotification, error)
	AckTelegramNotificationMessages(ctx context.Context, group string, messageIDs []string) error
	TrimStreamByAge(ctx context.Context, age time.Duration) error
}

type telegramService interface {
	SendWelcomeMessage(ctx context.Context, chatID int64) error
	SendMessageSimple(ctx context.Context, chatID int64, text string) error
	SetMessageReaction(ctx context.Context, chatID, messageID int64, emoji string) error
}

type marketDealChatService interface {
	CopyMessageToOtherTopic(ctx context.Context, chatID int64, messageThreadID int64, messageID int64) error
}

type service struct {
	telegramClient        telegramService
	eventService          eventService
	notificationEventSvc  telegramNotificationEventService
	marketDealChatService marketDealChatService
}

func NewService(
	telegramClient telegramService,
	eventService eventService,
	notificationEventSvc telegramNotificationEventService,
	marketDealChatService marketDealChatService,
) *service {
	return &service{
		telegramClient:        telegramClient,
		eventService:          eventService,
		notificationEventSvc:  notificationEventSvc,
		marketDealChatService: marketDealChatService,
	}
}

func (s *service) HandleUpdate(ctx context.Context, raw []byte) error {
	update, err := telegram.ParseUpdateData(raw)
	if err != nil {
		return nil
	}

	return s.eventService.AddTelegramUpdateEvent(ctx, update, time.Now())
}

func (s *service) StartBackgroundProcessingUpdates(ctx context.Context) {
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

func (s *service) processUpdates(ctx context.Context) error {
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

func (s *service) processUpdate(ctx context.Context, updateEvent *evententity.EventTelegramUpdate) error {
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

	// If message is in a forum topic (deal chat), mirror it to the other side's topic.
	if update.Message != nil && update.Message.Chat != nil && update.Message.MessageThreadID != 0 {
		if err := s.marketDealChatService.CopyMessageToOtherTopic(ctx, update.Message.Chat.ID, update.Message.MessageThreadID, update.Message.MessageID); err != nil {
			slog.Debug("deal chat copy message", "error", err, "chat_id", update.Message.Chat.ID, "thread_id", update.Message.MessageThreadID, "message_id", update.Message.MessageID)
		}
	}
	return nil
}

func (s *service) getUpdateType(update *telegram.Update) UpdateType {
	if update.Message != nil {
		if update.Message.Text == "/start" {
			return UpdateCommandStart
		}
	}
	return UpdateUnknown
}

func (s *service) processPendingUpdates(ctx context.Context) {
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

func (s *service) streamCleaner(ctx context.Context) {
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
