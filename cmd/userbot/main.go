package userbot

import (
	"context"
	"fmt"

	"ads-mrkt/internal/config"
	channelupdateevent "ads-mrkt/internal/event/application/channel_update_stats/event"
	telegramnotifyevent "ads-mrkt/internal/event/application/telegram_notification/event"
	eventredis "ads-mrkt/internal/event/repository/redis"
	"ads-mrkt/internal/helpers/telegram"
	"ads-mrkt/internal/market/repository/channel"
	"ads-mrkt/internal/market/repository/deal"
	"ads-mrkt/internal/market/repository/deal_action_lock"
	"ads-mrkt/internal/market/repository/deal_forum_topic"
	"ads-mrkt/internal/market/repository/deal_post_message"
	"ads-mrkt/internal/market/repository/listing"
	"ads-mrkt/internal/postgres"
	"ads-mrkt/internal/redis"
	userbotrepo "ads-mrkt/internal/userbot/repository/state"
	userbotservice "ads-mrkt/internal/userbot/service/userbot"

	"github.com/spf13/cobra"
)

func UserbotCmd(ctx context.Context, conf *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "userbot",
		Short: "Userbot (Telegram client) commands",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Usage()
		},
	}

	cmd.AddCommand(runCmd(ctx, conf))

	return cmd
}

func runCmd(ctx context.Context, cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "run userbot (polling for channel updates)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pg, err := postgres.New(ctx, cfg.Database)
			if err != nil {
				return fmt.Errorf("postgres: %w", err)
			}

			redisClient, err := redis.New(ctx, cfg.Redis)
			if err != nil {
				return fmt.Errorf("redis: %w", err)
			}
			defer redisClient.Close()

			stateStorage := userbotrepo.New(pg)
			channelRepo := channel.New(pg)
			listingRepo := listing.New(pg)
			dealRepo := deal.New(pg)
			dealPostMessageRepo := deal_post_message.New(pg)
			dealActionLockRepo := deal_action_lock.New(pg)
			eventRepo := eventredis.New(redisClient)
			channelUpdateStatsEventSvc, err := channelupdateevent.NewService(ctx, eventRepo)
			if err != nil {
				return fmt.Errorf("create channel update stats event service: %w", err)
			}
			telegramNotifyEventSvc, err := telegramnotifyevent.NewService(ctx, eventRepo)
			if err != nil {
				return fmt.Errorf("create telegram notification event service: %w", err)
			}
			dealForumTopicRepo := deal_forum_topic.New(pg)
			telegramBotClient := telegram.NewAPIClient(ctx, cfg.Telegram, redisClient)
			b := userbotservice.New(cfg.UserBot, stateStorage, channelRepo, listingRepo, dealRepo, dealPostMessageRepo, dealActionLockRepo, channelUpdateStatsEventSvc, telegramBotClient, telegramNotifyEventSvc, dealForumTopicRepo)

			if err := b.Start(ctx); err != nil {
				return fmt.Errorf("userbot start: %w", err)
			}
			return nil
		},
	}
}
