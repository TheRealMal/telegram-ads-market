package updates

import (
	"context"
	"log/slog"

	"ads-mrkt/internal/helpers/telegram"
	dealchat "ads-mrkt/internal/market/service/deal_chat"
)

func (s *service) handleForumMessage(ctx context.Context, message *telegram.UpdateMessage) error {
	// Intercept forum commands before mirroring
	if dealchat.IsForumCommand(message) {
		if err := s.marketDealChatService.HandleForumCommand(ctx, message); err != nil {
			slog.Error("handle forum command", "error", err,
				"chat_id", message.Chat.ID,
				"thread_id", message.MessageThreadID)
		}
		return nil // Do NOT mirror commands to the other side
	}

	// Check if the user is in an active wizard flow
	if message.From != nil {
		handled, err := s.marketDealChatService.HandleWizardStep(ctx, message.Chat.ID, message.MessageThreadID, message)
		if err != nil {
			slog.Error("handle wizard step", "error", err,
				"chat_id", message.Chat.ID,
				"thread_id", message.MessageThreadID)
		}
		if handled {
			return nil
		}
	}

	// Normal flow: mirror message to other side's topic
	if err := s.marketDealChatService.CopyMessageToOtherTopic(ctx, message.Chat.ID, message.MessageThreadID, message.MessageID); err != nil {
		slog.Error("deal chat copy message", "error", err,
			"chat_id", message.Chat.ID,
			"thread_id", message.MessageThreadID,
			"message_id", message.MessageID)
	}
	return nil
}
