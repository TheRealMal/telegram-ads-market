package userbot

import (
	"context"

	"ads-mrkt/internal/config"
	channelupdateevent "ads-mrkt/internal/event/application/channel_update_stats/event"
	eventredis "ads-mrkt/internal/event/repository/redis"
	marketrepo "ads-mrkt/internal/market/repository/market"
	"ads-mrkt/internal/postgres"
	"ads-mrkt/internal/redis"
	userbotrepo "ads-mrkt/internal/userbot/repository/state"
	userbotservice "ads-mrkt/internal/userbot/service/userbot"

	"github.com/pkg/errors"
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
				return errors.Wrap(err, "postgres")
			}

			redisClient, err := redis.New(ctx, cfg.Redis)
			if err != nil {
				return errors.Wrap(err, "redis")
			}
			defer redisClient.Close()

			stateStorage := userbotrepo.New(pg)
			marketRepo := marketrepo.New(pg)
			eventRepo := eventredis.New(redisClient)
			channelUpdateStatsEventSvc := channelupdateevent.NewService(eventRepo)
			b := userbotservice.New(cfg.UserBot, stateStorage, marketRepo, channelUpdateStatsEventSvc)

			if err := b.Start(ctx); err != nil {
				return errors.Wrap(err, "userbot start")
			}
			return nil
		},
	}
}
