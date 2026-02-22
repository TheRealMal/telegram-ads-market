package escrow

import (
	"context"
	"log/slog"
	"time"

	evententity "ads-mrkt/internal/event/domain/entity"
)

const (
	escrowDepositGroup        = "market"
	escrowDepositConsumer     = "escrow-deposit"
	escrowDepositReadCount    = 50
	escrowDepositPollInterval = 2 * time.Second
)

type escrowDepositEventService interface {
	ReadEscrowDepositEvents(ctx context.Context, group, consumer string, limit int64) ([]*evententity.EventEscrowDeposit, error)
	AckEscrowDepositMessages(ctx context.Context, group string, messageIDs []string) error
}

// DepositStreamWorker reads escrow_deposit events and sets deal status to escrow_deposit_confirmed when amount >= deal.EscrowAmount.
func (s *service) DepositStreamWorker(ctx context.Context, eventService escrowDepositEventService) {
	logger := slog.With("component", "escrow_deposit_worker")
	ticker := time.NewTicker(escrowDepositPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.processDepositEvents(ctx, eventService, logger)
		}
	}
}

func (s *service) processDepositEvents(ctx context.Context, eventService escrowDepositEventService, logger *slog.Logger) {
	events, err := eventService.ReadEscrowDepositEvents(ctx, escrowDepositGroup, escrowDepositConsumer, escrowDepositReadCount)
	if err != nil || len(events) == 0 {
		return
	}
	var ids []string
	for _, ev := range events {
		deal, err := s.dealRepo.GetDealByEscrowAddress(ctx, ev.Address)
		if err != nil {
			logger.Error("get deal", "address", ev.Address, "error", err)
			ids = append(ids, ev.ID)
			continue
		}
		if deal == nil {
			ids = append(ids, ev.ID)
			continue
		}
		if ev.Amount < deal.EscrowAmount {
			logger.Info("amount too low", "deal_id", deal.ID, "address", ev.Address, "amount", ev.Amount, "escrow_amount", deal.EscrowAmount)
			ids = append(ids, ev.ID)
			continue
		}
		if err := s.dealRepo.SetDealStatusEscrowDepositConfirmed(ctx, deal.ID); err != nil {
			logger.Error("set status", "deal_id", deal.ID, "error", err)
			continue
		}
		if deal.EscrowAddress != nil && *deal.EscrowAddress != "" {
			_ = s.redis.Del(ctx, *deal.EscrowAddress)
		}
		logger.Info("escrow deposit confirmed", "deal_id", deal.ID, "address", ev.Address)
		ids = append(ids, ev.ID)
	}
	if len(ids) > 0 {
		_ = eventService.AckEscrowDepositMessages(ctx, escrowDepositGroup, ids)
	}
}
