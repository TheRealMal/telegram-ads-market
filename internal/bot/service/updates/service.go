package updates

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	evententity "ads-mrkt/internal/event/domain/entity"
	"ads-mrkt/internal/helpers/telegram"
	marketentity "ads-mrkt/internal/market/domain/entity"
	dealmodel "ads-mrkt/internal/market/repository/deal/model"
	"ads-mrkt/internal/worker"
)

type UpdateType string

const (
	UpdateCommandStart UpdateType = "start"
	UpdateCallback     UpdateType = "callback"
	UpdateMyChatMember UpdateType = "my_chat_member"
	UpdateChatMember   UpdateType = "chat_member"
	UpdateForumMessage UpdateType = "forum_message"
	UpdateUnknown      UpdateType = "unknown"

	groupName                     = "master"
	consumerName                  = "updates"
	readTelegramUpdateEventsLimit = 100
	telegramUpdateEventsAge       = 48 * time.Hour
	streamCleanPeriod             = 24 * time.Hour
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
	SendNotification(ctx context.Context, msg telegram.NotificationMessage) error
	GetChatAdministrators(ctx context.Context, chatID int64) ([]*telegram.ChatMember, error)
	GetChannelPhotoBase64(ctx context.Context, chatID int64) (string, error)
	AnswerCallbackQuery(ctx context.Context, callbackQueryID string, text string, showAlert bool) error
}

type marketDealChatService interface {
	CopyMessageToOtherTopic(ctx context.Context, chatID int64, messageThreadID int64, messageID int64) error
	HandleForumCommand(ctx context.Context, message *telegram.UpdateMessage) error
	HandleApproveCallback(ctx context.Context, callbackQuery *telegram.CallbackQuery) error
	HandleConfirmSignCallback(ctx context.Context, callbackQuery *telegram.CallbackQuery) error
	HandleWizardStep(ctx context.Context, chatID, threadID int64, message *telegram.UpdateMessage) (bool, error)
}

type channelRepository interface {
	UpsertChannel(ctx context.Context, channel *marketentity.Channel) error
	UpdateChannelBotMemberStatus(ctx context.Context, channelID int64, status marketentity.BotMemberStatus) error
	UpdateBotAdminRights(ctx context.Context, channelID int64, rights marketentity.BotAdminRights) error
	ListChannelsWithBotAccess(ctx context.Context) ([]*marketentity.Channel, error)
	ResetChannelAccessHash(ctx context.Context, channelID int64) error
	UpdateChannelPhoto(ctx context.Context, channelID int64, photo string) error
}

type channelAdminRepository interface {
	DeleteChannelAdmin(ctx context.Context, userID, channelID int64) error
	DeleteChannelAdmins(ctx context.Context, channelID int64) error
	UpsertChannelAdmin(ctx context.Context, userID, channelID int64, role string) error
}

type listingRepository interface {
	DeactivateListingsByUserAndChannel(ctx context.Context, userID, channelID int64) (int64, error)
	DeactivateListingsByChannel(ctx context.Context, channelID int64) (int64, error)
}

type dealRepository interface {
	RejectDealsByUserAndChannel(ctx context.Context, userID, channelID int64) ([]dealmodel.RejectedDealRow, error)
	RejectDealsByChannel(ctx context.Context, channelID int64) ([]dealmodel.RejectedDealRow, error)
}

type notificationAdder interface {
	AddTelegramNotificationEvent(ctx context.Context, event *evententity.EventTelegramNotification) error
}

type channelStatsService interface {
	RequestStatsRefresh(ctx context.Context, channelID int64, userID int64) (*marketentity.Channel, error)
}

type userbotStateRepository interface {
	GetUserbotUserID(ctx context.Context) (int64, error)
}

type dealForumTopicRepository interface {
	GetDealForumTopicByDealID(ctx context.Context, dealID int64) (*marketentity.DealForumTopic, error)
}

type service struct {
	telegramClient        telegramService
	eventService          eventService
	notificationEventSvc  telegramNotificationEventService
	marketDealChatService marketDealChatService
	channelRepo           channelRepository
	channelAdminRepo      channelAdminRepository
	listingRepo           listingRepository
	dealRepo              dealRepository
	notificationAdder     notificationAdder
	userbotStateRepo      userbotStateRepository
	dealForumTopicRepo    dealForumTopicRepository
	channelStatsSvc       channelStatsService
	botUsername           string
	botUserID             int64
}

func NewService(
	telegramClient telegramService,
	eventService eventService,
	notificationEventSvc telegramNotificationEventService,
	marketDealChatService marketDealChatService,
	channelRepo channelRepository,
	channelAdminRepo channelAdminRepository,
	listingRepo listingRepository,
	dealRepo dealRepository,
	notificationAdder notificationAdder,
	userbotStateRepo userbotStateRepository,
	dealForumTopicRepo dealForumTopicRepository,
	channelStatsSvc channelStatsService,
	botUsername string,
	botUserID int64,
) *service {
	return &service{
		telegramClient:        telegramClient,
		eventService:          eventService,
		notificationEventSvc:  notificationEventSvc,
		marketDealChatService: marketDealChatService,
		channelRepo:           channelRepo,
		channelAdminRepo:      channelAdminRepo,
		listingRepo:           listingRepo,
		dealRepo:              dealRepo,
		notificationAdder:     notificationAdder,
		userbotStateRepo:      userbotStateRepo,
		dealForumTopicRepo:    dealForumTopicRepo,
		channelStatsSvc:       channelStatsSvc,
		botUsername:           botUsername,
		botUserID:             botUserID,
	}
}

func (s *service) HandleUpdate(ctx context.Context, raw []byte) error {
	update, err := telegram.ParseUpdateData(raw)
	if err != nil {
		return nil
	}

	return s.eventService.AddTelegramUpdateEvent(ctx, update, time.Now())
}

func (s *service) RunUpdateProcessorWorker(ctx context.Context) {
	logger := slog.With("component", "telegram_update_processor")
	logger.Info("started")
	defer logger.Info("stopped")

	go s.streamCleaner(ctx)
	go s.runPendingUpdateProcessorWorker(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if err := s.processUpdates(ctx); err != nil {
				logger.Error("process updates", "error", err)
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
			promUpdatesFailed.Inc()
			continue
		}
		promUpdatesProcessed.Inc()
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
		s.sendWelcomeNotification(ctx, update.Message.Chat.ID)
	case UpdateMyChatMember:
		return s.handleMyChatMember(ctx, update.MyChatMember)
	case UpdateChatMember:
		return s.handleChatMember(ctx, update.ChatMember)
	case UpdateForumMessage:
		return s.handleForumMessage(ctx, update.Message)
	case UpdateCallback:
		return s.handleCallback(ctx, update.CallbackQuery)
	case UpdateUnknown:
		slog.Debug("received unknown telegram update type, skipping", "update_id", updateEvent.Update.UpdateID)
	}

	return nil
}

func (s *service) getUpdateType(update *telegram.Update) UpdateType {
	if update.MyChatMember != nil {
		return UpdateMyChatMember
	}
	if update.ChatMember != nil {
		return UpdateChatMember
	}
	if update.CallbackQuery != nil {
		return UpdateCallback
	}
	if update.Message != nil {
		// Forum thread messages take priority (even /start in a thread)
		if update.Message.Chat != nil && update.Message.MessageThreadID != 0 {
			return UpdateForumMessage
		}
		if update.Message.Text == "/start" {
			return UpdateCommandStart
		}
	}
	return UpdateUnknown
}

func (s *service) runPendingUpdateProcessorWorker(ctx context.Context) {
	worker.RunTicker(ctx, "telegram_update_pending_processor", pendingPeriod, false, s.processPendingUpdatesBatch)
}

func (s *service) processPendingUpdatesBatch(ctx context.Context, logger *slog.Logger) {
	events, err := s.eventService.PendingTelegramUpdateEvents(ctx, groupName, consumerName, readTelegramUpdateEventsLimit, pendingMinIdle)
	if err != nil {
		logger.Error("read pending events", "error", err)
		return
	}
	if len(events) == 0 {
		return
	}
	messageIDs := make([]string, 0, len(events))
	for _, updateEvent := range events {
		if err := s.processUpdate(ctx, updateEvent); err != nil {
			logger.Error("process pending update", "error", err, "event_id", updateEvent.ID)
			continue
		}
		messageIDs = append(messageIDs, updateEvent.ID)
	}
	if len(messageIDs) > 0 {
		if err := s.eventService.AckMessages(ctx, groupName, messageIDs); err != nil {
			logger.Error("ack pending messages", "error", err)
		}
	}
}

func (s *service) streamCleaner(ctx context.Context) {
	worker.RunTicker(ctx, "telegram_update_stream_cleaner", streamCleanPeriod, false, func(ctx context.Context, logger *slog.Logger) {
		if err := s.eventService.TrimStreamByAge(ctx, telegramUpdateEventsAge); err != nil {
			logger.Error("trim stream by age", "err", err)
		}
	})
}

func (s *service) sendWelcomeNotification(ctx context.Context, chatID int64) {
	buttons, _ := json.Marshal([][]telegram.InlineKeyboardButton{
		{{Text: "Open", URL: fmt.Sprintf("https://t.me/%s?startapp=", s.botUsername)}},
	})
	if err := s.notificationAdder.AddTelegramNotificationEvent(ctx, &evententity.EventTelegramNotification{
		ChatID:  chatID,
		Message: "Start message",
		Buttons: string(buttons),
	}); err != nil {
		slog.Error("failed to add notification", "type", "welcome", "chat_id", chatID, "error", err)
	}
}
