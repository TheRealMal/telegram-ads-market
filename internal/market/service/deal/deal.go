package service

import (
	"context"
	"log/slog"
	"time"

	evententity "ads-mrkt/internal/event/domain/entity"
	"ads-mrkt/internal/market/domain/entity"
	marketerrors "ads-mrkt/internal/market/domain/errors"
)

const completedWorkerInterval = 30 * time.Second

func (s *dealService) CreateDeal(ctx context.Context, d *entity.Deal, otherSideID int64) error {
	if d.Status != entity.DealStatusApproved {
		d.Status = entity.DealStatusDraft
	}
	d.EscrowAmount = s.escrowSvc.ComputeEscrowAmount(d.Price)
	if err := s.dealRepo.CreateDeal(ctx, d); err != nil {
		return err
	}
	return nil
}

func (s *dealService) GetDeal(ctx context.Context, id int64) (*entity.Deal, error) {
	return s.dealRepo.GetDealByID(ctx, id)
}

// GetDealForUser returns the deal only if the user is lessor or lessee; otherwise nil.
func (s *dealService) GetDealForUser(ctx context.Context, id int64, userID int64) (*entity.Deal, error) {
	d, err := s.dealRepo.GetDealByID(ctx, id)
	if err != nil || d == nil {
		return nil, err
	}
	if d.LessorID != userID && d.LesseeID != userID {
		return nil, nil
	}
	return d, nil
}

func (s *dealService) GetDealsByListingID(ctx context.Context, listingID int64) ([]*entity.Deal, error) {
	return s.dealRepo.GetDealsByListingID(ctx, listingID)
}

// GetDealsByListingIDForUser returns deals for the listing where the user is lessor or lessee.
func (s *dealService) GetDealsByListingIDForUser(ctx context.Context, listingID int64, userID int64) ([]*entity.Deal, error) {
	return s.dealRepo.GetDealsByListingIDForUser(ctx, listingID, userID)
}

func (s *dealService) GetDealsByUserID(ctx context.Context, userID int64) ([]*entity.Deal, error) {
	return s.dealRepo.ListDealsByUserID(ctx, userID)
}

// UpdateDealDraft updates type, duration, price, details when status is draft. Clears both signatures.
// Caller must be lessor or lessee.
func (s *dealService) UpdateDealDraft(ctx context.Context, userID int64, d *entity.Deal) error {
	existing, err := s.dealRepo.GetDealByID(ctx, d.ID)
	if err != nil || existing == nil {
		return marketerrors.ErrNotFound
	}
	if existing.Status != entity.DealStatusDraft {
		return marketerrors.ErrDealNotDraft
	}
	if userID != existing.LessorID && userID != existing.LesseeID {
		return marketerrors.ErrUnauthorizedSide
	}
	d.LessorID = existing.LessorID
	d.LesseeID = existing.LesseeID
	d.ListingID = existing.ListingID
	d.Status = entity.DealStatusDraft
	d.EscrowAmount = s.escrowSvc.ComputeEscrowAmount(d.Price)
	return s.dealRepo.UpdateDealDraftFieldsAndClearSignatures(ctx, d)
}

// SignDeal sets the current user's signature on the deal. Delegates to signerSvc.
func (s *dealService) SignDeal(ctx context.Context, userID int64, dealID int64) error {
	return s.signerSvc.SignDeal(ctx, userID, dealID)
}

// SetDealPayoutAddress sets the current user's payout address on the deal (lessor or lessee). Only in draft.
func (s *dealService) SetDealPayoutAddress(ctx context.Context, userID int64, dealID int64, payoutAddressRaw string) error {
	existing, err := s.dealRepo.GetDealByID(ctx, dealID)
	if err != nil || existing == nil {
		return marketerrors.ErrNotFound
	}
	if existing.Status != entity.DealStatusDraft {
		return marketerrors.ErrDealNotDraft
	}
	if userID != existing.LessorID && userID != existing.LesseeID {
		return marketerrors.ErrUnauthorizedSide
	}
	if payoutAddressRaw == "" {
		return marketerrors.ErrWalletNotSet
	}
	return s.dealRepo.SetDealPayoutAddress(ctx, dealID, userID, payoutAddressRaw)
}

// RejectDeal sets deal status to rejected. Only allowed when status is draft; caller must be lessor or lessee.
func (s *dealService) RejectDeal(ctx context.Context, userID int64, dealID int64) error {
	existing, err := s.dealRepo.GetDealByID(ctx, dealID)
	if err != nil || existing == nil {
		return marketerrors.ErrNotFound
	}
	if userID != existing.LessorID && userID != existing.LesseeID {
		return marketerrors.ErrUnauthorizedSide
	}
	updated, err := s.dealRepo.SetDealStatusRejected(ctx, dealID)
	if err != nil {
		return err
	}
	if !updated {
		return marketerrors.ErrDealNotDraft
	}
	otherID := existing.LesseeID
	if userID == existing.LesseeID {
		otherID = existing.LessorID
	}
	if err := s.notificationAdder.AddTelegramNotificationEvent(
		ctx,
		&evententity.EventTelegramNotification{
			ChatID:  otherID,
			Message: "Deal was rejected",
		},
	); err != nil {
		slog.Error("failed to add notification", "type", "deal_rejected", "deal_id", dealID, "chat_id", otherID, "error", err)

	}

	return nil
}

// ExpireTimedOutDeposits marks deals in waiting_escrow_deposit with updated_at before the given time as expired (e.g. on startup: olderThan = now - 1h).
func (s *dealService) ExpireTimedOutDeposits(ctx context.Context, olderThan time.Time) error {
	deals, err := s.dealRepo.ListDealsWaitingEscrowDepositOlderThan(ctx, olderThan)
	if err != nil {
		return err
	}
	for _, d := range deals {
		slog.Info("expiring timed-out deposit deal", "deal_id", d.ID, "updated_at", d.UpdatedAt)
		if err = s.dealRepo.SetDealStatusExpiredByDealID(ctx, d.ID); err != nil {
			slog.Error("set deal status expired", "deal_id", d.ID, "error", err)
			continue
		}
	}
	return nil
}

// RunCompletedWorker moves deals from escrow_release_confirmed / escrow_refund_confirmed to completed (final status for frontend). Run in a goroutine.
func (s *dealService) RunCompletedWorker(ctx context.Context) {
	logger := slog.With("component", "deal_completed_worker")
	ticker := time.NewTicker(completedWorkerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			deals, err := s.dealRepo.ListDealsEscrowConfirmedToComplete(ctx)
			if err != nil {
				logger.Error("list deals to complete", "error", err)
				continue
			}
			for _, d := range deals {
				if err := s.dealRepo.SetDealStatusCompleted(ctx, d.ID); err != nil {
					logger.Error("set deal completed", "deal_id", d.ID, "error", err)
					continue
				}
				logger.Info("deal set completed", "deal_id", d.ID)
			}
		}
	}
}
