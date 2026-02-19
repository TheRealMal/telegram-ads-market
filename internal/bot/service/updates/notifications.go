package updates

import (
	"context"
	"log/slog"
	"time"
)

const (
	telegramNotificationGroup    = "bot"
	telegramNotificationConsumer = "notifications"
	telegramNotificationLimit    = 50
	telegramNotificationInterval = 1 * time.Second

	telegramNotificationPendingPeriod  = 15 * time.Second
	telegramNotificationPendingMinIdle = 30 * time.Second
	telegramNotificationStreamMaxAge   = 7 * 24 * time.Hour
)

func (s *service) StartBackgroundProcessingNotifications(ctx context.Context) {
	go s.notificationStreamCleaner(ctx)
	go s.processPendingNotifications(ctx)
	s.runNotificationWorker(ctx)
}

func (s *service) runNotificationWorker(ctx context.Context) {
	ticker := time.NewTicker(telegramNotificationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("telegram notification worker stopped")
			return
		case <-ticker.C:
			events, err := s.notificationEventSvc.ReadTelegramNotificationEvents(ctx, telegramNotificationGroup, telegramNotificationConsumer, telegramNotificationLimit)
			if err != nil || len(events) == 0 {
				continue
			}
			var ids []string
			for _, ev := range events {
				if err := s.telegramClient.SendMessageSimple(ctx, ev.ChatID, ev.Message); err != nil {
					slog.Error("send telegram notification", "chat_id", ev.ChatID, "error", err)
					continue
				}
				ids = append(ids, ev.ID)
			}
			if len(ids) > 0 {
				_ = s.notificationEventSvc.AckTelegramNotificationMessages(ctx, telegramNotificationGroup, ids)
			}
		}
	}
}

func (s *service) notificationStreamCleaner(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("telegram notification stream cleaner stopped")
			return
		case <-ticker.C:
			if err := s.notificationEventSvc.TrimStreamByAge(ctx, telegramNotificationStreamMaxAge); err != nil {
				slog.Error("failed to trim telegram notification stream by age", "err", err)
			}
			ticker.Reset(24 * time.Hour)
		}
	}
}

func (s *service) processPendingNotifications(ctx context.Context) {
	ticker := time.NewTicker(telegramNotificationPendingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("pending telegram notification events processor stopped")
			return
		case <-ticker.C:
			events, err := s.notificationEventSvc.PendingTelegramNotificationEvents(ctx, telegramNotificationGroup, telegramNotificationConsumer, telegramNotificationLimit, telegramNotificationPendingMinIdle)
			if err != nil {
				slog.Error("failed to read pending telegram notification events", "error", err)
				ticker.Reset(telegramNotificationPendingPeriod)
				continue
			}
			if len(events) == 0 {
				ticker.Reset(telegramNotificationPendingPeriod)
				continue
			}
			var ids []string
			for _, ev := range events {
				if err := s.telegramClient.SendMessageSimple(ctx, ev.ChatID, ev.Message); err != nil {
					slog.Error("send pending telegram notification", "chat_id", ev.ChatID, "error", err)
					continue
				}
				ids = append(ids, ev.ID)
			}
			if len(ids) > 0 {
				_ = s.notificationEventSvc.AckTelegramNotificationMessages(ctx, telegramNotificationGroup, ids)
			}
			ticker.Reset(telegramNotificationPendingPeriod)
		}
	}
}
