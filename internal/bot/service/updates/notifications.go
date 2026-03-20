package updates

import (
	"context"
	"log/slog"
	"time"

	"ads-mrkt/internal/helpers/telegram"
	"ads-mrkt/internal/worker"
)

const (
	telegramNotificationGroup    = "bot"
	telegramNotificationConsumer = "notifications"
	telegramNotificationLimit    = 50
	telegramNotificationInterval = 1 * time.Second

	telegramNotificationPendingPeriod     = 15 * time.Second
	telegramNotificationPendingMinIdle    = 30 * time.Second
	telegramNotificationStreamMaxAge      = 7 * 24 * time.Hour
	telegramNotificationStreamCleanPeriod = 24 * time.Hour
)

func (s *service) RunNotificationProcessorWorker(ctx context.Context) {
	go s.notificationStreamCleaner(ctx)
	go s.runPendingNotificationWorker(ctx)
	s.runNotificationWorker(ctx)
}

func (s *service) runNotificationWorker(ctx context.Context) {
	worker.RunTicker(ctx, "telegram_notification_worker", telegramNotificationInterval, false, s.processNotificationBatch)
}

func (s *service) processNotificationBatch(ctx context.Context, logger *slog.Logger) {
	events, err := s.notificationEventSvc.ReadTelegramNotificationEvents(ctx, telegramNotificationGroup, telegramNotificationConsumer, telegramNotificationLimit)
	if err != nil || len(events) == 0 {
		return
	}
	var ids []string
	for _, ev := range events {
		if err := s.telegramClient.SendNotification(ctx, telegram.NotificationMessage{
			ChatID:    ev.ChatID,
			Message:   ev.Message,
			ThreadID:  ev.ThreadID,
			Photo:     ev.Photo,
			Video:     ev.Video,
			Animation: ev.Animation,
			Entities:  ev.Entities,
			Buttons:   ev.Buttons,
		}); err != nil {
			logger.Error("send notification", "chat_id", ev.ChatID, "error", err)
			promNotificationsFailed.Inc()
			continue
		}
		promNotificationsProcessed.Inc()
		ids = append(ids, ev.ID)
	}
	if len(ids) > 0 {
		if err := s.notificationEventSvc.AckTelegramNotificationMessages(ctx, telegramNotificationGroup, ids); err != nil {
			logger.Error("ack notification messages", "error", err)
			promNotificationAcksFailed.Inc()
		}
	}
}

func (s *service) notificationStreamCleaner(ctx context.Context) {
	worker.RunTicker(ctx, "telegram_notification_stream_cleaner", telegramNotificationStreamCleanPeriod, false, func(ctx context.Context, logger *slog.Logger) {
		if err := s.notificationEventSvc.TrimStreamByAge(ctx, telegramNotificationStreamMaxAge); err != nil {
			logger.Error("trim stream by age", "err", err)
		}
	})
}

func (s *service) runPendingNotificationWorker(ctx context.Context) {
	worker.RunTicker(ctx, "telegram_notification_pending_processor", telegramNotificationPendingPeriod, false, s.processPendingNotificationBatch)
}

func (s *service) processPendingNotificationBatch(ctx context.Context, logger *slog.Logger) {
	events, err := s.notificationEventSvc.PendingTelegramNotificationEvents(ctx, telegramNotificationGroup, telegramNotificationConsumer, telegramNotificationLimit, telegramNotificationPendingMinIdle)
	if err != nil {
		logger.Error("read pending events", "error", err)
		return
	}
	if len(events) == 0 {
		return
	}
	var ids []string
	for _, ev := range events {
		if err := s.telegramClient.SendNotification(ctx, telegram.NotificationMessage{
			ChatID:    ev.ChatID,
			Message:   ev.Message,
			ThreadID:  ev.ThreadID,
			Photo:     ev.Photo,
			Video:     ev.Video,
			Animation: ev.Animation,
			Entities:  ev.Entities,
			Buttons:   ev.Buttons,
		}); err != nil {
			logger.Error("send pending notification", "chat_id", ev.ChatID, "error", err)
			promNotificationsFailed.Inc()
			continue
		}
		promNotificationsProcessed.Inc()
		ids = append(ids, ev.ID)
	}
	if len(ids) > 0 {
		if err := s.notificationEventSvc.AckTelegramNotificationMessages(ctx, telegramNotificationGroup, ids); err != nil {
			logger.Error("ack pending notification messages", "error", err)
			promNotificationAcksFailed.Inc()
		}
	}
}
