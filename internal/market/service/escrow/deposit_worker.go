package escrow

import (
	"context"
	"log/slog"
	"strings"
	"time"

	evententity "ads-mrkt/internal/event/domain/entity"

	"github.com/redis/go-redis/v9"
)

const (
	escrowDepositStream       = "events:escrow_deposit"
	escrowDepositGroup        = "market"
	escrowDepositConsumer     = "escrow-deposit"
	escrowDepositReadCount    = 50
	escrowDepositPollInterval = 2 * time.Second
)

// DepositStreamReader is used to read and ack escrow deposit events from Redis stream.
type DepositStreamReader interface {
	CreateGroup(ctx context.Context, stream, group, id string) error
	ReadEvents(ctx context.Context, args *redis.XReadGroupArgs) ([]redis.XMessage, error)
	AckMessages(ctx context.Context, stream, group string, messageIDs []string) error
}

// DepositStreamWorker reads events:escrow_deposit stream and sets deal status to escrow_deposit_confirmed when amount >= deal.Price.
func (s *service) DepositStreamWorker(ctx context.Context, stream DepositStreamReader) {
	if stream == nil {
		return
	}
	if err := stream.CreateGroup(ctx, escrowDepositStream, escrowDepositGroup, "0"); err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		slog.Error("escrow deposit worker: create group", "error", err)
		return
	}
	ticker := time.NewTicker(escrowDepositPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.processDepositEvents(ctx, stream)
		}
	}
}

func (s *service) processDepositEvents(ctx context.Context, stream DepositStreamReader) {
	msgs, err := stream.ReadEvents(ctx, &redis.XReadGroupArgs{
		Group:    escrowDepositGroup,
		Consumer: escrowDepositConsumer,
		Streams:  []string{escrowDepositStream, ">"},
		Count:    escrowDepositReadCount,
	})
	if err != nil || len(msgs) == 0 {
		return
	}
	var ids []string
	for i := range msgs {
		msg := &msgs[i]
		ev := &evententity.EventEscrowDeposit{}
		ev.FromMap(msg.Values)
		deal, err := s.marketRepository.GetDealByEscrowAddress(ctx, ev.Address)
		if err != nil {
			slog.Error("escrow deposit worker: get deal", "address", ev.Address, "error", err)
			ids = append(ids, msg.ID)
			continue
		}
		if deal == nil {
			ids = append(ids, msg.ID)
			continue
		}
		if ev.Amount < deal.EscrowAmount {
			slog.Info("escrow deposit worker: amount too low", "deal_id", deal.ID, "address", ev.Address, "amount", ev.Amount, "escrow_amount", deal.EscrowAmount)
			ids = append(ids, msg.ID)
			continue
		}
		if err := s.marketRepository.SetDealStatusEscrowDepositConfirmed(ctx, deal.ID); err != nil {
			slog.Error("escrow deposit worker: set status", "deal_id", deal.ID, "error", err)
			continue
		}
		slog.Info("escrow deposit confirmed", "deal_id", deal.ID, "address", ev.Address)
		ids = append(ids, msg.ID)
	}
	if len(ids) > 0 {
		_ = stream.AckMessages(ctx, escrowDepositStream, escrowDepositGroup, ids)
	}
}
