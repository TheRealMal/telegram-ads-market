package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	evententity "ads-mrkt/internal/event/domain/entity"
	"ads-mrkt/internal/market/domain/entity"
	marketerrors "ads-mrkt/internal/market/domain/errors"
)

const completedWorkerInterval = 30 * time.Second

// CreateDealFromRequest validates all inputs and creates a deal.
// Returns the created deal and the listing type (needed by the caller for forum topic setup).
func (s *dealService) CreateDealFromRequest(ctx context.Context, userID int64, input CreateDealInput) (*entity.Deal, entity.ListingType, error) {
	creator, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	if creator == nil || !creator.HasWallet() {
		return nil, "", marketerrors.ErrWalletRequired
	}

	listing, err := s.listingRepo.GetListingByID(ctx, input.ListingID)
	if err != nil {
		return nil, "", err
	}
	if listing == nil {
		return nil, "", marketerrors.ErrNotFound
	}
	if listing.UserID == userID {
		return nil, "", marketerrors.ErrOwnListing
	}
	if !entity.DealPriceMatchesListing(listing.Prices, input.Type, input.Duration, entity.TONToNanoton(input.PriceTON)) {
		return nil, "", marketerrors.ErrPriceMismatch
	}

	// Fetch listing owner (the other party). Creator is already fetched — reuse it.
	listingOwner, err := s.userRepo.GetUserByID(ctx, listing.UserID)
	if err != nil {
		return nil, "", err
	}
	if listingOwner == nil {
		return nil, "", marketerrors.ErrNotFound
	}

	var lessor, lessee *entity.User
	var dealChannelID *int64
	switch listing.Type {
	case entity.ListingTypeLessor:
		lessor, lessee = listingOwner, creator
		dealChannelID = listing.ChannelID
	case entity.ListingTypeLessee:
		lessor, lessee = creator, listingOwner
		if input.ChannelID == nil {
			return nil, "", marketerrors.ErrInvalidListingType
		}
		dealChannelID = input.ChannelID
	default:
		return nil, "", marketerrors.ErrInvalidListingType
	}

	canonDetails := json.RawMessage("{}")
	isInstantPost := entity.AdType(input.Type) == entity.AdTypeInstantPost
	if isInstantPost {
		canonDetails, err = entity.ValidateDealDetails(listing.PreparedPost)
		if err != nil {
			return nil, "", fmt.Errorf("%w: %s", marketerrors.ErrInvalidDealDetails, err.Error())
		}
	} else if len(input.Details) > 0 {
		canonDetails, err = entity.ValidateDealDetails(input.Details)
		if err != nil {
			return nil, "", fmt.Errorf("%w: %s", marketerrors.ErrInvalidDealDetails, err.Error())
		}
	}

	d := &entity.Deal{
		ListingID:           input.ListingID,
		LessorID:            lessor.ID,
		LesseeID:            lessee.ID,
		ChannelID:           dealChannelID,
		Type:                entity.AdType(input.Type),
		Duration:            input.Duration,
		Price:               entity.TONToNanoton(input.PriceTON),
		Details:             canonDetails,
		Message:             input.Message,
		LessorPayoutAddress: lessor.WalletAddress,
		LesseePayoutAddress: lessee.WalletAddress,
	}

	if d.IsInstantPost() {
		if err := s.prepareInstantPost(d, lessor, lessee); err != nil {
			return nil, "", err
		}
	}

	if d.Status != entity.DealStatusApproved {
		d.Status = entity.DealStatusDraft
	}
	d.EscrowAmount = s.escrowSvc.ComputeEscrowAmount(d.Price)
	if err := s.dealRepo.CreateDeal(ctx, d); err != nil {
		return nil, "", err
	}
	return d, listing.Type, nil
}

// prepareInstantPost validates wallets and sets status and signatures for an instant_post deal.
// Payout addresses are already set on d from the user profiles by the caller.
func (s *dealService) prepareInstantPost(d *entity.Deal, lessor, lessee *entity.User) error {
	if !lessor.HasWallet() {
		return marketerrors.ErrInstantPostWalletRequired
	}
	if !lessee.HasWallet() {
		return marketerrors.ErrInstantPostWalletRequired
	}

	d.Status = entity.DealStatusApproved

	lessorSig := entity.ComputeDealSignature(string(d.Type), d.Duration, d.Price, d.Details, lessor.ID, *d.LessorPayoutAddress, *d.LesseePayoutAddress)
	lesseeSig := entity.ComputeDealSignature(string(d.Type), d.Duration, d.Price, d.Details, lessee.ID, *d.LessorPayoutAddress, *d.LesseePayoutAddress)
	d.LessorSignature = &lessorSig
	d.LesseeSignature = &lesseeSig
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
// Caller must be lessor or lessee. Returns the updated deal.
func (s *dealService) UpdateDealDraft(ctx context.Context, userID int64, dealID int64, input UpdateDealDraftInput) (*entity.Deal, error) {
	existing, err := s.dealRepo.GetDealByID(ctx, dealID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, marketerrors.ErrNotFound
	}
	if !existing.CanBeEdited() {
		return nil, marketerrors.ErrDealNotDraft
	}
	if userID != existing.LessorID && userID != existing.LesseeID {
		return nil, marketerrors.ErrUnauthorizedSide
	}

	d := *existing
	d.ID = dealID
	if input.Type != nil {
		d.Type = entity.AdType(*input.Type)
	}
	if input.Duration != nil {
		d.Duration = *input.Duration
	}
	if input.PriceTON != nil {
		d.Price = entity.TONToNanoton(*input.PriceTON)
	}
	if len(input.Details) > 0 {
		canonDetails, err := entity.ValidateDealDetails(input.Details)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", marketerrors.ErrInvalidDealDetails, err.Error())
		}
		d.Details = canonDetails
	}

	if input.Type != nil || input.Duration != nil || input.PriceTON != nil {
		listing, err := s.listingRepo.GetListingByID(ctx, existing.ListingID)
		if err != nil {
			return nil, err
		}
		if listing != nil && !entity.DealPriceMatchesListing(listing.Prices, string(d.Type), d.Duration, d.Price) {
			return nil, marketerrors.ErrPriceMismatch
		}
	}

	d.EscrowAmount = s.escrowSvc.ComputeEscrowAmount(d.Price)
	return s.dealRepo.UpdateDealDraftFieldsAndClearSignatures(ctx, &d)
}

// SignDeal sets the current user's signature on the deal. Delegates to signerSvc. Returns the updated deal.
func (s *dealService) SignDeal(ctx context.Context, userID int64, dealID int64) (*entity.Deal, error) {
	return s.signerSvc.SignDeal(ctx, userID, dealID)
}

// SetDealPayoutAddress sets the current user's payout address on the deal (lessor or lessee). Only in draft. Returns the updated deal.
func (s *dealService) SetDealPayoutAddress(ctx context.Context, userID int64, dealID int64, payoutAddressRaw string) (*entity.Deal, error) {
	existing, err := s.dealRepo.GetDealByID(ctx, dealID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, marketerrors.ErrNotFound
	}
	if !existing.CanBeEdited() {
		return nil, marketerrors.ErrDealNotDraft
	}
	if userID != existing.LessorID && userID != existing.LesseeID {
		return nil, marketerrors.ErrUnauthorizedSide
	}
	if payoutAddressRaw == "" {
		return nil, marketerrors.ErrWalletNotSet
	}
	return s.dealRepo.SetDealPayoutAddress(ctx, dealID, userID, payoutAddressRaw)
}

// RejectDeal sets deal status to rejected. Only allowed when status is draft; caller must be lessor or lessee. Returns the updated deal.
func (s *dealService) RejectDeal(ctx context.Context, userID int64, dealID int64) (*entity.Deal, error) {
	existing, err := s.dealRepo.GetDealByID(ctx, dealID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, marketerrors.ErrNotFound
	}
	if userID != existing.LessorID && userID != existing.LesseeID {
		return nil, marketerrors.ErrUnauthorizedSide
	}
	d, err := s.dealRepo.SetDealStatusRejected(ctx, dealID)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, marketerrors.ErrDealNotDraft
	}
	s.dealChatSvc.UpdateDealForumTopicEmoji(ctx, dealID, entity.DealStatusRejected)
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

	return d, nil
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
		s.dealChatSvc.UpdateDealForumTopicEmoji(ctx, d.ID, entity.DealStatusExpired)
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
				s.dealChatSvc.UpdateDealForumTopicEmoji(ctx, d.ID, entity.DealStatusCompleted)
				logger.Info("deal set completed", "deal_id", d.ID)
			}
		}
	}
}
