package blockchain_observer

import (
	"context"
	"fmt"

	"ads-mrkt/cmd/builder"
	"ads-mrkt/internal/blockchain_observer"
	"ads-mrkt/internal/config"
	escrowdepositevent "ads-mrkt/internal/event/application/escrow_deposit/event"
	eventredis "ads-mrkt/internal/event/repository/redis"
	"ads-mrkt/internal/liteclient"
	"ads-mrkt/internal/market/repository/deal"
	"ads-mrkt/internal/postgres"
	"ads-mrkt/internal/redis"

	"github.com/spf13/cobra"
)

func Cmd(ctx context.Context, conf *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blockchain_observer",
		Short: "blockchain observer commands",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Usage()
		},
	}

	cmd.AddCommand(runCmd(ctx, conf))

	return cmd
}

func runCmd(ctx context.Context, conf *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "run blockchain observer (escrow wallet TTL + deposit stream)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctxRun, cancel := context.WithCancel(ctx)
			shutdownSrv := builder.NewShutdown()
			go func() {
				shutdownSrv.WaitShutdown(ctxRun)
				cancel()
			}()

			pg, err := postgres.New(ctxRun, conf.Database)
			if err != nil {
				return fmt.Errorf("postgres: %w", err)
			}

			redisClient, err := redis.New(ctxRun, conf.Redis)
			if err != nil {
				return fmt.Errorf("redis: %w", err)
			}
			defer redisClient.Close()

			lc, err := liteclient.NewClient(ctxRun, conf.Liteclient, conf.IsTestnet, conf.IsPublic)
			if err != nil {
				return fmt.Errorf("liteclient: %w", err)
			}

			dealRepo := deal.New(pg)
			eventRepo := eventredis.New(redisClient)
			escrowDepositEventSvc, err := escrowdepositevent.NewService(ctxRun, eventRepo)
			if err != nil {
				return fmt.Errorf("create escrow deposit event service: %w", err)
			}
			obs := blockchain_observer.New(lc, redisClient.Client(), dealRepo, escrowDepositEventSvc, conf.Redis.DB)

			go func() {
				if err := obs.Start(ctxRun); err != nil {
					shutdownSrv.MustShutdown(ctxRun, "blockchain_observer", err)
				}
			}()

			<-ctxRun.Done()
			return nil
		},
	}
}
