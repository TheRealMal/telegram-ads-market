package bot

import (
	"context"

	"ads-mrkt/cmd/builder"
	webhookhttp "ads-mrkt/internal/bot/application/webhook/http"
	botupdates "ads-mrkt/internal/bot/service/updates"
	"ads-mrkt/internal/config"
	eventtelegram "ads-mrkt/internal/event/application/telegram_update/event"
	eventredis "ads-mrkt/internal/event/repository/redis"
	"ads-mrkt/internal/helpers/telegram"
	marketrepo "ads-mrkt/internal/market/repository/market"
	dealchatservice "ads-mrkt/internal/market/service/deal_chat"
	"ads-mrkt/internal/postgres"
	"ads-mrkt/internal/redis"
	"ads-mrkt/internal/server"
	telegramrouter "ads-mrkt/internal/server/routers/telegram"
	"ads-mrkt/pkg/health"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func BotCmd(ctx context.Context, conf *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bot",
		Short: "Telegram bot commands",
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
		Short: "run Telegram bot webhook HTTP server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctxRun, cancel := context.WithCancel(ctx)
			shutdownSrv := builder.NewShutdown()
			go func() {
				shutdownSrv.WaitShutdown(ctxRun)
				cancel()
			}()

			// Redis for event stream and rate limiting
			redisClient, err := redis.New(ctxRun, cfg.Redis)
			if err != nil {
				return errors.Wrap(err, "redis")
			}
			defer redisClient.Close()

			// Telegram API client (for welcome message + middleware secret token)
			telegramClient := telegram.NewAPIClient(ctxRun, cfg.Telegram, redisClient)

			// Event stream: telegram updates
			eventRepo := eventredis.New(redisClient)
			telegramEventSvc := eventtelegram.NewService(eventRepo)

			pg, err := postgres.New(ctxRun, cfg.Database)
			if err != nil {
				return errors.Wrap(err, "postgres")
			}

			marketRepo := marketrepo.New(pg)
			dealChatSvc := dealchatservice.NewService(marketRepo, nil)

			// Bot updates service
			updatesSvc := botupdates.NewService(telegramClient, telegramEventSvc, dealChatSvc)
			go updatesSvc.StartBackgroundProcessingUpdates(ctxRun)

			// Webhook HTTP handler and router
			webhookHandler := webhookhttp.NewHandler(updatesSvc)
			healthChecker := health.NewChecker(cfg.Health)
			srv := server.NewServer(cfg.Server, healthChecker)

			shutdownSrv.Add(func(ctx context.Context) error {
				srv.Stop(ctx)
				return nil
			})

			router := telegramrouter.NewRouter(webhookHandler, telegramClient)

			go srv.Start(ctxRun, router.GetRoutes())

			<-ctxRun.Done()

			return nil
		},
	}
}
