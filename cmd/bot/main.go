package bot

import (
	"context"

	"ads-mrkt/cmd/builder"
	webhookhttp "ads-mrkt/internal/bot/application/webhook/http"
	userbotstate "ads-mrkt/internal/bot/repository/userbot_state"
	botupdates "ads-mrkt/internal/bot/service/updates"
	"ads-mrkt/internal/config"
	eventtelegramnotify "ads-mrkt/internal/event/application/telegram_notification/event"
	eventtelegram "ads-mrkt/internal/event/application/telegram_update/event"
	eventredis "ads-mrkt/internal/event/repository/redis"
	"ads-mrkt/internal/helpers/telegram"
	"ads-mrkt/internal/market/repository/channel"
	"ads-mrkt/internal/market/repository/channel_admin"
	"ads-mrkt/internal/market/repository/deal"
	"ads-mrkt/internal/market/repository/deal_forum_topic"
	"ads-mrkt/internal/market/repository/listing"
	userrepo "ads-mrkt/internal/market/repository/user"
	dealchatservice "ads-mrkt/internal/market/service/deal_chat"
	dealsigner "ads-mrkt/internal/market/service/deal_signer"
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

			// Event streams: telegram updates + telegram notifications
			eventRepo := eventredis.New(redisClient)
			telegramEventSvc, err := eventtelegram.NewService(ctxRun, eventRepo)
			if err != nil {
				return errors.Wrap(err, "create telegram update event service")
			}
			telegramNotifyEventSvc, err := eventtelegramnotify.NewService(ctxRun, eventRepo)
			if err != nil {
				return errors.Wrap(err, "create telegram notification event service")
			}

			pg, err := postgres.New(ctxRun, cfg.Database)
			if err != nil {
				return errors.Wrap(err, "postgres")
			}

			channelRepo := channel.New(pg)
			channelAdminRepo := channel_admin.New(pg)
			listingRepo := listing.New(pg)
			userbotStateRepo := userbotstate.New(pg)

			dealRepo := deal.New(pg)
			dealForumTopicRepo := deal_forum_topic.New(pg)
			userRepo := userrepo.New(pg)
			// TODO: (@TheRealMal) Remove at all if userbot flow is choosen
			// dealPostMessageRepo := deal_post_message.New(pg)
			// dealActionLockRepo := deal_action_lock.New(pg)
			dealSignerSvc := dealsigner.NewService(dealRepo, userRepo, telegramNotifyEventSvc, dealForumTopicRepo)
			dealChatSvc := dealchatservice.NewService(dealRepo, dealForumTopicRepo, telegramClient, dealSignerSvc, telegramNotifyEventSvc, redisClient, cfg.Telegram.BotUsername)

			// Bot updates service
			updatesSvc := botupdates.NewService(
				telegramClient, telegramEventSvc, telegramNotifyEventSvc, dealChatSvc,
				channelRepo, channelAdminRepo, listingRepo, dealRepo, telegramNotifyEventSvc, userbotStateRepo,
				dealForumTopicRepo,
				cfg.Telegram.BotUsername,
			)
			go updatesSvc.StartBackgroundProcessingUpdates(ctxRun)
			go updatesSvc.StartBackgroundProcessingNotifications(ctxRun)
			go updatesSvc.StartAdminMonitorWorker(ctxRun)

			// TODO: (@TheRealMal) Remove at all if userbot flow is choosen
			// dealPostSvc := botdealpost.NewService(
			// 	telegramClient,
			// 	channelRepo,
			// 	listingRepo,
			// 	dealRepo,
			// 	dealPostMessageRepo,
			// 	dealActionLockRepo,
			// 	cfg.Telegram.ServiceChatID,
			// )
			// go dealPostSvc.RunDealPostSenderWorker(ctxRun)
			// go dealPostSvc.RunDealPostCheckerWorker(ctxRun)

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
