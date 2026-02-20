package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	evententity "ads-mrkt/internal/event/domain/entity"
	marketentity "ads-mrkt/internal/market/domain/entity"
	"ads-mrkt/internal/userbot/config"

	"github.com/gotd/td/examples"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/updates"
	updhook "github.com/gotd/td/telegram/updates/hook"
	"github.com/gotd/td/tg"
)

type marketRepository interface {
	UpsertChannel(ctx context.Context, channel *marketentity.Channel) error
	UpsertChannelStats(ctx context.Context, channelID int64, stats json.RawMessage) error
	GetChannelByID(ctx context.Context, id int64) (*marketentity.Channel, error)
	DeleteChannelAdmins(ctx context.Context, channelID int64) error
	UpsertChannelAdmin(ctx context.Context, userID, channelID int64, role string) error
	// Deal post message workers
	ListDealsEscrowDepositConfirmedWithoutPostMessage(ctx context.Context) ([]*marketentity.Deal, error)
	GetListingByID(ctx context.Context, id int64) (*marketentity.Listing, error)
	CreateDealPostMessageAndSetDealInProgress(ctx context.Context, m *marketentity.DealPostMessage) error
	UpdateDealPostMessageStatus(ctx context.Context, id int64, status marketentity.DealPostMessageStatus) error
	UpdateDealPostMessageStatusAndNextCheck(ctx context.Context, id int64, status marketentity.DealPostMessageStatus, nextCheck time.Time) error
	ListDealPostMessageExistsWithNextCheckBefore(ctx context.Context, before time.Time) ([]*marketentity.DealPostMessage, error)
	TakeDealActionLock(ctx context.Context, dealID int64, actionType marketentity.DealActionType) (string, error)
	ReleaseDealActionLock(ctx context.Context, lockID string, status marketentity.DealActionLockStatus) error
	GetLastDealActionLock(ctx context.Context, dealID int64, actionType marketentity.DealActionType) (*marketentity.DealActionLock, error)
}

type channelUpdateStatsEventService interface {
	ReadChannelUpdateStatsEvents(ctx context.Context, group, consumer string, limit int64) ([]*evententity.EventChannelUpdateStats, error)
	PendingChannelUpdateStatsEvents(ctx context.Context, group, consumer string, limit int64, minIdle time.Duration) ([]*evententity.EventChannelUpdateStats, error)
	AckChannelUpdateStatsMessages(ctx context.Context, group string, messageIDs []string) error
	TrimStreamByAge(ctx context.Context, age time.Duration) error
}

type service struct {
	stateStorage               updates.StateStorage
	marketRepository           marketRepository
	channelUpdateStatsEventSvc channelUpdateStatsEventService
	telegramClient             *telegram.Client
	authFlow                   auth.Flow
	updatesManager             *updates.Manager
	userID                     int64
}

func New(cfg config.Config, stateStorage updates.StateStorage, marketRepository marketRepository, channelUpdateStatsEventSvc channelUpdateStatsEventService) *service {
	s := &service{
		stateStorage:               stateStorage,
		marketRepository:           marketRepository,
		channelUpdateStatsEventSvc: channelUpdateStatsEventSvc,
	}

	dispatcher := tg.NewUpdateDispatcher()
	dispatcher.OnChannel(s.handleChannelUpdate)

	s.updatesManager = updates.New(updates.Config{
		Handler: dispatcher,
		Storage: stateStorage,
	})

	s.telegramClient = telegram.NewClient(
		cfg.ApiID,
		cfg.ApiHash,
		telegram.Options{
			SessionStorage: &telegram.FileSessionStorage{
				Path: cfg.SessionFile,
			},
			UpdateHandler: s.updatesManager,
			Middlewares: []telegram.Middleware{
				updhook.UpdateHook(s.updatesManager.Handle),
			},
			Device: telegram.DeviceConfig{
				DeviceModel:    "ADS Market",
				SystemVersion:  "n/a",
				AppVersion:     "n/a",
				SystemLangCode: "en",
				LangPack:       "en",
				LangCode:       "en",
			},
		},
	)
	s.authFlow = auth.NewFlow(examples.Terminal{PhoneNumber: cfg.Phone}, auth.SendCodeOptions{})

	return s
}

func (s *service) getCurrentState(ctx context.Context) error {
	api := s.telegramClient.API()

	state, err := api.UpdatesGetState(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current state from Telegram: %w", err)
	}

	if err := s.stateStorage.SetState(
		ctx,
		s.userID,
		updates.State{
			Pts:  state.Pts,
			Qts:  state.Qts,
			Date: state.Date,
			Seq:  state.Seq,
		},
	); err != nil {
		return fmt.Errorf("failed to store state in database: %w", err)
	}

	slog.Info("current state retrieved and stored",
		"pts", state.Pts,
		"qts", state.Qts,
		"date", state.Date,
		"seq", state.Seq,
	)

	return nil
}

func (s *service) Start(ctx context.Context) error {
	return s.telegramClient.Run(ctx, s.run)
}

func (s *service) run(ctx context.Context) error {
	slog.Debug("performing auth if necessary")
	if err := s.telegramClient.Auth().IfNecessary(ctx, s.authFlow); err != nil {
		return fmt.Errorf("auth: %w", err)
	}

	slog.Debug("getting user info")
	user, err := s.telegramClient.Self(ctx)
	if err != nil {
		return fmt.Errorf("call self: %w", err)
	}

	s.userID = user.ID

	go s.RunDealPostSenderWorker(ctx)
	go s.RunDealPostCheckerWorker(ctx)
	go s.RunChannelUpdateStatsWorker(ctx)

	slog.Debug("getting current state")
	if err := s.getCurrentState(ctx); err != nil {
		slog.Warn("Failed to get current state, continuing anyway", "error", err)
	}

	api := s.telegramClient.API()
	authOpts := updates.AuthOptions{
		IsBot:  false,
		Forget: false,
		OnStart: func(ctx context.Context) {
			slog.Info("updates manager started successfully")
		},
	}

	slog.Info("starting updates polling", "user_id", user.ID)
	return s.updatesManager.Run(ctx, api, user.ID, authOpts)
}
