package deal_chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	evententity "ads-mrkt/internal/event/domain/entity"
	"ads-mrkt/internal/helpers/telegram"
	"ads-mrkt/internal/market/domain"
	"ads-mrkt/internal/market/domain/entity"
	marketerrors "ads-mrkt/internal/market/domain/errors"
)

type telegramForum interface {
	CreateForumTopic(ctx context.Context, chatID int64, name string) (messageThreadID int64, err error)
	DeleteForumTopic(ctx context.Context, chatID int64, messageThreadID int64) error
	CopyMessage(ctx context.Context, fromChatID int64, messageID int64, toChatID int64, toMessageThreadID *int64) (copiedMessageID int64, err error)
	AnswerCallbackQuery(ctx context.Context, callbackQueryID string, text string, showAlert bool) error
	EditMessageReplyMarkup(ctx context.Context, chatID int64, messageID int64, markup *telegram.InlineKeyboardMarkup) error
}

type notificationAdder interface {
	AddTelegramNotificationEvent(ctx context.Context, event *evententity.EventTelegramNotification) error
}

type dealRepository interface {
	GetDealByID(ctx context.Context, id int64) (*entity.Deal, error)
	UpdateDealDetailsAndClearSignatures(ctx context.Context, dealID int64, details json.RawMessage) error
}

type dealForumTopicRepository interface {
	InsertDealForumTopic(ctx context.Context, t *entity.DealForumTopic) (inserted bool, err error)
	GetDealForumTopicByDealID(ctx context.Context, dealID int64) (*entity.DealForumTopic, error)
	GetDealForumTopicByChatAndThread(ctx context.Context, chatID int64, messageThreadID int64) (*entity.DealForumTopic, string, error)
	DeleteDealForumTopic(ctx context.Context, dealID int64) error
}

type dealSigner interface {
	SignDeal(ctx context.Context, userID int64, dealID int64) error
}

type service struct {
	dealRepo          dealRepository
	forumTopicRepo    dealForumTopicRepository
	telegramForum     telegramForum
	dealSigner        dealSigner
	notificationAdder notificationAdder
	botUsername       string
}

func NewService(dealRepo dealRepository, forumTopicRepo dealForumTopicRepository, telegramForum telegramForum, dealSigner dealSigner, notificationAdder notificationAdder, botUsername string) *service {
	return &service{
		dealRepo:          dealRepo,
		forumTopicRepo:    forumTopicRepo,
		telegramForum:     telegramForum,
		dealSigner:        dealSigner,
		notificationAdder: notificationAdder,
		botUsername:       strings.TrimPrefix(strings.TrimSpace(botUsername), "@"),
	}
}

// CreateDealForumTopics creates forum topics for both parties of a deal. Idempotent:
// if topics already exist in the DB, returns nil without creating new ones.
// The flow is: check DB → create Telegram topics → INSERT ON CONFLICT DO NOTHING.
// If a concurrent caller inserted first, the orphaned Telegram topics are cleaned up.
func (s *service) CreateDealForumTopics(ctx context.Context, dealID int64, listingType entity.ListingType) error {
	if s.telegramForum == nil {
		return nil
	}
	deal, err := s.dealRepo.GetDealByID(ctx, dealID)
	if err != nil {
		return fmt.Errorf("get deal: %w", err)
	}
	if deal == nil {
		return marketerrors.ErrNotFound
	}

	existing, err := s.forumTopicRepo.GetDealForumTopicByDealID(ctx, dealID)
	if err != nil {
		return fmt.Errorf("get deal forum topic: %w", err)
	}
	if existing != nil {
		return nil
	}

	name := "Deal #" + strconv.FormatInt(dealID, 10)
	lessorThreadID, err := s.telegramForum.CreateForumTopic(ctx, deal.LessorID, name)
	if err != nil {
		return fmt.Errorf("create lessor forum topic: %w", err)
	}
	lesseeThreadID, err := s.telegramForum.CreateForumTopic(ctx, deal.LesseeID, name)
	if err != nil {
		_ = s.telegramForum.DeleteForumTopic(ctx, deal.LessorID, lessorThreadID)
		return fmt.Errorf("create lessee forum topic: %w", err)
	}

	t := &entity.DealForumTopic{
		DealID:                dealID,
		LessorChatID:          deal.LessorID,
		LesseeChatID:          deal.LesseeID,
		LessorMessageThreadID: lessorThreadID,
		LesseeMessageThreadID: lesseeThreadID,
	}
	inserted, err := s.forumTopicRepo.InsertDealForumTopic(ctx, t)
	if err != nil {
		_ = s.telegramForum.DeleteForumTopic(ctx, deal.LessorID, lessorThreadID)
		_ = s.telegramForum.DeleteForumTopic(ctx, deal.LesseeID, lesseeThreadID)
		return fmt.Errorf("insert deal forum topic: %w", err)
	}
	if !inserted {
		// Another process already inserted topics for this deal — clean up our orphans.
		_ = s.telegramForum.DeleteForumTopic(ctx, deal.LessorID, lessorThreadID)
		_ = s.telegramForum.DeleteForumTopic(ctx, deal.LesseeID, lesseeThreadID)
		return nil
	}

	lessorMessage, lesseeMessage := s.buildInitialMessages(deal, listingType)
	s.sendInitialMessage(ctx, deal.LessorID, lessorThreadID, lessorMessage)
	s.sendInitialMessage(ctx, deal.LesseeID, lesseeThreadID, lesseeMessage)

	return nil
}

func (s *service) buildInitialMessages(deal *entity.Deal, listingType entity.ListingType) (lessorMsg, lesseeMsg string) {
	priceTON := strconv.FormatFloat(domain.NanotonToTON(deal.Price), 'f', -1, 64)
	postDate := deal.CreatedAt.Format("02.01.2006 15:04")
	channelID := ""
	if deal.ChannelID != nil {
		channelID = strconv.FormatInt(*deal.ChannelID, 10)
	}

	buildMsg := func(prefix string) string {
		var b strings.Builder
		b.WriteString(prefix)
		b.WriteString(" ")
		b.WriteString(deal.Type)
		b.WriteString("\nChannel: ")
		b.WriteString(channelID)
		b.WriteString("\n\n💰 ")
		b.WriteString(priceTON)
		b.WriteString(" TON")
		b.WriteString("\n⏰ ")
		b.WriteString(postDate)
		if deal.Message != "" {
			b.WriteString("\n\n")
			b.WriteString(deal.Message)
		}
		return b.String()
	}

	switch listingType {
	case entity.ListingTypeLessee:
		lessorMsg = buildMsg("Applied for")
		lesseeMsg = buildMsg("Offer for")
	case entity.ListingTypeLessor:
		lessorMsg = buildMsg("Request for")
		lesseeMsg = buildMsg("Requested for")
	}

	return lessorMsg, lesseeMsg
}

func (s *service) sendInitialMessage(ctx context.Context, chatID int64, threadID int64, text string) {
	if s.notificationAdder == nil {
		return
	}
	if err := s.notificationAdder.AddTelegramNotificationEvent(ctx, &evententity.EventTelegramNotification{
		ChatID:   chatID,
		ThreadID: threadID,
		Message:  text,
	}); err != nil {
		slog.Error("failed to add notification", "type", "initial_deal_message", "chat_id", chatID, "thread_id", threadID, "error", err)
	}
}

// GetDealChatLink returns the chat link for an existing deal forum topic.
// If topics do not yet exist (e.g., CreateDealForumTopics failed at deal creation), it creates them as a fallback.
func (s *service) GetDealChatLink(ctx context.Context, dealID int64, userID int64) (string, error) {
	if s.telegramForum == nil {
		return "", ErrForumNotConfigured
	}
	deal, err := s.dealRepo.GetDealByID(ctx, dealID)
	if err != nil {
		return "", err
	}
	if deal == nil {
		return "", marketerrors.ErrNotFound
	}
	if userID != deal.LessorID && userID != deal.LesseeID {
		return "", marketerrors.ErrUnauthorizedSide
	}

	forumTopic, err := s.forumTopicRepo.GetDealForumTopicByDealID(ctx, dealID)
	if err != nil {
		return "", fmt.Errorf("get deal forum topic: %w", err)
	}

	return s.chatLinkForUser(forumTopic, deal, userID), nil
}

func (s *service) threadIDForUser(t *entity.DealForumTopic, deal *entity.Deal, userID int64) int64 {
	if deal.LessorID == userID {
		return t.LessorMessageThreadID
	}
	return t.LesseeMessageThreadID
}

func (s *service) chatLinkForUser(t *entity.DealForumTopic, deal *entity.Deal, userID int64) string {
	threadID := s.threadIDForUser(t, deal, userID)
	if s.botUsername != "" {
		return "https://t.me/" + s.botUsername + "/" + strconv.FormatInt(threadID, 10)
	}
	return ""
}

func (s *service) DeleteDealForumTopic(ctx context.Context, dealID int64) error {
	if s.telegramForum == nil {
		return nil
	}
	t, err := s.forumTopicRepo.GetDealForumTopicByDealID(ctx, dealID)
	if err != nil || t == nil {
		return err
	}
	_ = s.telegramForum.DeleteForumTopic(ctx, t.LessorChatID, t.LessorMessageThreadID)
	_ = s.telegramForum.DeleteForumTopic(ctx, t.LesseeChatID, t.LesseeMessageThreadID)
	return s.forumTopicRepo.DeleteDealForumTopic(ctx, dealID)
}

func (s *service) CopyMessageToOtherTopic(ctx context.Context, chatID int64, messageThreadID int64, messageID int64) error {
	if s.telegramForum == nil {
		return nil
	}
	t, side, err := s.forumTopicRepo.GetDealForumTopicByChatAndThread(ctx, chatID, messageThreadID)
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
