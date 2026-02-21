package market

import (
	"context"
	"log/slog"
	"time"

	"ads-mrkt/cmd/builder"
	"ads-mrkt/internal/config"
	channelupdateevent "ads-mrkt/internal/event/application/channel_update_stats/event"
	escrowdepositevent "ads-mrkt/internal/event/application/escrow_deposit/event"
	telegramnotifyevent "ads-mrkt/internal/event/application/telegram_notification/event"
	eventredis "ads-mrkt/internal/event/repository/redis"
	"ads-mrkt/internal/helpers/telegram"
	"ads-mrkt/internal/liteclient"
	"ads-mrkt/internal/market/application/market/http"
	marketrepo "ads-mrkt/internal/market/repository/market"
	channelservice "ads-mrkt/internal/market/service/channel"
	dealservice "ads-mrkt/internal/market/service/deal"
	dealchatservice "ads-mrkt/internal/market/service/deal_chat"
	dealpostmessage "ads-mrkt/internal/market/service/deal_post_message"
	escrowservice "ads-mrkt/internal/market/service/escrow"
	listingservice "ads-mrkt/internal/market/service/listing"
	userservice "ads-mrkt/internal/market/service/user"
	"ads-mrkt/internal/postgres"
	"ads-mrkt/internal/redis"
	"ads-mrkt/internal/server"
	marketrouter "ads-mrkt/internal/server/routers/market"
	"ads-mrkt/pkg/auth"
	"ads-mrkt/pkg/health"

	"github.com/pkg/errors"
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
				return errors.Wrap(err, "create postgres client")
			}

			redisClient, err := redis.New(ctxRun, cfg.Redis)
			if err != nil {
				return errors.Wrap(err, "redis")
			}
			defer redisClient.Close()

			lc, err := liteclient.NewClient(ctxRun, cfg.Liteclient, cfg.IsTestnet, cfg.IsPublic)
			if err != nil {
				return errors.Wrap(err, "create liteclient for escrow worker")
			}

			// Telegram API client (for welcome message + middleware secret token)
			telegramClient := telegram.NewAPIClient(ctxRun, cfg.Telegram, redisClient)

			repo := marketrepo.New(pg)

			userSvc := userservice.NewUserService(cfg.Telegram.Token, repo)
			listingSvc := listingservice.NewListingService(repo, repo)
			dealChatSvc := dealchatservice.NewService(repo, telegramClient, cfg.Telegram.BotUsername)
			escrowSvc := escrowservice.NewService(repo, lc, redisClient, dealChatSvc, cfg.MarketTransactionGasTON, cfg.MarketCommissionPercent)
			eventRepo := eventredis.New(redisClient)
			escrowDepositEventSvc := escrowdepositevent.NewService(eventRepo)
			channelUpdateStatsEventSvc := channelupdateevent.NewService(eventRepo)
			telegramNotifyEventSvc := telegramnotifyevent.NewService(eventRepo)

			channelSvc := channelservice.NewChannelService(repo, channelUpdateStatsEventSvc)
			dealSvc := dealservice.NewDealService(repo, repo, escrowSvc, telegramNotifyEventSvc)
			dealPostMessageSvc := dealpostmessage.NewService(repo)
			// Preload: mark deals in waiting_escrow_deposit past deposit deadline (updated_at + 1h) as expired
			preloadCtx, preloadCancel := context.WithTimeout(ctxRun, 30*time.Second)
			if errPreload := dealSvc.ExpireTimedOutDeposits(preloadCtx, time.Now().Add(-1*time.Hour)); errPreload != nil {
				slog.Error("preload expire timed-out deposits", "error", errPreload)
			}
			preloadCancel()

			go escrowSvc.Worker(ctxRun)
			go escrowSvc.DepositStreamWorker(ctxRun, escrowDepositEventSvc)
			go escrowSvc.ReleaseRefundWorker(ctxRun)
			go dealPostMessageSvc.RunPassedWorker(ctxRun)
			go dealSvc.RunCompletedWorker(ctxRun)

			jwtManager := auth.NewJWTManager(cfg.Auth.JWTSecret, time.Duration(cfg.Auth.JWTTimeToLive)*time.Hour)
			authMiddleware := auth.NewAuthMiddleware(jwtManager)
			handler := http.NewHandler(userSvc, listingSvc, dealSvc, dealChatSvc, channelSvc, jwtManager)

			healthChecker := health.NewChecker(cfg.Health, pg)
			srv := server.NewServer(cfg.Server, healthChecker)

			shutdownSrv.Add(func(ctx context.Context) error {
				srv.Stop(ctx)
				return nil
			})

			router := marketrouter.NewRouter(cfg.Server, handler, authMiddleware)

			go srv.Start(ctxRun, router.GetRoutes())

			<-ctxRun.Done()

			return nil
		},
	}
}
