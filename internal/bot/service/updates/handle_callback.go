package updates

import (
	"context"
	"log/slog"
	"strings"

	"ads-mrkt/internal/helpers/telegram"
)

func (s *service) handleCallback(ctx context.Context, callbackQuery *telegram.CallbackQuery) error {
	if callbackQuery == nil {
		return nil
	}

	// Route approve_edit callbacks to deal chat service
	if strings.HasPrefix(callbackQuery.Data, "approve_edit:") {
		if err := s.marketDealChatService.HandleApproveCallback(ctx, callbackQuery); err != nil {
			slog.Error("handle approve callback", "error", err, "callback_id", callbackQuery.ID)
		}
		return nil
	}

	// Route confirm_sign callbacks to deal chat service
	if strings.HasPrefix(callbackQuery.Data, "confirm_sign:") {
		if err := s.marketDealChatService.HandleConfirmSignCallback(ctx, callbackQuery); err != nil {
			slog.Error("handle confirm sign callback", "error", err, "callback_id", callbackQuery.ID)
		}
		return nil
	}

	// Unknown callback -- answer to dismiss Telegram's loading indicator
	_ = s.telegramClient.AnswerCallbackQuery(ctx, callbackQuery.ID, "", false)
	return nil
}
