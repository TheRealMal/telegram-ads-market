package deal_chat

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"ads-mrkt/internal/market/domain/entity"
	marketerrors "ads-mrkt/internal/market/domain/errors"
)

type telegramForum interface {
	CreateForumTopic(ctx context.Context, chatID int64, name string) (messageThreadID int64, err error)
	DeleteForumTopic(ctx context.Context, chatID int64, messageThreadID int64) error
	CopyMessage(ctx context.Context, fromChatID int64, messageID int64, toChatID int64, toMessageThreadID *int64) (copiedMessageID int64, err error)
}

type marketRepository interface {
	GetDealByID(ctx context.Context, id int64) (*entity.Deal, error)
	InsertDealForumTopic(ctx context.Context, t *entity.DealForumTopic) error
	GetDealForumTopicByDealID(ctx context.Context, dealID int64) (*entity.DealForumTopic, error)
	GetDealForumTopicByChatAndThread(ctx context.Context, chatID int64, messageThreadID int64) (*entity.DealForumTopic, string, error)
	DeleteDealForumTopic(ctx context.Context, dealID int64) error
}

type service struct {
	marketRepo    marketRepository
	telegramForum telegramForum
	botUsername   string
}

// NewService creates the deal chat service. Topics are created in each user's chat (lessor/lessee user id). botUsername is used for the "Jump into chat" link. Pass empty to disable.
func NewService(marketRepo marketRepository, telegramForum telegramForum, botUsername string) *service {
	return &service{
		marketRepo:    marketRepo,
		telegramForum: telegramForum,
		botUsername:   strings.TrimPrefix(strings.TrimSpace(botUsername), "@"),
	}
}

// GetOrCreateDealForumChat creates one topic in the lessor's chat and one in the lessee's chat (chat_id = user id), then returns the link for the requesting user.
func (s *service) GetOrCreateDealForumChat(ctx context.Context, dealID int64, userID int64) (chatLink string, err error) {
	if s.telegramForum == nil {
		return "", ErrForumNotConfigured
	}
	deal, err := s.marketRepo.GetDealByID(ctx, dealID)
	if err != nil {
		return "", err
	}
	if deal == nil {
		return "", marketerrors.ErrNotFound
	}
	if userID != deal.LessorID && userID != deal.LesseeID {
		return "", marketerrors.ErrUnauthorizedSide
	}

	existing, err := s.marketRepo.GetDealForumTopicByDealID(ctx, dealID)
	if err != nil {
		return "", fmt.Errorf("get deal forum topic: %w", err)
	}
	if existing != nil {
		return s.chatLinkForUser(existing, deal, userID), nil
	}

	name := "Deal #" + strconv.FormatInt(dealID, 10)
	lessorThreadID, err := s.telegramForum.CreateForumTopic(ctx, deal.LessorID, name)
	if err != nil {
		return "", fmt.Errorf("create lessor forum topic: %w", err)
	}
	lesseeThreadID, err := s.telegramForum.CreateForumTopic(ctx, deal.LesseeID, name)
	if err != nil {
		_ = s.telegramForum.DeleteForumTopic(ctx, deal.LessorID, lessorThreadID)
		return "", fmt.Errorf("create lessee forum topic: %w", err)
	}
	t := &entity.DealForumTopic{
		DealID:                dealID,
		LessorChatID:          deal.LessorID,
		LesseeChatID:          deal.LesseeID,
		LessorMessageThreadID: lessorThreadID,
		LesseeMessageThreadID: lesseeThreadID,
	}
	if err = s.marketRepo.InsertDealForumTopic(ctx, t); err != nil {
		_ = s.telegramForum.DeleteForumTopic(ctx, deal.LessorID, lessorThreadID)
		_ = s.telegramForum.DeleteForumTopic(ctx, deal.LesseeID, lesseeThreadID)
		return "", fmt.Errorf("insert deal forum topic: %w", err)
	}
	return s.chatLinkForUser(t, deal, userID), nil
}

func (s *service) threadIDForUser(t *entity.DealForumTopic, deal *entity.Deal, userID int64) int64 {
	if deal.LessorID == userID {
		return t.LessorMessageThreadID
	}
	return t.LesseeMessageThreadID
}

// chatLinkForUser returns the deal chat link in format https://t.me/<bot_username>/<thread_id> for use with web_app_open_tg_link.
func (s *service) chatLinkForUser(t *entity.DealForumTopic, deal *entity.Deal, userID int64) string {
	threadID := s.threadIDForUser(t, deal, userID)
	if s.botUsername != "" {
		return "https://t.me/" + s.botUsername + "/" + strconv.FormatInt(threadID, 10)
	}
	return ""
}

// DeleteDealForumTopic deletes both topics via Telegram deleteForumTopic (lessor and lessee chats), then removes the row.
func (s *service) DeleteDealForumTopic(ctx context.Context, dealID int64) error {
	if s.telegramForum == nil {
		return nil
	}
	t, err := s.marketRepo.GetDealForumTopicByDealID(ctx, dealID)
	if err != nil || t == nil {
		return err
	}
	_ = s.telegramForum.DeleteForumTopic(ctx, t.LessorChatID, t.LessorMessageThreadID)
	_ = s.telegramForum.DeleteForumTopic(ctx, t.LesseeChatID, t.LesseeMessageThreadID)
	return s.marketRepo.DeleteDealForumTopic(ctx, dealID)
}

// CopyMessageToOtherTopic copies a message from one side's topic to the other. chatID and messageThreadID identify the sender's chat and topic.
func (s *service) CopyMessageToOtherTopic(ctx context.Context, chatID int64, messageThreadID int64, messageID int64) error {
	if s.telegramForum == nil {
		return nil
	}
	t, side, err := s.marketRepo.GetDealForumTopicByChatAndThread(ctx, chatID, messageThreadID)
	if err != nil || t == nil {
		return err
	}
	var toChatID int64
	var toThreadID *int64
	if side == "lessor" {
		toChatID = t.LesseeChatID
		toThreadID = &t.LesseeMessageThreadID
	} else {
		toChatID = t.LessorChatID
		toThreadID = &t.LessorMessageThreadID
	}
	_, err = s.telegramForum.CopyMessage(ctx, chatID, messageID, toChatID, toThreadID)
	return err
}

var (
	ErrForumNotConfigured = errors.New("deal chat is not configured")
)
