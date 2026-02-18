package blockchain_observer

import (
	"context"

	"ads-mrkt/internal/blockchain_observer"
	"ads-mrkt/internal/config"
	escrowdepositevent "ads-mrkt/internal/event/application/escrow_deposit/event"
	eventredis "ads-mrkt/internal/event/repository/redis"
	"ads-mrkt/internal/liteclient"
	marketrepo "ads-mrkt/internal/market/repository/market"
	"ads-mrkt/internal/postgres"
	"ads-mrkt/internal/redis"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func Cmd(ctx context.Context, conf *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "blockchain_observer",
		Short: "run blockchain observer (escrow wallet TTL + deposit stream)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctxRun, cancel := context.WithCancel(ctx)
			defer cancel()

			pg, err := postgres.New(ctxRun, conf.Database)
			if err != nil {
				return errors.Wrap(err, "postgres")
			}

			redisClient, err := redis.New(ctxRun, conf.Redis)
			if err != nil {
				return errors.Wrap(err, "redis")
			}
			defer redisClient.Close()

			lc, err := liteclient.NewClient(ctxRun, conf.Liteclient, conf.IsTestnet, conf.IsPublic)
			if err != nil {
				return errors.Wrap(err, "liteclient")
			}

			dealRepo := marketrepo.New(pg)
			eventRepo := eventredis.New(redisClient)
			escrowDepositEventSvc := escrowdepositevent.NewService(eventRepo)
			obs := blockchain_observer.New(lc, redisClient.Client(), dealRepo, escrowDepositEventSvc, conf.Redis.DB)

			go obs.Start(ctxRun)

			<-ctxRun.Done()
			return nil
		},
	}
}
