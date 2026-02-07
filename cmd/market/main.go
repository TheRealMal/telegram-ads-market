package market

import (
	"context"
	"time"

	"ads-mrkt/cmd/builder"
	"ads-mrkt/internal/config"
	"ads-mrkt/internal/helpers/telegram"
	"ads-mrkt/internal/market/application/market/http"
	marketrepo "ads-mrkt/internal/market/repository/market"
	channelservice "ads-mrkt/internal/market/service/channel"
	dealservice "ads-mrkt/internal/market/service/deal"
	dealchatservice "ads-mrkt/internal/market/service/deal_chat"
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

			// Telegram API client (for welcome message + middleware secret token)
			telegramClient := telegram.NewAPIClient(ctxRun, cfg.Telegram, redisClient)

			repo := marketrepo.New(pg)
			userSvc := userservice.NewUserService(cfg.Telegram.Token, repo)
			listingSvc := listingservice.NewListingService(repo, repo)
			dealSvc := dealservice.NewDealService(repo)
			dealChatSvc := dealchatservice.NewService(repo, telegramClient) // pass TelegramSender to enable send-chat-message
			channelSvc := channelservice.NewChannelService(repo)

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
