package deal_chat

import (
	"context"
	"errors"
	"fmt"

	"ads-mrkt/internal/market/domain/entity"
	marketerrors "ads-mrkt/internal/market/domain/errors"
)

// SentMessage is the result of sending a Telegram message (for persistence).
type SentMessage struct {
	ChatID    int64
	MessageID int64
}

// TelegramSender sends a text message to a chat and returns the sent message identifiers.
type TelegramSender interface {
	SendMessage(ctx context.Context, chatID int64, text string) (*SentMessage, error)
}

type DealRepository interface {
	GetDealByID(ctx context.Context, id int64) (*entity.Deal, error)
}

type DealChatRepository interface {
	InsertDealChat(ctx context.Context, dc *entity.DealChat) error
	GetDealChatByReply(ctx context.Context, chatID, messageID int64) (*entity.DealChat, error)
	UpdateDealChatRepliedMessage(ctx context.Context, dealID, replyToChatID, replyToMessageID int64, repliedMessage string) error
	ListDealChatsByDealIDWhereReplied(ctx context.Context, dealID int64) ([]*entity.DealChat, error)
	HasActiveDealChatForUser(ctx context.Context, dealID, userID int64) (bool, error)
}

type Service struct {
	dealRepo       DealRepository
	dealChatRepo   DealChatRepository
	telegramSender TelegramSender
}

const dealChatInviteText = "Reply to this message to chat with the other side."

func NewService(dealRepo DealRepository, dealChatRepo DealChatRepository, telegramSender TelegramSender) *Service {
	return &Service{
		dealRepo:       dealRepo,
		dealChatRepo:   dealChatRepo,
		telegramSender: telegramSender,
	}
}

// SendDealChatMessage sends the "reply to chat" message to the requesting user and saves it to deal_chat.
// userID must be lessor or lessee of the deal.
func (s *Service) SendDealChatMessage(ctx context.Context, dealID, userID int64) (*entity.DealChat, error) {
	if s.telegramSender == nil {
		return nil, fmt.Errorf("telegram sender not configured")
	}
	deal, err := s.dealRepo.GetDealByID(ctx, dealID)
	if err != nil {
		return nil, err
	}
	if deal == nil {
		return nil, marketerrors.ErrNotFound
	}
	if userID != deal.LessorID && userID != deal.LesseeID {
		return nil, marketerrors.ErrUnauthorizedSide
	}
	hasActive, err := s.dealChatRepo.HasActiveDealChatForUser(ctx, dealID, userID)
	if err != nil {
		return nil, fmt.Errorf("check active deal chat: %w", err)
	}
	if hasActive {
		return nil, ErrActiveDealChatExists
	}
	sent, err := s.telegramSender.SendMessage(ctx, userID, dealChatInviteText)
	if err != nil {
		return nil, fmt.Errorf("send telegram message: %w", err)
	}
	dc := &entity.DealChat{
		DealID:           dealID,
		ReplyToChatID:    sent.ChatID,
		ReplyToMessageID: sent.MessageID,
		RepliedMessage:   nil,
	}
	if err = s.dealChatRepo.InsertDealChat(ctx, dc); err != nil {
		return nil, fmt.Errorf("insert deal chat: %w", err)
	}
	return dc, nil
}

// ListDealMessages returns all deal_chat rows for the deal in chronological order.
// userID must be lessor or lessee of the deal.
func (s *Service) ListDealMessages(ctx context.Context, dealID, userID int64) ([]*entity.DealChat, error) {
	deal, err := s.dealRepo.GetDealByID(ctx, dealID)
	if err != nil {
		return nil, err
	}
	if deal == nil {
		return nil, marketerrors.ErrNotFound
	}
	if userID != deal.LessorID && userID != deal.LesseeID {
		return nil, marketerrors.ErrUnauthorizedSide
	}
	return s.dealChatRepo.ListDealChatsByDealIDWhereReplied(ctx, dealID)
}

// SetRepliedMessageIfMatch finds a deal_chat by the message being replied to and sets replied_message.
// Returns nil if no matching row (not an error).
func (s *Service) SetRepliedMessageIfMatch(ctx context.Context, replyToChatID, replyToMessageID int64, repliedText string) error {
	dc, err := s.dealChatRepo.GetDealChatByReply(ctx, replyToChatID, replyToMessageID)
	if err != nil {
		return err
	}
	if dc == nil {
		return nil
	}
	return s.dealChatRepo.UpdateDealChatRepliedMessage(ctx, dc.DealID, dc.ReplyToChatID, dc.ReplyToMessageID, repliedText)
}

// ErrTelegramSenderNil is returned when the telegram sender is not configured.
var ErrTelegramSenderNil = errors.New("telegram sender not configured")

// ErrActiveDealChatExists is returned when the user already has an active (unreplied) chat invite for this deal.
var ErrActiveDealChatExists = errors.New("deal chat: user already has an active chat invite for this deal")
