package market

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"ads-mrkt/cmd/builder"
	analyticshttp "ads-mrkt/internal/analytics/application/http"
	analyticsrepo "ads-mrkt/internal/analytics/repository"
	analyticsservice "ads-mrkt/internal/analytics/service"
	"ads-mrkt/internal/config"
	channelupdateevent "ads-mrkt/internal/event/application/channel_update_stats/event"
	escrowdepositevent "ads-mrkt/internal/event/application/escrow_deposit/event"
	telegramnotifyevent "ads-mrkt/internal/event/application/telegram_notification/event"
	eventredis "ads-mrkt/internal/event/repository/redis"
	"ads-mrkt/internal/helpers/telegram"
	"ads-mrkt/internal/liteclient"
	"ads-mrkt/internal/market/application/market/http"
	"ads-mrkt/internal/market/repository/channel"
	"ads-mrkt/internal/market/repository/channel_admin"
	"ads-mrkt/internal/market/repository/deal"
	"ads-mrkt/internal/market/repository/deal_action_lock"
	"ads-mrkt/internal/market/repository/deal_forum_topic"
	"ads-mrkt/internal/market/repository/deal_post_message"
	"ads-mrkt/internal/market/repository/listing"
	"ads-mrkt/internal/market/repository/user"
	channelservice "ads-mrkt/internal/market/service/channel"
	dealservice "ads-mrkt/internal/market/service/deal"
	dealchatservice "ads-mrkt/internal/market/service/deal_chat"
	dealpostmessage "ads-mrkt/internal/market/service/deal_post_message"
	dealsigner "ads-mrkt/internal/market/service/deal_signer"
	escrowservice "ads-mrkt/internal/market/service/escrow"
	listingservice "ads-mrkt/internal/market/service/listing"
	userservice "ads-mrkt/internal/market/service/user"
	"ads-mrkt/internal/postgres"
	"ads-mrkt/internal/redis"
	"ads-mrkt/internal/server"
	marketrouter "ads-mrkt/internal/server/routers/market"
	"ads-mrkt/internal/vault"
	"ads-mrkt/pkg/auth"
	"ads-mrkt/pkg/health"

	"github.com/spf13/cobra"
)

func ApiCmd(ctx context.Context, conf *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "market",
		Short: "market API commands",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Usage()
		},
	}

	cmd.AddCommand(httpCmd(ctx, conf))

	return cmd
}

func httpCmd(ctx context.Context, cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "http",
		Short: "run market HTTP API server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctxRun, cancel := context.WithCancel(ctx)
			shutdownSrv := builder.NewShutdown()
			go func() {
				shutdownSrv.WaitShutdown(ctxRun)
				cancel()
			}()

			pg, err := postgres.New(ctx, cfg.Database)
			if err != nil {
				return fmt.Errorf("create postgres client: %w", err)
			}

			redisClient, err := redis.New(ctxRun, cfg.Redis)
			if err != nil {
				return fmt.Errorf("redis: %w", err)
			}
			defer redisClient.Close()

			lc, err := liteclient.NewClient(ctxRun, cfg.Liteclient, cfg.IsTestnet, cfg.IsPublic)
			if err != nil {
				return fmt.Errorf("create liteclient for escrow worker: %w", err)
			}

			// Telegram API client (for welcome message + middleware secret token)
			telegramClient := telegram.NewAPIClient(ctxRun, cfg.Telegram, redisClient)

			listingRepo := listing.New(pg)
			channelRepo := channel.New(pg)
			userRepo := user.New(pg)
			channelAdminRepo := channel_admin.New(pg)
			dealRepo := deal.New(pg)
			dealPostMessageRepo := deal_post_message.New(pg)
			dealActionLockRepo := deal_action_lock.New(pg)
			dealForumTopicRepo := deal_forum_topic.New(pg)

			analyticsRepo := analyticsrepo.New(pg)
			analyticsSvc := analyticsservice.New(analyticsRepo, cfg.MarketTransactionGasTON, cfg.MarketCommissionPercent)

			eventRepo := eventredis.New(redisClient)
			escrowDepositEventSvc, err := escrowdepositevent.NewService(ctxRun, eventRepo)
			if err != nil {
				return fmt.Errorf("create escrow deposit event service: %w", err)
			}
			channelUpdateStatsEventSvc, err := channelupdateevent.NewService(ctxRun, eventRepo)
			if err != nil {
				return fmt.Errorf("create channel update stats event service: %w", err)
			}
			telegramNotifyEventSvc, err := telegramnotifyevent.NewService(ctxRun, eventRepo)
			if err != nil {
				return fmt.Errorf("create telegram notification event service: %w", err)
			}

			userSvc := userservice.NewUserService(cfg.Environment, cfg.Telegram.Token, userRepo)
			listingSvc := listingservice.NewListingService(listingRepo, channelAdminRepo, userRepo)

			// Standalone deal signer for deal chat (breaks circular dep: dealChatSvc -> dealSigner -> escrowSvc -> dealChatSvc).
			dealSignerSvc := dealsigner.NewService(dealRepo, userRepo, telegramNotifyEventSvc, dealForumTopicRepo)
			dealChatSvc := dealchatservice.NewService(dealRepo, dealForumTopicRepo, telegramClient, dealSignerSvc, telegramNotifyEventSvc, redisClient, cfg.Telegram.BotUsername)
			vaultClient, err := vault.NewClient(cfg.Vault)
			if err != nil {
				return fmt.Errorf("create vault client: %w", err)
			}
			escrowSvc := escrowservice.NewService(dealRepo, vaultClient, dealActionLockRepo, lc, redisClient, dealChatSvc, cfg.MarketTransactionGasTON, cfg.MarketCommissionPercent)

			channelSvc := channelservice.NewChannelService(channelRepo, channelAdminRepo, listingRepo, channelUpdateStatsEventSvc)
			dealSvc := dealservice.NewDealService(dealRepo, userRepo, listingRepo, escrowSvc, telegramNotifyEventSvc, dealChatSvc, dealForumTopicRepo)
			dealPostMessageSvc := dealpostmessage.NewService(dealPostMessageRepo, dealChatSvc)
			// Preload: mark deals in waiting_escrow_deposit past deposit deadline (updated_at + 1h) as expired
			preloadCtx, preloadCancel := context.WithTimeout(ctxRun, 30*time.Second)
			if errPreload := dealSvc.ExpireTimedOutDeposits(preloadCtx, time.Now().Add(-1*time.Hour)); errPreload != nil {
				slog.Error("preload expire timed-out deposits", "error", errPreload)
			}
			preloadCancel()

			go escrowSvc.RunEscrowCreatorWorker(ctxRun)
			go escrowSvc.RunDepositStreamWorker(ctxRun, escrowDepositEventSvc)
			go escrowSvc.RunReleaseRefundWorker(ctxRun)
			go dealPostMessageSvc.RunDealPostMessageWorker(ctxRun)
			go dealSvc.RunCompletedWorker(ctxRun)
			go analyticsSvc.RunSnapshotWorker(ctxRun)

			jwtManager := auth.NewJWTManager(cfg.Auth.JWTSecret, time.Duration(cfg.Auth.JWTTimeToLive)*time.Hour)
			authMiddleware := auth.NewAuthMiddleware(jwtManager)
			handler := http.NewHandler(userSvc, listingSvc, dealSvc, dealChatSvc, channelSvc, jwtManager)

			healthChecker := health.NewChecker(cfg.Health, pg)
			srv := server.NewServer(cfg.Server, healthChecker)

			shutdownSrv.Add(func(ctx context.Context) error {
				srv.Stop(ctx)
				return nil
			})

			analyticsHandler := analyticshttp.NewHandler(analyticsSvc)
			router := marketrouter.NewRouter(cfg.Server, handler, authMiddleware, analyticsHandler)

			go srv.Start(ctxRun, router.GetRoutes())

			<-ctxRun.Done()

			return nil
		},
	}
}
